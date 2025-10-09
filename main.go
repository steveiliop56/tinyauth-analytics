package main

import (
	"context"
	"os"
	"time"
	"tinyauth-analytics/internal/controller"
	"tinyauth-analytics/internal/middleware"
	"tinyauth-analytics/internal/model"
	"tinyauth-analytics/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger().Level(zerolog.FatalLevel)

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
		log.Fatal().Err(err).Msg("failed to parse config")
	}

	switch conf.LogLevel {
	case "debug":
		log.Level(zerolog.DebugLevel)
	case "info":
		log.Level(zerolog.InfoLevel)
	case "warn":
		log.Level(zerolog.WarnLevel)
	case "error":
		log.Level(zerolog.ErrorLevel)
	case "fatal":
		log.Level(zerolog.FatalLevel)
	default:
		log.Fatal().Str("level", conf.LogLevel).Msg("invalid log level")
	}

	dbSvc := service.NewDatabaseService(service.DatabaseServiceConfig{
		DatabasePath: conf.DatabasePath,
	})

	if err := dbSvc.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	db := dbSvc.GetDatabase()

	cacheSvc := service.NewCacheService()

	zerologMiddleware := middleware.NewZerologMiddleware(log.Logger.GetLevel())

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(zerologMiddleware.Middleware())

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

	log.Info().Str("port", conf.Port).Str("address", conf.Address).Msg("starting server, version " + version)

	if err := engine.Run(conf.Address + ":" + conf.Port); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}

func clearOldSessions(db *gorm.DB) {
	ticker := time.NewTicker(time.Duration(24) * time.Hour)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		log.Info().Msg("clearing old sessions")

		ctx := context.Background()
		cutoffTime := time.Now().Add(time.Duration(-48) * time.Hour).UnixMilli()
		rowsAffected, err := gorm.G[model.Instance](db).Where("last_seen < ?", cutoffTime).Delete(ctx)

		if err != nil {
			log.Warn().Err(err).Msg("failed to clear old sessions")
			continue
		}

		log.Info().Msgf("cleared %d old sessions", rowsAffected)
	}
}
