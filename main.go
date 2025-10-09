package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"
	"tinyauth-analytics/internal/controller"
	"tinyauth-analytics/internal/middleware"
	"tinyauth-analytics/internal/model"
	"tinyauth-analytics/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var version = "development"

type config struct {
	DatabasePath       string   `mapstructure:"db_path"`
	Port               string   `mapstructure:"port"`
	Address            string   `mapstructure:"address"`
	RateLimitCount     int      `mapstructure:"rate_limit_count"`
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
	TrustedProxies     []string `mapstructure:"trusted_proxies"`
	LogLevel           string   `mapstructure:"log_level"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(log.Writer(), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	v := viper.New()

	v.AutomaticEnv()

	v.SetDefault("db_path", "/data/analytics.db")
	v.SetDefault("port", "8080")
	v.SetDefault("address", "0.0.0.0")
	v.SetDefault("rate_limit_count", 3)
	v.SetDefault("cors_allowed_origins", "*")
	v.SetDefault("trusted_proxies", "")
	v.SetDefault("log_level", "info")

	var conf config

	if err := v.Unmarshal(&conf); err != nil {
		slog.Error("failed to parse config", "error", err)
	}

	switch conf.LogLevel {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.Error("invalid log level", "level", conf.LogLevel)
	}

	dbSvc := service.NewDatabaseService(service.DatabaseServiceConfig{
		DatabasePath: conf.DatabasePath,
	})

	if err := dbSvc.Init(); err != nil {
		slog.Error("failed to initialize database", "error", err)
	}

	db := dbSvc.GetDatabase()

	cacheSvc := service.NewCacheService()

	engine := gin.Default()
	engine.Use(gin.Recovery())

	if conf.LogLevel == "debug" || conf.LogLevel == "info" {
		engine.Use(gin.Logger())
	}

	engine.Use(cors.New(
		cors.Config{
			AllowOrigins: conf.CORSAllowedOrigins,
		},
	))

	engine.SetTrustedProxies(conf.TrustedProxies)

	api := engine.Group("/v1")

	rateLimitMiddleware := middleware.NewRateLimitMiddleware(db, cacheSvc, conf.RateLimitCount)

	instancesCtrl := controller.NewInstancesController(api, db, rateLimitMiddleware)

	instancesCtrl.SetupRoutes()

	healthCtrl := controller.NewHealthController(api)

	healthCtrl.SetupRoutes()

	go clearOldSessions(db)

	slog.Info("starting analytics server", "address", conf.Address, "port", conf.Port, "version", version)

	if err := engine.Run(conf.Address + ":" + conf.Port); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}

func clearOldSessions(db *gorm.DB) {
	ticker := time.NewTicker(time.Duration(24) * time.Hour)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		slog.Info("clearing old sessions")

		ctx := context.Background()
		cutoffTime := time.Now().Add(time.Duration(-48) * time.Hour).UnixMilli()
		rowsAffected, err := gorm.G[model.Instance](db).Where("last_seen < ?", cutoffTime).Delete(ctx)

		if err != nil {
			slog.Warn("failed to clear old sessions: ", "error", err)
			continue
		}

		slog.Info(fmt.Sprintf("cleared %d old sessions", rowsAffected))
	}
}
