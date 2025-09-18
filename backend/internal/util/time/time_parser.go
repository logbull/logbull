package time_parser

import "time"

// parseTimestamp converts various timestamp formats to time.Time in UTC
// Supported formats:
//   - nil or empty string: uses current time
//   - ISO strings: RFC3339, RFC3339Nano, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02 15:04:05"
//   - Unix timestamps: seconds (< 1e12) or milliseconds (>= 1e12) as int, int64, or float64
//   - Returns error for unsupported types or invalid string formats
func ParseTimestamp(timestamp any) time.Time {
	// Use current time for nil timestamps
	if timestamp == nil {
		return time.Now().UTC()
	}

	switch v := timestamp.(type) {
	case string:
		// Use current time for empty strings
		if v == "" {
			return time.Now().UTC()
		}

		// Try multiple ISO string formats in order of preference
		formats := []string{
			time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
			time.RFC3339Nano,       // "2006-01-02T15:04:05.999999999Z07:00"
			"2006-01-02T15:04:05Z", // ISO with Z suffix
			"2006-01-02T15:04:05",  // ISO without timezone
			"2006-01-02 15:04:05",  // Space-separated format
		}

		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t.UTC()
			}
		}

		return time.Now().UTC()

	case float64:
		// JSON numbers are parsed as float64
		// Distinguish between seconds and milliseconds using threshold
		if v > 1e12 { // Milliseconds (timestamp > ~2001-09-09)
			return time.Unix(0, int64(v)*int64(time.Millisecond)).UTC()
		} else { // Seconds
			return time.Unix(int64(v), 0).UTC()
		}

	case int64:
		// Handle both unix seconds and milliseconds
		if v > 1e12 { // Milliseconds
			return time.Unix(0, v*int64(time.Millisecond)).UTC()
		} else { // Seconds
			return time.Unix(v, 0).UTC()
		}

	case int:
		// Convert int to int64 and recurse to avoid code duplication
		return ParseTimestamp(int64(v))

	default:
		// Reject unsupported types (bool, array, object, etc.)
		return time.Now().UTC()
	}
}
