package logs_querying

import (
	"strings"
	"testing"

	"logbull/internal/util/logger"

	logs_core "logbull/internal/features/logs/core"

	"github.com/stretchr/testify/assert"
)

// Main validation method tests
func Test_ValidateQuery_WithValidSimpleCondition_ReturnsNoError(t *testing.T) {
	validator := createValidator()
	query := createValidSimpleConditionQuery()

	err := validator.ValidateQuery(query)

	assert.NoError(t, err)
}

func Test_ValidateQuery_WithValidLogicalQuery_ReturnsNoError(t *testing.T) {
	validator := createValidator()
	query := createValidLogicalQuery()

	err := validator.ValidateQuery(query)

	assert.NoError(t, err)
}

func Test_ValidateQuery_WithNilQuery_ReturnsNoError(t *testing.T) {
	validator := createValidator()

	err := validator.ValidateQuery(nil)

	assert.NoError(t, err)
}

// Query complexity validation tests
func Test_ValidateComplexity_QueryDepthLimits_WorksCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		depth       int
		expectError bool
		errorCode   string
	}{
		{"At limit", 10, false, ""},
		{"Exceeds limit", 11, true, logs_core.ErrorQueryTooComplex},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := createDeepNestedQuery(tt.depth)
			err := validator.ValidateQuery(query)

			if tt.expectError {
				assertValidationErrorWithMessage(t, err, tt.errorCode, "depth")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ValidateComplexity_NodeCountLimits_WorksCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		nodeCount   int
		expectError bool
		errorCode   string
	}{
		{"At limit", 50, false, ""},
		{"Exceeds limit", 51, true, logs_core.ErrorQueryTooComplex},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := createQueryWithManyNodes(tt.nodeCount)
			err := validator.ValidateQuery(query)

			if tt.expectError {
				assertValidationErrorWithMessage(t, err, tt.errorCode, "nodes")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Query node validation tests
func Test_ValidateQueryNode_WithInvalidInputs_ReturnsErrors(t *testing.T) {
	tests := []struct {
		name      string
		node      *logs_core.QueryNode
		errorCode string
	}{
		{"Nil node", nil, logs_core.ErrorInvalidQueryStructure},
		{"Invalid node type", &logs_core.QueryNode{Type: "invalid_type"}, logs_core.ErrorInvalidQueryStructure},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateQueryNode(tt.node, 0)
			assertValidationError(t, err, tt.errorCode)
		})
	}
}

func Test_ValidateQueryNode_WithValidNodes_ReturnsNoError(t *testing.T) {
	tests := []struct {
		name string
		node *logs_core.QueryNode
	}{
		{"Valid condition node", createValidSimpleConditionQuery()},
		{"Valid logical node", createValidLogicalQuery()},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateQueryNode(tt.node, 0)
			assert.NoError(t, err)
		})
	}
}

// Logical node validation tests
func Test_ValidateLogicalNode_WithInvalidStructure_ReturnsErrors(t *testing.T) {
	tests := []struct {
		name      string
		node      *logs_core.QueryNode
		errorCode string
		message   string
	}{
		{
			"Nil logic",
			&logs_core.QueryNode{Type: logs_core.QueryNodeTypeLogical, Logic: nil},
			logs_core.ErrorInvalidQueryStructure,
			"",
		},
		{
			"Invalid operator",
			createLogicalNode("invalid_operator", []logs_core.QueryNode{*createValidSimpleConditionQuery()}),
			logs_core.ErrorInvalidQueryStructure,
			"",
		},
		{
			"No children",
			createLogicalNode(logs_core.LogicalOperatorAnd, []logs_core.QueryNode{}),
			logs_core.ErrorInvalidQueryStructure,
			"",
		},
		{
			"NOT with multiple children",
			createLogicalNode(logs_core.LogicalOperatorNot, []logs_core.QueryNode{
				*createValidSimpleConditionQuery(),
				*createValidSimpleConditionQuery(),
			}),
			logs_core.ErrorInvalidQueryStructure,
			"NOT operator should have exactly one child",
		},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateLogicalNode(tt.node, 0)
			if tt.message != "" {
				assertValidationErrorWithMessage(t, err, tt.errorCode, tt.message)
			} else {
				assertValidationError(t, err, tt.errorCode)
			}
		})
	}
}

func Test_ValidateLogicalNode_WithValidOperators_ReturnsNoError(t *testing.T) {
	operators := []logs_core.LogicalOperator{
		logs_core.LogicalOperatorAnd,
		logs_core.LogicalOperatorOr,
		logs_core.LogicalOperatorNot,
	}

	validator := createValidator()
	for _, op := range operators {
		t.Run(string(op), func(t *testing.T) {
			children := []logs_core.QueryNode{*createValidSimpleConditionQuery()}
			node := createLogicalNode(op, children)
			err := validator.validateLogicalNode(node, 0)
			assert.NoError(t, err)
		})
	}
}

func Test_ValidateLogicalNode_WithTooManyChildren_ReturnsQueryTooComplexError(t *testing.T) {
	validator := createValidator()
	children := make([]logs_core.QueryNode, 21) // Exceeds maxChildrenCount = 20
	for i := 0; i < 21; i++ {
		children[i] = *createValidSimpleConditionQuery()
	}
	node := createLogicalNode(logs_core.LogicalOperatorAnd, children)

	err := validator.validateLogicalNode(node, 0)

	assertValidationError(t, err, logs_core.ErrorQueryTooComplex)
}

func Test_ValidateLogicalNode_WithChildrenAtLimit_ReturnsNoError(t *testing.T) {
	validator := createValidator()
	children := make([]logs_core.QueryNode, 20) // At maxChildrenCount = 20
	for i := 0; i < 20; i++ {
		children[i] = *createValidSimpleConditionQuery()
	}
	node := createLogicalNode(logs_core.LogicalOperatorAnd, children)

	err := validator.validateLogicalNode(node, 0)

	assert.NoError(t, err)
}

// Condition node validation tests
func Test_ValidateConditionNode_WithInvalidConditions_ReturnsErrors(t *testing.T) {
	tests := []struct {
		name      string
		node      *logs_core.QueryNode
		errorCode string
	}{
		{
			"Nil condition",
			&logs_core.QueryNode{Type: logs_core.QueryNodeTypeCondition, Condition: nil},
			logs_core.ErrorInvalidQueryStructure,
		},
		{
			"Invalid field",
			createConditionNode("", logs_core.ConditionOperatorEquals, "test"),
			logs_core.ErrorInvalidQueryStructure,
		},
		{
			"Invalid operator",
			createConditionNode("message", "invalid_operator", "test"),
			logs_core.ErrorInvalidQueryStructure,
		},
		{
			"Invalid value",
			createConditionNode("message", logs_core.ConditionOperatorEquals, nil),
			logs_core.ErrorInvalidQueryStructure,
		},
		{
			"Incompatible field-operator",
			createConditionNode("timestamp", logs_core.ConditionOperatorContains, "test"),
			logs_core.ErrorInvalidQueryStructure,
		},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateConditionNode(tt.node)
			assertValidationError(t, err, tt.errorCode)
		})
	}
}

func Test_ValidateConditionNode_WithValidCondition_ReturnsNoError(t *testing.T) {
	validator := createValidator()
	node := createValidSimpleConditionQuery()

	err := validator.validateConditionNode(node)

	assert.NoError(t, err)
}

// Field validation tests
func Test_ValidateField_WithValidFields_ReturnsNoError(t *testing.T) {
	validFields := []string{
		"message", "level", "client_ip", "timestamp", "created_at",
		"user_id", "order_id", "session_id", "SomeField", "some-field",
		"_private_field", "field123", "multi-word-field", "CamelCaseField",
		"-field", "field(some)", "some.field", "some@field", "123field",
		" trimmed_field ", "special#chars", "field with spaces",
	}

	validator := createValidator()
	for _, field := range validFields {
		t.Run(field, func(t *testing.T) {
			err := validator.validateField(field)
			assert.NoError(t, err)
		})
	}
}

func Test_ValidateField_WithInvalidFields_ReturnsErrors(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		errorCode string
	}{
		{"Empty field", "", logs_core.ErrorInvalidQueryStructure},
		{"Only spaces", "   ", logs_core.ErrorInvalidQueryStructure},
		{"Only tabs", "\t\t", logs_core.ErrorInvalidQueryStructure},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateField(tt.field)
			assertValidationError(t, err, tt.errorCode)
		})
	}
}

// Operator validation tests
func Test_ValidateOperator_WithValidOperators_ReturnsNoError(t *testing.T) {
	validOperators := []logs_core.ConditionOperator{
		logs_core.ConditionOperatorEquals,
		logs_core.ConditionOperatorContains,
		logs_core.ConditionOperatorGreaterThan,
		logs_core.ConditionOperatorIn,
		logs_core.ConditionOperatorExists,
	}

	validator := createValidator()
	for _, op := range validOperators {
		t.Run(string(op), func(t *testing.T) {
			err := validator.validateOperator(op)
			assert.NoError(t, err)
		})
	}
}

func Test_ValidateOperator_WithInvalidOperator_ReturnsInvalidQueryStructureError(t *testing.T) {
	validator := createValidator()

	err := validator.validateOperator("invalid_operator")

	assertValidationError(t, err, logs_core.ErrorInvalidQueryStructure)
}

// Value validation tests
func Test_ValidateValue_WithNilValues_HandlesCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		operator    logs_core.ConditionOperator
		expectError bool
		errorCode   string
	}{
		{
			"Nil for non-existence operator",
			logs_core.ConditionOperatorEquals,
			true,
			logs_core.ErrorInvalidQueryStructure,
		},
		{"Nil for existence operator", logs_core.ConditionOperatorExists, false, ""},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateValue(nil, tt.operator)
			if tt.expectError {
				assertValidationError(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ValidateValue_WithValidValues_ReturnsNoError(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		operator logs_core.ConditionOperator
	}{
		{"String value", "test value", logs_core.ConditionOperatorEquals},
		{"Value at length limit", strings.Repeat("a", 1000), logs_core.ConditionOperatorEquals},
		{"Valid array", []string{"value1", "value2", "value3"}, logs_core.ConditionOperatorIn},
		{"Empty string", "", logs_core.ConditionOperatorEquals},
		{"Special characters", "!@#$%^&*()_+-=[]{}|;':,./<>?", logs_core.ConditionOperatorEquals},
		{"Unicode characters", "测试 🚀 Unicode", logs_core.ConditionOperatorEquals},
		{"Boolean value", true, logs_core.ConditionOperatorEquals},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateValue(tt.value, tt.operator)
			assert.NoError(t, err)
		})
	}
}

func Test_ValidateValue_WithInvalidValues_ReturnsErrors(t *testing.T) {
	validator := createValidator()

	tests := []struct {
		name      string
		value     interface{}
		operator  logs_core.ConditionOperator
		errorCode string
	}{
		{
			"Value too long",
			strings.Repeat("a", 1001),
			logs_core.ConditionOperatorEquals,
			logs_core.ErrorInvalidQueryStructure,
		},
		{"Empty array", []string{}, logs_core.ConditionOperatorIn, logs_core.ErrorInvalidQueryStructure},
		{"Array too large", make([]string, 101), logs_core.ConditionOperatorIn, logs_core.ErrorQueryTooComplex},
		{"Non-array for IN", "not_an_array", logs_core.ConditionOperatorIn, logs_core.ErrorInvalidQueryStructure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateValue(tt.value, tt.operator)
			assertValidationError(t, err, tt.errorCode)
		})
	}
}

// Field-operator compatibility tests
func Test_ValidateFieldOperatorCompatibility_ReturnsCorrectResults(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		operator    logs_core.ConditionOperator
		expectError bool
		errorCode   string
	}{
		{"Message with Contains", "message", logs_core.ConditionOperatorContains, false, ""},
		{
			"Message with GreaterThan",
			"message",
			logs_core.ConditionOperatorGreaterThan,
			true,
			logs_core.ErrorInvalidQueryStructure,
		},
		{"Timestamp with GreaterThan", "timestamp", logs_core.ConditionOperatorGreaterThan, false, ""},
		{
			"Timestamp with Contains",
			"timestamp",
			logs_core.ConditionOperatorContains,
			true,
			logs_core.ErrorInvalidQueryStructure,
		},
		{"Custom field with Equals", "user_id", logs_core.ConditionOperatorEquals, false, ""},
		{"Level with In", "level", logs_core.ConditionOperatorIn, false, ""},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateFieldOperatorCompatibility(tt.field, tt.operator)
			if tt.expectError {
				assertValidationError(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function tests
func Test_CalculateQueryDepth_ReturnsCorrectValues(t *testing.T) {
	tests := []struct {
		name          string
		query         *logs_core.QueryNode
		currentDepth  int
		expectedDepth int
	}{
		{"Simple condition", createValidSimpleConditionQuery(), 0, 0},
		{"Logical query", createValidLogicalQuery(), 0, 1},
		{"Nil node", nil, 5, 5},
		{"Deep nested", createDeepNestedQuery(5), 0, 5},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depth := validator.calculateQueryDepth(tt.query, tt.currentDepth)
			assert.Equal(t, tt.expectedDepth, depth)
		})
	}
}

func Test_CountQueryNodes_ReturnsCorrectCounts(t *testing.T) {
	tests := []struct {
		name          string
		query         *logs_core.QueryNode
		expectedCount int
	}{
		{"Simple condition", createValidSimpleConditionQuery(), 1},
		{"Logical with children", createValidLogicalQuery(), 3},
		{"Nil node", nil, 0},
		{"Many nodes", createQueryWithManyNodes(5), 5},
	}

	validator := createValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := validator.countQueryNodes(tt.query)
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

// Integration tests
func Test_ValidateQuery_WithComplexValidRealWorldQuery_ReturnsNoError(t *testing.T) {
	validator := createValidator()
	// (level=ERROR AND message contains "payment") OR (client_ip equals "192.168" AND fields.user_id exists)
	query := &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: logs_core.LogicalOperatorOr,
			Children: []logs_core.QueryNode{
				{
					Type: logs_core.QueryNodeTypeLogical,
					Logic: &logs_core.LogicalNode{
						Operator: logs_core.LogicalOperatorAnd,
						Children: []logs_core.QueryNode{
							{
								Type: logs_core.QueryNodeTypeCondition,
								Condition: &logs_core.ConditionNode{
									Field:    "level",
									Operator: logs_core.ConditionOperatorEquals,
									Value:    "ERROR",
								},
							},
							{
								Type: logs_core.QueryNodeTypeCondition,
								Condition: &logs_core.ConditionNode{
									Field:    "message",
									Operator: logs_core.ConditionOperatorContains,
									Value:    "payment",
								},
							},
						},
					},
				},
				{
					Type: logs_core.QueryNodeTypeLogical,
					Logic: &logs_core.LogicalNode{
						Operator: logs_core.LogicalOperatorAnd,
						Children: []logs_core.QueryNode{
							{
								Type: logs_core.QueryNodeTypeCondition,
								Condition: &logs_core.ConditionNode{
									Field:    "user_id",
									Operator: logs_core.ConditionOperatorExists,
									Value:    nil,
								},
							},
						},
					},
				},
			},
		},
	}

	err := validator.ValidateQuery(query)

	assert.NoError(t, err)
}

func Test_ValidateQuery_WithMixedValidAndInvalidChildren_ReturnsError(t *testing.T) {
	validator := createValidator()
	query := &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: logs_core.LogicalOperatorAnd,
			Children: []logs_core.QueryNode{
				*createValidSimpleConditionQuery(),
				{Type: "invalid_type"}, // Invalid child
			},
		},
	}

	err := validator.ValidateQuery(query)

	assert.Error(t, err)
}

func Test_ValidateQuery_WithChainedNOTOperators_ReturnsNoError(t *testing.T) {
	validator := createValidator()
	// NOT(NOT(condition))
	query := &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: logs_core.LogicalOperatorNot,
			Children: []logs_core.QueryNode{
				{
					Type: logs_core.QueryNodeTypeLogical,
					Logic: &logs_core.LogicalNode{
						Operator: logs_core.LogicalOperatorNot,
						Children: []logs_core.QueryNode{*createValidSimpleConditionQuery()},
					},
				},
			},
		},
	}

	err := validator.ValidateQuery(query)

	assert.NoError(t, err)
}

// Private helper functions - moved to bottom per coding standards

func createValidator() *QueryValidator {
	return &QueryValidator{
		logger: logger.GetLogger(),
	}
}

func createValidSimpleConditionQuery() *logs_core.QueryNode {
	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeCondition,
		Condition: &logs_core.ConditionNode{
			Field:    "message",
			Operator: logs_core.ConditionOperatorEquals,
			Value:    "test",
		},
	}
}

func createValidLogicalQuery() *logs_core.QueryNode {
	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: logs_core.LogicalOperatorAnd,
			Children: []logs_core.QueryNode{
				*createValidSimpleConditionQuery(),
				{
					Type: logs_core.QueryNodeTypeCondition,
					Condition: &logs_core.ConditionNode{
						Field:    "level",
						Operator: logs_core.ConditionOperatorEquals,
						Value:    "ERROR",
					},
				},
			},
		},
	}
}

func createConditionNode(field string, operator logs_core.ConditionOperator, value interface{}) *logs_core.QueryNode {
	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeCondition,
		Condition: &logs_core.ConditionNode{
			Field:    field,
			Operator: operator,
			Value:    value,
		},
	}
}

func createLogicalNode(operator logs_core.LogicalOperator, children []logs_core.QueryNode) *logs_core.QueryNode {
	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: operator,
			Children: children,
		},
	}
}

func createDeepNestedQuery(depth int) *logs_core.QueryNode {
	if depth <= 0 {
		return createValidSimpleConditionQuery()
	}

	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: logs_core.LogicalOperatorAnd,
			Children: []logs_core.QueryNode{
				*createDeepNestedQuery(depth - 1),
			},
		},
	}
}

func createQueryWithManyNodes(nodeCount int) *logs_core.QueryNode {
	if nodeCount <= 1 {
		return createValidSimpleConditionQuery()
	}

	return createNodeTree(nodeCount)
}

func createNodeTree(remainingNodes int) *logs_core.QueryNode {
	if remainingNodes <= 1 {
		return createValidSimpleConditionQuery()
	}

	maxChildren := 19
	childrenCount := remainingNodes - 1

	if childrenCount > maxChildren {
		childrenCount = maxChildren
	}

	children := make([]logs_core.QueryNode, childrenCount)
	remainingForChildren := remainingNodes - 1

	for i := 0; i < childrenCount; i++ {
		if i == childrenCount-1 {
			children[i] = *createNodeTree(remainingForChildren)
		} else {
			children[i] = *createValidSimpleConditionQuery()
			remainingForChildren--
		}
	}

	return &logs_core.QueryNode{
		Type: logs_core.QueryNodeTypeLogical,
		Logic: &logs_core.LogicalNode{
			Operator: logs_core.LogicalOperatorAnd,
			Children: children,
		},
	}
}

func assertValidationError(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var validationErr *ValidationError
	if assert.ErrorAs(t, err, &validationErr) {
		assert.Equal(t, expectedCode, validationErr.Code)
	}
}

func assertValidationErrorWithMessage(t *testing.T, err error, expectedCode string, expectedMessage string) {
	assert.Error(t, err)
	var validationErr *ValidationError
	if assert.ErrorAs(t, err, &validationErr) {
		assert.Equal(t, expectedCode, validationErr.Code)
		assert.Contains(t, validationErr.Message, expectedMessage)
	}
}
