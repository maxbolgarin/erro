package erro

var (
	ErrMaxWrapDepthExceeded = NewLight("maximum wrap depth exceeded")
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

// prepareFields prepares fields with validation and safe truncation
func prepareFields(fields []any) []any {
	if len(fields) == 0 {
		return fields
	}

	// Limit the number of fields to prevent DOS
	maxElements := maxFieldsCount * 2
	if len(fields) > maxElements {
		fields = fields[:maxElements]
	}

	// Ensure even number of elements (key-value pairs)
	if len(fields)%2 != 0 {
		result := make([]any, len(fields)+1)
		copy(result, fields)
		result[len(fields)] = "<missing>"
		return result
	}

	result := make([]any, len(fields))
	copy(result, fields)
	return result
}
