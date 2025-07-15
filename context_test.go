package erro_test

import (
	"errors"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestExtractError(t *testing.T) {
	stdErr := errors.New("std error")
	err := erro.ExtractError(stdErr)
	if err.Message() != "std error" {
		t.Errorf("Expected message 'std error', got '%s'", err.Message())
	}

	erroErr := erro.New("erro error")
	err = erro.ExtractError(erroErr)
	if err != erroErr {
		t.Errorf("Expected extracted error to be the same")
	}
}

func TestLogFields(t *testing.T) {
	err := erro.New("test message", "key", "value")
	fields := erro.LogFields(err)
	if len(fields) < 2 {
		t.Fatalf("Expected at least 2 fields, got %d", len(fields))
	}
	found := false
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "key" && fields[i+1] == "value" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find key 'key' with value 'value'")
	}
}

func TestLogFieldsMap(t *testing.T) {
	err := erro.New("test message", "key", "value")
	fields := erro.LogFieldsMap(err)
	if val, ok := fields["key"]; !ok || val != "value" {
		t.Errorf("Expected to find key 'key' with value 'value'")
	}
}

func TestLogError(t *testing.T) {
	err := erro.New("test message", "key", "value")
	logged := false
	logFunc := func(message string, fields ...any) {
		logged = true
		if message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", message)
		}
		if len(fields) < 2 {
			t.Fatalf("Expected at least 2 fields, got %d", len(fields))
		}
	}
	erro.LogError(err, logFunc)
	if !logged {
		t.Errorf("Expected log function to be called")
	}
}
