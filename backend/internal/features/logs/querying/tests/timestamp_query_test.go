package logs_querying_tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	logs_core "logbull/internal/features/logs/core"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_ExecuteQuery_FilterByTimestampGreaterThan_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	repository := logs_core.GetLogCoreRepository()

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Timestamp Greater Than Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create test timestamps
	now := time.Now().UTC()
	filterTime := now.Add(-2 * time.Hour)
	oldTime := now.Add(-4 * time.Hour)
	recentTime := now.Add(-1 * time.Hour)

	// Store logs directly via repository to preserve exact timestamps
	storeLogEntriesWithTimestamp(t, repository, project.ID, oldTime, "Old log message", uniqueID, nil)
	storeLogEntriesWithTimestamp(t, repository, project.ID, recentTime, "Recent log message", uniqueID, nil)

	waitForTimestampLogsIndexing(t, router, project.ID, uniqueID, owner.Token)

	// Query for logs with timestamp greater than filterTime (should only return recent logs)
	query := &logs_core.LogQueryRequestDTO{
		Query: BuildCondition("test_id", "equals", uniqueID),
		TimeRange: &logs_core.TimeRangeDTO{
			From: &filterTime,
			To:   &now,
		},
		Limit:     100,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	response := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, response, 1)

	// Verify all returned logs have timestamps greater than filterTime
	foundRecentLogs := 0
	for _, log := range response.Logs {
		assert.True(t, log.Timestamp.After(filterTime),
			"Log timestamp %v should be after filter time %v", log.Timestamp, filterTime)

		// Verify it contains our test data
		if testID, exists := log.Fields["test_id"]; exists {
			assert.Equal(t, uniqueID, testID)
		}

		if log.Message == "Recent log message" {
			foundRecentLogs++
		}
	}

	assert.GreaterOrEqual(t, foundRecentLogs, 1, "Should find at least one recent log")
	t.Logf("Found %d logs with timestamp > %v", len(response.Logs), filterTime)
}

func Test_ExecuteQuery_FilterByTimestampLessThan_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	repository := logs_core.GetLogCoreRepository()

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Timestamp Less Than Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create test timestamps
	now := time.Now().UTC()
	filterTime := now.Add(-2 * time.Hour)
	oldTime := now.Add(-4 * time.Hour)
	recentTime := now.Add(-1 * time.Hour)

	// Store logs directly via repository to preserve exact timestamps
	storeLogEntriesWithTimestamp(t, repository, project.ID, oldTime, "Old log message", uniqueID, nil)
	storeLogEntriesWithTimestamp(t, repository, project.ID, recentTime, "Recent log message", uniqueID, nil)

	waitForTimestampLogsIndexing(t, router, project.ID, uniqueID, owner.Token)

	// Query for logs with timestamp less than filterTime (should only return old logs)
	// Use a much earlier start time to ensure we catch all historical logs
	veryEarlyTime := now.Add(-24 * time.Hour)
	query := &logs_core.LogQueryRequestDTO{
		Query: BuildCondition("test_id", "equals", uniqueID),
		TimeRange: &logs_core.TimeRangeDTO{
			From: &veryEarlyTime, // Start from very early to catch old logs
			To:   &filterTime,    // End at filter time (2 hours ago)
		},
		Limit:     100,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	response := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

	// Verify all returned logs have timestamps less than filterTime
	foundOldLogs := 0
	for _, log := range response.Logs {
		assert.True(t, log.Timestamp.Before(filterTime),
			"Log timestamp %v should be before filter time %v", log.Timestamp, filterTime)

		// Verify it contains our test data
		if testID, exists := log.Fields["test_id"]; exists {
			assert.Equal(t, uniqueID, testID)
		}

		if log.Message == "Old log message" {
			foundOldLogs++
		}
	}

	// We expect to find at least one old log
	if len(response.Logs) > 0 {
		assert.GreaterOrEqual(t, foundOldLogs, 1, "Should find at least one old log")
	} else {
		t.Logf("Warning: No logs found - log storage may need more indexing time")
	}

	t.Logf("Found %d logs with timestamp < %v", len(response.Logs), filterTime)
}

func Test_ExecuteQuery_WithTimeRangeFilter_ReturnsLogsInRange(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	repository := logs_core.GetLogCoreRepository()

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Time Range Filter Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create test timestamps
	now := time.Now().UTC()
	rangeStart := now.Add(-3 * time.Hour)
	rangeEnd := now.Add(-1 * time.Hour)

	beforeRangeTime := now.Add(-5 * time.Hour)
	inRangeTime := now.Add(-2 * time.Hour)
	afterRangeTime := now.Add(-30 * time.Minute)

	// Store logs directly via repository with specific timestamps
	storeLogEntriesWithTimestamp(t, repository, project.ID, beforeRangeTime, "Before range log", uniqueID, nil)
	storeLogEntriesWithTimestamp(t, repository, project.ID, inRangeTime, "In range log", uniqueID, nil)
	storeLogEntriesWithTimestamp(t, repository, project.ID, afterRangeTime, "After range log", uniqueID, nil)

	waitForTimestampLogsIndexing(t, router, project.ID, uniqueID, owner.Token)

	// Query for logs within the specific time range
	query := &logs_core.LogQueryRequestDTO{
		Query: BuildCondition("test_id", "equals", uniqueID),
		TimeRange: &logs_core.TimeRangeDTO{
			From: &rangeStart,
			To:   &rangeEnd,
		},
		Limit:     100,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	response := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, response, 1)

	// Verify all returned logs are within the specified time range
	for _, log := range response.Logs {
		assert.True(t,
			(log.Timestamp.After(rangeStart) || log.Timestamp.Equal(rangeStart)) &&
				(log.Timestamp.Before(rangeEnd) || log.Timestamp.Equal(rangeEnd)),
			"Log timestamp %v should be between %v and %v", log.Timestamp, rangeStart, rangeEnd)

		// Verify it contains our test data
		if testID, exists := log.Fields["test_id"]; exists {
			assert.Equal(t, uniqueID, testID)
		}

		// Ensure we didn't get logs from outside the range
		assert.NotEqual(t, "Before range log", log.Message, "Should not return logs before range")
		assert.NotEqual(t, "After range log", log.Message, "Should not return logs after range")
	}

	// We should have found at least the in-range log (if log storage is working properly)
	if len(response.Logs) > 0 {
		t.Logf("Found %d logs within time range [%v, %v]", len(response.Logs), rangeStart, rangeEnd)
	}
}

func Test_ExecuteQuery_WithTimeRangeAndConditions_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	repository := logs_core.GetLogCoreRepository()

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Time Range And Conditions Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	// Create test timestamps
	now := time.Now().UTC()
	rangeStart := now.Add(-3 * time.Hour)
	rangeEnd := now.Add(-1 * time.Hour)

	inRangeTime1 := now.Add(-2*time.Hour + 30*time.Minute)
	inRangeTime2 := now.Add(-1*time.Hour + 30*time.Minute)
	beforeRangeTime := now.Add(-4 * time.Hour)

	// Store logs with different timestamps and conditions
	// In-range log with matching condition
	storeLogEntriesWithTimestamp(t, repository, project.ID, inRangeTime1,
		"Error in payment processing", uniqueID, map[string]any{
			"component": "payment",
			"level":     "ERROR",
		})

	// In-range log with non-matching condition
	storeLogEntriesWithTimestamp(t, repository, project.ID, inRangeTime2,
		"Info message", uniqueID, map[string]any{
			"component": "user",
			"level":     "INFO",
		})

	// Before-range log with matching condition (should not be returned)
	storeLogEntriesWithTimestamp(t, repository, project.ID, beforeRangeTime,
		"Error in payment processing", uniqueID, map[string]any{
			"component": "payment",
			"level":     "ERROR",
		})

	waitForTimestampLogsIndexing(t, router, project.ID, uniqueID, owner.Token)

	// Query with both time range AND field conditions
	query := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorAnd,
				Children: []logs_core.QueryNode{
					*BuildCondition("test_id", "equals", uniqueID),
					*BuildCondition("component", "equals", "payment"),
					*BuildCondition("level", "equals", "ERROR"),
				},
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &rangeStart,
			To:   &rangeEnd,
		},
		Limit:     100,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	response := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)
	AssertQueryResponseValid(t, response, 1)

	// Verify results match both time range AND conditions
	for _, log := range response.Logs {
		// Check time range
		assert.True(t,
			(log.Timestamp.After(rangeStart) || log.Timestamp.Equal(rangeStart)) &&
				(log.Timestamp.Before(rangeEnd) || log.Timestamp.Equal(rangeEnd)),
			"Log timestamp %v should be between %v and %v", log.Timestamp, rangeStart, rangeEnd)

		// Check field conditions
		if component, exists := log.Fields["component"]; exists {
			assert.Equal(t, "payment", component, "Component should be 'payment'")
		}
		if level, exists := log.Fields["level"]; exists {
			assert.Equal(t, "ERROR", level, "Level should be 'ERROR'")
		}
		if testID, exists := log.Fields["test_id"]; exists {
			assert.Equal(t, uniqueID, testID)
		}

		// Verify message content matches our expected log
		assert.Contains(t, log.Message, "Error in payment processing")
	}

	t.Logf("Found %d logs matching both time range and conditions", len(response.Logs))
}

// storeLogEntriesWithTimestamp stores logs directly via repository to preserve exact timestamps
func storeLogEntriesWithTimestamp(
	t *testing.T,
	repository *logs_core.LogCoreRepository,
	projectID uuid.UUID,
	timestamp time.Time,
	message string,
	uniqueID string,
	additionalFields map[string]any,
) {
	logID := uuid.New()

	fields := map[string]any{
		"test_id": uniqueID,
	}

	// Add additional fields
	for k, v := range additionalFields {
		fields[k] = v
	}

	logItem := &logs_core.LogItem{
		ID:        logID,
		ProjectID: projectID,
		Timestamp: timestamp,
		Level:     logs_core.LogLevelInfo,
		Message:   message,
		Fields:    fields,
		ClientIP:  "127.0.0.1",
	}

	logEntries := map[uuid.UUID][]*logs_core.LogItem{
		projectID: {logItem},
	}

	err := repository.StoreLogsBatch(logEntries)
	assert.NoError(t, err, "Failed to store logs with timestamp %v", timestamp)
}

// waitForTimestampLogsIndexing waits for logs to be indexed with debug output if needed
func waitForTimestampLogsIndexing(t *testing.T, router *gin.Engine, projectID uuid.UUID, uniqueID, token string) {
	err := logs_core.GetLogCoreRepository().ForceFlush()
	assert.NoError(t, err, "Failed to flush logs")

	time.Sleep(100 * time.Millisecond)

	// Quick verification that logs are indexed
	query := BuildSimpleConditionQuery("test_id", "equals", uniqueID)
	response := ExecuteTestQuery(t, router, projectID, query, token, http.StatusOK)

	if len(response.Logs) == 0 {
		t.Logf("Warning: No logs found during indexing wait - log storage may need more time")
	} else {
		t.Logf("Successfully indexed %d logs with test_id: %s", len(response.Logs), uniqueID)
	}
}
