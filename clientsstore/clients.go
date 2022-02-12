package clientsstore

import (
	"chromesdaddy/chrome"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	PortIntervalStart = 7500
	PortIntervalEnd   = 7756
)

type ClientsStore struct {
	sync.RWMutex
	chromes   map[string]chrome.Chrome
	busyPorts map[int]bool
}

func NewStore() ClientsStore {
	return ClientsStore{
		RWMutex:   sync.RWMutex{},
		chromes:   make(map[string]chrome.Chrome),
		busyPorts: make(map[int]bool),
	}
}

func (cs *ClientsStore) Get(chromeURL string) chrome.Chrome {
	cs.Lock()
	defer cs.Unlock()

	return cs.chromes[chromeURL]
}

func (cs *ClientsStore) Put(chrome chrome.Chrome) {
	cs.Lock()
	defer cs.Unlock()

	cs.chromes[chrome.ID] = chrome
	cs.busyPorts[chrome.Port] = true
}

func (cs *ClientsStore) Del(chrome chrome.Chrome) {
	cs.Lock()
	defer cs.Unlock()

	delete(cs.chromes, chrome.ID)

	cs.busyPorts[chrome.Port] = false
}

// CheckExpiredChromes periodically checks running chromes, and removes from the list if it finds a killed chrome
// In case, for some reason, the balancer launched the chrome, but the client does not use it
func (cs *ClientsStore) CheckExpiredChromes(limiter chan<- struct{}) {
	ticker := time.NewTicker(10 * time.Second)

	for range ticker.C {
		for _, c := range cs.chromes {
			if c.Ctx.Err() != nil {
				log.Warn().
					Str("chromeID", c.ID).
					Int("port", c.Port).
					Str("proxy", c.Proxy).
					Msg("detect expired chrome")

				if err := chrome.Kill(c); err != nil {
					log.Error().Err(err).Msg("fail to kill expired chrome")
				}

				cs.Del(c)
				limiter <- struct{}{}

				log.Warn().
					Str("chromeID", c.ID).
					Int("port", c.Port).
					Str("proxy", c.Proxy).
					Msg("expired chrome successfully deleted")
			}
		}
	}
}
