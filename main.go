package main

import (
	"chromebalancer/clientsstore"
	"chromebalancer/utils"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

// TODO Modify API for more user-friendly
// TODO Add Swagger

var ClientsStore clientsstore.ClientsStore

func main() {
	port, err := strconv.Atoi(utils.GetEnv("PORT", "9222"))
	if err != nil {
		log.Fatal().Err(err).Msg("fail to parse port")
	}

	maxChromes, err := strconv.Atoi(utils.GetEnv("MAX_CHROMES_NUM", "16"))
	if err != nil {
		log.Fatal().Err(err).Msg("fail to parse max chromes num")
	}

	rand.Seed(time.Now().UnixNano())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// for limit the number of running chrome instances
	limiterChan := make(chan struct{}, maxChromes)

	for i := 0; i < maxChromes; i++ {
		limiterChan <- struct{}{}
	}

	ClientsStore = clientsstore.NewStore()

	go ClientsStore.CheckExpiredChromes(limiterChan)

	router := chi.NewRouter()
	router.Handle("/json/version", initHandleFunc(limiterChan))
	router.Handle("/devtools/browser/*", connProxyHandleFunc(limiterChan))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Send()
		}
	}()

	log.Debug().Str("addr", srv.Addr).Int("max chromes num", maxChromes).Msg("server started")

	<-done

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Debug().Str("addr", srv.Addr).Msg("server stopped")
}
