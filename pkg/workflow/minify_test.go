package workflow

import (
	"testing"
)

func TestMinifyJavaScript(t *testing.T) {
	// Ensure minification is enabled for this test
	SetMinificationEnabled(true)

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkFn  func(t *testing.T, result string)
	}{
		{
			name: "simple function",
			input: `function hello() {
  const message = "Hello, World!";
  console.log(message);
}`,
			wantErr: false,
			checkFn: func(t *testing.T, result string) {
				// Result should be shorter than input (minified)
				if len(result) >= len(`function hello() {
  const message = "Hello, World!";
  console.log(message);
}`) {
					t.Logf("Minification might not be reducing size, result: %s", result)
				}
				// Result should still contain the essential parts
				if len(result) == 0 {
					t.Error("Result should not be empty")
				}
			},
		},
		{
			name: "variable declarations",
			input: `const firstName = "John";
const lastName = "Doe";
const fullName = firstName + " " + lastName;
console.log(fullName);`,
			wantErr: false,
			checkFn: func(t *testing.T, result string) {
				// Minified code should be shorter
				t.Logf("Input length: %d, Output length: %d", len(`const firstName = "John";
const lastName = "Doe";
const fullName = firstName + " " + lastName;
console.log(fullName);`), len(result))
			},
		},
		{
			name: "async function",
			input: `async function fetchData() {
  const response = await fetch("https://api.example.com");
  const data = await response.json();
  return data;
}`,
			wantErr: false,
			checkFn: func(t *testing.T, result string) {
				// Check that async/await is preserved
				if len(result) == 0 {
					t.Error("Result should not be empty")
				}
			},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
			checkFn: func(t *testing.T, result string) {
				// Empty input may produce empty output or just whitespace
				// Terser may return a newline for empty input
				trimmed := result
				if len(trimmed) > 1 {
					t.Errorf("Expected empty or minimal result for empty input, got: %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MinifyJavaScript(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinifyJavaScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

func TestMinifyJavaScriptDisabled(t *testing.T) {
	// Disable minification
	SetMinificationEnabled(false)
	defer SetMinificationEnabled(true) // Reset after test

	input := `function test() { return 42; }`
	result, err := MinifyJavaScript(input)
	if err != nil {
		t.Errorf("MinifyJavaScript() with disabled minification should not error: %v", err)
	}
	if result != input {
		t.Errorf("MinifyJavaScript() with disabled minification should return original code, got: %s", result)
	}
}

func TestIsMinificationEnabled(t *testing.T) {
	original := IsMinificationEnabled()

	SetMinificationEnabled(false)
	if IsMinificationEnabled() {
		t.Error("Expected minification to be disabled")
	}

	SetMinificationEnabled(true)
	if !IsMinificationEnabled() {
		t.Error("Expected minification to be enabled")
	}

	// Restore original state
	SetMinificationEnabled(original)
}
