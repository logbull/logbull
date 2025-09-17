package logs_querying_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	logs_core "logbull/internal/features/logs/core"
	logs_receiving "logbull/internal/features/logs/receiving"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_ExecuteQuery_FilterByLevelEquals_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Level Equals Test %s", uniqueID[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	logItems := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelInfo,
			Message: fmt.Sprintf("Info message %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "operation": "user_login"},
		},
		{
			Level:   logs_core.LogLevelError,
			Message: fmt.Sprintf("Error occurred %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "operation": "payment_failed"},
		},
		{
			Level:   logs_core.LogLevelWarn,
			Message: fmt.Sprintf("Warning detected %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "operation": "rate_limit_approaching"},
		},
		{
			Level:   logs_core.LogLevelError,
			Message: fmt.Sprintf("Another error %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "operation": "database_timeout"},
		},
		{
			Level:   logs_core.LogLevelFatal,
			Message: fmt.Sprintf("System failure %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "operation": "system_crash"},
		},
	}

	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	submitURL := fmt.Sprintf("/api/v1/logs/receiving/%s", project.ID.String())
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

	workerService := logs_receiving.GetLogWorkerService()
	if err := workerService.ExecuteBackgroundTasksForTest(); err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	targetLevel := "ERROR"
	query := BuildSimpleConditionQuery("level", "equals", targetLevel)
	queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)
	assertAllLogsHaveLevel(t, queryResponse.Logs, targetLevel)
	assert.Equal(t, 2, len(queryResponse.Logs), "Expected exactly 2 logs with level %s", targetLevel)
}

func Test_ExecuteQuery_FilterByLevelIn_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Level In Test %s", uniqueID[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	logItems := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelDebug,
			Message: fmt.Sprintf("Debug info %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "component": "auth"},
		},
		{
			Level:   logs_core.LogLevelInfo,
			Message: fmt.Sprintf("Information log %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "component": "api"},
		},
		{
			Level:   logs_core.LogLevelWarn,
			Message: fmt.Sprintf("Warning message %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "component": "database"},
		},
		{
			Level:   logs_core.LogLevelError,
			Message: fmt.Sprintf("Error occurred %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "component": "payment"},
		},
		{
			Level:   logs_core.LogLevelFatal,
			Message: fmt.Sprintf("Fatal system error %s", uniqueID),
			Fields:  map[string]any{"test_id": uniqueID, "component": "core"},
		},
	}

	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	submitURL := fmt.Sprintf("/api/v1/logs/receiving/%s", project.ID.String())
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

	workerService := logs_receiving.GetLogWorkerService()
	if err := workerService.ExecuteBackgroundTasksForTest(); err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	targetLevels := []string{"ERROR", "FATAL"}
	from := time.Now().UTC().Add(-2 * time.Hour)
	to := time.Now().UTC()
	query := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "level",
				Operator: logs_core.ConditionOperatorIn,
				Value:    targetLevels,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{From: &from, To: &to},
		Limit:     50,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)

	foundErrorLogs := 0
	foundFatalLogs := 0
	for _, log := range queryResponse.Logs {
		switch log.Level {
		case "ERROR":
			foundErrorLogs++
		case "FATAL":
			foundFatalLogs++
		default:
			t.Errorf("Query returned log with unexpected level: %s (expected: ERROR or FATAL)", log.Level)
		}
	}

	assert.Equal(t, 2, len(queryResponse.Logs), "Expected exactly 2 logs with levels ERROR or FATAL")
	assert.Equal(t, 1, foundErrorLogs, "Expected exactly 1 ERROR log")
	assert.Equal(t, 1, foundFatalLogs, "Expected exactly 1 FATAL log")
}

func Test_SubmitAndQuery_WithDifferentLogLevels_LogLevelsFilterable(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Level Filtering E2E %s", uniqueID[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	allLevels := []logs_core.LogLevel{
		logs_core.LogLevelDebug,
		logs_core.LogLevelInfo,
		logs_core.LogLevelWarn,
		logs_core.LogLevelError,
		logs_core.LogLevelFatal,
	}

	var allLogItems []logs_receiving.LogItemRequestDTO
	expectedCounts := make(map[string]int)

	for i, level := range allLevels {
		levelCount := i + 2
		expectedCounts[string(level)] = levelCount

		for j := range levelCount {
			logItem := logs_receiving.LogItemRequestDTO{
				Level:   level,
				Message: fmt.Sprintf("%s message %d for test %s", level, j+1, uniqueID),
				Fields: map[string]any{
					"test_id":     uniqueID,
					"level_test":  string(level),
					"log_number":  j + 1,
					"batch_index": i,
				},
			}
			allLogItems = append(allLogItems, logItem)
		}
	}

	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: allLogItems,
	}

	submitURL := fmt.Sprintf("/api/v1/logs/receiving/%s", project.ID.String())
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

	workerService := logs_receiving.GetLogWorkerService()
	if err := workerService.ExecuteBackgroundTasksForTest(); err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	WaitForLogsToBeIndexed(t, router, project.ID, expectedCounts["DEBUG"], uniqueID, "Bearer "+owner.Token)

	for _, level := range allLevels {
		levelStr := string(level)
		expectedCount := expectedCounts[levelStr]

		query := BuildSimpleConditionQuery("level", "equals", levelStr)
		queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

		AssertQueryResponseValid(t, queryResponse, 1)
		assertAllLogsHaveLevel(t, queryResponse.Logs, levelStr)
		assert.Equal(
			t,
			expectedCount,
			len(queryResponse.Logs),
			"Expected %d logs with level %s, but found %d",
			expectedCount,
			levelStr,
			len(queryResponse.Logs),
		)
	}

	errorQuery := BuildLogicalQuery("or",
		*BuildCondition("level", "equals", "ERROR"),
		*BuildCondition("level", "equals", "FATAL"),
	)

	errorQueryResponse := ExecuteTestQuery(t, router, project.ID, errorQuery, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, errorQueryResponse, 1)

	expectedErrorAndFatal := expectedCounts["ERROR"] + expectedCounts["FATAL"]
	assert.Equal(t, expectedErrorAndFatal, len(errorQueryResponse.Logs),
		"Expected %d ERROR+FATAL logs, but found %d", expectedErrorAndFatal, len(errorQueryResponse.Logs))
}

func assertAllLogsHaveLevel(t *testing.T, logs []logs_core.LogItemDTO, expectedLevel string) {
	for _, log := range logs {
		if log.Level != expectedLevel {
			t.Errorf("Query returned log with unexpected level: %s (expected: %s)", log.Level, expectedLevel)
		}
	}
}
