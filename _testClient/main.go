package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

const chromesDaddyURL = "http://127.0.0.1:9222"

type ChromeOpts struct {
	Proxy          string
	UserAgent      string
	DownloadImages string
}

func doInitReqToChrome(remoteConnector string, opts ChromeOpts) (string, error) {
	chromeURL := fmt.Sprintf("%s/json/version", remoteConnector)

	cu, err := url.Parse(chromeURL)
	if err != nil {
		return "", err
	}

	pu, err := url.Parse(opts.Proxy)
	if err != nil {
		return "", err
	}

	q := cu.Query()
	q.Set("proxy", pu.Host)
	q.Set("ua", opts.UserAgent)
	q.Set("images", "false")

	cu.RawQuery = q.Encode()

	resp, err := http.Get(cu.String())
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var result map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	val, ok := result["webSocketDebuggerUrl"]
	if !ok {
		return "", fmt.Errorf("webSocketDebuggerUrl not found in %s", chromeURL)
	}

	// i.e. ws://localhost:9222/devtools/browser/8b295fa2-3f26-4b2f-9369-3652b4d695d4
	return val.(string), nil
}

func main() {
	opts := ChromeOpts{
		Proxy:          "",
		UserAgent:      "",
		DownloadImages: "false",
	}

	connURL, err := doInitReqToChrome(chromesDaddyURL, opts)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	timeoutContext, timeoutCancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer timeoutCancel()

	remoteContext, remoteCancel := chromedp.NewRemoteAllocator(timeoutContext, connURL)
	defer remoteCancel()

	chromeContext, chromeCancel := chromedp.NewContext(remoteContext)
	defer chromeCancel()

	// first run for establish connection
	if err := chromedp.Run(chromeContext); err != nil {
		log.Fatal().Err(err).Msg("fail to connect to browser")
	}

	res := ""

	if err := chromedp.Run(
		chromeContext,
		chromedp.Navigate("https://api.myip.com"),
		chromedp.InnerHTML("body", &res),
	); err != nil {
		log.Fatal().Err(err).Msg("fail to connect to browser")
	}

	log.Debug().Str("location data", res).Msg("response success")
}
