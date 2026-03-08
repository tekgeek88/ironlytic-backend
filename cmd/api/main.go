package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tekgeek88/ironlytic-backend/internal/config"
	httpserver "github.com/tekgeek88/ironlytic-backend/internal/http"
	"github.com/tekgeek88/ironlytic-backend/internal/platform/db"
	apperlog "github.com/tekgeek88/ironlytic-backend/internal/platform/logger"
)

func main() {

	env := os.Getenv("ENV")
	if env != "staging" && env != "production" {
		env = "development"
		_ = godotenv.Load(".env." + env)
		log.Println("Loaded local env file for", env)
	}

	cfg := config.Load()

	logger := apperlog.New(cfg.AppEnv)
	logger.Info("starting API server",
		"env", cfg.AppEnv,
		"addr", cfg.AppAddr,
		"db_host", cfg.DBHost,
		"db_name", cfg.DBName,
	)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()

	postgres, err := db.NewPostgres(startupCtx, cfg.PostgresDSN())
	if err != nil {
		logger.Error("failed to initialize postgres", "error", err)
		os.Exit(1)
	}
	defer postgres.Close()

	router := httpserver.NewRouter(cfg, logger, postgres)

	srv := &http.Server{
		Addr:              cfg.AppAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			log.Fatal(err)
		}
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-stopCtx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return
	}

	logger.Info("server stopped cleanly")
}

func SayHello() string {
	return "Hello, World!"
}
