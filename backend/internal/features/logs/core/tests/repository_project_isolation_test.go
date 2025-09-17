package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_ExecuteQueryForProject_MultipleProjects_OnlyReturnsRequestedProject(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	// Create different projects with logs
	project1 := uuid.New()
	project2 := uuid.New()
	project3 := uuid.New()

	// Create logs for project1
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Project 1 log message", map[string]any{
			"test_session": uniqueTestSession,
			"project_name": "project_one",
			"priority":     "high",
		})

	// Create logs for project2 with different content
	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime.Add(1*time.Second),
		"Project 2 log message", map[string]any{
			"test_session": uniqueTestSession,
			"project_name": "project_two",
			"priority":     "medium",
		})

	// Create logs for project3
	project3Logs := CreateTestLogEntriesWithUniqueFields(project3, currentTime.Add(2*time.Second),
		"Project 3 log message", map[string]any{
			"test_session": uniqueTestSession,
			"project_name": "project_three",
			"priority":     "low",
		})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	allLogs = MergeLogEntries(allLogs, project3Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Query project1 - should only return project1 logs
	project1Query := &logs_core.LogQueryRequestDTO{
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

	project1Result, err := repository.ExecuteQueryForProject(project1, project1Query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project1Result.Total)
	assert.Len(t, project1Result.Logs, 1)
	assert.Equal(t, "Project 1 log message", project1Result.Logs[0].Message)
	assert.Equal(t, "project_one", project1Result.Logs[0].Fields["project_name"])

	// Query project2 - should only return project2 logs
	project2Result, err := repository.ExecuteQueryForProject(project2, project1Query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project2Result.Total)
	assert.Len(t, project2Result.Logs, 1)
	assert.Equal(t, "Project 2 log message", project2Result.Logs[0].Message)
	assert.Equal(t, "project_two", project2Result.Logs[0].Fields["project_name"])

	// Query project3 - should only return project3 logs
	project3Result, err := repository.ExecuteQueryForProject(project3, project1Query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project3Result.Total)
	assert.Len(t, project3Result.Logs, 1)
	assert.Equal(t, "Project 3 log message", project3Result.Logs[0].Message)
	assert.Equal(t, "project_three", project3Result.Logs[0].Fields["project_name"])
}

func Test_ExecuteQueryForProject_SameLogContent_DifferentProjects_IsolatedCorrectly(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create identical log content in both projects
	identicalMessage := "Identical log message for isolation test"
	identicalFields := map[string]any{
		"test_session": uniqueTestSession,
		"component":    "auth-service",
		"action":       "user_login",
		"status":       "success",
	}

	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime, identicalMessage, identicalFields)
	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime, identicalMessage, identicalFields)

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Query both projects with identical query
	query := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "component",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    "auth-service",
			},
		},
		Limit: 10,
	}

	// Query project1 - should return only project1's log
	project1Result, err := repository.ExecuteQueryForProject(project1, query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project1Result.Total)
	assert.Len(t, project1Result.Logs, 1)

	// Query project2 - should return only project2's log
	project2Result, err := repository.ExecuteQueryForProject(project2, query)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project2Result.Total)
	assert.Len(t, project2Result.Logs, 1)

	// Verify both results have identical content but are from different projects
	assert.Equal(t, project1Result.Logs[0].Message, project2Result.Logs[0].Message)
	assert.Equal(t, project1Result.Logs[0].Fields["component"], project2Result.Logs[0].Fields["component"])
	assert.NotEqual(t, project1Result.Logs[0].ID, project2Result.Logs[0].ID, "Should have different log IDs")
}

func Test_ExecuteQueryForProject_CrossProjectQuery_NeverReturnsOtherProjectLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create logs in project1 with specific content
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Payment processed successfully", map[string]any{
			"test_session": uniqueTestSession,
			"service":      "payment-api",
			"amount":       "100.00",
			"currency":     "USD",
		})

	// Create logs in project2 with content that would match broad queries
	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Payment failed with error", map[string]any{
			"test_session": uniqueTestSession,
			"service":      "payment-api",
			"amount":       "200.00",
			"currency":     "USD",
			"error":        "insufficient_funds",
		})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Broad query that would match logs in both projects
	broadQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "service",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    "payment-api",
			},
		},
		Limit: 10,
	}

	// Query project1 - should only return project1's payment log
	project1Result, err := repository.ExecuteQueryForProject(project1, broadQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project1Result.Total)
	assert.Len(t, project1Result.Logs, 1)
	assert.Contains(t, project1Result.Logs[0].Message, "successfully")
	assert.Equal(t, "100.00", project1Result.Logs[0].Fields["amount"])
	assert.NotContains(t, project1Result.Logs[0].Message, "failed", "Should not contain project2's log")

	// Query project2 - should only return project2's payment log
	project2Result, err := repository.ExecuteQueryForProject(project2, broadQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project2Result.Total)
	assert.Len(t, project2Result.Logs, 1)
	assert.Contains(t, project2Result.Logs[0].Message, "failed")
	assert.Equal(t, "200.00", project2Result.Logs[0].Fields["amount"])
	assert.NotContains(t, project2Result.Logs[0].Message, "successfully", "Should not contain project1's log")
}

func Test_ExecuteQueryForProject_NonExistentProject_ReturnsEmptyResults(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	nonExistentProject := uuid.New()

	query := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "level",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    "info",
			},
		},
		Limit: 10,
	}

	result, err := repository.ExecuteQueryForProject(nonExistentProject, query)
	assert.NoError(t, err, "Querying non-existent project should not return error")
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.Total, "Should return zero logs for non-existent project")
	assert.Empty(t, result.Logs, "Should return empty logs array for non-existent project")
}

func Test_ExecuteQueryForProject_ProjectIdFilter_AlwaysApplied(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create logs in both projects
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Test message", map[string]any{"test_session": uniqueTestSession})
	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Test message", map[string]any{"test_session": uniqueTestSession})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Empty query (should still be filtered by project)
	emptyQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{},
		Limit: 10,
	}

	project1Result, err := repository.ExecuteQueryForProject(project1, emptyQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project1Result.Total, "Empty query should still filter by project")
	assert.Len(t, project1Result.Logs, 1)

	project2Result, err := repository.ExecuteQueryForProject(project2, emptyQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project2Result.Total, "Empty query should still filter by project")
	assert.Len(t, project2Result.Logs, 1)

	// Verify results are from correct projects
	assert.NotEqual(t, project1Result.Logs[0].ID, project2Result.Logs[0].ID)
}

func Test_ExecuteQueryForProject_ProjectIdInQuery_DoesNotConflict(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create logs in both projects
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Test message project 1", map[string]any{"test_session": uniqueTestSession})
	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Test message project 2", map[string]any{"test_session": uniqueTestSession})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Query that tries to access project2 while querying project1
	// This should be ignored because the automatic project filter takes precedence
	queryTryingToAccessProject2 := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "project_id",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    project2.String(),
			},
		},
		Limit: 10,
	}

	// Execute against project1 - should return nothing because:
	// 1. Automatic filter limits to project1
	// 2. Manual condition looks for project2
	// 3. No logs match both conditions
	project1Result, err := repository.ExecuteQueryForProject(project1, queryTryingToAccessProject2)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), project1Result.Total, "Should find no logs due to conflicting project filters")
	assert.Empty(t, project1Result.Logs)

	// Query with matching project_id should work
	queryMatchingProject1 := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "project_id",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    project1.String(),
			},
		},
		Limit: 10,
	}

	matchingResult, err := repository.ExecuteQueryForProject(project1, queryMatchingProject1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), matchingResult.Total, "Should find logs when project_id matches")
}

func Test_DiscoverFields_MultipleProjects_OnlyReturnsProjectFields(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create logs with different custom fields for each project
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Project 1 log", map[string]any{
			"test_session":     uniqueTestSession,
			"project1_field_a": "value_a",
			"project1_field_b": "value_b",
			"shared_field":     "project1_value",
			"project1_unique":  "unique_to_p1",
		})

	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Project 2 log", map[string]any{
			"test_session":     uniqueTestSession,
			"project2_field_x": "value_x",
			"project2_field_y": "value_y",
			"shared_field":     "project2_value",
			"project2_unique":  "unique_to_p2",
		})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Discover fields for project1
	project1Fields, err := repository.DiscoverFields(project1)
	assert.NoError(t, err)
	assert.NotEmpty(t, project1Fields)

	// Convert to map for easier checking
	project1FieldMap := make(map[string]bool)
	for _, field := range project1Fields {
		project1FieldMap[field] = true
	}

	// Should contain project1's fields
	assert.True(t, project1FieldMap["project1_field_a"], "Should discover project1_field_a")
	assert.True(t, project1FieldMap["project1_field_b"], "Should discover project1_field_b")
	assert.True(t, project1FieldMap["project1_unique"], "Should discover project1_unique")
	assert.True(t, project1FieldMap["shared_field"], "Should discover shared_field")
	assert.True(t, project1FieldMap["test_session"], "Should discover test_session")

	// Should NOT contain project2's unique fields
	assert.False(t, project1FieldMap["project2_field_x"], "Should NOT discover project2_field_x")
	assert.False(t, project1FieldMap["project2_field_y"], "Should NOT discover project2_field_y")
	assert.False(t, project1FieldMap["project2_unique"], "Should NOT discover project2_unique")

	// Discover fields for project2
	project2Fields, err := repository.DiscoverFields(project2)
	assert.NoError(t, err)
	assert.NotEmpty(t, project2Fields)

	project2FieldMap := make(map[string]bool)
	for _, field := range project2Fields {
		project2FieldMap[field] = true
	}

	// Should contain project2's fields
	assert.True(t, project2FieldMap["project2_field_x"], "Should discover project2_field_x")
	assert.True(t, project2FieldMap["project2_field_y"], "Should discover project2_field_y")
	assert.True(t, project2FieldMap["project2_unique"], "Should discover project2_unique")
	assert.True(t, project2FieldMap["shared_field"], "Should discover shared_field")

	// Should NOT contain project1's unique fields
	assert.False(t, project2FieldMap["project1_field_a"], "Should NOT discover project1_field_a")
	assert.False(t, project2FieldMap["project1_field_b"], "Should NOT discover project1_field_b")
	assert.False(t, project2FieldMap["project1_unique"], "Should NOT discover project1_unique")
}

func Test_DeleteLogsByProject_OnlyDeletesSpecifiedProject(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()
	project3 := uuid.New()

	// Create logs in all three projects
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Project 1 log to be deleted", map[string]any{
			"test_session": uniqueTestSession,
			"project":      "project_1",
		})

	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Project 2 log should remain", map[string]any{
			"test_session": uniqueTestSession,
			"project":      "project_2",
		})

	project3Logs := CreateTestLogEntriesWithUniqueFields(project3, currentTime,
		"Project 3 log should remain", map[string]any{
			"test_session": uniqueTestSession,
			"project":      "project_3",
		})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	allLogs = MergeLogEntries(allLogs, project3Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Verify all projects have logs before deletion
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

	project1BeforeResult, _ := repository.ExecuteQueryForProject(project1, query)
	project2BeforeResult, _ := repository.ExecuteQueryForProject(project2, query)
	project3BeforeResult, _ := repository.ExecuteQueryForProject(project3, query)

	assert.Equal(t, int64(1), project1BeforeResult.Total, "Project1 should have logs before deletion")
	assert.Equal(t, int64(1), project2BeforeResult.Total, "Project2 should have logs before deletion")
	assert.Equal(t, int64(1), project3BeforeResult.Total, "Project3 should have logs before deletion")

	// Delete logs from project1 only
	deleteErr := repository.DeleteLogsByProject(project1)
	assert.NoError(t, deleteErr, "Delete operation should succeed")

	// Force refresh to make deletion visible
	flushErr := repository.ForceFlush()
	assert.NoError(t, flushErr)

	// Wait a bit for deletion to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify project1 has no logs, others remain unchanged
	project1AfterResult, _ := repository.ExecuteQueryForProject(project1, query)
	project2AfterResult, _ := repository.ExecuteQueryForProject(project2, query)
	project3AfterResult, _ := repository.ExecuteQueryForProject(project3, query)

	assert.Equal(t, int64(0), project1AfterResult.Total, "Project1 should have no logs after deletion")
	assert.Equal(t, int64(1), project2AfterResult.Total, "Project2 should still have logs after project1 deletion")
	assert.Equal(t, int64(1), project3AfterResult.Total, "Project3 should still have logs after project1 deletion")

	// Verify remaining logs have correct content
	assert.Contains(t, project2AfterResult.Logs[0].Message, "Project 2")
	assert.Contains(t, project3AfterResult.Logs[0].Message, "Project 3")
}

func Test_DeleteOldLogs_OnlyDeletesFromSpecifiedProject(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	baseTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create old and new logs for both projects
	oldTime := baseTime.Add(-2 * time.Hour)
	newTime := baseTime.Add(-30 * time.Minute)

	// Project1 logs
	project1OldLogs := CreateTestLogEntriesWithUniqueFields(project1, oldTime,
		"Project 1 old log", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "old",
			"project":      "project_1",
		})

	project1NewLogs := CreateTestLogEntriesWithUniqueFields(project1, newTime,
		"Project 1 new log", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "new",
			"project":      "project_1",
		})

	// Project2 logs (both old and new should remain after deleting project1 old logs)
	project2OldLogs := CreateTestLogEntriesWithUniqueFields(project2, oldTime,
		"Project 2 old log", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "old",
			"project":      "project_2",
		})

	project2NewLogs := CreateTestLogEntriesWithUniqueFields(project2, newTime,
		"Project 2 new log", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "new",
			"project":      "project_2",
		})

	allLogs := MergeLogEntries(project1OldLogs, project1NewLogs)
	allLogs = MergeLogEntries(allLogs, project2OldLogs)
	allLogs = MergeLogEntries(allLogs, project2NewLogs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Delete old logs from project1 only
	cutoffTime := baseTime.Add(-1 * time.Hour)
	deleteErr := repository.DeleteOldLogs(project1, cutoffTime)
	assert.NoError(t, deleteErr)

	// Force refresh
	flushErr := repository.ForceFlush()
	assert.NoError(t, flushErr)
	time.Sleep(1 * time.Second)

	// Verify results
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

	project1Result, _ := repository.ExecuteQueryForProject(project1, query)
	project2Result, _ := repository.ExecuteQueryForProject(project2, query)

	// Project1 should only have the new log
	assert.Equal(t, int64(1), project1Result.Total, "Project1 should only have new log")
	assert.Contains(t, project1Result.Logs[0].Message, "new")

	// Project2 should have both logs
	assert.Equal(t, int64(2), project2Result.Total, "Project2 should have both logs")
	messages := []string{project2Result.Logs[0].Message, project2Result.Logs[1].Message}
	assert.Contains(t, messages[0]+messages[1], "old")
	assert.Contains(t, messages[0]+messages[1], "new")
}

func Test_ExecuteQueryForProject_TotalCount_OnlyCountsProjectLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create 3 logs for project1 and 2 logs for project2
	project1Logs := CreateBatchLogEntries(project1, 3, currentTime, uniqueTestSession)
	project2Logs := CreateBatchLogEntries(project2, 2, currentTime.Add(1*time.Hour), uniqueTestSession)

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

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

	// Test project1 count
	project1Result, err := repository.ExecuteQueryForProject(project1, query)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), project1Result.Total, "Project1 total should only count project1 logs")

	// Test project2 count
	project2Result, err := repository.ExecuteQueryForProject(project2, query)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), project2Result.Total, "Project2 total should only count project2 logs")
}

func Test_ExecuteQueryForProject_Pagination_OnlyPaginatesProjectLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create 5 logs for project1 and 3 logs for project2
	project1Logs := CreateBatchLogEntries(project1, 5, currentTime, uniqueTestSession)
	project2Logs := CreateBatchLogEntries(project2, 3, currentTime.Add(1*time.Hour), uniqueTestSession)

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Test pagination on project1 (5 logs total)
	firstPageQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit:  2,
		Offset: 0,
	}

	firstPage, err := repository.ExecuteQueryForProject(project1, firstPageQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), firstPage.Total, "Total should reflect all project1 logs")
	assert.Len(t, firstPage.Logs, 2, "First page should have 2 logs")

	// Second page
	secondPageQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit:  2,
		Offset: 2,
	}

	secondPage, err := repository.ExecuteQueryForProject(project1, secondPageQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), secondPage.Total)
	assert.Len(t, secondPage.Logs, 2, "Second page should have 2 logs")

	// Third page
	thirdPageQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit:  2,
		Offset: 4,
	}

	thirdPage, err := repository.ExecuteQueryForProject(project1, thirdPageQuery)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), thirdPage.Total)
	assert.Len(t, thirdPage.Logs, 1, "Third page should have 1 remaining log")

	// Verify no logs from project2 appear in project1 pagination
	allProject1Logs := append(firstPage.Logs, secondPage.Logs...)
	allProject1Logs = append(allProject1Logs, thirdPage.Logs...)

	for i, log := range allProject1Logs {
		assert.Equal(t, uniqueTestSession, log.Fields["test_session"],
			"Log %d should be from correct test session", i)
		// All logs should be from project1's batch (which includes "api" in service field)
		assert.Equal(t, "api", log.Fields["service"],
			"Log %d should be from project1's batch", i)
	}
}

func Test_ExecuteQueryForProject_Sorting_OnlySortsProjectLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	baseTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create logs with different timestamps for both projects
	project1Time1 := baseTime.Add(-3 * time.Hour)
	project1Time2 := baseTime.Add(-1 * time.Hour)
	project2Time1 := baseTime.Add(-2 * time.Hour)
	project2Time2 := baseTime.Add(-30 * time.Minute)

	project1Logs1 := CreateTestLogEntriesWithUniqueFields(project1, project1Time1,
		"Project1 oldest", map[string]any{"test_session": uniqueTestSession, "order": 1})
	project1Logs2 := CreateTestLogEntriesWithUniqueFields(project1, project1Time2,
		"Project1 newest", map[string]any{"test_session": uniqueTestSession, "order": 2})

	project2Logs1 := CreateTestLogEntriesWithUniqueFields(project2, project2Time1,
		"Project2 oldest", map[string]any{"test_session": uniqueTestSession, "order": 1})
	project2Logs2 := CreateTestLogEntriesWithUniqueFields(project2, project2Time2,
		"Project2 newest", map[string]any{"test_session": uniqueTestSession, "order": 2})

	allLogs := MergeLogEntries(project1Logs1, project1Logs2)
	allLogs = MergeLogEntries(allLogs, project2Logs1)
	allLogs = MergeLogEntries(allLogs, project2Logs2)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Query project1 with descending sort (default)
	descQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		SortOrder: "desc",
		Limit:     10,
	}

	descResult, err := repository.ExecuteQueryForProject(project1, descQuery)
	assert.NoError(t, err)
	assert.Len(t, descResult.Logs, 2)
	// Should be sorted by timestamp descending within project1 only
	assert.True(t, descResult.Logs[0].Timestamp.After(descResult.Logs[1].Timestamp) ||
		descResult.Logs[0].Timestamp.Equal(descResult.Logs[1].Timestamp),
		"Logs should be sorted in descending order")
	assert.Contains(t, descResult.Logs[0].Message, "Project1 newest")
	assert.Contains(t, descResult.Logs[1].Message, "Project1 oldest")

	// Query project1 with ascending sort
	ascQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		SortOrder: "asc",
		Limit:     10,
	}

	ascResult, err := repository.ExecuteQueryForProject(project1, ascQuery)
	assert.NoError(t, err)
	assert.Len(t, ascResult.Logs, 2)
	// Should be sorted by timestamp ascending within project1 only
	assert.True(t, ascResult.Logs[0].Timestamp.Before(ascResult.Logs[1].Timestamp) ||
		ascResult.Logs[0].Timestamp.Equal(ascResult.Logs[1].Timestamp),
		"Logs should be sorted in ascending order")
	assert.Contains(t, ascResult.Logs[0].Message, "Project1 oldest")
	assert.Contains(t, ascResult.Logs[1].Message, "Project1 newest")

	// Verify that project2 logs are not affecting project1 sort order
	project2Result, err := repository.ExecuteQueryForProject(project2, descQuery)
	assert.NoError(t, err)
	assert.Len(t, project2Result.Logs, 2)
	assert.Contains(t, project2Result.Logs[0].Message, "Project2")
	assert.Contains(t, project2Result.Logs[1].Message, "Project2")
}

func Test_ExecuteQueryForProject_CannotAccessOtherProjectViaQuery(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	// Create sensitive data in project2 that project1 should never access
	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Project1 normal log", map[string]any{
			"test_session": uniqueTestSession,
			"data":         "project1_data",
		})

	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Project2 sensitive log", map[string]any{
			"test_session": uniqueTestSession,
			"data":         "sensitive_project2_data",
			"secret":       "top_secret_key",
		})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Try various query techniques to access project2 data from project1 context
	maliciousQueries := []*logs_core.LogQueryRequestDTO{
		// Try to query for project2's secret field
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    "secret",
					Operator: logs_core.ConditionOperatorExists,
				},
			},
			Limit: 10,
		},
		// Try to query for project2's data
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    "data",
					Operator: logs_core.ConditionOperatorContains,
					Value:    "sensitive",
				},
			},
			Limit: 10,
		},
		// Try to query for project2's message content
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeCondition,
				Condition: &logs_core.ConditionNode{
					Field:    "message",
					Operator: logs_core.ConditionOperatorContains,
					Value:    "sensitive",
				},
			},
			Limit: 10,
		},
		// Try logical OR to access both projects
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeLogical,
				Logic: &logs_core.LogicalNode{
					Operator: logs_core.LogicalOperatorOr,
					Children: []logs_core.QueryNode{
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "data",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    "project1_data",
							},
						},
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "data",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    "sensitive_project2_data",
							},
						},
					},
				},
			},
			Limit: 10,
		},
	}

	for i, maliciousQuery := range maliciousQueries {
		result, err := repository.ExecuteQueryForProject(project1, maliciousQuery)
		assert.NoError(t, err, "Query %d should not error but should return isolated results", i)

		// Should never return project2's sensitive data
		for j, log := range result.Logs {
			assert.NotContains(t, log.Message, "sensitive", "Query %d, log %d should not contain sensitive data", i, j)
			assert.NotContains(t, log.Message, "Project2", "Query %d, log %d should not contain Project2 data", i, j)

			if log.Fields != nil {
				assert.NotEqual(t, "sensitive_project2_data", log.Fields["data"],
					"Query %d, log %d should not contain project2's data", i, j)
				assert.Nil(t, log.Fields["secret"],
					"Query %d, log %d should not contain project2's secret field", i, j)
			}
		}
	}
}

func Test_ExecuteQueryForProject_ProjectIdManipulation_HasNoEffect(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	project1 := uuid.New()
	project2 := uuid.New()

	project1Logs := CreateTestLogEntriesWithUniqueFields(project1, currentTime,
		"Project 1 message", map[string]any{"test_session": uniqueTestSession})
	project2Logs := CreateTestLogEntriesWithUniqueFields(project2, currentTime,
		"Project 2 message", map[string]any{"test_session": uniqueTestSession})

	allLogs := MergeLogEntries(project1Logs, project2Logs)
	StoreTestLogsAndFlush(t, repository, allLogs)

	// Try to manipulate project_id in various ways in the query
	manipulationAttempts := []*logs_core.LogQueryRequestDTO{
		// Try to override project_id with different value
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeLogical,
				Logic: &logs_core.LogicalNode{
					Operator: logs_core.LogicalOperatorAnd,
					Children: []logs_core.QueryNode{
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "project_id",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    project2.String(),
							},
						},
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "test_session",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    uniqueTestSession,
							},
						},
					},
				},
			},
			Limit: 10,
		},
		// Try NOT project_id to negate the automatic filter
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeLogical,
				Logic: &logs_core.LogicalNode{
					Operator: logs_core.LogicalOperatorNot,
					Children: []logs_core.QueryNode{
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "project_id",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    project1.String(),
							},
						},
					},
				},
			},
			Limit: 10,
		},
		// Try OR with different project_id
		{
			Query: &logs_core.QueryNode{
				Type: logs_core.QueryNodeTypeLogical,
				Logic: &logs_core.LogicalNode{
					Operator: logs_core.LogicalOperatorOr,
					Children: []logs_core.QueryNode{
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "project_id",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    project1.String(),
							},
						},
						{
							Type: logs_core.QueryNodeTypeCondition,
							Condition: &logs_core.ConditionNode{
								Field:    "project_id",
								Operator: logs_core.ConditionOperatorEquals,
								Value:    project2.String(),
							},
						},
					},
				},
			},
			Limit: 10,
		},
	}

	for i, query := range manipulationAttempts {
		// Execute against project1 - should always be limited to project1 regardless of query content
		result, err := repository.ExecuteQueryForProject(project1, query)
		assert.NoError(t, err, "Manipulation attempt %d should not cause error", i)

		// Should never return more than project1's logs
		assert.LessOrEqual(t, result.Total, int64(1),
			"Manipulation attempt %d should not bypass project isolation", i)

		// If any logs are returned, they should only be from project1
		for j, log := range result.Logs {
			assert.Contains(t, log.Message, "Project 1",
				"Manipulation attempt %d, log %d should only return Project 1 data", i, j)
			assert.NotContains(t, log.Message, "Project 2",
				"Manipulation attempt %d, log %d should never return Project 2 data", i, j)
		}
	}
}
