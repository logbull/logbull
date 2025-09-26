package logs_core

import (
	"time"

	"github.com/google/uuid"
)

type LogItem struct {
	ID        uuid.UUID      `json:"id"`
	ProjectID uuid.UUID      `json:"projectId"`
	Timestamp time.Time      `json:"timestamp"`
	Level     LogLevel       `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	ClientIP  string         `json:"clientIp,omitempty"`
}
