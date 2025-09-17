package system_healthcheck

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthcheckController struct {
	healthcheckService *HealthcheckService
}

func (c *HealthcheckController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/system/health", c.CheckHealth)
}

// CheckHealth
// @Summary Check system health
// @Description Check if the system is healthy by testing database connection
// @Tags system/health
// @Produce json
// @Success 200 {object} HealthcheckResponse
// @Failure 503 {object} HealthcheckResponse
// @Router /system/health [get]
func (c *HealthcheckController) CheckHealth(ctx *gin.Context) {
	err := c.healthcheckService.IsHealthy()

	if err == nil {
		ctx.JSON(
			http.StatusOK,
			HealthcheckResponse{
				Status: "Application is healthy, internal DB working fine and disk usage is below 95%. You can connect downdetector to this endpoint",
			},
		)
		return
	}

	ctx.JSON(http.StatusServiceUnavailable, HealthcheckResponse{Status: err.Error()})
}
