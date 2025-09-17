package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_ExecuteQueryForProject_WithSpecificTimeRange_ReturnsOnlyLogsInRange(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()

	currentTime := time.Now().UTC()
	oldLogTime := currentTime.Add(-48 * time.Hour)
	recentLogTime := currentTime.Add(-2 * time.Hour)

	oldLogEntries := CreateTestLogEntriesWithTimestamp(projectID, oldLogTime, "Old log message")
	recentLogEntries := CreateTestLogEntriesWithTimestamp(projectID, recentLogTime, "Recent log message")

	oldStoreErr := repository.StoreLogsBatch(oldLogEntries)
	assert.NoError(t, oldStoreErr, "Failed to store old test data")

	recentStoreErr := repository.StoreLogsBatch(recentLogEntries)
	assert.NoError(t, recentStoreErr, "Failed to store recent test data")

	flushErr := repository.ForceFlush()
	assert.NoError(t, flushErr, "Failed to refresh index")

	timeRangeStart := currentTime.Add(-4 * time.Hour)
	timeRangeEnd := currentTime

	timeRangeQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "message",
				Operator: logs_core.ConditionOperatorContains,
				Value:    "log message",
			},
		},
		TimeRange: &logs_core.TimeRangeDTO{
			From: &timeRangeStart,
			To:   &timeRangeEnd,
		},
		Limit: 100,
	}

	timeRangeResult, timeRangeErr := repository.ExecuteQueryForProject(projectID, timeRangeQuery)
	assert.NoError(t, timeRangeErr, "Failed to execute time range query")
	assert.NotNil(t, timeRangeResult)

	// Verify all returned logs are within the specified time range
	for _, logEntry := range timeRangeResult.Logs {
		assert.True(t, logEntry.Timestamp.After(timeRangeStart) || logEntry.Timestamp.Equal(timeRangeStart),
			"Log timestamp %v should be after or equal to range start %v", logEntry.Timestamp, timeRangeStart)
		assert.True(t, logEntry.Timestamp.Before(timeRangeEnd) || logEntry.Timestamp.Equal(timeRangeEnd),
			"Log timestamp %v should be before or equal to range end %v", logEntry.Timestamp, timeRangeEnd)
	}
}
