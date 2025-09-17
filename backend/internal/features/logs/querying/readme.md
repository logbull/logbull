# Log Query System API Documentation

This document provides examples of how to structure queries for the Log Query API. The frontend can use these examples to understand the query format and build a query builder interface.

## API Endpoints

### Execute Query

```
POST /api/v1/logs/query/execute/{projectId}
```

### Get Queryable Fields

```
GET /api/v1/logs/query/fields/{projectId}?query=optional_search
```

---

## Query Structure Overview

All queries follow a tree-like structure with two types of nodes:

1. **Condition Node**: Represents a single field condition (e.g., `message contains "error"`)
2. **Logical Node**: Combines multiple nodes with AND, OR, or NOT operators

```typescript
interface QueryNode {
  type: "condition" | "logical";
  condition?: ConditionNode; // Required if type === "condition"
  logic?: LogicalNode; // Required if type === "logical"
}
```

---

## Sorting Behavior

All queries are **automatically sorted by timestamp**. Sorting behavior:

- **Field**: Always sorted by `timestamp` (cannot be changed)
- **Order**: Defaults to `desc` (newest first) if `sortOrder` is not specified
- **Optional Parameter**: `sortOrder` can be set to `"asc"` or `"desc"`

```json
{
  "query": {
    /* your query */
  },
  "sortOrder": "asc" // Optional: defaults to "desc" if not specified
}
```

---

## Simple Query Examples

### 1. Message Contains Text

Find logs where the message contains "error":

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "message",
      "operator": "contains",
      "value": "error"
    }
  },
  "timeRange": {
    "from": "2024-01-15T00:00:00Z",
    "to": "2024-01-15T23:59:59Z"
  },
  "limit": 100,
  "sortOrder": "desc"
}
```

### 2. Specific Log Level

Find all ERROR level logs:

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "level",
      "operator": "equals",
      "value": "ERROR"
    }
  },
  "limit": 50
}
```

### 3. Client IP Filter

Find logs from a specific IP range:

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "client_ip",
      "operator": "equals",
      "value": "192.168.1"
    }
  }
}
```

### 4. Custom Field Query

Find logs where custom field "user_id" equals "12345":

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "user_id",
      "operator": "equals",
      "value": "12345"
    }
  }
}
```

### 4b. Custom Field with Dashes

Find logs where custom field "order-status" equals "completed":

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "order-status",
      "operator": "equals",
      "value": "completed"
    }
  }
}
```

### 4c. Custom Field with Mixed Case

Find logs where custom field "SessionId" exists:

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "SessionId",
      "operator": "exists",
      "value": null
    }
  }
}
```

### 4d. Custom Field Starting with Number

Find logs where custom field "123field" equals "value":

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "123field",
      "operator": "equals",
      "value": "value"
    }
  }
}
```

### 4e. Custom Field with Special Characters

Find logs where custom field "some@field" contains "data":

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "some@field",
      "operator": "contains",
      "value": "data"
    }
  }
}
```

### 4f. Custom Field with Parentheses

Find logs where custom field "method(param)" equals "result":

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "method(param)",
      "operator": "equals",
      "value": "result"
    }
  }
}
```

### 4g. Custom Field with Dots

Find logs where custom field "config.database.host" exists:

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "config.database.host",
      "operator": "exists",
      "value": null
    }
  }
}
```

---

## Complex Query Examples

### 5. AND Logic - Multiple Conditions

Find ERROR logs from a specific IP:

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "and",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "level",
            "operator": "equals",
            "value": "ERROR"
          }
        },
        {
          "type": "condition",
          "condition": {
            "field": "client_ip",
            "operator": "equals",
            "value": "192.168"
          }
        }
      ]
    }
  }
}
```

### 6. OR Logic - Alternative Conditions

Find logs that are either ERROR or WARN level:

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "or",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "level",
            "operator": "equals",
            "value": "ERROR"
          }
        },
        {
          "type": "condition",
          "condition": {
            "field": "level",
            "operator": "equals",
            "value": "WARN"
          }
        }
      ]
    }
  }
}
```

### 7. NOT Logic - Exclusion

Find logs that are NOT INFO level:

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "not",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "level",
            "operator": "equals",
            "value": "INFO"
          }
        }
      ]
    }
  }
}
```

### 8. Nested Logic - Complex Query

Find logs that contain "order" AND (are from IP starting with "88.34" OR have user_id "12345"):

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "and",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "message",
            "operator": "contains",
            "value": "order"
          }
        },
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "condition",
                "condition": {
                  "field": "client_ip",
                  "operator": "equals",
                  "value": "88.34"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "user_id",
                  "operator": "equals",
                  "value": "12345"
                }
              }
            ]
          }
        }
      ]
    }
  }
}
```

### 9. Deep Nested Logic (3 levels) - Advanced Query

Find ERROR or WARN logs that contain "payment" AND ((are from IP range "192.168.\*" AND have session_id) OR (have user_id in specific list AND NOT contain "test")):

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "and",
      "children": [
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "condition",
                "condition": {
                  "field": "level",
                  "operator": "equals",
                  "value": "ERROR"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "level",
                  "operator": "equals",
                  "value": "WARN"
                }
              }
            ]
          }
        },
        {
          "type": "condition",
          "condition": {
            "field": "message",
            "operator": "contains",
            "value": "payment"
          }
        },
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "logical",
                "logic": {
                  "operator": "and",
                  "children": [
                    {
                      "type": "condition",
                      "condition": {
                        "field": "client_ip",
                        "operator": "equals",
                        "value": "192.168"
                      }
                    },
                    {
                      "type": "condition",
                      "condition": {
                        "field": "session_id",
                        "operator": "exists",
                        "value": null
                      }
                    }
                  ]
                }
              },
              {
                "type": "logical",
                "logic": {
                  "operator": "and",
                  "children": [
                    {
                      "type": "condition",
                      "condition": {
                        "field": "user_id",
                        "operator": "in",
                        "value": ["user123", "user456", "user789"]
                      }
                    },
                    {
                      "type": "logical",
                      "logic": {
                        "operator": "not",
                        "children": [
                          {
                            "type": "condition",
                            "condition": {
                              "field": "message",
                              "operator": "contains",
                              "value": "test"
                            }
                          }
                        ]
                      }
                    }
                  ]
                }
              }
            ]
          }
        }
      ]
    }
  }
}
```

### 10. Multi-Level NOT Logic - Exclusion Patterns

Find logs that are NOT (INFO level OR (contain "debug" AND are from localhost)):

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "not",
      "children": [
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "condition",
                "condition": {
                  "field": "level",
                  "operator": "equals",
                  "value": "INFO"
                }
              },
              {
                "type": "logical",
                "logic": {
                  "operator": "and",
                  "children": [
                    {
                      "type": "condition",
                      "condition": {
                        "field": "message",
                        "operator": "contains",
                        "value": "debug"
                      }
                    },
                    {
                      "type": "condition",
                      "condition": {
                        "field": "client_ip",
                        "operator": "equals",
                        "value": "127.0.0.1"
                      }
                    }
                  ]
                }
              }
            ]
          }
        }
      ]
    }
  }
}
```

---

## Available Fields and Operators

### Standard Fields

| Field        | Display Name  | Type      | Available Operators                                                          |
| ------------ | ------------- | --------- | ---------------------------------------------------------------------------- |
| `message`    | Message       | string    | equals, not_equals, contains, not_contains                                   |
| `level`      | Log Level     | string    | equals, not_equals, in, not_in                                               |
| `client_ip`  | Client IP     | string    | equals, not_equals, contains, not_contains                                   |
| `timestamp`  | Timestamp     | timestamp | equals, not_equals, greater_than, greater_or_equal, less_than, less_or_equal |
| `*` (custom) | Custom Fields | string    | equals, not_equals, contains, not_contains, exists, not_exists               |

## Complete Operator Reference

### String Operators

#### equals / not_equals

```json
// Exact match
{"field": "level", "operator": "equals", "value": "ERROR"}

// Not equal to
{"field": "level", "operator": "not_equals", "value": "INFO"}
```

#### contains / not_contains

```json
// Contains text
{"field": "message", "operator": "contains", "value": "payment"}

// Does not contain text
{"field": "message", "operator": "not_contains", "value": "debug"}
```

#### in / not_in (Array Operations)

```json
// Value is in array
{"field": "level", "operator": "in", "value": ["ERROR", "WARN", "FATAL"]}

// Value is not in array
{"field": "level", "operator": "not_in", "value": ["DEBUG", "TRACE"]}
```

### Numeric/Comparison Operators

#### greater_than / greater_or_equal

```json
// Timestamp after specific time (nanoseconds)
{"field": "timestamp", "operator": "greater_than", "value": "1705334400000000000"}

// Timestamp on or after specific time
{"field": "timestamp", "operator": "greater_or_equal", "value": "1705334400000000000"}
```

#### less_than / less_or_equal

```json
// Timestamp before specific time
{"field": "timestamp", "operator": "less_than", "value": "1705420800000000000"}

// Timestamp on or before specific time
{"field": "timestamp", "operator": "less_or_equal", "value": "1705420800000000000"}
```

### Existence Operators

#### exists / not_exists

```json
// Field exists (has any value)
{"field": "user_id", "operator": "exists", "value": null}

// Field doesn't exist (is null/undefined)
{"field": "fields.session_id", "operator": "not_exists", "value": null}
```

## Field-Operator Compatibility

### Custom Field Naming

Custom fields support extremely flexible naming conventions. Almost any field name is supported:

- **Underscore style**: `user_id`, `session_id`, `order_total`
- **Dash style**: `order-status`, `user-name`, `payment-method`
- **CamelCase style**: `SessionId`, `OrderTotal`, `PaymentMethod`
- **Starting with numbers**: `123field`, `456_status`
- **Starting with dashes**: `-field`, `--special-field`
- **With parentheses**: `field(some)`, `method(param)`
- **With dots**: `some.field`, `namespace.property`
- **With special characters**: `some@field`, `field#special`, `field$value`
- **With spaces**: `field with spaces`, `long field name`
- **Mixed styles**: `_private_field`, `field123`, `multi-word-field`

**Rules:**

- Field names cannot be empty or contain only whitespace
- Leading and trailing spaces are automatically trimmed
- Any other characters are allowed

### Complete Compatibility Matrix

| Field Type                 | Available Operators                                                                                              |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **message (string)**       | `equals`, `not_equals`, `contains`, `not_contains`, `exists`, `not_exists`                                       |
| **level (string)**         | `equals`, `not_equals`, `contains`, `not_contains`, `in`, `not_in`, `exists`, `not_exists`                       |
| **client_ip (string)**     | `equals`, `not_equals`, `contains`, `not_contains`, `exists`, `not_exists`                                       |
| **timestamp (timestamp)**  | `equals`, `not_equals`, `greater_than`, `greater_or_equal`, `less_than`, `less_or_equal`, `exists`, `not_exists` |
| **custom fields (string)** | `equals`, `not_equals`, `contains`, `not_contains`, `in`, `not_in`, `exists`, `not_exists`                       |

### Time Range Query Examples

```json
// Find logs from specific time period using AND logic
{
  "type": "logical",
  "logic": {
    "operator": "and",
    "children": [
      {
        "type": "condition",
        "condition": {
          "field": "timestamp",
          "operator": "greater_or_equal",
          "value": "2024-01-15T09:00:00Z"
        }
      },
      {
        "type": "condition",
        "condition": {
          "field": "timestamp",
          "operator": "less_or_equal",
          "value": "2024-01-15T17:00:00Z"
        }
      }
    ]
  }
}

// Last 24 hours (using timeRange is preferred for time filtering)
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "level",
      "operator": "equals",
      "value": "ERROR"
    }
  },
  "timeRange": {
    "from": "2024-01-14T12:00:00Z",
    "to": "2024-01-15T12:00:00Z"
  }
}
```

### Advanced Operator Examples

```json
// Multiple log levels
{"field": "level", "operator": "in", "value": ["ERROR", "FATAL", "CRITICAL"]}

// Exclude test environments
{"field": "fields.environment", "operator": "not_in", "value": ["test", "development", "staging"]}
```

---

## Logical Operators Reference

### AND Operator

Combines conditions where ALL must be true:

```json
{
  "type": "logical",
  "logic": {
    "operator": "and",
    "children": [
      {
        "type": "condition",
        "condition": {
          "field": "level",
          "operator": "equals",
          "value": "ERROR"
        }
      },
      {
        "type": "condition",
        "condition": {
          "field": "message",
          "operator": "contains",
          "value": "payment"
        }
      }
    ]
  }
}
```

### OR Operator

Combines conditions where ANY can be true:

```json
{
  "type": "logical",
  "logic": {
    "operator": "or",
    "children": [
      {
        "type": "condition",
        "condition": {
          "field": "level",
          "operator": "equals",
          "value": "ERROR"
        }
      },
      {
        "type": "condition",
        "condition": {
          "field": "level",
          "operator": "equals",
          "value": "FATAL"
        }
      }
    ]
  }
}
```

### NOT Operator

Inverts the condition (must have exactly one child):

```json
{
  "type": "logical",
  "logic": {
    "operator": "not",
    "children": [
      {
        "type": "condition",
        "condition": {
          "field": "level",
          "operator": "equals",
          "value": "DEBUG"
        }
      }
    ]
  }
}
```

---

## Real-World Query Examples

### E-commerce Application Monitoring

#### Failed Payment Processing

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "and",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "level",
            "operator": "equals",
            "value": "ERROR"
          }
        },
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "condition",
                "condition": {
                  "field": "message",
                  "operator": "contains",
                  "value": "payment failed"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "message",
                  "operator": "contains",
                  "value": "transaction declined"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "payment_status",
                  "operator": "equals",
                  "value": "failed"
                }
              }
            ]
          }
        }
      ]
    }
  },
  "timeRange": {
    "from": "2024-01-15T00:00:00Z",
    "to": "2024-01-15T23:59:59Z"
  }
}
```

### API Monitoring

#### High Error Rate Investigation

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "and",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "level",
            "operator": "in",
            "value": ["ERROR", "WARN"]
          }
        },
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "condition",
                "condition": {
                  "field": "response_time",
                  "operator": "greater_than",
                  "value": "5000"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "message",
                  "operator": "contains",
                  "value": "timeout"
                }
              }
            ]
          }
        },
        {
          "type": "condition",
          "condition": {
            "field": "endpoint",
            "operator": "not_contains",
            "value": "/health"
          }
        }
      ]
    }
  }
}
```

### User Behavior Analysis

#### User Session Tracking

```json
{
  "query": {
    "type": "logical",
    "logic": {
      "operator": "and",
      "children": [
        {
          "type": "condition",
          "condition": {
            "field": "user_id",
            "operator": "equals",
            "value": "user123"
          }
        },
        {
          "type": "logical",
          "logic": {
            "operator": "or",
            "children": [
              {
                "type": "condition",
                "condition": {
                  "field": "message",
                  "operator": "contains",
                  "value": "login"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "message",
                  "operator": "contains",
                  "value": "logout"
                }
              },
              {
                "type": "condition",
                "condition": {
                  "field": "action",
                  "operator": "in",
                  "value": ["session_start", "session_end", "password_change"]
                }
              }
            ]
          }
        }
      ]
    }
  },
  "sortOrder": "asc"
}
```

---

## Complete Request Examples

### Basic Request with Time Range

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "level",
      "operator": "equals",
      "value": "ERROR"
    }
  },
  "timeRange": {
    "from": "2024-01-15T00:00:00Z",
    "to": "2024-01-15T23:59:59Z"
  },
  "limit": 100,
  "offset": 0,
  "sortOrder": "desc"
}
```

### Pagination Example

```json
{
  "query": {
    "type": "condition",
    "condition": {
      "field": "message",
      "operator": "contains",
      "value": "payment"
    }
  },
  "limit": 50,
  "offset": 100,
  "sortOrder": "desc"
}
```

---

## Response Format

### Successful Query Response

```json
{
  "logs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "timestamp": "2024-01-15T14:30:00Z",
      "level": "ERROR",
      "message": "Payment processing failed for order #12345",
      "fields": {
        "user_id": "user123",
        "order_id": "12345",
        "amount": "99.99"
      },
      "clientIp": "192.168.1.100",
      "createdAt": "2024-01-15T14:30:00Z"
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0,
  "executedIn": "45ms"
}
```

### Error Response

```json
{
  "error": "invalid query structure",
  "code": "INVALID_QUERY_STRUCTURE"
}
```

---

## Getting Available Fields

### Request

```
GET /api/v1/logs/query/fields/{projectId}?query=user
```

### Response

```json
{
  "fields": [
    {
      "name": "message",
      "displayName": "Message",
      "type": "string",
      "operations": ["equals", "contains"],
      "isCustom": false
    },
    {
      "name": "user_id",
      "displayName": "User Id",
      "type": "string",
      "operations": ["equals", "contains", "exists", "not_exists"],
      "isCustom": true
    },
    {
      "name": "order-status",
      "displayName": "Order Status",
      "type": "string",
      "operations": ["equals", "contains", "exists", "not_exists"],
      "isCustom": true
    },
    {
      "name": "SessionId",
      "displayName": "Session Id",
      "type": "string",
      "operations": ["equals", "contains", "exists", "not_exists"],
      "isCustom": true
    },
    {
      "name": "123field",
      "displayName": "123 Field",
      "type": "string",
      "operations": ["equals", "contains", "exists", "not_exists"],
      "isCustom": true
    },
    {
      "name": "some@field",
      "displayName": "Some @ Field",
      "type": "string",
      "operations": ["equals", "contains", "exists", "not_exists"],
      "isCustom": true
    },
    {
      "name": "method(param)",
      "displayName": "Method (Param)",
      "type": "string",
      "operations": ["equals", "contains", "exists", "not_exists"],
      "isCustom": true
    }
  ]
}
```

---

## Query Builder UI Recommendations

### 1. Field Selection

- Provide dropdown/autocomplete for field selection
- Use the `/fields` endpoint to populate available fields
- Group fields by type (Standard vs Custom)

### 2. Operator Selection

- Show only compatible operators for selected field type
- Use field's `operations` array to determine available operators

### 3. Value Input

- Text input for string values
- Date/time picker for timestamp fields
- Multi-select for IN/NOT IN operators
- Checkbox for EXISTS/NOT EXISTS (no value needed)

### 4. Logical Operators

- Visual tree structure for nested queries
- Drag-and-drop interface for building complex queries
- Add/remove condition buttons

### 5. Query Validation

- Real-time validation as user builds query
- Show error messages for invalid combinations
- Preview generated query structure

### 6. Sort Order

- Toggle for ascending/descending order (defaults to descending)
- Note: All results are automatically sorted by timestamp

### 7. Time Range

- Quick presets (Last hour, Last 24 hours, Last week)
- Custom date/time range picker
- Timezone handling

---

## Error Codes

| Code                          | Description                     | HTTP Status |
| ----------------------------- | ------------------------------- | ----------- |
| `TOO_MANY_CONCURRENT_QUERIES` | User has 3+ active queries      | 429         |
| `INVALID_QUERY_STRUCTURE`     | Query format is invalid         | 400         |
| `QUERY_TOO_COMPLEX`           | Query exceeds complexity limits | 400         |
| `QUERY_TIMEOUT`               | Query took too long to execute  | 408         |

---

## Limits and Constraints

- **Maximum query depth**: 10 levels
- **Maximum query nodes**: 50 nodes total
- **Maximum concurrent queries per user**: 3
- **Query timeout**: 30 seconds
- **Maximum results per query**: 1000
- **Maximum value length**: 1000 characters
- **Maximum array size for IN operator**: 100 items

---

## Security Notes

- Project ID is always taken from URL path parameter, never from request body
- All queries are automatically scoped to the specified project
- Users can only query projects they have access to
- Global admins can query any project
- All field names and values are validated and escaped to prevent injection attacks

---

## Quick Reference

### All Available Operators

| Category             | Operators                                                        | Description             |
| -------------------- | ---------------------------------------------------------------- | ----------------------- |
| **Equality**         | `equals`, `not_equals`                                           | Exact matches           |
| **Text Search**      | `contains`, `not_contains`                                       | Partial text matching   |
| **Array Operations** | `in`, `not_in`                                                   | Multiple value matching |
| **Numeric/Time**     | `greater_than`, `greater_or_equal`, `less_than`, `less_or_equal` | Comparison operations   |
| **Existence**        | `exists`, `not_exists`                                           | Field presence checking |

### All Logical Operators

| Operator | Children | Description                 |
| -------- | -------- | --------------------------- |
| `and`    | 1+       | All conditions must be true |
| `or`     | 1+       | Any condition can be true   |
| `not`    | 1        | Inverts the condition       |

### Field Types

| Type          | Examples                                           | Compatible Operators             |
| ------------- | -------------------------------------------------- | -------------------------------- |
| **string**    | message, level, client_ip                          | All string + existence operators |
| **timestamp** | timestamp, created_at                              | Comparison + existence operators |
| **custom**    | user_id, 123field, some@field, method(param), etc. | All string + existence operators |

### Best Practices

1. **Time Filtering**: Use `timeRange` parameter for time-based filtering instead of timestamp conditions when possible
2. **Sorting**: All results are automatically sorted by timestamp in descending order (newest first). Use `sortOrder: "asc"` for ascending order if needed
3. **Performance**: Limit query complexity - max 10 levels deep, 50 nodes total
4. **Security**: Project ID comes from URL path, not request body
5. **Pagination**: Use `limit` and `offset` for large result sets (max 1000 per query)
6. **Field Discovery**: Use `/fields` endpoint to get available fields dynamically
7. **Validation**: All queries are validated for structure, complexity, and security

### Common Patterns

```typescript
// Simple condition
{
  type: "condition",
  condition: { field: "level", operator: "equals", value: "ERROR" }
}

// AND logic
{
  type: "logical",
  logic: {
    operator: "and",
    children: [condition1, condition2, ...]
  }
}

// OR logic
{
  type: "logical",
  logic: {
    operator: "or",
    children: [condition1, condition2, ...]
  }
}

// NOT logic (exactly one child)
{
  type: "logical",
  logic: {
    operator: "not",
    children: [condition1]
  }
}
```

This documentation provides a complete reference for implementing the log query system frontend interface.
