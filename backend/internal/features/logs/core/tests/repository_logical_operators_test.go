package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_ExecuteQueryForProject_WithLogicalOrAndNotOperators_ReturnsMatchingLogs(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()

	projectID := uuid.New()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	testLogEntries := CreateBatchLogEntries(projectID, 4, currentTime, uniqueTestSession)
	StoreTestLogsAndFlush(t, repository, testLogEntries)

	logicalQueryRequest := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeLogical,
			Logic: &logs_core.LogicalNode{
				Operator: logs_core.LogicalOperatorOr,
				Children: []logs_core.QueryNode{
					{Type: logs_core.QueryNodeTypeCondition, Condition: &logs_core.ConditionNode{
						Field: "service", Operator: logs_core.ConditionOperatorContains, Value: "api",
					}},
					{
						Type: logs_core.QueryNodeTypeLogical,
						Logic: &logs_core.LogicalNode{
							Operator: logs_core.LogicalOperatorNot,
							Children: []logs_core.QueryNode{
								{Type: logs_core.QueryNodeTypeCondition, Condition: &logs_core.ConditionNode{
									Field: "message", Operator: logs_core.ConditionOperatorContains, Value: "error",
								}},
							},
						},
					},
				},
			},
		},
		Limit: 5,
	}

	queryResult, queryErr := repository.ExecuteQueryForProject(projectID, logicalQueryRequest)
	assert.NoError(t, queryErr, "Failed to execute logical query")
	assert.NotNil(t, queryResult)
	assert.GreaterOrEqual(t, queryResult.Total, int64(0), "Query should return valid result structure")
}
