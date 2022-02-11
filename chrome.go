package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	PortIntervalStart = 7500
	PortIntervalEnd   = 7628
)

const chromeBaseURL = "http://localhost"
const chromeTimeout = 5 * time.Minute

type RunChromeOpts struct {
	Port           int
	Proxy          string
	DownloadImages string
	UserAgent      string
}

type ChromeInstance struct {
	ID     string
	Port   int
	Proxy  string
	URLRaw string
	URL    *url.URL

	CMD        *exec.Cmd
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

var lock = sync.RWMutex{}

// Connections contains client connections as map of chromeID to ChromeInstance
var Connections = make(map[string]ChromeInstance)

// BusyPorts contains busy ports on which chromes are running
var BusyPorts = make(map[int]bool)

// RunChrome runs the chrome and returns ChromeInstance
func RunChrome(opts RunChromeOpts) (ChromeInstance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), chromeTimeout)

	app := "/headless-shell/headless-shell"
	args := []string{
		"--headless",
		"--disable-infobars",
		"--disable-crash-reporter",
		"--disable-session-crashed-bubble",
		"--disable-setuid-sandbox",
		"--no-sandbox",
		"--remote-debugging-address=0.0.0.0",
		fmt.Sprintf("--remote-debugging-port=%d", opts.Port),
		"--enable-features=NetworkService",
		"--window-size=1920,1080",
		"--disable-gpu",
		"--dbus-stub",
		fmt.Sprintf("--proxy-server=%s", opts.Proxy),
		"--window-position=0,0",
	}

	if opts.UserAgent != "" {
		args = append(args, fmt.Sprintf("--user-agent=%s", opts.UserAgent))
	}

	if opts.DownloadImages != "true" {
		args = append(args, "--blink-settings=imagesEnabled=false")
	}

	cmd := exec.CommandContext(ctx, app, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("fail to get chrome stdout pipe")
		cancel()

		return ChromeInstance{}, err
	}

	// this is necessary for the read operation from stdoutPipe to be non-blocking
	cmd.Stderr = cmd.Stdout

	reader := bufio.NewReader(stdoutPipe)

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("fail to start chrome")
		cancel()

		return ChromeInstance{}, err
	}

	chromeID := GetChromeIDFromStdout(reader)

	// After got the chrome id from the stdout pipe, we no longer need the Stdout of Chrome.
	// We need to redirect it to dev/null, otherwise chrome will fall after some time.

	// redirect to dev/null
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := stdoutPipe.Close(); err != nil {
		log.Error().Err(err).Msg("fail to close chrome stdout pipe")
		cancel()

		return ChromeInstance{}, err
	}

	if chromeID == "" {
		log.Error().Msg("fail to get chromeID")

		if err := cmd.Process.Kill(); err != nil {
			log.Error().Err(err).Msg("fail to kill chrome")
		}

		cancel()

		return ChromeInstance{}, errors.New("fail to get chromeID")
	}

	urlRaw := fmt.Sprintf("%s:%d", chromeBaseURL, opts.Port)

	u, err := url.Parse(urlRaw)
	if err != nil {
		log.Error().Err(err).Msg("fail to parse chrome url")

		if err := cmd.Process.Kill(); err != nil {
			log.Error().Err(err).Msg("fail to kill chrome")
		}

		cancel()

		return ChromeInstance{}, err
	}

	chrome := ChromeInstance{
		ID:         chromeID,
		Port:       opts.Port,
		Proxy:      opts.Proxy,
		URLRaw:     urlRaw,
		URL:        u,
		CMD:        cmd,
		Ctx:        ctx,
		CancelFunc: cancel,
	}

	return chrome, nil
}

// KillChrome kills the unnecessary browser instance
func KillChrome(chrome ChromeInstance) error {
	if err := chrome.CMD.Process.Kill(); err != nil {
		return err
	}

	if _, err := chrome.CMD.Process.Wait(); err != nil {
		return err
	}

	chrome.CancelFunc()

	return nil
}

func GetChromeIDFromStdout(reader *bufio.Reader) string {
	ticker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ticker.C:
			log.Error().Msg("time out for read chrome's stdout")
			return ""
		default:
			str, err := reader.ReadString('\n')
			if err != nil {
				log.Error().Err(err).Msg("fail to read chrome's stdout")
				continue
			}

			id, err := extractChromeID(str)
			if err != nil {
				continue
			}

			return id
		}
	}
}

func StoreChromeConnection(chrome ChromeInstance) {
	lock.Lock()
	defer lock.Unlock()

	Connections[chrome.ID] = chrome
	BusyPorts[chrome.Port] = true
}

func RemoveChromeConnection(chrome ChromeInstance) {
	lock.Lock()
	defer lock.Unlock()

	delete(Connections, chrome.ID)

	BusyPorts[chrome.Port] = false
}

// CheckCtxDeadChromes periodically checks running chromes, and removes from the list if it finds a killed chrome
// In case, for some reason, the balancer launched the chrome, but the client does not use it
func CheckCtxDeadChromes(limiter chan<- bool) {
	for {
		for _, chrome := range Connections {
			if chrome.Ctx.Err() != nil {
				log.Warn().
					Str("chromeID", chrome.ID).
					Int("port", chrome.Port).
					Str("proxy", chrome.Proxy).
					Msg("detected killed chrome")

				if err := KillChrome(chrome); err != nil {
					log.Error().Err(err).Msg("fail to kill chrome in check ctxs routine")
				}

				RemoveChromeConnection(chrome)
				limiter <- true

				log.Warn().
					Str("chromeID", chrome.ID).
					Int("port", chrome.Port).
					Str("proxy", chrome.Proxy).
					Msg("killed chrome successfully deleted")
			}
		}

		time.Sleep(10 * time.Second)
	}
}
