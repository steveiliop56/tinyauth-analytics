package controller

import (
	"context"
	"errors"
	"strings"
	"time"
	"tinyauth-analytics/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type RateLimit interface {
	Middleware() gin.HandlerFunc
}

type Hearbeat struct {
	Version string `json:"version"`
	UUID    string `json:"uuid"`
}

type InstancesController struct {
	database  *gorm.DB
	router    *gin.RouterGroup
	rateLimit RateLimit
	warn      string
}

func NewInstancesController(router *gin.RouterGroup, database *gorm.DB, rateLimit RateLimit, warn string) *InstancesController {
	return &InstancesController{
		database:  database,
		router:    router,
		rateLimit: rateLimit,
		warn:      warn,
	}
}

func (ic *InstancesController) SetupRoutes() {
	instancesGroup := ic.router.Group("/instances")
	instancesGroup.GET("/all", ic.listAllInstances)
	instancesGroup.POST("/heartbeat", ic.rateLimit.Middleware(), ic.heartbeat)
}

func (ic *InstancesController) listAllInstances(c *gin.Context) {
	ctx := context.Background()

	instances, err := gorm.G[model.Instance](ic.database).Find(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to fetch instances")
		c.JSON(500, gin.H{
			"status":  500,
			"message": "Database error",
		})
		return
	}

	c.JSON(200, map[string]any{
		"status":    200,
		"total":     len(instances),
		"instances": instances,
		"warning":   ic.warn,
	})
}

func (ic *InstancesController) heartbeat(c *gin.Context) {
	var heartbeat Hearbeat

	if err := c.BindJSON(&heartbeat); err != nil {
		c.JSON(400, gin.H{
			"status":  400,
			"message": "Invalid request body",
		})
		return
	}

	if strings.TrimSpace(heartbeat.UUID) == "" || strings.TrimSpace(heartbeat.Version) == "" {
		c.JSON(400, gin.H{
			"status":  400,
			"message": "Missing required fields",
		})
		return
	}

	ctx := context.Background()
	instance, err := gorm.G[model.Instance](ic.database).Where("uuid = ?", heartbeat.UUID).First(ctx)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error().Err(err).Msg("failed to fetch instance")
		c.JSON(500, gin.H{
			"status":  500,
			"message": "Database error",
		})
		return
	}

	t := time.Now().UnixMilli()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		err := gorm.G[model.Instance](ic.database).Create(ctx, &model.Instance{
			UUID:     heartbeat.UUID,
			Version:  heartbeat.Version,
			LastSeen: t,
		})

		if err != nil {
			log.Error().Err(err).Msg("failed to create instance")
			c.JSON(500, gin.H{
				"status":  500,
				"message": "Database error",
			})
			return
		}

		c.JSON(201, gin.H{
			"status":  201,
			"message": "Instance created",
			"warning": ic.warn,
		})
		return
	}

	_, err = gorm.G[model.Instance](ic.database).Where("id = ?", instance.ID).Updates(ctx, model.Instance{
		Version:  heartbeat.Version,
		LastSeen: t,
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to update instance")
		c.JSON(500, gin.H{
			"status":  500,
			"message": "Database error",
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  200,
		"message": "Instance updated",
		"warning": ic.warn,
	})
}
