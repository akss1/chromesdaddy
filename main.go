package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog/log"
)

func main() {
	port, err := strconv.Atoi(getEnv("PORT", "9222"))
	if err != nil {
		log.Fatal().Err(err).Msg("fail to parse port")
	}

	maxChromes, err := strconv.Atoi(getEnv("MAX_CHROMES_NUM", "16"))
	if err != nil {
		log.Fatal().Err(err).Msg("fail to parse max chromes num")
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// for limit the number of running chrome instances
	limiterChan := make(chan bool, maxChromes)

	for i := 0; i < maxChromes; i++ {
		limiterChan <- true
	}

	go CheckCtxDeadChromes(limiterChan)

	router := chi.NewRouter()
	router.Handle("/json/version", initReqHandleFunc(limiterChan))
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

	log.Debug().Str("addr", srv.Addr).Msg("server stopped")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal().Err(err).Send()
	}
}
