package erro

var (
	ErrMaxWrapDepthExceeded = New("maximum wrap depth exceeded")
)

// Security configuration constants
const (
	// Maximum string lengths to prevent memory exhaustion
	maxMessageLength    = 1000 // Maximum length for error messages
	maxFieldKeyLength   = 128  // Maximum length for field keys
	maxFieldValueLength = 1024 // Maximum length for field values (when converted to string)
	maxCodeLength       = 128  // Maximum length for error codes
	maxCategoryLength   = 128  // Maximum length for error categories
	maxSeverityLength   = 64   // Maximum length for error severity
	maxTraceIDLength    = 256  // Maximum length for trace IDs
	maxTagLength        = 128  // Maximum length for individual tags

	// Maximum array/slice lengths to prevent array bombing
	maxFieldsCount = 100 // Maximum number of fields (key-value pairs)
	maxTagsCount   = 50  // Maximum number of tags

	// Wrapping depth limits to prevent stack overflow
	maxWrapDepth = 50 // Maximum depth of error wrapping

	// Stack trace limits
	maxStackDepth = 50 // Maximum stack depth
)

// wrapDepthTracker tracks the depth of error wrapping to prevent stack overflow
type wrapDepthTracker struct {
	depth int
}

// getWrapDepth extracts the current wrap depth from an error by counting the wrap chain
func getWrapDepth(err error) int {
	if err == nil {
		return 0
	}

	depth := 0
	current := err

	// Count wrap depth by traversing the error chain
	for current != nil {
		if _, ok := current.(Error); ok {
			// If this is a wrapError, count it and continue to wrapped error
			if wrapErr, isWrap := current.(*wrapError); isWrap {
				depth++
				current = wrapErr.wrapped
				continue
			}
			// If this is a baseError, we've reached the base
			break
		}
		// For non-erro errors, we can't traverse further
		break
	}

	return depth
}

// incrementWrapDepth creates a new depth tracker with incremented depth
func incrementWrapDepth(err error) wrapDepthTracker {
	currentDepth := getWrapDepth(err)
	return wrapDepthTracker{depth: currentDepth + 1}
}

// SafeAppendFields safely appends fields with validation
func safeAppendFields[T any](existing []T, newFields []T) []T {
	if len(newFields) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return newFields
	}

	if len(newFields)+len(existing) > maxFieldsCount*2 {
		newFields = newFields[:maxFieldsCount*2-len(existing)]
		if len(newFields) == 0 {
			return existing
		}
	}

	if len(existing)+len(newFields) > cap(existing) {
		existing = append(
			make([]T, 0, cap(existing)+len(newFields)),
			existing...,
		)
	}

	return append(existing, newFields...)
}

// truncateString safely truncates a string to maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// truncateTags safely truncates tags array and individual tag lengths
func truncateTags(tags []string) []string {
	if len(tags) > maxTagsCount {
		tags = tags[:maxTagsCount]
	}

	// Truncate individual tag lengths
	for i, tag := range tags {
		tags[i] = truncateString(tag, maxTagLength)
	}

	return tags
}

// prepareFields prepares fields with validation and safe truncation
func prepareFields(fields []any) []any {
	if len(fields) == 0 {
		return fields
	}

	// Ensure even number of elements (key-value pairs)
	if len(fields)%2 != 0 {
		fields = append(fields, "<missing>")
	}

	// Limit the number of fields to prevent DOS
	maxElements := maxFieldsCount * 2
	if len(fields) > maxElements {
		fields = fields[:maxElements]
	}

	return fields
}
