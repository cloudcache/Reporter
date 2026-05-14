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
	appStore := store.NewMemoryStore()
	if cfg.Database.DSN != "" {
		appStore.ConfigureSQL(cfg.Database.Driver, cfg.Database.DSN)
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		if err := appStore.LoadIdentityFromSQL(ctx, cfg.Database.Driver, cfg.Database.DSN); err != nil {
			log.Warn().Err(err).Msg("database identity load skipped; using in-memory users")
		}
		if err := appStore.LoadFormLibraryFromSQL(ctx, cfg.Database.Driver, cfg.Database.DSN); err != nil {
			log.Warn().Err(err).Msg("database form library load skipped; using built-in form library")
		}
		if err := appStore.LoadFollowupConfigFromSQL(ctx, cfg.Database.Driver, cfg.Database.DSN); err != nil {
			log.Warn().Err(err).Msg("database followup config load skipped; using built-in followup config")
		}
		if err := appStore.EnsureEvaluationComplaintTables(ctx); err != nil {
			log.Warn().Err(err).Msg("database evaluation complaint tables ensure skipped")
		}
		if err := appStore.EnsurePatientGroupTables(ctx); err != nil {
			log.Warn().Err(err).Msg("database patient group tables ensure skipped")
		}
		if err := appStore.EnsureSurveyChannelTables(ctx); err != nil {
			log.Warn().Err(err).Msg("database survey channel tables ensure skipped")
		}
		cancel()
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

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server shutdown failed")
	}
}
