package erro_test

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/maxbolgarin/erro"
)

// TestInputValidationEdgeCases tests various edge cases in input validation
func TestInputValidationEdgeCases(t *testing.T) {
	t.Run("Extreme numeric values", func(t *testing.T) {
		extremeValues := []any{
			math.MaxInt64,
			math.MinInt64,
			math.MaxFloat64,
			math.SmallestNonzeroFloat64,
			math.Inf(1),
			math.Inf(-1),
			math.NaN(),
			uint64(math.MaxUint64),
		}

		for i, val := range extremeValues {
			err := erro.New("extreme value test", fmt.Sprintf("key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle extreme value %v", val)
			}
		}
	})

	t.Run("Complex data types", func(t *testing.T) {
		complexValues := []any{
			complex(1.5, 2.5),
			complex(math.Inf(1), math.NaN()),
			make(chan int),
			make(chan struct{}, 100),
			func() {},
			unsafe.Pointer(nil),
		}

		for i, val := range complexValues {
			err := erro.New("complex type test", fmt.Sprintf("key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle complex value %v of type %T", val, val)
			}
		}
	})

	t.Run("Reflection edge cases", func(t *testing.T) {
		// Test with reflect.Value types
		reflectValues := []any{
			reflect.ValueOf(42),
			reflect.ValueOf("string"),
			reflect.ValueOf([]int{1, 2, 3}),
			reflect.Zero(reflect.TypeOf("")),
			reflect.ValueOf(nil),
		}

		for i, val := range reflectValues {
			err := erro.New("reflect test", fmt.Sprintf("reflect_key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle reflect value %v", val)
			}
		}
	})

	t.Run("Interface edge cases", func(t *testing.T) {
		// Test with various interface{} values
		var nilInterface interface{}
		var typedNil *string
		var emptyInterface interface{} = (*string)(nil)

		interfaceValues := []any{
			nilInterface,
			typedNil,
			emptyInterface,
			interface{}(42),
			interface{}(nil),
		}

		for i, val := range interfaceValues {
			err := erro.New("interface test", fmt.Sprintf("interface_key_%d", i), val)
			if err.Error() == "" {
				t.Errorf("Error should handle interface value %v", val)
			}
		}
	})
}

// TestBoundaryConditionValidation tests boundary conditions
func TestBoundaryConditionValidation(t *testing.T) {
	t.Run("String length boundaries", func(t *testing.T) {
		testCases := []struct {
			name   string
			length int
		}{
			{"single char", 1},
			{"at max key length", erro.MaxKeyLength},
			{"beyond max key length", erro.MaxKeyLength + 1},
			{"at max value length", erro.MaxValueLength},
			{"beyond max value length", erro.MaxValueLength + 1},
			{"at max message length", erro.MaxMessageLength},
			{"beyond max message length", erro.MaxMessageLength + 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testString := strings.Repeat("x", tc.length)

				// Test as message
				err := erro.New(testString)
				if err.Error() == "" {
					t.Error("Should handle string as message")
				}

				// Test as key
				err = erro.New("test", testString, "value")
				if err.Error() == "" {
					t.Error("Should handle string as key")
				}

				// Test as value
				err = erro.New("test", "key", testString)
				if err.Error() == "" {
					t.Error("Should handle string as value")
				}
			})
		}
	})

	t.Run("Field count boundaries", func(t *testing.T) {
		testCases := []struct {
			name       string
			fieldCount int
		}{
			{"no fields", 0},
			{"single field pair", 2},
			{"at max fields", erro.MaxFieldsCount * 2},
			{"beyond max fields", erro.MaxFieldsCount*2 + 10},
			{"odd field count", erro.MaxFieldsCount*2 + 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fields := make([]any, tc.fieldCount)
				for i := 0; i < tc.fieldCount; i++ {
					if i%2 == 0 {
						fields[i] = fmt.Sprintf("key_%d", i/2)
					} else {
						fields[i] = fmt.Sprintf("value_%d", i/2)
					}
				}

				err := erro.New("boundary test", fields...)
				if err.Error() == "" {
					t.Error("Should handle field boundary conditions")
				}

				// Check that fields are properly limited
				actualFields := err.Fields()
				maxExpected := erro.MaxFieldsCount * 2
				if len(actualFields) > maxExpected {
					t.Errorf("Fields should be limited to %d, got %d", maxExpected, len(actualFields))
				}
			})
		}
	})

	t.Run("Stack depth boundaries", func(t *testing.T) {
		// Test with maximum stack depth
		err := erro.New("stack depth test", erro.StackTrace())
		stack := err.Stack()

		if len(stack) > erro.MaxStackDepth {
			t.Errorf("Stack depth should be limited to %d, got %d", erro.MaxStackDepth, len(stack))
		}

		// Test stack trace with deep call stack
		deepErr := createDeepStackTrace(50)
		deepStack := deepErr.Stack()

		if len(deepStack) > erro.MaxStackDepth {
			t.Errorf("Deep stack should be limited to %d, got %d", erro.MaxStackDepth, len(deepStack))
		}
	})
}

// TestMalformedDataHandling tests handling of malformed or corrupted data
func TestMalformedDataHandling(t *testing.T) {
	t.Run("Invalid UTF-8 sequences", func(t *testing.T) {
		invalidUTF8 := []string{
			"\xff\xfe\xfd",                         // Invalid byte sequences
			"valid\xff\xfeinvalid",                 // Mixed valid/invalid
			string([]byte{0xff, 0xfe, 0xfd, 0xfc}), // Raw invalid bytes
		}

		for i, invalid := range invalidUTF8 {
			err := erro.New("utf8 test", fmt.Sprintf("key_%d", i), invalid)
			if err.Error() == "" {
				t.Error("Should handle invalid UTF-8")
			}

			// Should not panic when converting to string
			_ = err.Error()
		}
	})

	t.Run("Zero-width and invisible characters", func(t *testing.T) {
		invisibleChars := []string{
			"\u200b", // Zero width space
			"\u200c", // Zero width non-joiner
			"\u200d", // Zero width joiner
			"\ufeff", // Byte order mark
			"\u2060", // Word joiner
		}

		for i, char := range invisibleChars {
			err := erro.New("invisible char test", fmt.Sprintf("key_%d", i), char)
			if err.Error() == "" {
				t.Error("Should handle invisible characters")
			}
		}
	})

	t.Run("Control characters and escapes", func(t *testing.T) {
		controlChars := []string{
			"\x00",     // Null
			"\x07",     // Bell
			"\x08",     // Backspace
			"\x1b[31m", // ANSI escape sequence
			"\x1b[0m",  // ANSI reset
		}

		for i, char := range controlChars {
			err := erro.New("control char test", fmt.Sprintf("key_%d", i), char)
			if err.Error() == "" {
				t.Error("Should handle control characters")
			}
		}
	})

	t.Run("Extremely nested structures", func(t *testing.T) {
		// Create deeply nested map
		depth := 100
		nested := make(map[string]interface{})
		current := nested

		for i := 0; i < depth; i++ {
			next := make(map[string]interface{})
			current[fmt.Sprintf("level_%d", i)] = next
			current = next
		}
		current["final"] = "value"

		err := erro.New("nested test", "nested_data", nested)
		if err.Error() == "" {
			t.Error("Should handle deeply nested structures")
		}

		// Should not cause stack overflow
		_ = err.Error()
	})
}

// TestDataTypeValidation tests validation of various Go data types
func TestDataTypeValidation(t *testing.T) {
	t.Run("Slice and array edge cases", func(t *testing.T) {
		sliceTests := []any{
			[]int{},                 // Empty slice
			[]int{1, 2, 3},          // Regular slice
			make([]int, 0, 1000),    // High capacity, zero length
			make([]string, 1000),    // Large slice
			[...]int{1, 2, 3, 4, 5}, // Array
			(*[]int)(nil),           // Nil slice pointer
		}

		for i, slice := range sliceTests {
			err := erro.New("slice test", fmt.Sprintf("slice_%d", i), slice)
			if err.Error() == "" {
				t.Errorf("Should handle slice type %T", slice)
			}
		}
	})

	t.Run("Map edge cases", func(t *testing.T) {
		mapTests := []any{
			map[string]int{},                      // Empty map
			map[string]int{"key": 1},              // Regular map
			make(map[string]int, 1000),            // Large capacity map
			map[interface{}]interface{}{1: "one"}, // Interface{} map
			(*map[string]int)(nil),                // Nil map pointer
		}

		for i, m := range mapTests {
			err := erro.New("map test", fmt.Sprintf("map_%d", i), m)
			if err.Error() == "" {
				t.Errorf("Should handle map type %T", m)
			}
		}
	})

	t.Run("Pointer edge cases", func(t *testing.T) {
		value := 42
		var nilPtr *int

		pointerTests := []any{
			&value,                          // Valid pointer
			nilPtr,                          // Nil pointer
			&nilPtr,                         // Pointer to nil pointer
			unsafe.Pointer(&value),          // Unsafe pointer
			uintptr(unsafe.Pointer(&value)), // Uintptr
		}

		for i, ptr := range pointerTests {
			err := erro.New("pointer test", fmt.Sprintf("ptr_%d", i), ptr)
			if err.Error() == "" {
				t.Errorf("Should handle pointer type %T", ptr)
			}
		}
	})

	t.Run("Time and duration edge cases", func(t *testing.T) {
		timeTests := []any{
			time.Time{},     // Zero time
			time.Now(),      // Current time
			time.Unix(0, 0), // Unix epoch
			time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC), // Far future
			time.Duration(0),           // Zero duration
			time.Nanosecond,            // Smallest duration
			time.Hour * 24 * 365 * 100, // Large duration
			-time.Hour,                 // Negative duration
		}

		for i, timeVal := range timeTests {
			err := erro.New("time test", fmt.Sprintf("time_%d", i), timeVal)
			if err.Error() == "" {
				t.Errorf("Should handle time type %T", timeVal)
			}
		}
	})
}

// TestMemoryAndGarbageCollection tests memory-related edge cases
func TestMemoryAndGarbageCollection(t *testing.T) {
	t.Run("Large field values with GC pressure", func(t *testing.T) {
		// Create errors with large data and force GC
		for i := 0; i < 100; i++ {
			largeData := make([]byte, 10000)
			for j := range largeData {
				largeData[j] = byte(j % 256)
			}

			err := erro.New("gc test", "large_data", largeData, "iteration", i)
			_ = err.Error() // Force string creation

			if i%10 == 0 {
				runtime.GC() // Force garbage collection
			}
		}

		// Test should complete without memory issues
		runtime.GC()
	})

	t.Run("Reference cycles with error fields", func(t *testing.T) {
		type cyclicStruct struct {
			Name string
			Ref  *cyclicStruct
		}

		root := &cyclicStruct{Name: "root"}
		child := &cyclicStruct{Name: "child", Ref: root}
		root.Ref = child // Create cycle

		err := erro.New("cycle test", "cyclic_data", root)
		_ = err.Error()

		// Should not prevent garbage collection
		root = nil
		child = nil
		runtime.GC()
	})
}

// Helper function to create deep stack trace
func createDeepStackTrace(depth int) erro.Error {
	if depth <= 0 {
		return erro.New("deep stack trace", erro.StackTrace())
	}
	return createDeepStackTrace(depth - 1)
}
