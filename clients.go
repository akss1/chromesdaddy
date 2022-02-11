package main

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Clients struct {
	sync.RWMutex
	chromes   map[string]Chrome
	busyPorts map[int]bool
}

func NewClientsStore() Clients {
	return Clients{
		RWMutex:   sync.RWMutex{},
		chromes:   make(map[string]Chrome),
		busyPorts: make(map[int]bool),
	}
}

func (c *Clients) genIdlePort() int {
	p := 0

	for {
		p = RandInt(PortIntervalStart, PortIntervalEnd)

		if !c.busyPorts[p] {
			break
		}
	}

	return p
}

func (c *Clients) Get(chromeURL string) Chrome {
	c.Lock()
	defer c.Unlock()

	return c.chromes[chromeURL]
}

func (c *Clients) Put(chrome Chrome) {
	c.Lock()
	defer c.Unlock()

	c.chromes[chrome.ID] = chrome
	c.busyPorts[chrome.Port] = true
}

func (c *Clients) Del(chrome Chrome) {
	c.Lock()
	defer c.Unlock()

	delete(c.chromes, chrome.ID)

	c.busyPorts[chrome.Port] = false
}

// CheckExpiredChromes periodically checks running chromes, and removes from the list if it finds a killed chrome
// In case, for some reason, the balancer launched the chrome, but the client does not use it
func (c *Clients) CheckExpiredChromes(limiter chan<- struct{}) {
	for {
		for _, chrome := range c.chromes {
			if chrome.Ctx.Err() != nil {
				log.Warn().
					Str("chromeID", chrome.ID).
					Int("port", chrome.Port).
					Str("proxy", chrome.Proxy).
					Msg("detected killed chrome")

				if err := KillChrome(chrome); err != nil {
					log.Error().Err(err).Msg("fail to kill chrome in check ctxs routine")
				}

				c.Del(chrome)
				limiter <- struct{}{}

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
