package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_DeleteOldLogs_WhenOldLogsExist_DeletesLogsOlderThanSpecifiedTime(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs at different times
	oldTime := baseTime.Add(-48 * time.Hour)
	mediumTime := baseTime.Add(-18 * time.Hour)
	recentTime := baseTime.Add(-1 * time.Hour)
	cutoffTime := baseTime.Add(-12 * time.Hour)

	// Create logs with different timestamps
	oldLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, oldTime,
		"Old log that should be deleted", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "old",
		})

	mediumLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, mediumTime,
		"Medium log that should also be deleted", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "medium",
		})

	recentLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, recentTime,
		"Recent log that should remain", map[string]any{
			"test_session": uniqueTestSession,
			"log_type":     "recent",
		})

	// Store all logs
	allLogEntries := MergeLogEntries(
		MergeLogEntries(oldLogEntries, mediumLogEntries),
		recentLogEntries,
	)
	StoreTestLogsAndFlush(t, repository, allLogEntries)

	// Verify logs exist before deletion
	beforeDeletionQuery := &logs_core.LogQueryRequestDTO{
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

	beforeDeletionResult, err := repository.ExecuteQueryForProject(projectID, beforeDeletionQuery)
	assert.NoError(t, err)

	// Delete old logs (older than cutoffTime)
	err = repository.DeleteOldLogs(projectID, cutoffTime)
	assert.NoError(t, err)

	// Wait for deletion to complete with condition check
	afterDeletionResult := waitForDeletionWithCondition(t, repository, projectID, beforeDeletionQuery,
		func(result *logs_core.LogQueryResponseDTO) bool {
			return result.Total < beforeDeletionResult.Total
		},
		"logs should be deleted and total count should decrease", 60_000)

	assert.NotNil(t, afterDeletionResult)

	// Should have fewer logs than before
	assert.Less(t, afterDeletionResult.Total, beforeDeletionResult.Total)

	// Verify only recent logs remain
	hasRecentLogs := false
	hasOldLogs := false
	hasMediumLogs := false

	for _, log := range afterDeletionResult.Logs {
		if log.Fields != nil {
			if logType, ok := log.Fields["log_type"].(string); ok {
				switch logType {
				case "recent":
					hasRecentLogs = true
				case "old":
					hasOldLogs = true
				case "medium":
					hasMediumLogs = true
				}
			}
		}
	}

	// Recent logs should remain, old and medium should be deleted
	assert.True(t, hasRecentLogs, "Recent logs should still exist")
	assert.False(t, hasOldLogs, "Old logs should be deleted")
	assert.False(t, hasMediumLogs, "Medium logs should be deleted")
}

func Test_DeleteLogsByProject_WhenProjectLogsExist_DeletesAllProjectLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	project1ID := uuid.New()
	project2ID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create logs for project 1
	project1LogEntries := CreateBatchLogEntries(project1ID, 3, currentTime, uniqueTestSession+"_p1")

	// Create logs for project 2
	project2LogEntries := CreateBatchLogEntries(project2ID, 3, currentTime, uniqueTestSession+"_p2")

	// Store logs for both projects
	StoreTestLogsAndFlush(t, repository, project1LogEntries)
	StoreTestLogsAndFlush(t, repository, project2LogEntries)

	// Verify both projects have logs before deletion
	project1BeforeDeletionQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession + "_p1",
			},
		},
		Limit: 10,
	}

	project2BeforeDeletionQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession + "_p2",
			},
		},
		Limit: 10,
	}

	project1BeforeDeletionResult, err := repository.ExecuteQueryForProject(project1ID, project1BeforeDeletionQuery)
	assert.NoError(t, err)

	project2BeforeDeletionResult, err := repository.ExecuteQueryForProject(project2ID, project2BeforeDeletionQuery)
	assert.NoError(t, err)

	assert.GreaterOrEqual(
		t,
		project1BeforeDeletionResult.Total,
		int64(3),
		"Project 1 should have at least 3 test logs before deletion",
	)
	assert.GreaterOrEqual(
		t,
		project2BeforeDeletionResult.Total,
		int64(3),
		"Project 2 should have at least 3 test logs before deletion",
	)

	// Delete all logs for project 1
	err = repository.DeleteLogsByProject(project1ID)
	assert.NoError(t, err)

	// Wait for project 1 logs to be completely deleted
	project1AfterDeletionResult := waitForDeletionCompletion(t, repository, project1ID,
		project1BeforeDeletionQuery, 0, 60_000)

	assert.Equal(t, int64(0), project1AfterDeletionResult.Total, "Project 1 logs should be deleted")
	assert.Empty(t, project1AfterDeletionResult.Logs, "Project 1 should have no logs")

	// Verify project 2 logs still exist (wait to ensure they weren't accidentally deleted)
	project2AfterDeletionResult := waitForDeletionCompletion(t, repository, project2ID,
		project2BeforeDeletionQuery, project2BeforeDeletionResult.Total, 60_000)

	assert.Equal(
		t,
		project2BeforeDeletionResult.Total,
		project2AfterDeletionResult.Total,
		"Project 2 logs should remain unchanged",
	)
	assert.Len(
		t,
		project2AfterDeletionResult.Logs,
		len(project2BeforeDeletionResult.Logs),
		"Project 2 should still have all logs",
	)
}

func Test_DeleteLogsByProject_WithNonExistentProject_DoesNotFail(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	nonExistentProjectID := uuid.New()

	// Delete logs for non-existent project should not fail
	err := repository.DeleteLogsByProject(nonExistentProjectID)
	assert.NoError(t, err, "Deleting logs for non-existent project should not fail")
}

func Test_DeleteOldLogs_WithNoOldLogs_DoesNotFail(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	// Create only recent logs
	recentLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, currentTime.Add(-1*time.Hour),
		"Recent log", map[string]any{
			"test_session": uniqueTestSession,
		})

	StoreTestLogsAndFlush(t, repository, recentLogEntries)

	// Try to delete logs older than 48 hours (should find nothing to delete)
	cutoffTime := currentTime.Add(-48 * time.Hour)
	err := repository.DeleteOldLogs(projectID, cutoffTime)
	assert.NoError(t, err, "Deleting old logs when none exist should not fail")

	// Verify recent logs still exist using helper function
	verificationQuery := &logs_core.LogQueryRequestDTO{
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

	verificationResult, err := repository.ExecuteQueryForProject(projectID, verificationQuery)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, verificationResult.Total, int64(1), "Recent logs should still exist")
}
func waitForDeletionCompletion(
	t *testing.T,
	repository *logs_core.LogCoreRepository,
	projectID uuid.UUID,
	query *logs_core.LogQueryRequestDTO,
	expectedTotal int64,
	timeoutMs int,
) *logs_core.LogQueryResponseDTO {
	const pollIntervalMs = 50
	maxAttempts := timeoutMs / pollIntervalMs

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := repository.ForceFlush()
		assert.NoError(t, err, "Force flush should not fail on attempt %d", attempt+1)

		result, err := repository.ExecuteQueryForProject(projectID, query)
		assert.NoError(t, err, "Query should not fail on attempt %d", attempt+1)

		if result.Total == expectedTotal {
			return result
		}

		time.Sleep(pollIntervalMs * time.Millisecond)
	}

	// Final attempt after timeout
	err := repository.ForceFlush()
	assert.NoError(t, err, "Final force flush should not fail")

	result, err := repository.ExecuteQueryForProject(projectID, query)
	assert.NoError(t, err, "Final query should not fail")

	assert.Equal(t, expectedTotal, result.Total,
		"Expected %d logs after deletion, but found %d (timeout after %dms)",
		expectedTotal, result.Total, timeoutMs)

	return result
}

func waitForDeletionWithCondition(
	t *testing.T,
	repository *logs_core.LogCoreRepository,
	projectID uuid.UUID,
	query *logs_core.LogQueryRequestDTO,
	conditionCheck func(*logs_core.LogQueryResponseDTO) bool,
	conditionDescription string,
	timeoutMs int,
) *logs_core.LogQueryResponseDTO {
	const pollIntervalMs = 50
	maxAttempts := timeoutMs / pollIntervalMs

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := repository.ForceFlush()
		assert.NoError(t, err, "Force flush should not fail on attempt %d", attempt+1)

		result, err := repository.ExecuteQueryForProject(projectID, query)
		assert.NoError(t, err, "Query should not fail on attempt %d", attempt+1)

		if conditionCheck(result) {
			return result
		}

		time.Sleep(pollIntervalMs * time.Millisecond)
	}

	// Final attempt after timeout
	err := repository.ForceFlush()
	assert.NoError(t, err, "Final force flush should not fail")

	result, err := repository.ExecuteQueryForProject(projectID, query)
	assert.NoError(t, err, "Final query should not fail")

	assert.True(t, conditionCheck(result),
		"Deletion condition not met: %s (timeout after %dms)",
		conditionDescription, timeoutMs)

	return result
}
