package logs_querying_tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	logs_core "logbull/internal/features/logs/core"
	logs_receiving "logbull/internal/features/logs/receiving"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"

	"github.com/google/uuid"
)

func Test_ExecuteQuery_MessageOperators_ReturnsMatchingLogs(t *testing.T) {
	testCases := []struct {
		name          string
		operator      string
		searchValue   string
		testMessages  []string
		expectedCount int
		verifyFunc    func(t *testing.T, logs []logs_core.LogItemDTO, searchValue string, expectedCount int)
	}{
		{
			name:        "Equals",
			operator:    "equals",
			searchValue: "User authentication successful",
			testMessages: []string{
				"User authentication successful",
				"Payment processing failed",
				"User authentication successful", // Duplicate
				"User authentication timeout",
			},
			expectedCount: 2,
			verifyFunc: func(t *testing.T, logs []logs_core.LogItemDTO, searchValue string, expectedCount int) {
				foundMatching := 0
				for _, log := range logs {
					if log.Message == searchValue {
						foundMatching++
					} else {
						t.Errorf("Equals query returned log with wrong message: %s", log.Message)
					}
				}
				if foundMatching != expectedCount {
					t.Errorf("Expected %d logs with exact message match, got %d", expectedCount, foundMatching)
				}
			},
		},
		{
			name:        "Contains",
			operator:    "contains",
			searchValue: "Database",
			testMessages: []string{
				"Database connection established successfully",
				"Failed to connect to database server",
				"Database pool size exceeded threshold",
				"User session created",
				"Cache invalidation completed",
			},
			expectedCount: 2,
			verifyFunc: func(t *testing.T, logs []logs_core.LogItemDTO, searchValue string, expectedCount int) {
				foundMatching := 0
				for _, log := range logs {
					if strings.Contains(log.Message, searchValue) {
						foundMatching++
					} else {
						t.Errorf("Contains query returned log without '%s' in message: %s", searchValue, log.Message)
					}
				}
				if foundMatching != expectedCount {
					t.Errorf("Expected %d logs containing '%s', got %d", expectedCount, searchValue, foundMatching)
				}
			},
		},
		{
			name:        "NotContains",
			operator:    "not_contains",
			searchValue: "error",
			testMessages: []string{
				"Authentication error occurred",
				"Authorization failure detected",
				"User logged in successfully",
				"Data processing completed",
			},
			expectedCount: 1, // At least one non-error log
			verifyFunc: func(t *testing.T, logs []logs_core.LogItemDTO, searchValue string, expectedCount int) {
				foundNonError := 0
				for _, log := range logs {
					if !strings.Contains(strings.ToLower(log.Message), searchValue) {
						foundNonError++
					}
				}
				if foundNonError == 0 {
					t.Errorf("NOT contains query should return logs without '%s' in message", searchValue)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router, owner, project, uniqueID := SetupBasicQueryTest(t, fmt.Sprintf("Message %s Test", tc.name))

			// Create test logs with messages
			CreateTestLogsWithMessages(
				t,
				router,
				project.ID,
				uniqueID,
				tc.testMessages,
				logs_core.LogLevelInfo,
				map[string]any{
					"operator_test": tc.name,
				},
			)

			WaitForLogsToBeIndexed(t, router, project.ID, tc.expectedCount, uniqueID, "Bearer "+owner.Token)

			query := BuildSimpleConditionQuery("message", tc.operator, tc.searchValue)
			queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

			AssertQueryResponseValid(t, queryResponse, 1)
			tc.verifyFunc(t, queryResponse.Logs, tc.searchValue, tc.expectedCount)
		})
	}
}

func Test_ExecuteQuery_ComplexMessageQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Complex Message Query Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create diverse test logs for complex querying
	testLogs := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelError,
			Message: "Authentication failed for user john.doe@example.com",
			Fields:  map[string]any{"test_id": uniqueID, "module": "auth", "user_type": "customer"},
		},
		{
			Level:   logs_core.LogLevelError,
			Message: "Database connection timeout after 30 seconds",
			Fields:  map[string]any{"test_id": uniqueID, "module": "database", "user_type": "system"},
		},
		{
			Level:   logs_core.LogLevelWarn,
			Message: "Authentication attempt from suspicious IP",
			Fields:  map[string]any{"test_id": uniqueID, "module": "auth", "user_type": "unknown"},
		},
		{
			Level:   logs_core.LogLevelInfo,
			Message: "User registration completed successfully",
			Fields:  map[string]any{"test_id": uniqueID, "module": "registration", "user_type": "customer"},
		},
	}

	SubmitLogsAndProcess(t, router, project.ID, testLogs)
	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	// Complex query: (message contains "Authentication" AND level = ERROR) OR (message contains "successfully")
	complexQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorOr,
				Children: []logs_core.QueryNode{
					{
						Type: logs_core.QueryNodeTypeLogical,
						Logic: &logs_core.LogicalNode{
							Operator: logs_core.LogicalOperatorAnd,
							Children: []logs_core.QueryNode{
								*BuildCondition("message", "contains", "Authentication"),
								*BuildCondition("level", "equals", "ERROR"),
							},
						},
					},
					*BuildCondition("message", "contains", "successfully"),
				},
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: func() *time.Time { t := time.Now().UTC().Add(-2 * time.Hour); return &t }(),
			To:   func() *time.Time { t := time.Now().UTC(); return &t }(),
		},
		Limit:     50,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	queryResponse := ExecuteTestQuery(t, router, project.ID, complexQuery, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)

	// Verify query results match expected conditions
	foundAuthErrorLogs := false
	foundSuccessfulLogs := false

	for _, log := range queryResponse.Logs {
		// Check for first condition: message contains "Authentication" AND level = ERROR
		if strings.Contains(log.Message, "Authentication") && log.Level == "ERROR" {
			foundAuthErrorLogs = true
		}
		// Check for second condition: message contains "successfully"
		if strings.Contains(log.Message, "successfully") {
			foundSuccessfulLogs = true
		}
	}

	if !foundAuthErrorLogs && !foundSuccessfulLogs {
		t.Errorf("Complex message query should match at least one condition")
	}

	t.Logf("Complex query found auth errors: %v, successful operations: %v, total logs: %d",
		foundAuthErrorLogs, foundSuccessfulLogs, len(queryResponse.Logs))
}

func Test_ExecuteQuery_MessageWithSpecialCharacters_HandlesCorrectly(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Special Characters Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create test logs with special characters and unicode
	testLogs := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelInfo,
			Message: `File path: /home/user/documents/file.txt`,
			Fields:  map[string]any{"test_id": uniqueID, "type": "file_path"},
		},
		{
			Level:   logs_core.LogLevelError,
			Message: `JSON error: field name is required`,
			Fields:  map[string]any{"test_id": uniqueID, "type": "json_error"},
		},
		{
			Level:   logs_core.LogLevelInfo,
			Message: "User greeting: Hello! 👋 Welcome 🚀",
			Fields:  map[string]any{"test_id": uniqueID, "type": "unicode"},
		},
		{
			Level:   logs_core.LogLevelWarn,
			Message: "Special chars: @#$%^&*()_+-=|;',./<>?",
			Fields:  map[string]any{"test_id": uniqueID, "type": "special_chars"},
		},
	}

	SubmitLogsAndProcess(t, router, project.ID, testLogs)
	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	// Test message contains unix-style path
	query1 := BuildSimpleConditionQuery("message", "contains", "/home/user")
	queryResponse1 := ExecuteTestQuery(t, router, project.ID, query1, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, queryResponse1, 1)

	// Test message contains unicode emoji
	query2 := BuildSimpleConditionQuery("message", "contains", "👋")
	queryResponse2 := ExecuteTestQuery(t, router, project.ID, query2, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, queryResponse2, 1)

	// Test message contains field keyword
	query3 := BuildSimpleConditionQuery("message", "contains", "field name")
	queryResponse3 := ExecuteTestQuery(t, router, project.ID, query3, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, queryResponse3, 1)

	t.Logf("Special characters test completed - File paths: %d, Unicode: %d, JSON: %d",
		len(queryResponse1.Logs), len(queryResponse2.Logs), len(queryResponse3.Logs))
}

func Test_ExecuteQuery_ShortMessages_HandlesCorrectly(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Short Message Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create test logs with very short messages
	testLogs := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelInfo,
			Message: "x",
			Fields:  map[string]any{"test_id": uniqueID, "type": "single_char"},
		},
		{
			Level:   logs_core.LogLevelInfo,
			Message: "OK",
			Fields:  map[string]any{"test_id": uniqueID, "type": "short"},
		},
		{
			Level:   logs_core.LogLevelInfo,
			Message: "Normal message here",
			Fields:  map[string]any{"test_id": uniqueID, "type": "normal"},
		},
	}

	SubmitLogsAndProcess(t, router, project.ID, testLogs)
	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	// Test message equals single character
	query1 := BuildSimpleConditionQuery("message", "equals", "x")
	queryResponse1 := ExecuteTestQuery(t, router, project.ID, query1, owner.Token, http.StatusOK)

	// Test message contains "OK"
	query2 := BuildSimpleConditionQuery("message", "contains", "OK")
	queryResponse2 := ExecuteTestQuery(t, router, project.ID, query2, owner.Token, http.StatusOK)

	t.Logf("Short message test - Single char matches: %d logs, 'OK' matches: %d logs",
		len(queryResponse1.Logs), len(queryResponse2.Logs))
}
