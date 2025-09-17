package logs_querying_tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	logs_core "logbull/internal/features/logs/core"
	logs_querying "logbull/internal/features/logs/querying"
	logs_receiving "logbull/internal/features/logs/receiving"
	logs_receiving_tests "logbull/internal/features/logs/receiving/tests"
	projects_controllers "logbull/internal/features/projects/controllers"
	projects_models "logbull/internal/features/projects/models"
	projects_testing "logbull/internal/features/projects/testing"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// CreateLogQueryTestRouter creates unified router for log querying tests
func CreateLogQueryTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Logs receiving endpoints - no authentication required
	logs_receiving.GetReceivingController().RegisterRoutes(v1)

	// Protected routes for other controllers
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	// Register controllers that need authentication
	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		logs_querying.GetLogQueryController().RegisterRoutes(routerGroup)
		projects_controllers.GetProjectController().RegisterRoutes(routerGroup)
		projects_controllers.GetMembershipController().RegisterRoutes(routerGroup)
	}

	return router
}

// ExecuteTestQuery executes a query via HTTP API and returns the response
func ExecuteTestQuery(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	query *logs_core.LogQueryRequestDTO,
	token string,
	expectedStatus int,
) *logs_core.LogQueryResponseDTO {
	var response logs_core.LogQueryResponseDTO
	if expectedStatus == 200 {
		test_utils.MakePostRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/logs/query/execute/%s", projectID.String()),
			"Bearer "+token,
			query,
			expectedStatus,
			&response,
		)
		return &response
	} else {
		test_utils.MakePostRequest(
			t,
			router,
			fmt.Sprintf("/api/v1/logs/query/execute/%s", projectID.String()),
			"Bearer "+token,
			query,
			expectedStatus,
		)
		return nil
	}
}

// BuildSimpleConditionQuery creates a simple condition query
func BuildSimpleConditionQuery(field, operator, value string) *logs_core.LogQueryRequestDTO {
	to := time.Now().UTC()
	from := to.Add(-2 * time.Hour)

	return &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    field,
				Operator: logs_core.ConditionOperator(operator),
				Value:    value,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &from,
			To:   &to,
		},
		Limit:     50,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
}

// BuildLogicalQuery creates a logical query with given operator and children
func BuildLogicalQuery(operator string, children ...logs_core.QueryNode) *logs_core.LogQueryRequestDTO {
	to := time.Now().UTC()
	from := to.Add(-2 * time.Hour)

	return &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperator(operator),
				Children: children,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &from,
			To:   &to,
		},
		Limit:     50,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
}

// BuildCondition creates a QueryNode condition (helper for building logical queries)
func BuildCondition(field, operator string, value interface{}) *logs_core.QueryNode {
	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeCondition,
		Condition: &logs_core.ConditionNode{
			Field:    field,
			Operator: logs_core.ConditionOperator(operator),
			Value:    value,
		},
	}
}

// CreateTestLogsWithUniqueID submits logs to a project via the receiving API
func CreateTestLogsWithUniqueID(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	uniqueID string,
	count int,
) []logs_receiving.LogItemRequestDTO {
	logItems := logs_receiving_tests.CreateValidLogItems(count, uniqueID)

	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	var response logs_receiving.SubmitLogsResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		"",
		request,
		202, // HTTP Accepted
		&response,
	)

	// Execute background tasks to process logs immediately in tests
	workerService := logs_receiving.GetLogWorkerService()
	err := workerService.ExecuteBackgroundTasksForTest()
	if err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	// Wait for logs to be indexed in log storage
	WaitForLogsToBeIndexed(t, router, projectID, count, uniqueID, "")

	return logItems
}

// CreateTestLogsWithFields submits logs with custom fields to a project
func CreateTestLogsWithFields(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	customFields map[string]any,
	count int,
) []logs_receiving.LogItemRequestDTO {
	// Use test_id from customFields if provided, otherwise generate new
	uniqueID := uuid.New().String()
	if testID, exists := customFields["test_id"]; exists {
		if testIDStr, ok := testID.(string); ok {
			uniqueID = testIDStr
		}
	}

	logItems := logs_receiving_tests.CreateValidLogItems(count, uniqueID)

	// Add custom fields to each log item
	for i := range logItems {
		for key, value := range customFields {
			logItems[i].Fields[key] = value
		}
	}

	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	var response logs_receiving.SubmitLogsResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		"",
		request,
		202,
		&response,
	)

	// Execute background tasks to process logs immediately in tests
	workerService := logs_receiving.GetLogWorkerService()
	err := workerService.ExecuteBackgroundTasksForTest()
	if err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	// Wait for logs to be indexed in log storage (use uniqueID from test logs)
	WaitForLogsToBeIndexed(t, router, projectID, count, uniqueID, "")

	return logItems
}

// AssertLogContainsUniqueID verifies that logs contain expected unique ID
func AssertLogContainsUniqueID(t *testing.T, logs []logs_core.LogItemDTO, uniqueID string, expectedCount int) {
	matchingLogs := 0
	for _, log := range logs {
		if testID, exists := log.Fields["test_id"]; exists {
			if testID == uniqueID {
				matchingLogs++
			}
		}
	}
	assert.GreaterOrEqual(t, matchingLogs, expectedCount,
		"Expected at least %d logs with unique ID %s, but found %d",
		expectedCount, uniqueID, matchingLogs)
}

// AssertQueryResponseValid verifies basic query response structure
func AssertQueryResponseValid(t *testing.T, response *logs_core.LogQueryResponseDTO, expectedMinCount int) {
	assert.NotNil(t, response, "Query response should not be nil")
	assert.GreaterOrEqual(t, len(response.Logs), expectedMinCount,
		"Expected at least %d logs in response", expectedMinCount)
	assert.GreaterOrEqual(t, response.Total, int64(expectedMinCount),
		"Total count should be at least %d", expectedMinCount)
	assert.NotEmpty(t, response.ExecutedInMs, "ExecutedIn should be populated")
}

// AssertFieldsResponse verifies queryable fields response
func AssertFieldsResponse(t *testing.T, fields []logs_core.QueryableField, expectedFieldNames []string) {
	fieldNames := make([]string, len(fields))
	for i, field := range fields {
		fieldNames[i] = field.Name
	}

	for _, expectedName := range expectedFieldNames {
		assert.Contains(t, fieldNames, expectedName,
			"Expected field %s not found in response", expectedName)
	}
}

// WaitForLogsToBeIndexed polls log storage until logs appear or timeout
func WaitForLogsToBeIndexed(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	expectedCount int,
	uniqueID string,
	token string,
) {
	maxWaitTime := 10 * time.Second
	pollInterval := 100 * time.Millisecond
	startTime := time.Now()

	err := logs_core.GetLogCoreRepository().ForceFlush()
	if err != nil {
		t.Fatalf("Failed to flush logs: %v", err)
	}

	// If no token provided, skip authentication for internal testing
	if token == "" {
		token = "Bearer dummy" // Will be bypassed in test router
	}

	for time.Since(startTime) < maxWaitTime {
		query := BuildSimpleConditionQuery("test_id", "equals", uniqueID)

		// Try querying without failing the test on HTTP errors
		resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
			Method: "POST",
			URL:    fmt.Sprintf("/api/v1/logs/query/execute/%s", projectID.String()),
			Headers: map[string]string{
				"Authorization": token,
			},
			Body:           query,
			ExpectedStatus: 0, // Don't validate status
		})

		if resp.StatusCode == 200 {
			var queryResponse logs_core.LogQueryResponseDTO
			if err := json.Unmarshal(resp.Body, &queryResponse); err == nil {
				if len(queryResponse.Logs) >= expectedCount {
					// Logs are available!
					t.Logf("Logs indexed after %v (expected: %d, found: %d)",
						time.Since(startTime), expectedCount, len(queryResponse.Logs))
					return
				}
			}
		}

		time.Sleep(pollInterval)
	}

	t.Fatalf("Logs not indexed after %v (expected: %d logs with unique ID: %s)",
		maxWaitTime, expectedCount, uniqueID)
}

// CreateTestProjectWithLogs creates a project and submits logs for testing
func CreateTestProjectWithLogs(
	t *testing.T,
	router *gin.Engine,
	projectName string,
	ownerToken string,
	uniqueID string,
	logCount int,
) *projects_models.Project {
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, ownerToken, router)
	CreateTestLogsWithUniqueID(t, router, project.ID, uniqueID, logCount)
	return project
}

// SubmitLogsAndProcess submits logs and processes them immediately in tests
func SubmitLogsAndProcess(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	logItems []logs_receiving.LogItemRequestDTO,
) {
	submitURL := fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String())
	workerService := logs_receiving.GetLogWorkerService()

	// Send logs one by one with 1ms wait between each
	for _, logItem := range logItems {
		request := &logs_receiving.SubmitLogsRequestDTO{
			Logs: []logs_receiving.LogItemRequestDTO{logItem},
		}

		var response logs_receiving.SubmitLogsResponseDTO
		resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
			Method:         "POST",
			URL:            submitURL,
			Body:           request,
			ExpectedStatus: 202,
		})

		if err := json.Unmarshal(resp.Body, &response); err != nil {
			t.Fatalf("Failed to unmarshal submit response: %v", err)
		}

		if err := workerService.ExecuteBackgroundTasksForTest(); err != nil {
			t.Fatalf("Failed to execute background tasks: %v", err)
		}

		time.Sleep(1 * time.Millisecond)
	}
}

// BuildQueryWithPagination creates a condition query with custom limit/offset
func BuildQueryWithPagination(
	field, operator string,
	value interface{},
	limit, offset int,
) *logs_core.LogQueryRequestDTO {
	to := time.Now().UTC()
	from := to.Add(-2 * time.Hour)

	return &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    field,
				Operator: logs_core.ConditionOperator(operator),
				Value:    value,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &from,
			To:   &to,
		},
		Limit:     limit,
		Offset:    offset,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
}

// SetupTestProjectWithLogs creates a complete test environment with logs
func SetupTestProjectWithLogs(
	t *testing.T,
	testName string,
	logCount int,
) (router *gin.Engine, owner *users_dto.SignInResponseDTO, project *projects_models.Project, uniqueID string) {
	router = CreateLogQueryTestRouter()
	owner = users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID = uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testName, uniqueID[:8])
	project, _ = projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	if logCount > 0 {
		testLogs := logs_receiving_tests.CreateValidLogItems(logCount, uniqueID)
		SubmitLogsAndProcess(t, router, project.ID, testLogs)
		WaitForLogsToBeIndexed(t, router, project.ID, logCount, uniqueID, "Bearer "+owner.Token)
	}

	return
}

// CreateLogItemsWithMessages creates log items with specific messages and fields
func CreateLogItemsWithMessages(
	uniqueID string,
	messages []string,
	level logs_core.LogLevel,
	extraFields map[string]any,
) []logs_receiving.LogItemRequestDTO {
	logItems := make([]logs_receiving.LogItemRequestDTO, len(messages))

	for i, message := range messages {
		fields := map[string]any{"test_id": uniqueID}
		for k, v := range extraFields {
			fields[k] = v
		}

		logItems[i] = logs_receiving.LogItemRequestDTO{
			Level:   level,
			Message: message,
			Fields:  fields,
		}
	}

	return logItems
}

// AssertNoPaginationOverlap verifies that pagination results have no overlap
func AssertNoPaginationOverlap(t *testing.T, firstLogs, secondLogs []logs_core.LogItemDTO) {
	if len(firstLogs) == 0 || len(secondLogs) == 0 {
		return // No overlap possible with empty sets
	}

	firstLogIDs := make(map[string]bool)
	for _, log := range firstLogs {
		if log.ID != "" {
			firstLogIDs[log.ID] = true
		}
	}

	overlapCount := 0
	for _, log := range secondLogs {
		if log.ID != "" && firstLogIDs[log.ID] {
			overlapCount++
		}
	}

	assert.Equal(t, 0, overlapCount,
		"Pagination windows should not overlap - found %d overlapping logs", overlapCount)
}

// SetupBasicQueryTest creates basic test setup with router, user, project and unique ID
func SetupBasicQueryTest(t *testing.T, testName string) (
	router *gin.Engine,
	owner *users_dto.SignInResponseDTO,
	project *projects_models.Project,
	uniqueID string,
) {
	router = CreateLogQueryTestRouter()
	owner = users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID = uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testName, uniqueID[:8])
	project, _ = projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)
	return
}

// CreateTestLogsWithMessages creates and submits logs with specific messages
func CreateTestLogsWithMessages(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	uniqueID string,
	messages []string,
	level logs_core.LogLevel,
	extraFields map[string]any,
) {
	logItems := CreateLogItemsWithMessages(uniqueID, messages, level, extraFields)
	SubmitLogsAndProcess(t, router, projectID, logItems)
}

// SubmitLogsWithCustomFields creates logs with custom fields and submits them
func SubmitLogsWithCustomFields(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	uniqueID string,
	count int,
	customFields map[string]any,
) {
	logItems := logs_receiving_tests.CreateValidLogItems(count, uniqueID)

	// Add custom fields to each log item
	for i := range logItems {
		for key, value := range customFields {
			logItems[i].Fields[key] = value
		}
	}

	SubmitLogsAndProcess(t, router, projectID, logItems)
}
