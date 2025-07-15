package erro_test

import (
	"encoding/json"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestMarshalJSON(t *testing.T) {
	err := erro.New("test message",
		erro.ID("test_id"),
		erro.ClassValidation,
		erro.CategoryDatabase,
		erro.SeverityHigh,
		erro.Retryable(),
		"key", "value",
		"sensitive", erro.Redact("secret"),
	)

	b, jErr := json.Marshal(err)
	if jErr != nil {
		t.Fatalf("json.Marshal failed: %v", jErr)
	}

	var data map[string]interface{}
	if jErr = json.Unmarshal(b, &data); jErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", jErr)
	}

	if data["id"] != "test_id" {
		t.Errorf("Expected id 'test_id', got '%s'", data["id"])
	}
	if data["class"] != "validation" {
		t.Errorf("Expected class 'validation', got '%s'", data["class"])
	}
	if data["category"] != "database" {
		t.Errorf("Expected category 'database', got '%s'", data["category"])
	}
	if data["severity"] != "high" {
		t.Errorf("Expected severity 'high', got '%s'", data["severity"])
	}
	if data["message"] != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", data["message"])
	}
	if data["retryable"] != true {
		t.Errorf("Expected retryable to be true, got %v", data["retryable"])
	}
	if _, ok := data["created"]; !ok {
		t.Errorf("Expected 'created' field to exist")
	}

	fields, ok := data["fields"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'fields' to be a slice, got %T", data["fields"])
	}
	if len(fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(fields))
	}
	if fields[0] != "key" || fields[1] != "value" {
		t.Errorf("Unexpected fields: %v", fields)
	}
	if fields[2] != "sensitive" || fields[3] != erro.RedactedPlaceholder {
		t.Errorf("Expected redacted field, got '%s' and '%s'", fields[2], fields[3])
	}
}

func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test_id",
		"class": "validation",
		"category": "database",
		"severity": "high",
		"message": "test message",
		"retryable": true,
		"fields": ["key", "value"]
	}`

	err := erro.New("")
	jErr := json.Unmarshal([]byte(jsonData), err)
	if jErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", jErr)
	}

	if err.ID() != "test_id" {
		t.Errorf("Expected ID 'test_id', got '%s'", err.ID())
	}
	if err.Class() != erro.ClassValidation {
		t.Errorf("Expected class 'validation', got '%s'", err.Class())
	}
	if err.Category() != erro.CategoryDatabase {
		t.Errorf("Expected category 'database', got '%s'", err.Category())
	}
	if err.Severity() != erro.SeverityHigh {
		t.Errorf("Expected severity 'high', got '%s'", err.Severity())
	}
	if err.Message() != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message())
	}
	if !err.IsRetryable() {
		t.Errorf("Expected retryable to be true")
	}
	fields := err.Fields()
	if len(fields) != 2 || fields[0] != "key" || fields[1] != "value" {
		t.Errorf("Unexpected fields: %v", fields)
	}
}

func TestMarshalJSONWithStack(t *testing.T) {
	err := erro.New("test error", erro.StackTrace())

	b, jErr := json.Marshal(err)
	if jErr != nil {
		t.Fatalf("json.Marshal failed: %v", jErr)
	}

	var data map[string]interface{}
	if jErr = json.Unmarshal(b, &data); jErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", jErr)
	}

	if _, ok := data["stack_trace"]; !ok {
		t.Errorf("Expected 'stack_trace' field to exist")
	}
}
