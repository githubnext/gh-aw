package parser

import (
	"encoding/json"
	"testing"
)

func TestJSONParsing(t *testing.T) {
	// Test SHA resolution JSON parsing
	t.Run("parse SHA from commit response", func(t *testing.T) {
		response := `{"sha":"1e366aa4518cf83d25defd84e454b9a41e87cf7c","node_id":"C_kwDOKr1234","commit":{"message":"test"}}`

		var parsed struct {
			SHA     string `json:"sha"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(response), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed.SHA != "1e366aa4518cf83d25defd84e454b9a41e87cf7c" {
			t.Errorf("Expected SHA 1e366aa4518cf83d25defd84e454b9a41e87cf7c, got %s", parsed.SHA)
		}
	})

	// Test file content JSON parsing
	t.Run("parse content from file response", func(t *testing.T) {
		response := `{"content":"IyBUZXN0IGNvbnRlbnQ=\n","encoding":"base64","name":"test.md"}`

		var parsed struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
			Message  string `json:"message"`
		}

		if err := json.Unmarshal([]byte(response), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed.Encoding != "base64" {
			t.Errorf("Expected encoding base64, got %s", parsed.Encoding)
		}

		if parsed.Content == "" {
			t.Error("Expected non-empty content")
		}
	})

	// Test error response parsing
	t.Run("parse error response", func(t *testing.T) {
		response := `{"message":"Not Found","documentation_url":"https://docs.github.com/rest"}`

		var parsed struct {
			SHA     string `json:"sha"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(response), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if parsed.Message != "Not Found" {
			t.Errorf("Expected message 'Not Found', got %s", parsed.Message)
		}

		if parsed.SHA != "" {
			t.Errorf("Expected empty SHA for error response, got %s", parsed.SHA)
		}
	})
}
