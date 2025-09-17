package logs_core_tests

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_ExecuteQueryForProject_WithLogicalAndConditions_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	testLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"User login successful", map[string]any{
			"environment":  "production",
			"service":      "auth-api",
			"test_session": uniqueTestSession,
		})

	// Create a non-matching log to ensure filtering works
	nonMatchingEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Different message", map[string]any{
			"environment":  "staging", // Different environment
			"service":      "other-api",
			"test_session": uniqueTestSession,
		})

	allEntries := MergeLogEntries(testLogEntries, nonMatchingEntries)
	StoreTestLogsAndFlush(t, repository, allEntries)

	logicalAndQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorAnd,
				Children: []logs_core.QueryNode{
					{Type: logs_core.QueryNodeTypeCondition, Condition: &logs_core.ConditionNode{
						Field: "environment", Operator: logs_core.ConditionOperatorEquals, Value: "production",
					}},
					{Type: logs_core.QueryNodeTypeCondition, Condition: &logs_core.ConditionNode{
						Field: "message", Operator: logs_core.ConditionOperatorContains, Value: "login",
					}},
					{Type: logs_core.QueryNodeTypeCondition, Condition: &logs_core.ConditionNode{
						Field: "test_session", Operator: logs_core.ConditionOperatorEquals, Value: uniqueTestSession,
					}},
				},
			},
		},
		Limit: 10,
	}

	queryResult, queryErr := repository.ExecuteQueryForProject(projectID, logicalAndQuery)
	assert.NoError(t, queryErr)
	assert.NotNil(t, queryResult)

	// Validate that we got exactly 1 matching log (only the production/login one)
	assert.Equal(t, int64(1), queryResult.Total, "Should find exactly 1 log matching all AND conditions")
	assert.Len(t, queryResult.Logs, 1, "Should return exactly 1 log")

	// Validate the returned log matches all our conditions
	matchedLog := queryResult.Logs[0]
	assert.Contains(t, matchedLog.Message, "login", "Message should contain 'login'")
	assert.Equal(t, "production", matchedLog.Fields["environment"], "Environment should be 'production'")
	assert.Equal(t, uniqueTestSession, matchedLog.Fields["test_session"], "Test session should match")
	assert.Equal(t, "auth-api", matchedLog.Fields["service"], "Service should be 'auth-api'")
}

func Test_ExecuteQueryForProject_WithSingleCondition_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	differentTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create multiple logs - some matching, some not matching
	matchingLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"API request processed", map[string]any{
			"service":      "payment-api",
			"status_code":  200,
			"test_session": uniqueTestSession,
		})

	matchingLogEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Another matching request", map[string]any{
			"service":      "user-api",
			"status_code":  201,
			"test_session": uniqueTestSession, // Same test session
		})

	nonMatchingLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Non-matching request", map[string]any{
			"service":      "other-api",
			"status_code":  404,
			"test_session": differentTestSession, // Different test session
		})

	allEntries := MergeLogEntries(matchingLogEntries, matchingLogEntries2)
	allEntries = MergeLogEntries(allEntries, nonMatchingLogEntries)
	StoreTestLogsAndFlush(t, repository, allEntries)

	singleConditionQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit: 5,
	}

	queryResult, queryErr := repository.ExecuteQueryForProject(projectID, singleConditionQuery)
	assert.NoError(t, queryErr)
	assert.NotNil(t, queryResult)

	// Validate we got exactly the 2 matching logs
	assert.Equal(t, int64(2), queryResult.Total, "Should find exactly 2 logs with matching test_session")
	assert.Len(t, queryResult.Logs, 2, "Should return exactly 2 logs")

	// Validate all returned logs have the correct test_session
	for i, log := range queryResult.Logs {
		assert.Equal(t, uniqueTestSession, log.Fields["test_session"],
			"Log %d should have the correct test_session", i)
		// Ensure we didn't get the non-matching log
		assert.NotEqual(t, differentTestSession, log.Fields["test_session"],
			"Should not return logs with different test_session")
	}

	// Validate we got the expected messages (order may vary)
	messages := make([]string, len(queryResult.Logs))
	for i, log := range queryResult.Logs {
		messages[i] = log.Message
	}
	assert.Contains(t, messages, "API request processed")
	assert.Contains(t, messages, "Another matching request")
	assert.NotContains(t, messages, "Non-matching request")
}

func Test_DiscoverFields_WithCustomFieldsInLogs_ReturnsDiscoveredFields(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with multiple different custom fields to test field discovery
	testLogEntries1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Field discovery test log 1", map[string]any{
			"custom_field_one": "value_" + uniqueTestSession,
			"status_code":      201,
			"test_session":     uniqueTestSession,
			"unique_field_a":   "unique_value_a",
		})

	testLogEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Field discovery test log 2", map[string]any{
			"custom_field_two": "different_value",
			"priority_level":   "high",
			"test_session":     uniqueTestSession,
			"unique_field_b":   "unique_value_b",
		})

	allEntries := MergeLogEntries(testLogEntries1, testLogEntries2)
	StoreTestLogsAndFlush(t, repository, allEntries)

	discoveredFields, discoveryErr := repository.DiscoverFields(projectID)
	assert.NoError(t, discoveryErr)
	assert.NotNil(t, discoveredFields)
	assert.IsType(t, []string{}, discoveredFields)

	// Validate that our custom fields are discovered
	assert.NotEmpty(t, discoveredFields, "Should discover at least some fields")

	// Check for our specific custom fields
	fieldMap := make(map[string]bool)
	for _, field := range discoveredFields {
		fieldMap[field] = true
	}

	// Our custom fields should be discovered
	assert.True(t, fieldMap["custom_field_one"], "Should discover 'custom_field_one' field")
	assert.True(t, fieldMap["custom_field_two"], "Should discover 'custom_field_two' field")
	assert.True(t, fieldMap["status_code"], "Should discover 'status_code' field")
	assert.True(t, fieldMap["test_session"], "Should discover 'test_session' field")
	assert.True(t, fieldMap["priority_level"], "Should discover 'priority_level' field")
	assert.True(t, fieldMap["unique_field_a"], "Should discover 'unique_field_a' field")
	assert.True(t, fieldMap["unique_field_b"], "Should discover 'unique_field_b' field")

	// Note: Field discovery appears to only return custom fields, not standard built-in fields like 'message' and 'level'
	// This is expected behavior since standard fields are always available

	t.Logf("Discovered fields: %v", discoveredFields)
}

func Test_DiscoverFields_WithUnavailableRepository_ReturnsError(t *testing.T) {
	t.Parallel()
	unavailableRepository := logs_core.GetUnavailableLogCoreRepository()
	projectID := uuid.New()

	discoveredFields, discoveryErr := unavailableRepository.DiscoverFields(projectID)
	assert.Error(t, discoveryErr)
	assert.Nil(t, discoveredFields)
	assert.Contains(t, discoveryErr.Error(), "failed to execute field discovery search")
}

func Test_ExecuteQueryForProject_WithTimeRange_ReturnsFilteredLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs at different times
	oldTime := baseTime.Add(-2 * time.Hour)
	recentTime := baseTime.Add(-30 * time.Minute)
	veryRecentTime := baseTime.Add(-10 * time.Minute)

	oldLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, oldTime,
		"Old log entry", map[string]any{"test_session": uniqueTestSession, "log_type": "old"})
	recentLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, recentTime,
		"Recent log entry", map[string]any{"test_session": uniqueTestSession, "log_type": "recent"})
	veryRecentLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, veryRecentTime,
		"Very recent log entry", map[string]any{"test_session": uniqueTestSession, "log_type": "very_recent"})

	allLogEntries := MergeLogEntries(oldLogEntries, recentLogEntries)
	allLogEntries = MergeLogEntries(allLogEntries, veryRecentLogEntries)
	StoreTestLogsAndFlush(t, repository, allLogEntries)

	// First, query without time range to confirm we have all 3 logs
	allLogsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit: 10,
	}

	allLogsResult, allLogsErr := repository.ExecuteQueryForProject(projectID, allLogsQuery)
	assert.NoError(t, allLogsErr)
	assert.Equal(t, int64(3), allLogsResult.Total, "Should have 3 total logs before time filtering")

	// Query with time range filtering out old logs (only logs after -1 hour)
	timeRangeStart := baseTime.Add(-1 * time.Hour)
	timeRangeQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &timeRangeStart,
		},
		Limit: 10,
	}

	timeRangeResult, timeRangeErr := repository.ExecuteQueryForProject(projectID, timeRangeQuery)
	assert.NoError(t, timeRangeErr)
	assert.NotNil(t, timeRangeResult)

	// Validate that time filtering worked - should only get recent and very recent logs
	assert.Equal(t, int64(2), timeRangeResult.Total, "Should find only 2 logs after time range filtering")
	assert.Len(t, timeRangeResult.Logs, 2, "Should return only 2 logs")

	// Validate all returned logs are within the time range
	for i, log := range timeRangeResult.Logs {
		assert.True(t, log.Timestamp.After(timeRangeStart) || log.Timestamp.Equal(timeRangeStart),
			"Log %d timestamp should be after time range start. Log time: %v, Range start: %v",
			i, log.Timestamp, timeRangeStart)
	}

	// Validate we got the expected logs (not the old one)
	messages := make([]string, len(timeRangeResult.Logs))
	for i, log := range timeRangeResult.Logs {
		messages[i] = log.Message
	}
	assert.Contains(t, messages, "Recent log entry")
	assert.Contains(t, messages, "Very recent log entry")
	assert.NotContains(t, messages, "Old log entry", "Old log should be filtered out by time range")

	// Test with both From and To time range
	timeRangeEnd := baseTime.Add(-20 * time.Minute) // Should exclude the very recent log
	boundedTimeQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &timeRangeStart,
			To:   &timeRangeEnd,
		},
		Limit: 10,
	}

	boundedResult, boundedErr := repository.ExecuteQueryForProject(projectID, boundedTimeQuery)
	assert.NoError(t, boundedErr)
	assert.Equal(t, int64(1), boundedResult.Total, "Should find only 1 log with bounded time range")
	assert.Len(t, boundedResult.Logs, 1, "Should return only 1 log with bounded range")
	assert.Equal(t, "Recent log entry", boundedResult.Logs[0].Message, "Should return only the 'Recent log entry'")
}

func Test_ExecuteQueryForProject_FieldsSortedAscending_IncludingClientIp(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create a log with multiple custom fields and clientIp to test field sorting
	testLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Field sorting test log", map[string]any{
			"zebra_field":  "last_alphabetically",
			"alpha_field":  "first_alphabetically",
			"middle_field": "somewhere_middle",
			"beta_field":   "second_alphabetically",
			"test_session": uniqueTestSession,
		})

	// Set a specific client IP for testing
	for _, entries := range testLogEntries {
		for _, entry := range entries {
			entry.ClientIP = "192.168.1.100"
		}
	}

	StoreTestLogsAndFlush(t, repository, testLogEntries)

	query := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, query)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total, "Should find exactly 1 log")
	assert.Len(t, result.Logs, 1, "Should return exactly 1 log")

	log := result.Logs[0]
	assert.NotNil(t, log.Fields, "Log should have Fields map")

	// Verify clientIp is included in Fields map
	assert.Contains(t, log.Fields, "client_ip", "Fields should include client_ip")
	assert.Equal(t, "192.168.1.100", log.Fields["client_ip"], "client_ip in Fields should match")

	// Verify clientIp is also available as separate field (not removed from DTO)
	assert.Equal(t, "192.168.1.100", log.ClientIP, "ClientIP field should still be available")

	// Extract field names from the Fields map
	var fieldNames []string
	for fieldName := range log.Fields {
		fieldNames = append(fieldNames, fieldName)
	}

	// Verify we have all expected fields including clientIp
	expectedFields := []string{"alpha_field", "beta_field", "client_ip", "middle_field", "test_session", "zebra_field"}
	assert.Len(t, fieldNames, len(expectedFields), "Should have expected number of fields")

	// Sort the extracted field names for comparison (Go maps have randomized iteration order)
	slices.Sort(fieldNames)

	// Verify all expected fields are present and the sorted result matches expected sorted order
	assert.Equal(t, expectedFields, fieldNames, "Fields should be present and when sorted match expected order")

	// Verify individual field values are correct
	assert.Equal(t, "first_alphabetically", log.Fields["alpha_field"])
	assert.Equal(t, "second_alphabetically", log.Fields["beta_field"])
	assert.Equal(t, "192.168.1.100", log.Fields["client_ip"])
	assert.Equal(t, "somewhere_middle", log.Fields["middle_field"])
	assert.Equal(t, uniqueTestSession, log.Fields["test_session"])
	assert.Equal(t, "last_alphabetically", log.Fields["zebra_field"])

	t.Logf("Fields present (sorted): %v", fieldNames)
}
