package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reporter/internal/api"
	"reporter/internal/config"
	"reporter/internal/logger"
	"reporter/internal/store"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Environment, cfg.Log.Level)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var appStore *store.Store
	if cfg.Database.DSN != "" {
		var err error
		appStore, err = store.Open(ctx, cfg.Database.Driver, cfg.Database.DSN)
		if err != nil {
			log.Fatal().Err(err).Msg("database store init failed")
		}
	} else {
		appStore = store.InstallOnly()
		log.Warn().Msg("database dsn is empty; only installation endpoints are available")
	}

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           api.NewRouter(api.Dependencies{Config: cfg, Log: log, Store: appStore}),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	go func() {
		log.Info().
			Str("addr", cfg.HTTP.Addr).
			Str("env", cfg.Environment).
			Str("logLevel", cfg.Log.Level).
			Str("dbDriver", cfg.Database.Driver).
			Bool("dbConfigured", cfg.Database.DSN != "").
			Bool("redisEnabled", cfg.Redis.Enabled).
			Bool("businessConfigDB", cfg.BusinessConfigDB).
			Msg("reporter API listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server shutdown failed")
	}
}
