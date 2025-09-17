package logs_core

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

func (l LogLevel) IsValid() bool {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal:
		return true
	default:
		return false
	}
}

type QueryNodeType string

const (
	QueryNodeTypeLogical   QueryNodeType = "logical"
	QueryNodeTypeCondition QueryNodeType = "condition"
)

type LogicalOperator string

const (
	LogicalOperatorAnd LogicalOperator = "and"
	LogicalOperatorOr  LogicalOperator = "or"
	LogicalOperatorNot LogicalOperator = "not"
)

type ConditionOperator string

const (
	// String operations
	ConditionOperatorEquals      ConditionOperator = "equals"
	ConditionOperatorNotEquals   ConditionOperator = "not_equals"
	ConditionOperatorContains    ConditionOperator = "contains"
	ConditionOperatorNotContains ConditionOperator = "not_contains"

	// Numeric operations
	ConditionOperatorGreaterThan    ConditionOperator = "greater_than"
	ConditionOperatorGreaterOrEqual ConditionOperator = "greater_or_equal"
	ConditionOperatorLessThan       ConditionOperator = "less_than"
	ConditionOperatorLessOrEqual    ConditionOperator = "less_or_equal"

	// Array operations
	ConditionOperatorIn    ConditionOperator = "in"
	ConditionOperatorNotIn ConditionOperator = "not_in"

	// Existence operations
	ConditionOperatorExists    ConditionOperator = "exists"
	ConditionOperatorNotExists ConditionOperator = "not_exists"
)

type QueryableFieldType string

const (
	QueryableFieldTypeString    QueryableFieldType = "string"
	QueryableFieldTypeNumber    QueryableFieldType = "number"
	QueryableFieldTypeBoolean   QueryableFieldType = "boolean"
	QueryableFieldTypeTimestamp QueryableFieldType = "timestamp"
	QueryableFieldTypeArray     QueryableFieldType = "array"
)
