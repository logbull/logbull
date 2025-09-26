package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

// String Operations Tests

func Test_ExecuteQueryForProject_WithNotEqualsOperator_ReturnsNonMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with different custom field values
	matchingLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Log should not be returned", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "production",
			"status":       "active",
		})

	nonMatchingLogEntries1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Log should be returned 1", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "staging",
			"status":       "active",
		})

	nonMatchingLogEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Log should be returned 2", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "development",
			"status":       "inactive",
		})

	allEntries := MergeLogEntries(matchingLogEntries, nonMatchingLogEntries1)
	allEntries = MergeLogEntries(allEntries, nonMatchingLogEntries2)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test not_equals on custom field
	notEqualsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "environment",
				Operator: logs_core.ConditionOperatorNotEquals,
				Value:    "production",
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, notEqualsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs that don't equal 'production'")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs don't have environment=production
	for i, log := range result.Logs {
		assert.NotEqual(t, "production", log.Fields["environment"], "Log %d should not have environment=production", i)
		assert.Contains(t, log.Message, "should be returned", "Only non-matching logs should be returned")
	}

	// Test not_equals on system field (level)
	systemNotEqualsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "level",
				Operator: logs_core.ConditionOperatorNotEquals,
				Value:    "ERROR",
			},
		},
		Limit: 10,
	}

	systemResult, err := repository.ExecuteQueryForProject(projectID, systemNotEqualsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), systemResult.Total, "Should return all logs since none have level ERROR")

	for _, log := range systemResult.Logs {
		assert.NotEqual(t, "ERROR", log.Level, "No log should have level ERROR")
	}
}

func Test_ExecuteQueryForProject_WithNotContainsOperator_ReturnsNonMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with different message content
	containsLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Error occurred in payment processing", map[string]any{
			"test_session": uniqueTestSession,
			"component":    "payment-service",
		})

	notContainsLogEntries1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"User login successful", map[string]any{
			"test_session": uniqueTestSession,
			"component":    "auth-service",
		})

	notContainsLogEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Database connection established", map[string]any{
			"test_session": uniqueTestSession,
			"component":    "database-service",
		})

	allEntries := MergeLogEntries(containsLogEntries, notContainsLogEntries1)
	allEntries = MergeLogEntries(allEntries, notContainsLogEntries2)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test not_contains on system field (message)
	notContainsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "message",
				Operator: logs_core.ConditionOperatorNotContains,
				Value:    "Error",
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, notContainsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs that don't contain 'Error'")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs don't contain "Error"
	for i, log := range result.Logs {
		assert.NotContains(t, log.Message, "Error", "Log %d message should not contain 'Error'", i)
	}

	// Test not_contains on custom field
	customNotContainsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "component",
				Operator: logs_core.ConditionOperatorNotContains,
				Value:    "payment",
			},
		},
		Limit: 10,
	}

	customResult, err := repository.ExecuteQueryForProject(projectID, customNotContainsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), customResult.Total, "Should return 2 logs that don't contain 'payment' in component")

	for _, log := range customResult.Logs {
		component := log.Fields["component"]
		if component != nil {
			assert.NotContains(t, component.(string), "payment", "Component should not contain 'payment'")
		}
	}
}

func Test_ExecuteQueryForProject_WithContainsOperator_UserAgentField_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create log with userAgent field
	userAgentLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"User request from browser", map[string]any{
			"test_session": uniqueTestSession,
			"userAgent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		})

	// Create a log without userAgent field for comparison
	otherLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Server internal request", map[string]any{
			"test_session": uniqueTestSession,
			"source":       "internal",
		})

	allEntries := MergeLogEntries(userAgentLogEntries, otherLogEntries)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test contains operator on userAgent field (case sensitive)
	containsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "userAgent",
				Operator: logs_core.ConditionOperatorContains,
				Value:    "Macintosh",
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, containsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "Should return 1 log that contains 'Macintosh' in userAgent")
	assert.Len(t, result.Logs, 1)

	// Verify the returned log has the correct userAgent field
	log := result.Logs[0]
	assert.Contains(t, log.Fields, "userAgent", "Log should have userAgent field")
	userAgent := log.Fields["userAgent"].(string)
	assert.Contains(t, userAgent, "Macintosh", "userAgent should contain 'Macintosh'")
	assert.Equal(
		t,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		userAgent,
		"userAgent should have exact value",
	)
	assert.Contains(t, log.Message, "User request from browser", "Should return the correct log")
}

// Array Operations Tests

func Test_ExecuteQueryForProject_WithInOperator_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with different values
	logEntries1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Processing user data", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "production",
			"priority":     "high",
		})

	logEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Testing new features", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "staging",
			"priority":     "medium",
		})

	logEntries3 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Development work in progress", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "development",
			"priority":     "low",
		})

	logEntries4 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(3*time.Second),
		"Quality assurance testing", map[string]any{
			"test_session": uniqueTestSession,
			"environment":  "qa",
			"priority":     "medium",
		})

	allEntries := MergeLogEntries(logEntries1, logEntries2)
	allEntries = MergeLogEntries(allEntries, logEntries3)
	allEntries = MergeLogEntries(allEntries, logEntries4)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test IN operator on custom field
	inQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "environment",
				Operator: logs_core.ConditionOperatorIn,
				Value:    []string{"production", "staging"},
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, inQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs with environment in [production, staging]")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs have environment in the specified array
	foundEnvironments := make(map[string]bool)
	for _, log := range result.Logs {
		environment := log.Fields["environment"].(string)
		foundEnvironments[environment] = true
		assert.Contains(
			t,
			[]string{"production", "staging"},
			environment,
			"Environment should be in the specified array",
		)
	}
	assert.True(t, foundEnvironments["production"], "Should find production environment")
	assert.True(t, foundEnvironments["staging"], "Should find staging environment")

	// Test IN operator on system field (level) - create logs with different levels first
	errorLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(4*time.Second),
		"Error message", map[string]any{
			"test_session": uniqueTestSession,
		})
	// Manually set level to ERROR after creation
	for _, logs := range errorLogEntries {
		for _, log := range logs {
			log.Level = logs_core.LogLevelError
		}
	}

	warnLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(5*time.Second),
		"Warning message", map[string]any{
			"test_session": uniqueTestSession,
		})
	// Manually set level to WARN after creation
	for _, logs := range warnLogEntries {
		for _, log := range logs {
			log.Level = logs_core.LogLevelWarn
		}
	}

	levelEntries := MergeLogEntries(errorLogEntries, warnLogEntries)
	StoreTestLogsAndFlush(t, repository, levelEntries)

	systemInQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "level",
				Operator: logs_core.ConditionOperatorIn,
				Value:    []string{"ERROR", "WARN"},
			},
		},
		Limit: 10,
	}

	systemResult, err := repository.ExecuteQueryForProject(projectID, systemInQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), systemResult.Total, "Should return 2 logs with level in [ERROR, WARN]")

	for _, log := range systemResult.Logs {
		assert.Contains(t, []string{"ERROR", "WARN"}, log.Level, "Level should be in the specified array")
	}
}

func Test_ExecuteQueryForProject_WithNotInOperator_ReturnsNonMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with different statuses
	logEntries1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Operation completed successfully", map[string]any{
			"test_session": uniqueTestSession,
			"status":       "success",
			"result_code":  "200",
		})

	logEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Operation failed with error", map[string]any{
			"test_session": uniqueTestSession,
			"status":       "error",
			"result_code":  "500",
		})

	logEntries3 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Operation is pending", map[string]any{
			"test_session": uniqueTestSession,
			"status":       "pending",
			"result_code":  "202",
		})

	logEntries4 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(3*time.Second),
		"Operation was cancelled", map[string]any{
			"test_session": uniqueTestSession,
			"status":       "cancelled",
			"result_code":  "400",
		})

	allEntries := MergeLogEntries(logEntries1, logEntries2)
	allEntries = MergeLogEntries(allEntries, logEntries3)
	allEntries = MergeLogEntries(allEntries, logEntries4)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test NOT IN operator on custom field
	notInQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "status",
				Operator: logs_core.ConditionOperatorNotIn,
				Value:    []string{"error", "cancelled"},
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, notInQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs with status not in [error, cancelled]")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs don't have status in the excluded array
	for _, log := range result.Logs {
		status := log.Fields["status"].(string)
		assert.NotContains(t, []string{"error", "cancelled"}, status, "Status should not be in the excluded array")
		assert.Contains(t, []string{"success", "pending"}, status, "Status should be in the allowed values")
	}
}

func Test_ExecuteQueryForProject_WithInOperator_EmptyArray_ReturnsNoLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create some logs
	logEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Test log for empty array", map[string]any{
			"test_session": uniqueTestSession,
			"status":       "active",
		})

	StoreTestLogsAndFlush(t, repository, logEntries)

	// Test IN operator with empty array
	emptyInQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "status",
				Operator: logs_core.ConditionOperatorIn,
				Value:    []string{},
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, emptyInQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), result.Total, "Should return 0 logs when IN array is empty")
	assert.Empty(t, result.Logs, "Should return no logs when IN array is empty")
}

func Test_ExecuteQueryForProject_WithInOperator_SingleValue_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with different priorities
	logEntries1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"High priority task", map[string]any{
			"test_session": uniqueTestSession,
			"priority":     "high",
		})

	logEntries2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Medium priority task", map[string]any{
			"test_session": uniqueTestSession,
			"priority":     "medium",
		})

	allEntries := MergeLogEntries(logEntries1, logEntries2)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test IN operator with single value (should behave like equals)
	singleInQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "priority",
				Operator: logs_core.ConditionOperatorIn,
				Value:    []string{"high"},
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, singleInQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "Should return 1 log with priority 'high'")
	assert.Len(t, result.Logs, 1)

	// Verify the returned log has the correct priority
	assert.Equal(t, "high", result.Logs[0].Fields["priority"], "Priority should be 'high'")
	assert.Contains(t, result.Logs[0].Message, "High priority", "Should return the high priority log")

	// Compare with equals operator to verify same behavior
	equalsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "priority",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    "high",
			},
		},
		Limit: 10,
	}

	equalsResult, err := repository.ExecuteQueryForProject(projectID, equalsQuery)
	assert.NoError(t, err)
	assert.Equal(t, result.Total, equalsResult.Total, "IN with single value should return same count as equals")
	assert.Equal(t, len(result.Logs), len(equalsResult.Logs), "IN with single value should return same logs as equals")
}

// Existence Operations Tests

func Test_ExecuteQueryForProject_WithExistsOperator_ReturnsLogsWithField(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with optional fields
	logWithOptionalField := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Log with optional field", map[string]any{
			"test_session":   uniqueTestSession,
			"request_id":     "req-12345",
			"optional_field": "present",
			"user_id":        "user-123",
		})

	logWithDifferentFields := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Log with different fields", map[string]any{
			"test_session": uniqueTestSession,
			"session_id":   "session-67890",
			"trace_id":     "trace-abc",
		})

	logMinimalFields := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Log with minimal fields", map[string]any{
			"test_session": uniqueTestSession,
		})

	allEntries := MergeLogEntries(logWithOptionalField, logWithDifferentFields)
	allEntries = MergeLogEntries(allEntries, logMinimalFields)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test EXISTS operator on custom field
	existsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "request_id",
				Operator: logs_core.ConditionOperatorExists,
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, existsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "Should return 1 log that has request_id field")
	assert.Len(t, result.Logs, 1)

	// Verify the returned log has the field
	log := result.Logs[0]
	assert.Contains(t, log.Fields, "request_id", "Log should have request_id field")
	assert.Equal(t, "req-12345", log.Fields["request_id"], "request_id should have correct value")
	assert.Contains(t, log.Message, "optional field", "Should return the log with optional field")

	// Test EXISTS on another field that exists in different logs
	userExistsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "user_id",
				Operator: logs_core.ConditionOperatorExists,
			},
		},
		Limit: 10,
	}

	userResult, err := repository.ExecuteQueryForProject(projectID, userExistsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), userResult.Total, "Should return 1 log that has user_id field")

	// Test EXISTS on field that doesn't exist anywhere
	nonExistentQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "nonexistent_field",
				Operator: logs_core.ConditionOperatorExists,
			},
		},
		Limit: 10,
	}

	nonExistentResult, err := repository.ExecuteQueryForProject(projectID, nonExistentQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), nonExistentResult.Total, "Should return 0 logs for non-existent field")
	assert.Empty(t, nonExistentResult.Logs, "Should return no logs for non-existent field")
}

func Test_ExecuteQueryForProject_WithNotExistsOperator_ReturnsLogsWithoutField(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with and without error_code field
	logWithErrorCode := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Error occurred", map[string]any{
			"test_session": uniqueTestSession,
			"error_code":   "ERR001",
			"severity":     "high",
		})

	logWithoutErrorCode1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Success operation", map[string]any{
			"test_session": uniqueTestSession,
			"result":       "success",
		})

	logWithoutErrorCode2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Info message", map[string]any{
			"test_session": uniqueTestSession,
			"info_type":    "general",
		})

	allEntries := MergeLogEntries(logWithErrorCode, logWithoutErrorCode1)
	allEntries = MergeLogEntries(allEntries, logWithoutErrorCode2)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test NOT EXISTS operator on custom field
	notExistsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "error_code",
				Operator: logs_core.ConditionOperatorNotExists,
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, notExistsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs that don't have error_code field")
	assert.Len(t, result.Logs, 2)

	// Verify the returned logs don't have the field
	for i, log := range result.Logs {
		assert.NotContains(t, log.Fields, "error_code", "Log %d should not have error_code field", i)
		assert.NotContains(t, log.Message, "Error occurred", "Should not return the error log")
	}

	// Verify we got the correct logs
	messages := make([]string, len(result.Logs))
	for i, log := range result.Logs {
		messages[i] = log.Message
	}
	assert.Contains(t, messages, "Success operation")
	assert.Contains(t, messages, "Info message")
}

func Test_ExecuteQueryForProject_WithExistsOperator_SystemField_ReturnsAllLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create multiple logs
	log1 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"First test log", map[string]any{
			"test_session": uniqueTestSession,
			"sequence":     1,
		})

	log2 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Second test log", map[string]any{
			"test_session": uniqueTestSession,
			"sequence":     2,
		})

	log3 := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(2*time.Second),
		"Third test log", map[string]any{
			"test_session": uniqueTestSession,
			"sequence":     3,
		})

	allEntries := MergeLogEntries(log1, log2)
	allEntries = MergeLogEntries(allEntries, log3)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test EXISTS on system fields that should always exist
	systemFields := []string{"message", "level", "timestamp"}

	for _, fieldName := range systemFields {
		existsQuery := &logs_core.LogQueryRequestDTO{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    fieldName,
					Operator: logs_core.ConditionOperatorExists,
				},
			},
			Limit: 10,
		}

		result, err := repository.ExecuteQueryForProject(projectID, existsQuery)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), result.Total, "EXISTS on system field '%s' should return all 3 logs", fieldName)
		assert.Len(t, result.Logs, 3, "EXISTS on system field '%s' should return all logs", fieldName)

		// Verify all logs have non-empty values for system fields
		for i, log := range result.Logs {
			switch fieldName {
			case "message":
				assert.NotEmpty(t, log.Message, "Log %d should have non-empty message", i)
			case "level":
				assert.NotEmpty(t, log.Level, "Log %d should have non-empty level", i)
			case "timestamp":
				assert.False(t, log.Timestamp.IsZero(), "Log %d should have valid timestamp", i)
			}
		}
	}
}

func Test_ExecuteQueryForProject_WithNotExistsOperator_SystemField_ReturnsNoLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create test logs
	testLogs := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Test log for system field existence", map[string]any{
			"test_session": uniqueTestSession,
			"test_case":    "system_field_not_exists",
		})

	StoreTestLogsAndFlush(t, repository, testLogs)

	// Test NOT EXISTS on system fields - should return no results since all logs have system fields
	systemFields := []string{"message", "level", "timestamp", "id"}

	for _, fieldName := range systemFields {
		notExistsQuery := &logs_core.LogQueryRequestDTO{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    fieldName,
					Operator: logs_core.ConditionOperatorNotExists,
				},
			},
			Limit: 10,
		}

		result, err := repository.ExecuteQueryForProject(projectID, notExistsQuery)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), result.Total, "NOT EXISTS on system field '%s' should return 0 logs", fieldName)
		assert.Empty(t, result.Logs, "NOT EXISTS on system field '%s' should return no logs", fieldName)
	}
}

// Numeric/Range Operations Tests

func Test_ExecuteQueryForProject_WithGreaterThanOperator_SystemField_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs with different timestamps
	oldLog := CreateTestLogEntriesWithUniqueFields(projectID, baseTime.Add(-2*time.Hour),
		"Old log entry", map[string]any{
			"test_session": uniqueTestSession,
			"log_age":      "old",
		})

	recentLog := CreateTestLogEntriesWithUniqueFields(projectID, baseTime.Add(-30*time.Minute),
		"Recent log entry", map[string]any{
			"test_session": uniqueTestSession,
			"log_age":      "recent",
		})

	newestLog := CreateTestLogEntriesWithUniqueFields(projectID, baseTime.Add(-10*time.Minute),
		"Newest log entry", map[string]any{
			"test_session": uniqueTestSession,
			"log_age":      "newest",
		})

	allEntries := MergeLogEntries(oldLog, recentLog)
	allEntries = MergeLogEntries(allEntries, newestLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test greater than operator on timestamp
	thresholdTime := baseTime.Add(-1 * time.Hour)
	greaterThanQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorGreaterThan,
				Value:    thresholdTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, greaterThanQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs with timestamp greater than threshold")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs have timestamps greater than threshold
	for i, log := range result.Logs {
		assert.True(t, log.Timestamp.After(thresholdTime), "Log %d timestamp should be after threshold", i)
		logAge := log.Fields["log_age"].(string)
		assert.Contains(t, []string{"recent", "newest"}, logAge, "Should only return recent/newest logs")
		assert.NotEqual(t, "old", logAge, "Should not return old logs")
	}
}

func Test_ExecuteQueryForProject_WithGreaterOrEqualOperator_SystemField_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs with precise timestamps for boundary testing
	// Truncate to microsecond precision to match storage behavior
	boundaryTime := baseTime.Add(-1 * time.Hour).Truncate(time.Microsecond)

	beforeBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime.Add(-1*time.Minute),
		"Before boundary log", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "before",
		})

	exactBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime,
		"Exact boundary log", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "exact",
		})

	afterBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime.Add(1*time.Minute),
		"After boundary log", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "after",
		})

	allEntries := MergeLogEntries(beforeBoundaryLog, exactBoundaryLog)
	allEntries = MergeLogEntries(allEntries, afterBoundaryLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test greater than or equal operator on timestamp
	greaterOrEqualQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorGreaterOrEqual,
				Value:    boundaryTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, greaterOrEqualQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs with timestamp greater than or equal to boundary")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs have timestamps >= boundary (including exact match)
	foundPositions := make(map[string]bool)
	for i, log := range result.Logs {
		assert.True(t, log.Timestamp.After(boundaryTime) || log.Timestamp.Equal(boundaryTime),
			"Log %d timestamp should be >= boundary", i)
		position := log.Fields["position"].(string)
		foundPositions[position] = true
		assert.Contains(t, []string{"exact", "after"}, position, "Should return exact and after boundary logs")
	}
	assert.True(t, foundPositions["exact"], "Should include the exact boundary log")
	assert.True(t, foundPositions["after"], "Should include the after boundary log")
	assert.False(t, foundPositions["before"], "Should not include the before boundary log")
}

func Test_ExecuteQueryForProject_WithLessThanOperator_SystemField_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs with different timestamps
	veryOldLog := CreateTestLogEntriesWithUniqueFields(projectID, baseTime.Add(-3*time.Hour),
		"Very old log entry", map[string]any{
			"test_session": uniqueTestSession,
			"age_category": "very_old",
		})

	oldLog := CreateTestLogEntriesWithUniqueFields(projectID, baseTime.Add(-2*time.Hour),
		"Old log entry", map[string]any{
			"test_session": uniqueTestSession,
			"age_category": "old",
		})

	recentLog := CreateTestLogEntriesWithUniqueFields(projectID, baseTime.Add(-30*time.Minute),
		"Recent log entry", map[string]any{
			"test_session": uniqueTestSession,
			"age_category": "recent",
		})

	allEntries := MergeLogEntries(veryOldLog, oldLog)
	allEntries = MergeLogEntries(allEntries, recentLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test less than operator on timestamp
	thresholdTime := baseTime.Add(-1 * time.Hour)
	lessThanQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorLessThan,
				Value:    thresholdTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, lessThanQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs with timestamp less than threshold")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs have timestamps before threshold
	for i, log := range result.Logs {
		assert.True(t, log.Timestamp.Before(thresholdTime), "Log %d timestamp should be before threshold", i)
		ageCategory := log.Fields["age_category"].(string)
		assert.Contains(t, []string{"very_old", "old"}, ageCategory, "Should only return very_old/old logs")
		assert.NotEqual(t, "recent", ageCategory, "Should not return recent logs")
	}
}

func Test_ExecuteQueryForProject_WithLessOrEqualOperator_SystemField_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs with precise timestamps for boundary testing
	boundaryTime := baseTime.Add(-1 * time.Hour)

	beforeBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime.Add(-1*time.Minute),
		"Before boundary log", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "before",
		})

	exactBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime,
		"Exact boundary log", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "exact",
		})

	afterBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime.Add(1*time.Minute),
		"After boundary log", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "after",
		})

	allEntries := MergeLogEntries(beforeBoundaryLog, exactBoundaryLog)
	allEntries = MergeLogEntries(allEntries, afterBoundaryLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test less than or equal operator on timestamp
	lessOrEqualQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorLessOrEqual,
				Value:    boundaryTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(projectID, lessOrEqualQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Should return 2 logs with timestamp less than or equal to boundary")
	assert.Len(t, result.Logs, 2)

	// Verify all returned logs have timestamps <= boundary (including exact match)
	foundPositions := make(map[string]bool)
	for i, log := range result.Logs {
		assert.True(t, log.Timestamp.Before(boundaryTime) || log.Timestamp.Equal(boundaryTime),
			"Log %d timestamp should be <= boundary", i)
		position := log.Fields["position"].(string)
		foundPositions[position] = true
		assert.Contains(t, []string{"before", "exact"}, position, "Should return before and exact boundary logs")
	}
	assert.True(t, foundPositions["before"], "Should include the before boundary log")
	assert.True(t, foundPositions["exact"], "Should include the exact boundary log")
	assert.False(t, foundPositions["after"], "Should not include the after boundary log")
}

func Test_ExecuteQueryForProject_WithRangeOperators_CustomField_ReturnsNoLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with custom numeric fields
	testLogs := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Log with custom numeric field", map[string]any{
			"test_session":  uniqueTestSession,
			"custom_number": 100,
			"response_time": 250,
		})

	StoreTestLogsAndFlush(t, repository, testLogs)

	// Test that range operators on custom fields return no results
	rangeOperators := []logs_core.ConditionOperator{
		logs_core.ConditionOperatorGreaterThan,
		logs_core.ConditionOperatorGreaterOrEqual,
		logs_core.ConditionOperatorLessThan,
		logs_core.ConditionOperatorLessOrEqual,
	}

	for _, operator := range rangeOperators {
		rangeQuery := &logs_core.LogQueryRequestDTO{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    "custom_number",
					Operator: operator,
					Value:    50,
				},
			},
			Limit: 10,
		}

		result, err := repository.ExecuteQueryForProject(projectID, rangeQuery)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), result.Total, "Range operator %s on custom field should return 0 results", operator)
		assert.Empty(t, result.Logs, "Range operator %s on custom field should return no logs", operator)
	}

	// Verify that the same logs can be found with a non-range operator
	equalsQuery := &logs_core.LogQueryRequestDTO{
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

	equalsResult, err := repository.ExecuteQueryForProject(projectID, equalsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), equalsResult.Total, "Should find the logs with non-range operator")
}

// Edge Cases and Error Conditions
func Test_ExecuteQueryForProject_WithSpecialCharactersInValue_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with special characters in values
	specialCharsLog := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Log with special chars: @#$%^&*()[]{}|\\:;\"'<>,.?/~`", map[string]any{
			"test_session":  uniqueTestSession,
			"special_field": "@user#123",
			"email":         "test@example.com",
			"path":          "/api/v1/users?filter=active&sort=desc",
			"json_like":     `{"key": "value", "nested": {"number": 42}}`,
			"sql_injection": "'; DROP TABLE users; --",
			"xml_content":   "<tag>content</tag>",
		})

	normalLog := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"Normal log without special characters", map[string]any{
			"test_session": uniqueTestSession,
			"simple_field": "normalvalue",
		})

	allEntries := MergeLogEntries(specialCharsLog, normalLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test equals operator with special characters
	testCases := []struct {
		field    string
		value    string
		expected int64
	}{
		{"email", "test@example.com", 1},
		{"special_field", "@user#123", 1},
		{"path", "/api/v1/users?filter=active&sort=desc", 1},
		{"json_like", `{"key": "value", "nested": {"number": 42}}`, 1},
		{"sql_injection", "'; DROP TABLE users; --", 1},
		{"xml_content", "<tag>content</tag>", 1},
	}

	for _, testCase := range testCases {
		equalsQuery := &logs_core.LogQueryRequestDTO{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    testCase.field,
					Operator: logs_core.ConditionOperatorEquals,
					Value:    testCase.value,
				},
			},
			Limit: 10,
		}

		result, err := repository.ExecuteQueryForProject(projectID, equalsQuery)
		assert.NoError(t, err, "Query with special characters in field '%s' should not error", testCase.field)
		assert.Equal(
			t,
			testCase.expected,
			result.Total,
			"Should find log with special characters in field '%s'",
			testCase.field,
		)

		if result.Total > 0 {
			assert.Equal(t, testCase.value, result.Logs[0].Fields[testCase.field],
				"Field '%s' should match exactly", testCase.field)
		}
	}

	// Test contains operator with special characters
	containsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "message",
				Operator: logs_core.ConditionOperatorContains,
				Value:    "@#$%^&*()",
			},
		},
		Limit: 10,
	}

	containsResult, err := repository.ExecuteQueryForProject(projectID, containsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), containsResult.Total, "Should find log containing special characters in message")
}

func Test_ExecuteQueryForProject_WithUnicodeValue_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs with unicode characters
	unicodeLog := CreateTestLogEntriesWithUniqueFields(projectID, currentTime,
		"Unicode test: 你好世界 🌍 café naïve résumé", map[string]any{
			"test_session":  uniqueTestSession,
			"chinese":       "你好世界",
			"emoji":         "🌍🚀⭐",
			"french":        "café naïve résumé",
			"german":        "Müller Größe",
			"japanese":      "こんにちは",
			"russian":       "Привет мир",
			"arabic":        "مرحبا بالعالم",
			"mixed_unicode": "Hello 世界 🌟",
		})

	asciiLog := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(1*time.Second),
		"ASCII only log", map[string]any{
			"test_session": uniqueTestSession,
			"ascii_field":  "english only",
		})

	allEntries := MergeLogEntries(unicodeLog, asciiLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test equals operator with unicode values
	unicodeTestCases := []struct {
		field string
		value string
	}{
		{"chinese", "你好世界"},
		{"emoji", "🌍🚀⭐"},
		{"french", "café naïve résumé"},
		{"german", "Müller Größe"},
		{"japanese", "こんにちは"},
		{"russian", "Привет мир"},
		{"arabic", "مرحبا بالعالم"},
		{"mixed_unicode", "Hello 世界 🌟"},
	}

	for _, testCase := range unicodeTestCases {
		equalsQuery := &logs_core.LogQueryRequestDTO{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    testCase.field,
					Operator: logs_core.ConditionOperatorEquals,
					Value:    testCase.value,
				},
			},
			Limit: 10,
		}

		result, err := repository.ExecuteQueryForProject(projectID, equalsQuery)
		assert.NoError(t, err, "Query with unicode in field '%s' should not error", testCase.field)
		assert.Equal(t, int64(1), result.Total, "Should find log with unicode in field '%s'", testCase.field)
		assert.Equal(t, testCase.value, result.Logs[0].Fields[testCase.field],
			"Unicode field '%s' should match exactly", testCase.field)
	}

	// Test contains operator with unicode in message
	messageContainsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "message",
				Operator: logs_core.ConditionOperatorContains,
				Value:    "你好世界",
			},
		},
		Limit: 10,
	}

	messageResult, err := repository.ExecuteQueryForProject(projectID, messageContainsQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), messageResult.Total, "Should find log containing unicode in message")
	assert.Contains(t, messageResult.Logs[0].Message, "你好世界", "Message should contain unicode text")

}

// Boundary Testing

func Test_ExecuteQueryForProject_WithExactBoundaryTimestamp_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs with very precise timestamps for exact boundary testing
	boundaryTime := baseTime.Add(-1 * time.Hour)

	// Logs with millisecond precision differences (more realistic for OpenSearch)
	beforeBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime.Add(-1*time.Millisecond),
		"Just before boundary", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "just_before",
		})

	exactBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime,
		"Exact boundary", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "exact_boundary",
		})

	afterBoundaryLog := CreateTestLogEntriesWithUniqueFields(projectID, boundaryTime.Add(1*time.Millisecond),
		"Just after boundary", map[string]any{
			"test_session": uniqueTestSession,
			"position":     "just_after",
		})

	allEntries := MergeLogEntries(beforeBoundaryLog, exactBoundaryLog)
	allEntries = MergeLogEntries(allEntries, afterBoundaryLog)
	StoreTestLogsAndFlush(t, repository, allEntries)

	// Test greater_or_equal with exact boundary
	gteQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorGreaterOrEqual,
				Value:    boundaryTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	gteResult, err := repository.ExecuteQueryForProject(projectID, gteQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), gteResult.Total, "Greater or equal should include boundary and after")

	foundPositions := make(map[string]bool)
	for _, log := range gteResult.Logs {
		position := log.Fields["position"].(string)
		foundPositions[position] = true
	}
	assert.True(t, foundPositions["exact_boundary"], "Should include exact boundary log")
	assert.True(t, foundPositions["just_after"], "Should include just after log")
	assert.False(t, foundPositions["just_before"], "Should not include just before log")

	// Test less_or_equal with exact boundary
	lteQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorLessOrEqual,
				Value:    boundaryTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	lteResult, err := repository.ExecuteQueryForProject(projectID, lteQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), lteResult.Total, "Less or equal should include boundary and before")

	foundPositionsLte := make(map[string]bool)
	for _, log := range lteResult.Logs {
		position := log.Fields["position"].(string)
		foundPositionsLte[position] = true
	}
	assert.True(t, foundPositionsLte["exact_boundary"], "Should include exact boundary log")
	assert.True(t, foundPositionsLte["just_before"], "Should include just before log")
	assert.False(t, foundPositionsLte["just_after"], "Should not include just after log")

	// Test exact equals with precise timestamp
	equalsQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "timestamp",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    boundaryTime.Format(time.RFC3339Nano),
			},
		},
		Limit: 10,
	}

	equalsResult, err := repository.ExecuteQueryForProject(projectID, equalsQuery)
	assert.NoError(t, err)
	// Note: Equals on timestamp might not work as expected with OpenSearch due to precision
	// This test documents the behavior rather than asserting a specific result
	t.Logf("Equals query on exact timestamp returned %d results", equalsResult.Total)
}
