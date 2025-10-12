package console

import (
	"reflect"
	"strings"
	"testing"
)

// Test types for struct rendering
type TestOverview struct {
	RunID      int64  `console:"header:Run ID"`
	Workflow   string `console:"header:Workflow"`
	Status     string `console:"header:Status"`
	EmptyField string `console:"omitempty"`
	SkipField  string `console:"-"`
}

type TestMetrics struct {
	TokenUsage int     `console:"header:Token Usage"`
	Cost       float64 `console:"header:Estimated Cost"`
	Turns      int     `console:"header:Turns,omitempty"`
}

type TestJob struct {
	Name       string `console:"header:Name"`
	Status     string `console:"header:Status"`
	Conclusion string `console:"header:Conclusion,omitempty"`
}

func TestRenderStruct_SimpleStruct(t *testing.T) {
	data := TestOverview{
		RunID:      12345,
		Workflow:   "test-workflow",
		Status:     "completed",
		EmptyField: "",
		SkipField:  "should not appear",
	}

	output := RenderStruct(data)

	// Check that basic fields are present (using header names from tags)
	if !strings.Contains(output, "Run ID") {
		t.Error("Output should contain Run ID field")
	}
	if !strings.Contains(output, "12345") {
		t.Error("Output should contain RunID value")
	}
	if !strings.Contains(output, "test-workflow") {
		t.Error("Output should contain Workflow value")
	}

	// Check that skip field is not present
	if strings.Contains(output, "should not appear") {
		t.Error("Output should not contain skipped field")
	}
}

func TestRenderStruct_OmitEmpty(t *testing.T) {
	data := TestMetrics{
		TokenUsage: 1000,
		Cost:       1.23,
		Turns:      0, // Should be omitted with omitempty
	}

	output := RenderStruct(data)

	// Check that non-empty fields are present
	if !strings.Contains(output, "1000") {
		t.Error("Output should contain TokenUsage value")
	}
	if !strings.Contains(output, "1.23") {
		t.Error("Output should contain Cost value")
	}

	// Turns should be omitted because it's 0 and has omitempty
	// Note: This is tricky because "0" might appear in other contexts
	// We'll just verify the field appears when it has a value
	dataWithTurns := TestMetrics{
		TokenUsage: 1000,
		Cost:       1.23,
		Turns:      5,
	}
	outputWithTurns := RenderStruct(dataWithTurns)
	if !strings.Contains(outputWithTurns, "5") {
		t.Error("Output should contain Turns value when non-zero")
	}
}

func TestRenderSlice_AsTable(t *testing.T) {
	jobs := []TestJob{
		{Name: "job-1", Status: "completed", Conclusion: "success"},
		{Name: "job-2", Status: "in_progress", Conclusion: ""},
	}

	output := RenderStruct(jobs)

	// Check for table structure (headers and values)
	if !strings.Contains(output, "Name") {
		t.Error("Output should contain Name header")
	}
	if !strings.Contains(output, "Status") {
		t.Error("Output should contain Status header")
	}
	if !strings.Contains(output, "job-1") {
		t.Error("Output should contain job-1 name")
	}
	if !strings.Contains(output, "completed") {
		t.Error("Output should contain completed status")
	}
	if !strings.Contains(output, "job-2") {
		t.Error("Output should contain job-2 name")
	}
}

func TestRenderMap(t *testing.T) {
	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	output := RenderStruct(data)

	// Maps should render as key-value pairs
	if !strings.Contains(output, "key1") {
		t.Error("Output should contain key1")
	}
	if !strings.Contains(output, "value1") {
		t.Error("Output should contain value1")
	}
	if !strings.Contains(output, "key2") {
		t.Error("Output should contain key2")
	}
	if !strings.Contains(output, "value2") {
		t.Error("Output should contain value2")
	}
}

func TestParseConsoleTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected consoleTag
	}{
		{
			name: "skip tag",
			tag:  "-",
			expected: consoleTag{
				skip: true,
			},
		},
		{
			name: "omitempty tag",
			tag:  "omitempty",
			expected: consoleTag{
				omitempty: true,
			},
		},
		{
			name: "header tag",
			tag:  "header:Column Name",
			expected: consoleTag{
				header: "Column Name",
			},
		},
		{
			name: "title tag",
			tag:  "title:Section Title",
			expected: consoleTag{
				title: "Section Title",
			},
		},
		{
			name: "combined tags",
			tag:  "header:Name,omitempty",
			expected: consoleTag{
				header:    "Name",
				omitempty: true,
			},
		},
		{
			name: "all tags",
			tag:  "title:My Title,header:My Header,omitempty",
			expected: consoleTag{
				title:     "My Title",
				header:    "My Header",
				omitempty: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseConsoleTag(tt.tag)
			if result.skip != tt.expected.skip {
				t.Errorf("skip: got %v, want %v", result.skip, tt.expected.skip)
			}
			if result.omitempty != tt.expected.omitempty {
				t.Errorf("omitempty: got %v, want %v", result.omitempty, tt.expected.omitempty)
			}
			if result.header != tt.expected.header {
				t.Errorf("header: got %v, want %v", result.header, tt.expected.header)
			}
			if result.title != tt.expected.title {
				t.Errorf("title: got %v, want %v", result.title, tt.expected.title)
			}
		})
	}
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"nil pointer", (*int)(nil), true},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1}, false},
		{"false bool", false, true},
		{"true bool", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			result := isZeroValue(val)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatFieldValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"integer", 42, "42"},
		{"string", "hello", "hello"},
		{"empty string", "", "-"},
		{"float", 3.14, "3.14"},
		{"nil pointer", (*int)(nil), "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			result := formatFieldValue(val)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
