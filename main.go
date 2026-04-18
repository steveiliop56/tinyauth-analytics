package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tinyauthapp/analytics/queries"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"
)

var version = "development"

type Config struct {
	Port               int      `mapstructure:"port"`
	Address            string   `mapstructure:"address"`
	RateLimitCount     int      `mapstructure:"rate_limit_count"`
	DatabasePath       string   `mapstructure:"database_path"`
	TrustedProxies     []string `mapstructure:"trusted_proxies"`
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
	DashboardEnabled   bool     `mapstructure:"dashboard_enabled"`
}

func main() {
	v := viper.New()

	v.SetDefault("port", 8080)
	v.SetDefault("address", "0.0.0.0")
	v.SetDefault("rate_limit_count", 3)
	v.SetDefault("database_path", "analytics.db")
	v.SetDefault("trusted_proxies", []string{""})
	v.SetDefault("cors_allowed_origins", []string{"*"})
	v.SetDefault("dashboard_enabled", true)

	v.AutomaticEnv()

	var config Config

	err := v.Unmarshal(&config)

	if err != nil {
		slog.Error("failed to parse configuration: ", "error", err)
		os.Exit(1)
	}

	slog.Info("starting tinyauth analytics", "version", version, "config", config)

	sqlDb, err := sql.Open("sqlite", config.DatabasePath)

	if err != nil {
		slog.Error("failed to open database: ", "error", err)
		os.Exit(1)
	}

	defer sqlDb.Close()

	sqlDb.Exec(`PRAGMA journal_mode=WAL;`)

	sqlDb.Exec(`CREATE TABLE IF NOT EXISTS "instances" (
		"uuid" TEXT NOT NULL PRIMARY KEY,
		"version" TEXT NOT NULL,
		"last_seen" INTEGER NOT NULL
	);`)

	queries := queries.New(sqlDb)
	cache := NewCache()
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	rateLimiter := NewRateLimiter(RateLimitConfig{
		RateLimitCount: config.RateLimitCount,
		TrustedProxies: config.TrustedProxies,
	}, cache)

	instancesHandler := NewInstancesHandler(queries)
	healthHandler := NewHealthHandler()
	dashboardHandler := NewDashboardHandler(queries)

	router.Get("/v1/healthz", healthHandler.Health)

	router.Group(func(r chi.Router) {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: config.CORSAllowedOrigins,
		}))
		r.Get("/v1/instances/all", instancesHandler.GetInstances)
	})

	router.Group(func(r chi.Router) {
		r.Use(rateLimiter.limit)
		r.Post("/v1/instances/heartbeat", instancesHandler.Heartbeat)
	})

	if config.DashboardEnabled {
		router.Get("/dashboard", dashboardHandler.Dashboard)
	}

	router.Get("/favicon.txt", dashboardHandler.Favicon)
	router.Get("/robots.txt", dashboardHandler.Robots)

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

	for range ticker.C {
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
