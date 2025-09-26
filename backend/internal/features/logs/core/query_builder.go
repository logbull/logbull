package logs_core

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type QueryBuilder struct {
	logger *slog.Logger
}

// BuildSearchBody builds OpenSearch DSL body for the given project and structured request.
func (builder *QueryBuilder) BuildSearchBody(projectID uuid.UUID, request *LogQueryRequestDTO) (map[string]any, error) {
	boolQuery := map[string]any{
		"filter": []any{
			map[string]any{"term": map[string]any{"project_id.keyword": projectID.String()}},
		},
	}

	// Time range filter
	if request.TimeRange != nil && (request.TimeRange.From != nil || request.TimeRange.To != nil) {
		timeRange := map[string]any{}

		if request.TimeRange.From != nil {
			timeRange["gte"] = timestampToMicros(*request.TimeRange.From)
		}

		if request.TimeRange.To != nil {
			timeRange["lte"] = timestampToMicros(*request.TimeRange.To)
		}

		filterSlice, ok := boolQuery["filter"].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid filter type in bool query")
		}

		boolQuery["filter"] = append(filterSlice, map[string]any{
			"range": map[string]any{
				"timestamp": timeRange,
			},
		})
	}

	// User query
	if queryNode := builder.buildQueryNode(request.Query); queryNode != nil {
		// Attach to must
		if _, exists := boolQuery["must"]; !exists {
			boolQuery["must"] = []any{}
		}
		mustSlice, ok := boolQuery["must"].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid must type in bool query")
		}
		boolQuery["must"] = append(mustSlice, queryNode)
	}

	searchBody := map[string]any{
		"query":            map[string]any{"bool": boolQuery},
		"track_total_hits": true,
	}

	// Sort
	sortOrder := "desc"
	if strings.ToLower(request.SortOrder) == "asc" {
		sortOrder = "asc"
	}

	// Use numeric timestamp for precise microsecond sorting
	searchBody["sort"] = []any{
		map[string]any{"timestamp": map[string]any{"order": sortOrder}},
	}

	// Pagination
	if request.Offset > 0 {
		searchBody["from"] = request.Offset
	}
	if request.Limit > 0 {
		searchBody["size"] = request.Limit
	}

	return searchBody, nil
}

func (builder *QueryBuilder) buildQueryNode(node *QueryNode) map[string]any {
	if node == nil {
		return nil
	}

	switch node.Type {
	case QueryNodeTypeCondition:
		if node.Condition == nil {
			return nil
		}
		return builder.buildConditionNode(node.Condition)
	case QueryNodeTypeLogical:
		if node.Logic == nil || len(node.Logic.Children) == 0 {
			return nil
		}
		return builder.buildLogicalNode(node.Logic)
	default:
		return nil
	}
}

func (builder *QueryBuilder) buildLogicalNode(logic *LogicalNode) map[string]any {
	queryParts := make([]any, 0, len(logic.Children))
	for _, child := range logic.Children {
		if queryNode := builder.buildQueryNode(&child); queryNode != nil {
			queryParts = append(queryParts, queryNode)
		}
	}
	if len(queryParts) == 0 {
		return nil
	}

	switch logic.Operator {
	case LogicalOperatorAnd:
		return map[string]any{"bool": map[string]any{"must": queryParts}}
	case LogicalOperatorOr:
		return map[string]any{"bool": map[string]any{"should": queryParts, "minimum_should_match": 1}}
	case LogicalOperatorNot:
		// NOT over one or multiple => must_not
		return map[string]any{"bool": map[string]any{"must_not": queryParts}}
	default:
		return nil
	}
}

func (builder *QueryBuilder) buildConditionNode(condition *ConditionNode) map[string]any {
	fieldName := strings.TrimSpace(condition.Field)
	if fieldName == "" {
		return matchNone()
	}

	// System fields mapped directly; unknown fields go via attrs strategy
	isSystemField := builder.isSystemField(fieldName)

	switch condition.Operator {
	case ConditionOperatorEquals:
		if isSystemField {
			value := condition.Value
			// Convert timestamp strings to microseconds for consistency with storage
			if fieldName == "timestamp" {
				if stringValue, ok := value.(string); ok {
					if parsedTime, err := time.Parse(time.RFC3339Nano, stringValue); err == nil {
						value = timestampToMicros(parsedTime)
					}
				}
			}

			return term(builder.getSystemFieldName(fieldName), value)
		}
		return term("attrs_tokens.keyword", fmt.Sprintf("%s=%v", fieldName, condition.Value))

	case ConditionOperatorNotEquals:
		if isSystemField {
			value := condition.Value
			// Convert timestamp strings to microseconds for consistency with storage
			if fieldName == "timestamp" {
				if stringValue, ok := value.(string); ok {
					if parsedTime, err := time.Parse(time.RFC3339Nano, stringValue); err == nil {
						value = timestampToMicros(parsedTime)
					}
				}
			}
			return mustNot(term(builder.getSystemFieldName(fieldName), value))
		}
		return mustNot(term("attrs_tokens.keyword", fmt.Sprintf("%s=%v", fieldName, condition.Value)))

	case ConditionOperatorIn:
		values := asStringSlice(condition.Value)
		if len(values) == 0 {
			// Empty IN array should match nothing
			return matchNone()
		}
		if isSystemField {
			// Convert timestamp strings to microseconds for consistency with storage
			if fieldName == "timestamp" {
				microValues := make([]string, 0, len(values))
				for _, value := range values {
					if parsedTime, err := time.Parse(time.RFC3339Nano, value); err == nil {
						microValues = append(microValues, strconv.FormatInt(timestampToMicros(parsedTime), 10))
					} else {
						microValues = append(microValues, value) // Keep original if parsing fails
					}
				}
				return terms(builder.getSystemFieldName(fieldName), microValues)
			}
			return terms(builder.getSystemFieldName(fieldName), values)
		}
		// map to tokens "key=value"
		tokens := make([]string, 0, len(values))
		for _, value := range values {
			tokens = append(tokens, fmt.Sprintf("%s=%s", fieldName, value))
		}
		return terms("attrs_tokens.keyword", tokens)

	case ConditionOperatorNotIn:
		inCondition := (&ConditionNode{Field: fieldName, Operator: ConditionOperatorIn, Value: condition.Value})
		return mustNot(builder.buildConditionNode(inCondition))

	case ConditionOperatorExists:
		if isSystemField {
			return exists(fieldName)
		}
		// existence of any token with "field=" prefix
		return prefix("attrs_tokens.keyword", fieldName+"=")

	case ConditionOperatorNotExists:
		if isSystemField {
			return mustNot(exists(fieldName))
		}
		return mustNot(prefix("attrs_tokens.keyword", fieldName+"="))

	case ConditionOperatorContains:
		if isSystemField {
			// For strings: wildcard "*v*"; for message we also search in attrs_text
			return wildcard(builder.getSystemFieldName(fieldName), fmt.Sprintf("*%v*", condition.Value))
		}
		// Search attrs_tokens.keyword using wildcard for fieldName=*value*
		return wildcard("attrs_tokens.keyword", fmt.Sprintf("%s=*%v*", fieldName, condition.Value))

	case ConditionOperatorNotContains:
		if isSystemField {
			return mustNot(wildcard(builder.getSystemFieldName(fieldName), fmt.Sprintf("*%v*", condition.Value)))
		}
		return mustNot(wildcard("attrs_tokens.keyword", fmt.Sprintf("%s=*%v*", fieldName, condition.Value)))

	case ConditionOperatorGreaterThan, ConditionOperatorGreaterOrEqual,
		ConditionOperatorLessThan, ConditionOperatorLessOrEqual:

		// Only for system numeric/date fields; otherwise not supported
		if !isSystemField {
			// Range operators on custom fields are not supported and should match nothing
			return matchNone()
		}
		return rangeQuery(fieldName, condition.Operator, fmt.Sprintf("%v", condition.Value))

	default:
		return matchNone()
	}
}

func term(field string, value any) map[string]any {
	return map[string]any{"term": map[string]any{field: value}}
}

func terms(field string, values []string) map[string]any {
	arr := make([]any, len(values))
	for i, v := range values {
		arr[i] = v
	}
	return map[string]any{"terms": map[string]any{field: arr}}
}

func exists(field string) map[string]any {
	return map[string]any{"exists": map[string]any{"field": field}}
}

func prefix(field, value string) map[string]any {
	return map[string]any{"prefix": map[string]any{field: value}}
}

func wildcard(field, pattern string) map[string]any {
	return map[string]any{"wildcard": map[string]any{field: pattern}}
}

func mustNot(query map[string]any) map[string]any {
	return map[string]any{"bool": map[string]any{"must_not": []any{query}}}
}

func matchNone() map[string]any {
	return map[string]any{"match_none": map[string]any{}}
}

func rangeQuery(field string, operator ConditionOperator, value string) map[string]any {
	rangeKey := ""
	switch operator {
	case ConditionOperatorGreaterThan:
		rangeKey = "gt"
	case ConditionOperatorGreaterOrEqual:
		rangeKey = "gte"
	case ConditionOperatorLessThan:
		rangeKey = "lt"
	case ConditionOperatorLessOrEqual:
		rangeKey = "lte"
	}

	// Convert timestamp strings to microseconds for consistency with storage
	queryValue := value
	if field == "timestamp" {
		if parsedTime, err := time.Parse(time.RFC3339Nano, value); err == nil {
			queryValue = strconv.FormatInt(timestampToMicros(parsedTime), 10)
		}
	}

	return map[string]any{"range": map[string]any{field: map[string]any{rangeKey: queryValue}}}
}

// timestampToMicros converts a time to microseconds, ensuring consistent precision
func timestampToMicros(t time.Time) int64 {
	// Truncate to microseconds to ensure consistency with storage
	return t.Truncate(time.Microsecond).UnixMicro()
}

func asStringSlice(value any) []string {
	switch typedValue := value.(type) {
	case []string:
		return typedValue
	case []any:
		result := make([]string, 0, len(typedValue))
		for _, item := range typedValue {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	default:
		return nil
	}
}

func (builder *QueryBuilder) isSystemField(field string) bool {
	switch field {
	case "timestamp", "project_id", "id", "level", "client_ip", "created_at", "message":
		return true
	default:
		return false
	}
}

func (builder *QueryBuilder) getSystemFieldName(field string) string {
	// For exact term queries, some system fields need .keyword suffix
	switch field {
	case "project_id", "id", "level", "client_ip", "message":
		return field + ".keyword"
	default:
		return field
	}
}
