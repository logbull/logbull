package logs_core_tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	logs_core "logbull/internal/features/logs/core"
	projects_controllers "logbull/internal/features/projects/controllers"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_LogsCleanedUpOnProjectDeletion(t *testing.T) {
	logs_core.SetupDependencies()

	router := createProjectDeletionTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	repository := logs_core.GetLogCoreRepository()

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Project Deletion Test %s", uniqueID[:8])

	// Create project via API
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create couple of logs via repository
	currentTime := time.Now().UTC()
	testLogEntries1 := CreateTestLogEntriesWithUniqueFields(project.ID, currentTime,
		"First test log for deletion", map[string]any{
			"test_session": uniqueID,
			"log_type":     "test",
			"sequence":     1,
		})

	testLogEntries2 := CreateTestLogEntriesWithUniqueFields(project.ID, currentTime.Add(1*time.Second),
		"Second test log for deletion", map[string]any{
			"test_session": uniqueID,
			"log_type":     "test",
			"sequence":     2,
		})

	testLogEntries3 := CreateTestLogEntriesWithUniqueFields(project.ID, currentTime.Add(2*time.Second),
		"Third test log for deletion", map[string]any{
			"test_session": uniqueID,
			"log_type":     "test",
			"sequence":     3,
		})

	// Merge and store all log entries
	mergedEntries := MergeLogEntries(testLogEntries1, testLogEntries2)
	mergedEntries = MergeLogEntries(mergedEntries, testLogEntries3)
	StoreTestLogsAndFlush(t, repository, mergedEntries)

	// Verify logs exist before deletion
	queryBeforeDeletion := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueID,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: func() *time.Time { t := currentTime.Add(-1 * time.Hour); return &t }(),
			To:   func() *time.Time { t := currentTime.Add(1 * time.Hour); return &t }(),
		},
		Limit:     50,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	logsBeforeDeletion, err := repository.ExecuteQueryForProject(project.ID, queryBeforeDeletion)
	assert.NoError(t, err, "Query should succeed before project deletion")
	assert.GreaterOrEqual(t, len(logsBeforeDeletion.Logs), 3, "Should have at least 3 logs before deletion")

	// Remove project via API (this should trigger log cleanup)
	deleteURL := fmt.Sprintf("/api/v1/projects/%s", project.ID.String())
	test_utils.MakeDeleteRequest(t, router, deleteURL, "Bearer "+owner.Token, http.StatusOK)

	// Force flush to ensure deletion is processed
	flushErr := repository.ForceFlush()
	assert.NoError(t, flushErr, "Should be able to flush after deletion")

	// Wait up to 60 seconds for logs to be deleted
	timeout := 60 * time.Second
	WaitForLogsDeletion(t, repository, project.ID, queryBeforeDeletion, timeout)

	// Final verification that logs are completely removed
	logsAfterDeletion, err := repository.ExecuteQueryForProject(project.ID, queryBeforeDeletion)
	assert.NoError(t, err, "Repository query should still work even if project is deleted")
	assert.Equal(t, 0, len(logsAfterDeletion.Logs), "All logs should be removed after project deletion")
	assert.Equal(t, int64(0), logsAfterDeletion.Total, "Total count should be 0 after project deletion")
}

func createProjectDeletionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		projects_controllers.GetProjectController().RegisterRoutes(routerGroup)
		projects_controllers.GetMembershipController().RegisterRoutes(routerGroup)
	}

	return router
}
