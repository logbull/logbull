package time_parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ParseTimestamp_WithNilInput_ReturnsCurrentTime(t *testing.T) {
	before := time.Now().UTC()
	result := ParseTimestamp(nil)
	after := time.Now().UTC()

	assert.True(t, result.After(before.Add(-time.Second)) && result.Before(after.Add(time.Second)),
		"Expected result to be close to current time")
	assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
}

func Test_ParseTimestamp_WithEmptyString_ReturnsCurrentTime(t *testing.T) {
	before := time.Now().UTC()
	result := ParseTimestamp("")
	after := time.Now().UTC()

	assert.True(t, result.After(before.Add(-time.Second)) && result.Before(after.Add(time.Second)),
		"Expected result to be close to current time")
	assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
}

func Test_ParseTimestamp_WithValidISOStrings_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "RFC3339 format",
			input:    "2023-12-25T15:30:45Z",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "RFC3339 with timezone",
			input:    "2023-12-25T15:30:45+02:00",
			expected: time.Date(2023, 12, 25, 13, 30, 45, 0, time.UTC), // Converted to UTC
		},
		{
			name:     "RFC3339Nano format",
			input:    "2023-12-25T15:30:45.123456789Z",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 123456789, time.UTC),
		},
		{
			name:     "ISO with Z suffix",
			input:    "2023-12-25T15:30:45Z",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "ISO without timezone",
			input:    "2023-12-25T15:30:45",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
		{
			name:     "Space-separated format",
			input:    "2023-12-25 15:30:45",
			expected: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithInvalidStrings_ReturnsCurrentTime(t *testing.T) {
	invalidInputs := []string{
		"invalid-date",
		"2023-13-45",
		"not-a-timestamp",
		"2023/12/25 15:30:45",
		"25-12-2023",
		"just text",
	}

	for _, input := range invalidInputs {
		t.Run("Invalid input: "+input, func(t *testing.T) {
			before := time.Now().UTC()
			result := ParseTimestamp(input)
			after := time.Now().UTC()

			assert.True(t, result.After(before.Add(-time.Second)) && result.Before(after.Add(time.Second)),
				"Expected result to be close to current time for invalid input: %s", input)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithUnixSecondsFloat64_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected time.Time
	}{
		{
			name:     "Unix timestamp 0",
			input:    0,
			expected: time.Unix(0, 0).UTC(),
		},
		{
			name:     "Unix timestamp 1640000000",
			input:    1640000000, // 2021-12-20 13:46:40 UTC
			expected: time.Unix(1640000000, 0).UTC(),
		},
		{
			name:     "Unix timestamp with decimal",
			input:    1640000000.5,
			expected: time.Unix(1640000000, 0).UTC(), // Truncated to seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithUnixMillisecondsFloat64_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected time.Time
	}{
		{
			name:     "Unix milliseconds timestamp",
			input:    1640000000000, // 2021-12-20 13:46:40 UTC in milliseconds
			expected: time.Unix(0, 1640000000000*int64(time.Millisecond)).UTC(),
		},
		{
			name:     "Unix milliseconds with decimal",
			input:    1640000000123.456,
			expected: time.Unix(0, 1640000000123*int64(time.Millisecond)).UTC(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithUnixSecondsInt64_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected time.Time
	}{
		{
			name:     "Unix timestamp 0",
			input:    0,
			expected: time.Unix(0, 0).UTC(),
		},
		{
			name:     "Unix timestamp 1640000000",
			input:    1640000000,
			expected: time.Unix(1640000000, 0).UTC(),
		},
		{
			name:     "Negative unix timestamp",
			input:    -1640000000,
			expected: time.Unix(-1640000000, 0).UTC(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithUnixMillisecondsInt64_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected time.Time
	}{
		{
			name:     "Unix milliseconds timestamp",
			input:    1640000000000,
			expected: time.Unix(0, 1640000000000*int64(time.Millisecond)).UTC(),
		},
		{
			name:     "Large milliseconds timestamp",
			input:    1700000000000, // Future timestamp in milliseconds
			expected: time.Unix(0, 1700000000000*int64(time.Millisecond)).UTC(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithUnixSecondsInt_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Time
	}{
		{
			name:     "Unix timestamp 0",
			input:    0,
			expected: time.Unix(0, 0).UTC(),
		},
		{
			name:     "Unix timestamp 1640000000",
			input:    1640000000,
			expected: time.Unix(1640000000, 0).UTC(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_WithUnsupportedTypes_ReturnsCurrentTime(t *testing.T) {
	unsupportedInputs := []struct {
		name  string
		input any
	}{
		{"boolean true", true},
		{"boolean false", false},
		{"array", []string{"test"}},
		{"map", map[string]string{"key": "value"}},
		{"struct", struct{ Name string }{Name: "test"}},
		{"channel", make(chan int)},
	}

	for _, tt := range unsupportedInputs {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC()
			result := ParseTimestamp(tt.input)
			after := time.Now().UTC()

			assert.True(t, result.After(before.Add(-time.Second)) && result.Before(after.Add(time.Second)),
				"Expected result to be close to current time for unsupported type: %T", tt.input)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}

func Test_ParseTimestamp_ThresholdBehavior_DistinguishesSecondsFromMilliseconds(t *testing.T) {
	// Test the 1e12 threshold that distinguishes seconds from milliseconds
	tests := []struct {
		name        string
		input       int64
		expectType  string
		expectedSec int64
		expectedNs  int64
	}{
		{
			name:        "Just below threshold - seconds",
			input:       999999999999, // < 1e12, treated as seconds
			expectType:  "seconds",
			expectedSec: 999999999999,
			expectedNs:  0,
		},
		{
			name:        "Just above threshold - milliseconds",
			input:       1000000000001, // > 1e12, treated as milliseconds
			expectType:  "milliseconds",
			expectedSec: 1000000000,
			expectedNs:  1000000, // 1000000000001ms = 1000000000.001s = 1000000000s + 1000000ns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			expected := time.Unix(tt.expectedSec, tt.expectedNs).UTC()

			assert.Equal(t, expected, result)
			assert.Equal(t, time.UTC, result.Location(), "Expected UTC timezone")
		})
	}
}
