package chrome

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
)

const DefaultBaseURL = "http://127.0.0.1"
const DefaultTimeout = 5 * time.Minute

type Opts struct {
	Port           int
	Proxy          string
	DownloadImages string
	UserAgent      string
}

type Chrome struct {
	ID    string
	Port  int
	Proxy string
	URL   *url.URL

	CMD *exec.Cmd
	Ctx context.Context
}

// Run runs the chrome and returns Chrome
func Run(ctx context.Context, opts Opts) (Chrome, error) {
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
		return Chrome{}, err
	}

	// this is necessary for the read operation from stdoutPipe to be non-blocking
	cmd.Stderr = cmd.Stdout

	reader := bufio.NewReader(stdoutPipe)

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("fail to start chrome")
		return Chrome{}, err
	}

	chromeID := GetChromeIDFromStdout(reader)

	// After got the chrome id from the stdout pipe, we no longer need the Stdout of Chrome.
	// We need to redirect it to dev/null, otherwise chrome will fall after some time.

	// redirect to dev/null
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := stdoutPipe.Close(); err != nil {
		log.Error().Err(err).Msg("fail to close chrome stdout pipe")
		return Chrome{}, err
	}

	if chromeID == "" {
		log.Error().Msg("fail to get chromeID")

		if err := cmd.Process.Kill(); err != nil {
			log.Error().Err(err).Msg("fail to kill chrome")
		}

		return Chrome{}, errors.New("fail to get chromeID")
	}

	u, err := url.Parse(fmt.Sprintf("%s:%d", DefaultBaseURL, opts.Port))
	if err != nil {
		log.Error().Err(err).Msg("fail to parse chrome url")

		if err := cmd.Process.Kill(); err != nil {
			log.Error().Err(err).Msg("fail to kill chrome")
		}

		return Chrome{}, err
	}

	chrome := Chrome{
		ID:    chromeID,
		Port:  opts.Port,
		Proxy: opts.Proxy,
		URL:   u,
		CMD:   cmd,
		Ctx:   ctx,
	}

	return chrome, nil
}

// Kill kills the unnecessary browser instance
func Kill(chrome Chrome) error {
	if err := chrome.CMD.Process.Kill(); err != nil {
		return err
	}

	if _, err := chrome.CMD.Process.Wait(); err != nil {
		return err
	}

	return nil
}
