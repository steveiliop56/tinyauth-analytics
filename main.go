package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
	"tinyauth-analytics/internal/controller"
	"tinyauth-analytics/internal/middleware"
	"tinyauth-analytics/internal/model"
	"tinyauth-analytics/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var version = "development"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	dbPath := os.Getenv("DB_PATH")

	if dbPath == "" {
		dbPath = "/data/analytics.db"
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	address := os.Getenv("ADDRESS")

	if address == "" {
		address = "0.0.0.0"
	}

	trustedProxies := os.Getenv("TRUSTED_PROXIES")

	dbSvc := services.NewDatabaseService(services.DatabaseServiceConfig{
		DatabasePath: dbPath,
	})

	if err := dbSvc.Init(); err != nil {
		log.Fatal("failed to initialize database:", err)
	}

	db := dbSvc.GetDatabase()

	engine := gin.Default()

	engine.SetTrustedProxies(strings.Split(trustedProxies, ","))

	if version != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	api := engine.Group("/v1")

	rateLimitMiddleware := middleware.NewRateLimitMiddleware(db)

	instancesCtrl := controller.NewInstancesController(api, db, rateLimitMiddleware)

	instancesCtrl.SetupRoutes()

	go clearOldSessions(db)

	log.Printf("Starting analytics server on %s:%s (version: %s)", address, port, version)

	if err := engine.Run(address + ":" + port); err != nil {
		log.Fatal("failed to start server:", err)
	}
}

func clearOldSessions(db *gorm.DB) {
	ticker := time.NewTicker(time.Duration(24) * time.Hour)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		log.Println("Clearing old sessions")

		ctx := context.Background()
		cutoffTime := time.Now().Add(time.Duration(-48) * time.Hour).UnixMilli()
		rowsAffected, err := gorm.G[model.Instance](db).Where("last_seen < ?", cutoffTime).Delete(ctx)

		if err != nil {
			log.Println("Failed to clear old sessions:", err)
			continue
		}

		log.Printf("Cleared %d old sessions\n", rowsAffected)
	}
}
