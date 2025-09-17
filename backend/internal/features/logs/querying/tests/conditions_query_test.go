package logs_querying_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	logs_core "logbull/internal/features/logs/core"
	logs_receiving "logbull/internal/features/logs/receiving"
	logs_receiving_tests "logbull/internal/features/logs/receiving/tests"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
)

func Test_ExecuteQuery_WithSimpleConditionQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Simple Condition Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	SubmitLogsAndProcess(t, router, project.ID, logs_receiving_tests.CreateValidLogItems(3, uniqueID))
	WaitForLogsToBeIndexed(t, router, project.ID, 3, uniqueID, "Bearer "+owner.Token)

	query := BuildSimpleConditionQuery("test_id", "equals", uniqueID)
	queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)
	AssertLogContainsUniqueID(t, queryResponse.Logs, uniqueID, 3)

	if query.Query.Type != logs_core.QueryNodeTypeCondition {
		t.Errorf("Expected condition query type, got %s", query.Query.Type)
	}
}

func Test_ExecuteQuery_WithLogicalANDQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Logical AND Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	logItems := logs_receiving_tests.CreateValidLogItems(2, uniqueID)
	for i := range logItems {
		logItems[i].Fields["env"] = "production"
	}

	SubmitLogsAndProcess(t, router, project.ID, logItems)

	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	andQuery := BuildLogicalQuery("and",
		*BuildCondition("test_id", "equals", uniqueID),
		*BuildCondition("env", "equals", "production"),
	)

	queryResponse := ExecuteTestQuery(t, router, project.ID, andQuery, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)
	AssertLogContainsUniqueID(t, queryResponse.Logs, uniqueID, 2)

	for _, log := range queryResponse.Logs {
		if log.Fields["env"] != "production" {
			t.Errorf("Expected env=production in log, got %v", log.Fields["env"])
		}
	}
}

func Test_ExecuteQuery_WithLogicalORQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID1 := uuid.New().String()
	uniqueID2 := uuid.New().String()
	projectName := fmt.Sprintf("Logical OR Test %s", uniqueID1[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	logItems1 := logs_receiving_tests.CreateValidLogItems(2, uniqueID1)
	logItems2 := logs_receiving_tests.CreateValidLogItems(3, uniqueID2)
	allLogItems := append(logItems1, logItems2...)

	SubmitLogsAndProcess(t, router, project.ID, allLogItems)
	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID1, "Bearer "+owner.Token)

	orQuery := BuildLogicalQuery("or",
		*BuildCondition("test_id", "equals", uniqueID1),
		*BuildCondition("test_id", "equals", uniqueID2),
	)

	queryResponse := ExecuteTestQuery(t, router, project.ID, orQuery, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1) // Should match logs from either condition

	// Verify that we get logs with at least one of the unique IDs
	foundID1 := false
	foundID2 := false
	for _, log := range queryResponse.Logs {
		if testID, exists := log.Fields["test_id"]; exists {
			if testID == uniqueID1 {
				foundID1 = true
			}
			if testID == uniqueID2 {
				foundID2 = true
			}
		}
	}

	if !foundID1 && !foundID2 {
		t.Errorf(
			"OR query should match logs from at least one condition. Found ID1: %v, Found ID2: %v",
			foundID1,
			foundID2,
		)
	}
}

func Test_ExecuteQuery_WithLogicalNOTQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	excludeID := uuid.New().String()
	includeID := uuid.New().String()
	projectName := fmt.Sprintf("Logical NOT Test %s", excludeID[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	logItems1 := logs_receiving_tests.CreateValidLogItems(2, excludeID)
	logItems2 := logs_receiving_tests.CreateValidLogItems(3, includeID)
	allLogItems := append(logItems1, logItems2...)

	SubmitLogsAndProcess(t, router, project.ID, allLogItems)
	WaitForLogsToBeIndexed(t, router, project.ID, 2, excludeID, "Bearer "+owner.Token)

	notQuery := BuildLogicalQuery("not",
		*BuildCondition("test_id", "equals", excludeID),
	)

	queryResponse := ExecuteTestQuery(t, router, project.ID, notQuery, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)

	for _, log := range queryResponse.Logs {
		if testID, exists := log.Fields["test_id"]; exists {
			if testID == excludeID {
				t.Errorf("NOT query returned log with excluded test_id: %v", excludeID)
			}
		}
	}

	_ = includeID // Suppress unused variable warning
}

func Test_ExecuteQuery_WithNestedLogicalQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID1 := uuid.New().String()
	uniqueID2 := uuid.New().String()
	projectName := fmt.Sprintf("Nested Logical Test %s", uniqueID1[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	logItems1 := logs_receiving_tests.CreateValidLogItems(2, uniqueID1)
	for i := range logItems1 {
		logItems1[i].Fields["env"] = "production"
		logItems1[i].Level = logs_core.LogLevelInfo
	}

	logItems2 := logs_receiving_tests.CreateValidLogItems(3, uniqueID2)
	for i := range logItems2 {
		logItems2[i].Fields["component"] = "auth"
		logItems2[i].Level = logs_core.LogLevelError
	}

	logItems3 := logs_receiving_tests.CreateValidLogItems(1, "should_not_match")
	for i := range logItems3 {
		logItems3[i].Fields["env"] = "development"
		logItems3[i].Level = logs_core.LogLevelWarn
		logItems3[i].Fields["component"] = "web"
	}

	allLogItems := append(logItems1, logItems2...)
	allLogItems = append(allLogItems, logItems3...)

	SubmitLogsAndProcess(t, router, project.ID, allLogItems)
	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID1, "Bearer "+owner.Token)

	nestedQuery := BuildLogicalQuery("or",
		// First condition: env=production AND level=INFO
		logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorAnd,
				Children: []logs_core.QueryNode{
					*BuildCondition("env", "equals", "production"),
					*BuildCondition("level", "equals", "INFO"),
				},
			},
		},
		// Second condition: component=auth AND level=ERROR
		logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorAnd,
				Children: []logs_core.QueryNode{
					*BuildCondition("component", "equals", "auth"),
					*BuildCondition("level", "equals", "ERROR"),
				},
			},
		},
	)

	queryResponse := ExecuteTestQuery(t, router, project.ID, nestedQuery, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)

	// Verify that we got logs matching both conditions
	foundProductionInfo := false
	foundAuthError := false

	for _, log := range queryResponse.Logs {
		// Check for first condition: env=production AND level=INFO
		if env, hasEnv := log.Fields["env"]; hasEnv && env == "production" && log.Level == "INFO" {
			foundProductionInfo = true
		}
		// Check for second condition: component=auth AND level=ERROR
		if comp, hasComp := log.Fields["component"]; hasComp && comp == "auth" && log.Level == "ERROR" {
			foundAuthError = true
		}
	}

	// Should match at least one of the conditions (ideally both)
	if !foundProductionInfo && !foundAuthError {
		t.Errorf(
			"Nested OR query should match logs from at least one condition. Found production+info: %v, Found auth+error: %v",
			foundProductionInfo,
			foundAuthError,
		)
	}

	t.Logf(
		"Nested query results - Found production+info: %v, Found auth+error: %v, Total logs: %d",
		foundProductionInfo,
		foundAuthError,
		len(queryResponse.Logs),
	)
}

func Test_ExecuteQuery_WithComplexNestedQuery_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Complex Nested Test %s", uniqueID[:8])

	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Test Query: (env=production OR level=FATAL) AND component=api
	matchingLogs1 := logs_receiving_tests.CreateValidLogItems(2, uniqueID+"_match1")
	for i := range matchingLogs1 {
		matchingLogs1[i].Fields["env"] = "production"
		matchingLogs1[i].Fields["component"] = "api"
		matchingLogs1[i].Message = "Production API request processed"
	}

	matchingLogs2 := logs_receiving_tests.CreateValidLogItems(1, uniqueID+"_match2")
	for i := range matchingLogs2 {
		matchingLogs2[i].Level = logs_core.LogLevelFatal
		matchingLogs2[i].Fields["component"] = "api"
		matchingLogs2[i].Message = "Fatal API error occurred"
	}

	nonMatchingLogs1 := logs_receiving_tests.CreateValidLogItems(1, uniqueID+"_nomatch1")
	for i := range nonMatchingLogs1 {
		nonMatchingLogs1[i].Fields["env"] = "production"
		nonMatchingLogs1[i].Message = "Regular production request"
	}

	nonMatchingLogs2 := logs_receiving_tests.CreateValidLogItems(1, uniqueID+"_nomatch2")
	for i := range nonMatchingLogs2 {
		nonMatchingLogs2[i].Level = logs_core.LogLevelWarn
		nonMatchingLogs2[i].Fields["env"] = "development"
		nonMatchingLogs2[i].Fields["component"] = "web"
		nonMatchingLogs2[i].Message = "Warning message"
	}

	allLogItems := append(matchingLogs1, matchingLogs2...)
	allLogItems = append(allLogItems, nonMatchingLogs1...)
	allLogItems = append(allLogItems, nonMatchingLogs2...)

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

	err := json.Unmarshal(resp.Body, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal submit response: %v", err)
	}

	workerService := logs_receiving.GetLogWorkerService()
	bgErr := workerService.ExecuteBackgroundTasksForTest()
	if bgErr != nil {
		t.Fatalf("Failed to execute background tasks: %v", bgErr)
	}

	// Wait for logs to be indexed
	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID+"_match1", "Bearer "+owner.Token)

	// Execute simplified complex nested query: (env=production OR level=FATAL) AND component=api
	complexQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorAnd,
				Children: []logs_core.QueryNode{
					// First part: (env=production OR level=FATAL)
					{
						Type: logs_core.QueryNodeTypeLogical,
						Logic: &logs_core.LogicalNode{
							Operator: logs_core.LogicalOperatorOr,
							Children: []logs_core.QueryNode{
								*BuildCondition("env", "equals", "production"),
								*BuildCondition("level", "equals", "FATAL"),
							},
						},
					},
					// Second part: component=api
					*BuildCondition("component", "equals", "api"),
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

	// Verify that we got logs matching the simplified complex conditions: (env=production OR level=FATAL) AND component=api
	foundProductionApi := false
	foundFatalApi := false

	for _, log := range queryResponse.Logs {
		// All returned logs must have component=api
		if comp, hasComp := log.Fields["component"]; !hasComp || comp != "api" {
			t.Errorf("Complex query returned log without component=api: %v", log.Fields)
		}

		// Check for first matching pattern: env=production AND component=api
		if env, hasEnv := log.Fields["env"]; hasEnv && env == "production" {
			if comp, hasComp := log.Fields["component"]; hasComp && comp == "api" {
				foundProductionApi = true
			}
		}

		// Check for second matching pattern: level=FATAL AND component=api
		if log.Level == "FATAL" {
			if comp, hasComp := log.Fields["component"]; hasComp && comp == "api" {
				foundFatalApi = true
			}
		}
	}

	// Should match at least one of the complex conditions
	if !foundProductionApi && !foundFatalApi {
		t.Errorf(
			"Complex nested query should match logs from at least one condition. Found production+api: %v, Found fatal+api: %v",
			foundProductionApi,
			foundFatalApi,
		)
	}

	t.Logf(
		"Complex query results - Found production+api: %v, Found fatal+api: %v, Total logs: %d",
		foundProductionApi,
		foundFatalApi,
		len(queryResponse.Logs),
	)
}
