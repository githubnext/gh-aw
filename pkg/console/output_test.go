package console

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestOutputStructOrJSON_JSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name" console:"header:Name"`
		Count int    `json:"count" console:"header:Count"`
		Value string `json:"value" console:"header:Value"`
	}

	data := TestStruct{
		Name:  "test",
		Count: 42,
		Value: "example",
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function with asJSON=true
	err := OutputStructOrJSON(data, true)
	w.Close()

	// Read captured output
	var buf strings.Builder
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("OutputStructOrJSON returned error: %v", err)
	}

	output := buf.String()

	// Verify it's valid JSON
	var parsed TestStruct
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify values
	if parsed.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", parsed.Name)
	}
	if parsed.Count != 42 {
		t.Errorf("Expected count 42, got %d", parsed.Count)
	}
	if parsed.Value != "example" {
		t.Errorf("Expected value 'example', got '%s'", parsed.Value)
	}
}

func TestOutputStructOrJSON_Console(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name" console:"header:Name"`
		Count int    `json:"count" console:"header:Count"`
	}

	data := TestStruct{
		Name:  "test",
		Count: 42,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function with asJSON=false
	err := OutputStructOrJSON(data, false)
	w.Close()

	// Read captured output
	var buf strings.Builder
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("OutputStructOrJSON returned error: %v", err)
	}

	output := buf.String()

	// Verify it contains console-formatted output
	if !strings.Contains(output, "Name") {
		t.Errorf("Console output should contain 'Name' header")
	}
	if !strings.Contains(output, "Count") {
		t.Errorf("Console output should contain 'Count' header")
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Console output should contain value 'test'")
	}
	if !strings.Contains(output, "42") {
		t.Errorf("Console output should contain value '42'")
	}

	// Verify it's NOT JSON
	var parsed TestStruct
	if err := json.Unmarshal([]byte(output), &parsed); err == nil {
		t.Error("Console output should not be valid JSON")
	}
}

func TestOutputStructOrJSON_ComplexStruct(t *testing.T) {
	type NestedStruct struct {
		Field1 string `json:"field1" console:"header:Field 1"`
		Field2 int    `json:"field2" console:"header:Field 2"`
	}

	type ComplexStruct struct {
		Title   string       `json:"title" console:"header:Title"`
		Count   int          `json:"count" console:"header:Count"`
		Nested  NestedStruct `json:"nested"`
		Enabled bool         `json:"enabled" console:"header:Enabled"`
	}

	data := ComplexStruct{
		Title: "Test Report",
		Count: 100,
		Nested: NestedStruct{
			Field1: "nested value",
			Field2: 200,
		},
		Enabled: true,
	}

	// Test JSON output
	t.Run("JSON output", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := OutputStructOrJSON(data, true)
		w.Close()

		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("OutputStructOrJSON returned error: %v", err)
		}

		output := buf.String()

		// Verify it's valid JSON
		var parsed ComplexStruct
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		// Verify nested structure is preserved
		if parsed.Nested.Field1 != "nested value" {
			t.Errorf("Expected nested field1 'nested value', got '%s'", parsed.Nested.Field1)
		}
	})

	// Test Console output
	t.Run("Console output", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := OutputStructOrJSON(data, false)
		w.Close()

		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("OutputStructOrJSON returned error: %v", err)
		}

		output := buf.String()

		// Verify console formatting
		if !strings.Contains(output, "Title") {
			t.Errorf("Console output should contain 'Title'")
		}
		if !strings.Contains(output, "Test Report") {
			t.Errorf("Console output should contain 'Test Report'")
		}
	})
}

func TestOutputStructOrJSON_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	data := EmptyStruct{}

	// Test JSON output
	t.Run("JSON output", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := OutputStructOrJSON(data, true)
		w.Close()

		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("OutputStructOrJSON returned error: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		// Empty struct should produce {}
		if output != "{}" {
			t.Errorf("Expected '{}' for empty struct, got '%s'", output)
		}
	})

	// Test Console output
	t.Run("Console output", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := OutputStructOrJSON(data, false)
		w.Close()

		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("OutputStructOrJSON returned error: %v", err)
		}

		// Console output for empty struct should not crash
		// It may be empty or contain minimal formatting
	})
}
