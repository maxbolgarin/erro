package erro_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maxbolgarin/erro"
)

// TestJSONSerializationEdgeCases tests edge cases in JSON serialization
func TestJSONSerializationEdgeCases(t *testing.T) {
	t.Run("Large JSON serialization", func(t *testing.T) {
		// Create error with many fields that could cause large JSON
		fields := make([]any, 0, erro.MaxFieldsCount*2)
		for i := 0; i < erro.MaxFieldsCount; i++ {
			fields = append(fields,
				fmt.Sprintf("key_%d", i),
				strings.Repeat("large_value_", 10)+fmt.Sprintf("_%d", i),
			)
		}

		err := erro.New("large serialization test", fields...)
		err = erro.Wrap(err, "wrapped error", erro.StackTrace())

		// Marshal to JSON
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal large error: %v", marshalErr)
		}

		// Should produce reasonable size JSON
		if len(jsonData) > 1000000 { // 1MB limit
			t.Errorf("JSON size too large: %d bytes", len(jsonData))
		}

		// Should be able to unmarshal back
		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal large error: %v", unmarshalErr)
		}
	})

	t.Run("Invalid JSON unmarshaling", func(t *testing.T) {
		testCases := []struct {
			name     string
			jsonData string
		}{
			{"malformed JSON", `{"id": "test", "message": "test"`},
			{"invalid field types", `{"id": 123, "message": ["array"]}`},
			{"null values", `{"id": null, "message": null, "fields": null}`},
			{"empty JSON", `{}`},
			{"array instead of object", `["not", "an", "object"]`},
			{"string instead of object", `"not an object"`},
			{"very large JSON", `{"message": "` + strings.Repeat("x", 10000) + `"}`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := erro.New("")
				unmarshalErr := json.Unmarshal([]byte(tc.jsonData), err)
				// Should handle invalid JSON gracefully without panicking
				if unmarshalErr == nil && tc.name != "empty JSON" && tc.name != "null values" {
					t.Logf("Expected error for %s, but got none", tc.name)
				}
			})
		}
	})

	t.Run("Special characters in JSON", func(t *testing.T) {
		// Test with characters that could break JSON
		specialValues := []any{
			"unicode: 测试 🚀 émojis 🔥",
			"quotes: \"double\" and 'single'",
			"backslashes: \\ and \\n and \\t",
			"control chars: \x00\x01\x02\x1f",
			"json injection: \"},\"injected\":\"value",
		}

		for i, val := range specialValues {
			err := erro.New("special chars test", fmt.Sprintf("key_%d", i), val)

			// Should marshal without error
			jsonData, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Errorf("Failed to marshal special chars: %v", marshalErr)
			}

			// Should unmarshal without error
			newErr := erro.New("")
			unmarshalErr := json.Unmarshal(jsonData, newErr)
			if unmarshalErr != nil {
				t.Errorf("Failed to unmarshal special chars: %v", unmarshalErr)
			}
		}
	})

	t.Run("Redacted values in JSON", func(t *testing.T) {
		err := erro.New("redaction test",
			"public", "visible_value",
			"secret", erro.Redact("sensitive_data"),
			"password", erro.Redact("super_secret_password"),
			"api_key", erro.Redact(map[string]string{"key": "secret_key_value"}),
		)

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal redacted error: %v", marshalErr)
		}

		jsonStr := string(jsonData)

		// Should contain redacted placeholder
		if !strings.Contains(jsonStr, erro.RedactedPlaceholder) {
			t.Error("JSON should contain redacted placeholder")
		}

		// Should NOT contain sensitive data
		if strings.Contains(jsonStr, "sensitive_data") ||
			strings.Contains(jsonStr, "super_secret_password") ||
			strings.Contains(jsonStr, "secret_key_value") {
			t.Error("JSON should not contain sensitive data")
		}

		// Should contain public data
		if !strings.Contains(jsonStr, "visible_value") {
			t.Error("JSON should contain public data")
		}
	})

	t.Run("Stack trace in JSON", func(t *testing.T) {
		err := erro.New("stack trace test", erro.StackTrace())

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal error with stack: %v", marshalErr)
		}

		var parsed map[string]interface{}
		parseErr := json.Unmarshal(jsonData, &parsed)
		if parseErr != nil {
			t.Errorf("Failed to parse JSON: %v", parseErr)
		}

		// Should contain stack trace
		if _, exists := parsed["stack_trace"]; !exists {
			t.Error("JSON should contain stack_trace field")
		}

		// Unmarshal back to error
		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal error with stack: %v", unmarshalErr)
		}
	})
}

// TestComplexDataTypeSerialization tests serialization of complex data types
func TestComplexDataTypeSerialization(t *testing.T) {
	t.Run("Nested structures", func(t *testing.T) {
		type complexStruct struct {
			Name     string            `json:"name"`
			Values   []int             `json:"values"`
			Metadata map[string]string `json:"metadata"`
			Time     time.Time         `json:"time"`
		}

		complex := complexStruct{
			Name:   "test_struct",
			Values: []int{1, 2, 3, 4, 5},
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Time: time.Now(),
		}

		err := erro.New("complex data test",
			"complex_struct", complex,
			"nested_map", map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": []string{"a", "b", "c"},
				},
			},
		)

		// Should marshal successfully
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal complex data: %v", marshalErr)
		}

		// Should unmarshal successfully
		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal complex data: %v", unmarshalErr)
		}
	})

	t.Run("Circular reference handling", func(t *testing.T) {
		type circularStruct struct {
			Name string          `json:"name"`
			Self *circularStruct `json:"self,omitempty"`
		}

		circular := &circularStruct{Name: "circular"}
		circular.Self = circular

		err := erro.New("circular test", "circular_data", circular)

		// Should handle circular references gracefully
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			// Circular references should either be handled or fail gracefully
			t.Logf("Circular reference marshal failed as expected: %v", marshalErr)
		} else {
			// If it succeeds, should produce valid JSON
			var result map[string]interface{}
			parseErr := json.Unmarshal(jsonData, &result)
			if parseErr != nil {
				t.Errorf("Produced invalid JSON for circular reference: %v", parseErr)
			}
		}
	})

	t.Run("Large slice/array handling", func(t *testing.T) {
		largeSlice := make([]int, 10000)
		for i := range largeSlice {
			largeSlice[i] = i
		}

		err := erro.New("large slice test", "large_slice", largeSlice)

		// Should handle large slices efficiently
		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal large slice: %v", marshalErr)
		}

		// Should have reasonable size (truncation might occur)
		if len(jsonData) > 2000000 { // 2MB limit
			t.Errorf("JSON for large slice too big: %d bytes", len(jsonData))
		}
	})

	t.Run("Nil and empty values", func(t *testing.T) {
		var nilSlice []string
		var nilMap map[string]string
		var nilInterface interface{}

		err := erro.New("nil values test",
			"nil_slice", nilSlice,
			"nil_map", nilMap,
			"nil_interface", nilInterface,
			"empty_slice", []string{},
			"empty_map", map[string]string{},
			"empty_string", "",
		)

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal nil values: %v", marshalErr)
		}

		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal nil values: %v", unmarshalErr)
		}
	})
}

// TestSerializationBoundaryConditions tests boundary conditions in serialization
func TestSerializationBoundaryConditions(t *testing.T) {
	t.Run("Maximum field count serialization", func(t *testing.T) {
		// Test serialization at maximum field capacity
		fields := make([]any, 0, erro.MaxFieldsCount*2)
		for i := 0; i < erro.MaxFieldsCount; i++ {
			fields = append(fields, fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
		}

		err := erro.New("max fields test", fields...)

		// Multiple serialization/deserialization cycles
		for cycle := 0; cycle < 5; cycle++ {
			jsonData, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Errorf("Cycle %d: marshal failed: %v", cycle, marshalErr)
			}

			newErr := erro.New("")
			unmarshalErr := json.Unmarshal(jsonData, newErr)
			if unmarshalErr != nil {
				t.Errorf("Cycle %d: unmarshal failed: %v", cycle, unmarshalErr)
			}

			err = newErr // Use for next cycle
		}
	})

	t.Run("Empty error serialization", func(t *testing.T) {
		err := erro.New("")

		jsonData, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Errorf("Failed to marshal empty error: %v", marshalErr)
		}

		newErr := erro.New("")
		unmarshalErr := json.Unmarshal(jsonData, newErr)
		if unmarshalErr != nil {
			t.Errorf("Failed to unmarshal empty error: %v", unmarshalErr)
		}
	})

	t.Run("Concurrent serialization", func(t *testing.T) {
		err := erro.New("concurrent test",
			"field1", "value1",
			"field2", 42,
			"field3", time.Now(),
			erro.StackTrace(),
		)

		const numGoroutines = 100
		errors := make(chan error, numGoroutines)

		// Serialize concurrently
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("panic: %v", r)
					}
				}()

				jsonData, marshalErr := json.Marshal(err)
				if marshalErr != nil {
					errors <- marshalErr
					return
				}

				newErr := erro.New("")
				unmarshalErr := json.Unmarshal(jsonData, newErr)
				if unmarshalErr != nil {
					errors <- unmarshalErr
					return
				}

				errors <- nil
			}()
		}

		// Check results
		for i := 0; i < numGoroutines; i++ {
			if err := <-errors; err != nil {
				t.Errorf("Concurrent serialization error: %v", err)
			}
		}
	})
}

// TestSerializationPerformance tests performance characteristics of serialization
func TestSerializationPerformance(t *testing.T) {
	t.Run("Serialization performance with stack traces", func(t *testing.T) {
		err := erro.New("performance test", erro.StackTrace())

		start := time.Now()
		for i := 0; i < 1000; i++ {
			jsonData, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Errorf("Marshal failed: %v", marshalErr)
			}

			newErr := erro.New("")
			unmarshalErr := json.Unmarshal(jsonData, newErr)
			if unmarshalErr != nil {
				t.Errorf("Unmarshal failed: %v", unmarshalErr)
			}
		}
		duration := time.Since(start)

		if duration > 3*time.Second {
			t.Errorf("Serialization performance too slow: %v", duration)
		}
	})

	t.Run("Memory allocation during serialization", func(t *testing.T) {
		err := erro.New("memory test",
			"data", strings.Repeat("x", 1000),
			erro.StackTrace(),
		)

		// Test that repeated serialization doesn't cause memory leaks
		for i := 0; i < 1000; i++ {
			jsonData, _ := json.Marshal(err)
			_ = jsonData // Use the data to prevent optimization
		}

		// This test mainly ensures no panics or excessive memory growth
		// In a real scenario, you'd monitor memory usage here
	})
}
