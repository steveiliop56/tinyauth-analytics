package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"tinyauth-analytics/database/queries"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

type Config struct {
	Port               int      `mapstructure:"port"`
	Address            string   `mapstructure:"address"`
	RateLimitCount     int      `mapstructure:"rate_limit_count"`
	DatabasePath       string   `mapstructure:"database_path"`
	TrustedProxies     []string `mapstructure:"trusted_proxies"`
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
}

func main() {
	v := viper.New()

	v.SetDefault("port", 8080)
	v.SetDefault("address", "0.0.0.0")
	v.SetDefault("rate_limit_count", 3)
	v.SetDefault("database_path", "analytics.db")
	v.SetDefault("trusted_proxies", []string{""})
	v.SetDefault("cors_allowed_origins", []string{"*"})

	v.AutomaticEnv()

	var config Config

	err := v.Unmarshal(&config)

	if err != nil {
		slog.Error("failed to parse configuration: ", "error", err)
		os.Exit(1)
	}

	slog.Info("starting tinyauth analytics", "config", config)

	sqlDb, err := sql.Open("sqlite", config.DatabasePath)

	if err != nil {
		slog.Error("failed to open database: ", "error", err)
		os.Exit(1)
	}

	defer sqlDb.Close()

	sqlDb.Exec(`CREATE TABLE IF NOT EXISTS "instances" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"uuid" TEXT NOT NULL,
		"version" TEXT NOT NULL,
		"last_seen" INTEGER NOT NULL
	);`)

	queries := queries.New(sqlDb)
	cache := NewCache()
	router := chi.NewRouter()

	rateLimiter := NewRateLimiter(RateLimitConfig{
		RateLimitCount: config.RateLimitCount,
		TrustedProxies: config.TrustedProxies,
	}, cache)

	instancesHandler := NewInstancesHandler(queries)
	healthHandler := NewHealthHandler()

	router.Get("/v1/healthz", healthHandler.health)
	router.Get("/v1/instances/all", instancesHandler.GetInstances)

	router.Group(func(r chi.Router) {
		r.Use(rateLimiter.limit)
		r.Post("/v1/instances/heartbeat", instancesHandler.Heartbeat)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Address, config.Port),
		Handler: router,
	}

	go cleanUpOldInstances(queries)

	slog.Info("server listening", "address", srv.Addr)

	err = srv.ListenAndServe()

	if err != nil {
		slog.Error("server error: ", "error", err)
		os.Exit(1)
	}
}

func cleanUpOldInstances(queries *queries.Queries) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		slog.Info("cleaning up old instances")

		cutoffTime := time.Now().Add(-48 * time.Hour).UnixMilli()
		rowsAffected, err := queries.DeleteOldInstances(context.Background(), cutoffTime)

		if err != nil {
			slog.Error("failed to clean up old instances: ", "error", err)
			continue
		}

		slog.Info("old instances cleaned up", "rows_affected", rowsAffected)
	}

}
