package audit_logs

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	user_enums "logbull/internal/features/users/enums"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_GetGlobalAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly(t *testing.T) {
	adminUser := users_testing.CreateTestUser(user_enums.UserRoleAdmin)
	memberUser := users_testing.CreateTestUser(user_enums.UserRoleMember)
	router := createRouter()
	service := GetAuditLogService()
	projectID := uuid.New()
	testID := uuid.New().String()

	// Create test logs with unique identifiers
	userLogMessage := fmt.Sprintf("Test log with user %s", testID)
	projectLogMessage := fmt.Sprintf("Test log with project %s", testID)
	standaloneLogMessage := fmt.Sprintf("Test log standalone %s", testID)

	createAuditLog(service, userLogMessage, &adminUser.UserID, nil)
	createAuditLog(service, projectLogMessage, nil, &projectID)
	createAuditLog(service, standaloneLogMessage, nil, nil)

	// Test ADMIN can access global logs
	var response GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/audit-logs/global?limit=100", "Bearer "+adminUser.Token, http.StatusOK, &response)

	// Verify our specific test logs are present
	messages := extractMessages(response.AuditLogs)
	assert.Contains(t, messages, userLogMessage)
	assert.Contains(t, messages, projectLogMessage)
	assert.Contains(t, messages, standaloneLogMessage)

	// Test MEMBER cannot access global logs
	resp := test_utils.MakeGetRequest(t, router, "/api/v1/audit-logs/global",
		"Bearer "+memberUser.Token, http.StatusForbidden)
	assert.Contains(t, string(resp.Body), "only administrators can view global audit logs")
}

func Test_GetUserAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly(t *testing.T) {
	adminUser := users_testing.CreateTestUser(user_enums.UserRoleAdmin)
	user1 := users_testing.CreateTestUser(user_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(user_enums.UserRoleMember)
	router := createRouter()
	service := GetAuditLogService()
	projectID := uuid.New()
	testID := uuid.New().String()

	// Create test logs for different users with unique identifiers
	user1FirstMessage := fmt.Sprintf("Test log user1 first %s", testID)
	user1SecondMessage := fmt.Sprintf("Test log user1 second %s", testID)
	user2FirstMessage := fmt.Sprintf("Test log user2 first %s", testID)
	user2SecondMessage := fmt.Sprintf("Test log user2 second %s", testID)
	projectLogMessage := fmt.Sprintf("Test project log %s", testID)

	createAuditLog(service, user1FirstMessage, &user1.UserID, nil)
	createAuditLog(service, user1SecondMessage, &user1.UserID, &projectID)
	createAuditLog(service, user2FirstMessage, &user2.UserID, nil)
	createAuditLog(service, user2SecondMessage, &user2.UserID, &projectID)
	createAuditLog(service, projectLogMessage, nil, &projectID)

	// Test ADMIN can view any user's logs
	var user1Response GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		fmt.Sprintf("/api/v1/audit-logs/users/%s?limit=100", user1.UserID.String()),
		"Bearer "+adminUser.Token, http.StatusOK, &user1Response)

	// Verify user1's specific logs are present
	messages := extractMessages(user1Response.AuditLogs)
	assert.Contains(t, messages, user1FirstMessage)
	assert.Contains(t, messages, user1SecondMessage)

	// Count only our test logs for user1
	testLogsCount := 0
	for _, message := range messages {
		if message == user1FirstMessage || message == user1SecondMessage {
			testLogsCount++
		}
	}
	assert.Equal(t, 2, testLogsCount)

	// Test user can view own logs
	var ownLogsResponse GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		fmt.Sprintf("/api/v1/audit-logs/users/%s?limit=100", user2.UserID.String()),
		"Bearer "+user2.Token, http.StatusOK, &ownLogsResponse)

	// Verify user2's specific logs are present
	ownMessages := extractMessages(ownLogsResponse.AuditLogs)
	assert.Contains(t, ownMessages, user2FirstMessage)
	assert.Contains(t, ownMessages, user2SecondMessage)

	// Test user cannot view other user's logs
	resp := test_utils.MakeGetRequest(t, router,
		fmt.Sprintf("/api/v1/audit-logs/users/%s", user1.UserID.String()),
		"Bearer "+user2.Token, http.StatusForbidden)

	assert.Contains(t, string(resp.Body), "insufficient permissions")
}

func Test_GetGlobalAuditLogs_WithBeforeDateFilter_ReturnsFilteredLogs(t *testing.T) {
	adminUser := users_testing.CreateTestUser(user_enums.UserRoleAdmin)
	router := createRouter()
	baseTime := time.Now().UTC()

	// Set filter time to 30 minutes ago
	beforeTime := baseTime.Add(-30 * time.Minute)

	var filteredResponse GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		fmt.Sprintf("/api/v1/audit-logs/global?beforeDate=%s&limit=1000", beforeTime.Format(time.RFC3339)),
		"Bearer "+adminUser.Token, http.StatusOK, &filteredResponse)

	// Verify ALL returned logs are older than the filter time
	for _, log := range filteredResponse.AuditLogs {
		assert.True(t, log.CreatedAt.Before(beforeTime),
			fmt.Sprintf("Log created at %s should be before filter time %s",
				log.CreatedAt.Format(time.RFC3339), beforeTime.Format(time.RFC3339)))
	}
}

func createRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupDependencies()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetAuditLogController().RegisterRoutes(protected.(*gin.RouterGroup))

	return router
}
