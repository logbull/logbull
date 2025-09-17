package logs_receiving_tests

import (
	"fmt"

	api_keys "logbull/internal/features/api_keys"
	logs_core "logbull/internal/features/logs/core"
	logs_receiving "logbull/internal/features/logs/receiving"
	projects_controllers "logbull/internal/features/projects/controllers"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"

	"github.com/gin-gonic/gin"
)

func CreateLogsTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Logs receiving endpoints - no authentication required
	logs_receiving.GetReceivingController().RegisterRoutes(v1)

	// Protected routes for other controllers (projects, api keys, etc.)
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	// Register other controllers that need authentication
	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		projects_controllers.GetProjectController().RegisterRoutes(routerGroup)
		projects_controllers.GetMembershipController().RegisterRoutes(routerGroup)
		api_keys.GetApiKeyController().RegisterRoutes(routerGroup)
	}

	return router
}

func CreateValidLogItems(count int, uniqueID string) []logs_receiving.LogItemRequestDTO {
	logItems := make([]logs_receiving.LogItemRequestDTO, count)

	levels := []logs_core.LogLevel{
		logs_core.LogLevelDebug,
		logs_core.LogLevelInfo,
		logs_core.LogLevelWarn,
		logs_core.LogLevelError,
		logs_core.LogLevelFatal,
	}

	for i := range count {
		logItems[i] = logs_receiving.LogItemRequestDTO{
			Level:   levels[i%len(levels)],
			Message: fmt.Sprintf("Test log message %s - %d", uniqueID, i+1),
			Fields: map[string]any{
				"test_id":    uniqueID,
				"log_index":  i + 1,
				"component":  "test_component",
				"request_id": fmt.Sprintf("req_%s_%d", uniqueID[:8], i+1),
			},
		}
	}

	return logItems
}
