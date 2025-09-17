package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_ExecuteQueryForProject_WithPaginationAndOffset_ReturnsSameTotalCount(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	testLogEntries := CreateBatchLogEntries(projectID, 12, currentTime, uniqueTestSession)
	StoreTestLogsAndFlush(t, repository, testLogEntries)

	// Add additional wait to ensure indexing is complete
	time.Sleep(1 * time.Second)

	firstPageQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit:     5,
		Offset:    0,
		SortOrder: "desc",
	}

	firstPageResult, firstPageErr := repository.ExecuteQueryForProject(projectID, firstPageQuery)
	assert.NoError(t, firstPageErr)
	assert.NotNil(t, firstPageResult)

	// Verify we have the expected test data
	assert.GreaterOrEqual(t, firstPageResult.Total, int64(12), "Should have at least 12 logs total")

	secondPageQuery := *firstPageQuery
	secondPageQuery.Offset = 5

	secondPageResult, secondPageErr := repository.ExecuteQueryForProject(projectID, &secondPageQuery)
	assert.NoError(t, secondPageErr)
	assert.NotNil(t, secondPageResult)

	// Both pages should report the same total count
	assert.Equal(t, firstPageResult.Total, secondPageResult.Total, "Total count should be consistent across pages")
	assert.GreaterOrEqual(t, firstPageResult.Total, int64(12), "Should have at least 12 logs total")
	assert.LessOrEqual(t, len(firstPageResult.Logs), 5, "First page should have at most 5 logs")
	assert.LessOrEqual(t, len(secondPageResult.Logs), 5, "Second page should have at most 5 logs")
}
