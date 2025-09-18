package logs_receiving

import (
	logs_core "logbull/internal/features/logs/core"
)

type SubmitLogsRequestDTO struct {
	Logs []LogItemRequestDTO `json:"logs" binding:"required,min=1"`
}

type LogItemRequestDTO struct {
	Level     logs_core.LogLevel `json:"level"               binding:"required"`
	Message   string             `json:"message"             binding:"required,max=10000"`
	Timestamp any                `json:"timestamp,omitempty"`
	Fields    map[string]any     `json:"fields,omitempty"`
}

type SubmitLogsResponseDTO struct {
	Accepted int                  `json:"accepted"`
	Rejected int                  `json:"rejected"`
	Errors   []LogSubmissionError `json:"errors,omitempty"`
}

type LogSubmissionError struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}
