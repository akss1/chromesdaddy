package main

import (
	"chromebalancer/chrome"
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

		ctx, cancel := context.WithTimeout(r.Context(), chrome.DefaultTimeout)
		defer cancel()

		opts := chrome.RunChromeOpts{
			Port:           ClientsStore.GenIdlePort(),
			Proxy:          r.URL.Query().Get("proxy"),
			UserAgent:      r.URL.Query().Get("ua"),
			DownloadImages: r.URL.Query().Get("images"),
		}

		c, err := chrome.Run(ctx, opts)
		if err != nil {
			log.Error().Err(err).Msg("fail to run chrome")
			return
		}

		ClientsStore.Put(c)

		proxyServer := httputil.NewSingleHostReverseProxy(c.URL)

		proxyServer.ServeHTTP(w, r)

		log.Debug().
			Str("chromeID", c.ID).
			Int("port", c.Port).
			Str("proxy", c.Proxy).
			Msg("chrome started")
	}
}

// connProxyHandleFunc establishes a client-browser connection and controls it.
// Kills the chrome after the connection ends
func connProxyHandleFunc(limiterChan chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := ClientsStore.Get(r.RequestURI)

		logger := log.With().
			Str("chromeID", c.ID).
			Int("port", c.Port).
			Str("proxy", c.Proxy).Logger()

		proxyServer := httputil.NewSingleHostReverseProxy(c.URL)

		logger.Debug().Msg("client connected")

		proxyServer.ServeHTTP(w, r)

		logger.Debug().Msg("client disconnected")

		if err := chrome.Kill(c); err != nil {
			logger.Error().Err(err).Msg("fail to kill chrome")
			return
		}

		ClientsStore.Del(c)

		limiterChan <- struct{}{}

		logger.Debug().Msg("chrome killed")
	}
}
