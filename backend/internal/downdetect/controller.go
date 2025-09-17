package downdetect

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DowndetectController struct {
	service *DowndetectService
}

func (c *DowndetectController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/downdetect/is-available", c.IsAvailable)
}

// @Summary Check API availability
// @Description Checks if the API service is available
// @Tags downdetect
// @Accept json
// @Produce json
// @Success 200
// @Failure 500
// @Router /downdetect/api [get]
func (c *DowndetectController) IsAvailable(ctx *gin.Context) {
	err := c.service.IsDbAvailable()
	if err != nil {
		ctx.JSON(
			http.StatusInternalServerError,
			gin.H{"error": fmt.Sprintf("Database is not available: %v", err)},
		)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "API and DB are available"})
}
