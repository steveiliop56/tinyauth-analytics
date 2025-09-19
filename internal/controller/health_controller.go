package controller

import "github.com/gin-gonic/gin"

type HealthController struct {
	router *gin.RouterGroup
}

func NewHealthController(router *gin.RouterGroup) *HealthController {
	return &HealthController{
		router: router,
	}
}

func (hc *HealthController) SetupRoutes() {
	hc.router.GET("/health", hc.health)
	hc.router.HEAD("/health", hc.health)
}

func (hc *HealthController) health(c *gin.Context) {
	c.JSON(200, map[string]any{
		"status":  200,
		"message": "OK",
	})
}
