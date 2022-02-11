package main

import (
	"context"
	"net/http"
	"net/http/httputil"

	"github.com/rs/zerolog/log"
)

// initHandleFunc serves the first request from a client,
// starts a chrome instance on a random port from the pool between PortIntervalStart and PortIntervalEnd
// and sends data to the client to create a connection with chrome
func initHandleFunc(limiterChan chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		<-limiterChan
		defer func() {
			limiterChan <- struct{}{}
		}()

		ctx, cancel := context.WithTimeout(r.Context(), chromeTimeout)
		defer cancel()

		opts := RunChromeOpts{
			Port:           generateRandomPort(),
			Proxy:          r.URL.Query().Get("proxy"),
			UserAgent:      r.URL.Query().Get("ua"),
			DownloadImages: r.URL.Query().Get("images"),
		}

		chrome, err := RunChrome(ctx, opts)
		if err != nil {
			log.Error().Err(err).Msg("fail to run chrome")
			return
		}

		StoreChromeConnection(chrome)

		proxyServer := httputil.NewSingleHostReverseProxy(chrome.URL)

		proxyServer.ServeHTTP(w, r)

		log.Debug().
			Str("chromeID", chrome.ID).
			Int("port", chrome.Port).
			Str("proxy", chrome.Proxy).
			Msg("chrome started")
	}
}

// connProxyHandleFunc establishes a client-browser connection and controls it.
// Kills the chrome after the connection ends
func connProxyHandleFunc(limiterChan chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lock.RLock()
		chrome := Connections[r.RequestURI]
		lock.RUnlock()

		logger := log.With().
			Str("chromeID", chrome.ID).
			Int("port", chrome.Port).
			Str("proxy", chrome.Proxy).Logger()

		proxyServer := httputil.NewSingleHostReverseProxy(chrome.URL)

		logger.Debug().Msg("client connected")

		proxyServer.ServeHTTP(w, r)

		logger.Debug().Msg("client disconnected")

		if err := KillChrome(chrome); err != nil {
			logger.Error().Err(err).Msg("fail to kill chrome")
			return
		}

		RemoveChromeConnection(chrome)

		limiterChan <- struct{}{}

		logger.Debug().Msg("chrome killed")
	}
}
