package logs_querying

import (
	"fmt"
	"log/slog"
	"strings"

	logs_core "logbull/internal/features/logs/core"
)

type QueryValidator struct {
	logger *slog.Logger
}

const (
	// Query complexity limits
	maxQueryDepth    = 10   // Maximum nesting depth
	maxQueryNodes    = 50   // Maximum total nodes in query tree
	maxValueLength   = 1000 // Maximum value length
	maxChildrenCount = 20   // Maximum children per logical node
)

func (v *QueryValidator) ValidateQuery(query *logs_core.QueryNode) error {
	// Allow nil queries - they represent "return all logs within time period"
	if query == nil {
		return nil
	}

	if err := v.validateComplexity(query); err != nil {
		return err
	}

	if err := v.validateQueryNode(query, 0); err != nil {
		return err
	}

	return nil
}

func (v *QueryValidator) validateComplexity(query *logs_core.QueryNode) error {
	depth := v.calculateQueryDepth(query, 0)
	if depth > maxQueryDepth {
		return &ValidationError{
			Code:    logs_core.ErrorQueryTooComplex,
			Message: fmt.Sprintf("query depth %d exceeds maximum %d", depth, maxQueryDepth),
		}
	}

	nodeCount := v.countQueryNodes(query)
	if nodeCount > maxQueryNodes {
		return &ValidationError{
			Code:    logs_core.ErrorQueryTooComplex,
			Message: fmt.Sprintf("query has %d nodes, maximum allowed is %d", nodeCount, maxQueryNodes),
		}
	}

	return nil
}

func (v *QueryValidator) validateQueryNode(node *logs_core.QueryNode, depth int) error {
	if node == nil {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "query node cannot be nil",
		}
	}

	if node.Type != logs_core.QueryNodeTypeLogical && node.Type != logs_core.QueryNodeTypeCondition {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: fmt.Sprintf("invalid query node type: %s", node.Type),
		}
	}

	switch node.Type {
	case logs_core.QueryNodeTypeLogical:
		return v.validateLogicalNode(node, depth)
	case logs_core.QueryNodeTypeCondition:
		return v.validateConditionNode(node)
	}

	return nil
}

func (v *QueryValidator) validateLogicalNode(node *logs_core.QueryNode, depth int) error {
	if node.Logic == nil {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "logical node must have logic specified",
		}
	}

	logic := node.Logic

	if logic.Operator != logs_core.LogicalOperatorAnd &&
		logic.Operator != logs_core.LogicalOperatorOr &&
		logic.Operator != logs_core.LogicalOperatorNot {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: fmt.Sprintf("invalid logical operator: %s", logic.Operator),
		}
	}

	if len(logic.Children) == 0 {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "logical node must have at least one child",
		}
	}

	if len(logic.Children) > maxChildrenCount {
		return &ValidationError{
			Code: logs_core.ErrorQueryTooComplex,
			Message: fmt.Sprintf(
				"logical node has %d children, maximum allowed is %d",
				len(logic.Children),
				maxChildrenCount,
			),
		}
	}

	// Special case for NOT: should have exactly one child
	if logic.Operator == logs_core.LogicalOperatorNot && len(logic.Children) > 1 {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "NOT operator should have exactly one child",
		}
	}

	for i, child := range logic.Children {
		if err := v.validateQueryNode(&child, depth+1); err != nil {
			return fmt.Errorf("child %d: %w", i, err)
		}
	}

	return nil
}

func (v *QueryValidator) validateConditionNode(node *logs_core.QueryNode) error {
	if node.Condition == nil {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "condition node must have condition specified",
		}
	}

	condition := node.Condition

	if err := v.validateField(condition.Field); err != nil {
		return err
	}

	if err := v.validateOperator(condition.Operator); err != nil {
		return err
	}

	if err := v.validateValue(condition.Value, condition.Operator); err != nil {
		return err
	}

	if err := v.validateFieldOperatorCompatibility(condition.Field, condition.Operator); err != nil {
		return err
	}

	return nil
}

func (v *QueryValidator) validateField(field string) error {
	// Trim spaces from field name
	field = strings.TrimSpace(field)

	// Only validate that field is not empty
	if field == "" {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "field name cannot be empty",
		}
	}

	return nil
}

func (v *QueryValidator) validateOperator(operator logs_core.ConditionOperator) error {
	validOperators := map[logs_core.ConditionOperator]bool{
		logs_core.ConditionOperatorEquals:         true,
		logs_core.ConditionOperatorNotEquals:      true,
		logs_core.ConditionOperatorContains:       true,
		logs_core.ConditionOperatorNotContains:    true,
		logs_core.ConditionOperatorGreaterThan:    true,
		logs_core.ConditionOperatorGreaterOrEqual: true,
		logs_core.ConditionOperatorLessThan:       true,
		logs_core.ConditionOperatorLessOrEqual:    true,
		logs_core.ConditionOperatorIn:             true,
		logs_core.ConditionOperatorNotIn:          true,
		logs_core.ConditionOperatorExists:         true,
		logs_core.ConditionOperatorNotExists:      true,
	}

	if !validOperators[operator] {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: fmt.Sprintf("invalid operator: %s", operator),
		}
	}

	return nil
}

func (v *QueryValidator) validateValue(value any, operator logs_core.ConditionOperator) error {
	if value == nil && operator != logs_core.ConditionOperatorExists &&
		operator != logs_core.ConditionOperatorNotExists {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "value cannot be nil for this operator",
		}
	}

	strValue := fmt.Sprintf("%v", value)
	if len(strValue) > maxValueLength {
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: fmt.Sprintf("value length %d exceeds maximum %d", len(strValue), maxValueLength),
		}
	}

	if operator == logs_core.ConditionOperatorIn || operator == logs_core.ConditionOperatorNotIn {
		return v.validateArrayValue(value)
	}

	return nil
}

func (v *QueryValidator) validateArrayValue(value interface{}) error {
	switch v := value.(type) {
	case []interface{}:
		if len(v) == 0 {
			return &ValidationError{
				Code:    logs_core.ErrorInvalidQueryStructure,
				Message: "array value cannot be empty for IN/NOT IN operators",
			}
		}
		if len(v) > 100 { // Reasonable limit for IN clauses
			return &ValidationError{
				Code:    logs_core.ErrorQueryTooComplex,
				Message: "array value has too many elements (max 100)",
			}
		}
	case []string:
		if len(v) == 0 {
			return &ValidationError{
				Code:    logs_core.ErrorInvalidQueryStructure,
				Message: "array value cannot be empty for IN/NOT IN operators",
			}
		}
		if len(v) > 100 {
			return &ValidationError{
				Code:    logs_core.ErrorQueryTooComplex,
				Message: "array value has too many elements (max 100)",
			}
		}
	default:
		return &ValidationError{
			Code:    logs_core.ErrorInvalidQueryStructure,
			Message: "IN/NOT IN operators require array values",
		}
	}

	return nil
}

func (v *QueryValidator) validateFieldOperatorCompatibility(field string, operator logs_core.ConditionOperator) error {
	stringOperators := map[logs_core.ConditionOperator]bool{
		logs_core.ConditionOperatorEquals:      true,
		logs_core.ConditionOperatorNotEquals:   true,
		logs_core.ConditionOperatorContains:    true,
		logs_core.ConditionOperatorNotContains: true,
		logs_core.ConditionOperatorIn:          true,
		logs_core.ConditionOperatorNotIn:       true,
		logs_core.ConditionOperatorExists:      true,
		logs_core.ConditionOperatorNotExists:   true,
	}

	numericOperators := map[logs_core.ConditionOperator]bool{
		logs_core.ConditionOperatorEquals:         true,
		logs_core.ConditionOperatorNotEquals:      true,
		logs_core.ConditionOperatorGreaterThan:    true,
		logs_core.ConditionOperatorGreaterOrEqual: true,
		logs_core.ConditionOperatorLessThan:       true,
		logs_core.ConditionOperatorLessOrEqual:    true,
		logs_core.ConditionOperatorExists:         true,
		logs_core.ConditionOperatorNotExists:      true,
	}

	timestampOperators := numericOperators

	switch field {
	case "message", "level", "client_ip":
		if !stringOperators[operator] {
			return &ValidationError{
				Code:    logs_core.ErrorInvalidQueryStructure,
				Message: fmt.Sprintf("operator %s is not compatible with string field %s", operator, field),
			}
		}
	case "timestamp", "created_at":
		if !timestampOperators[operator] {
			return &ValidationError{
				Code:    logs_core.ErrorInvalidQueryStructure,
				Message: fmt.Sprintf("operator %s is not compatible with timestamp field %s", operator, field),
			}
		}
	default:
		// Custom fields - allow string operations by default
		if !stringOperators[operator] {
			return &ValidationError{
				Code:    logs_core.ErrorInvalidQueryStructure,
				Message: fmt.Sprintf("operator %s is not compatible with custom field %s", operator, field),
			}
		}
	}

	return nil
}

func (v *QueryValidator) calculateQueryDepth(node *logs_core.QueryNode, currentDepth int) int {
	if node == nil {
		return currentDepth
	}

	maxDepth := currentDepth

	if node.Type == logs_core.QueryNodeTypeLogical && node.Logic != nil {
		for _, child := range node.Logic.Children {
			childDepth := v.calculateQueryDepth(&child, currentDepth+1)
			if childDepth > maxDepth {
				maxDepth = childDepth
			}
		}
	}

	return maxDepth
}

func (v *QueryValidator) countQueryNodes(node *logs_core.QueryNode) int {
	if node == nil {
		return 0
	}

	count := 1 // Count current node

	if node.Type == logs_core.QueryNodeTypeLogical && node.Logic != nil {
		for _, child := range node.Logic.Children {
			count += v.countQueryNodes(&child)
		}
	}

	return count
}
