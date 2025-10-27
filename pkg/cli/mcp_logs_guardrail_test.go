package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckLogsOutputSize_SmallOutput(t *testing.T) {
	// Create a small output (less than 100KB)
	smallOutput := `{"summary": {"total_runs": 1}, "runs": []}`

	result, triggered := checkLogsOutputSize(smallOutput)

	if triggered {
		t.Error("Guardrail should not be triggered for small output")
	}

	if result != smallOutput {
		t.Error("Output should be unchanged for small output")
	}
}

func TestCheckLogsOutputSize_LargeOutput(t *testing.T) {
	// Create a large output (more than 100KB)
	largeOutput := strings.Repeat("x", MaxMCPLogsOutputSize+1)

	result, triggered := checkLogsOutputSize(largeOutput)

	if !triggered {
		t.Error("Guardrail should be triggered for large output")
	}

	if result == largeOutput {
		t.Error("Output should be replaced with guardrail response for large output")
	}

	// Verify the result contains a valid JSON guardrail response
	var guardrail MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result), &guardrail); err != nil {
		t.Errorf("Guardrail response should be valid JSON: %v", err)
	}

	// Verify guardrail response structure
	if guardrail.Message == "" {
		t.Error("Guardrail response should have a message")
	}

	if guardrail.OutputSize != len(largeOutput) {
		t.Errorf("Guardrail should report correct output size: expected %d, got %d", len(largeOutput), guardrail.OutputSize)
	}

	if guardrail.OutputSizeLimit != MaxMCPLogsOutputSize {
		t.Errorf("Guardrail should report correct limit: expected %d, got %d", MaxMCPLogsOutputSize, guardrail.OutputSizeLimit)
	}

	if len(guardrail.SuggestedQueries) == 0 {
		t.Error("Guardrail response should have suggested queries")
	}

	if len(guardrail.Schema.Fields) == 0 {
		t.Error("Guardrail response should have schema fields")
	}
}

func TestCheckLogsOutputSize_ExactLimit(t *testing.T) {
	// Create output exactly at the limit
	exactOutput := strings.Repeat("x", MaxMCPLogsOutputSize)

	result, triggered := checkLogsOutputSize(exactOutput)

	if triggered {
		t.Error("Guardrail should not be triggered for output at exact limit")
	}

	if result != exactOutput {
		t.Error("Output should be unchanged for output at exact limit")
	}
}

func TestCheckLogsOutputSize_JustOverLimit(t *testing.T) {
	// Create output just over the limit (100KB + 1 byte)
	overOutput := strings.Repeat("x", MaxMCPLogsOutputSize+1)

	_, triggered := checkLogsOutputSize(overOutput)

	if !triggered {
		t.Error("Guardrail should be triggered for output just over limit")
	}
}

func TestGetLogsDataSchema(t *testing.T) {
	schema := getLogsDataSchema()

	// Verify basic schema structure
	if schema.Type != "object" {
		t.Errorf("Schema type should be 'object', got '%s'", schema.Type)
	}

	if schema.Description == "" {
		t.Error("Schema should have a description")
	}

	// Verify expected fields are present
	expectedFields := []string{
		"summary",
		"runs",
		"tool_usage",
		"errors_and_warnings",
		"missing_tools",
		"mcp_failures",
		"access_log",
		"firewall_log",
		"continuation",
		"logs_location",
	}

	for _, field := range expectedFields {
		if _, ok := schema.Fields[field]; !ok {
			t.Errorf("Schema should have field '%s'", field)
		}
	}

	// Verify each field has type and description
	for fieldName, field := range schema.Fields {
		if field.Type == "" {
			t.Errorf("Field '%s' should have a type", fieldName)
		}
		if field.Description == "" {
			t.Errorf("Field '%s' should have a description", fieldName)
		}
	}
}

func TestGetSuggestedJqQueries(t *testing.T) {
	queries := getSuggestedJqQueries()

	if len(queries) == 0 {
		t.Error("Should have at least one suggested query")
	}

	// Verify each query has required fields
	for i, query := range queries {
		if query.Description == "" {
			t.Errorf("Query %d should have a description", i)
		}
		if query.Query == "" {
			t.Errorf("Query %d should have a query string", i)
		}
	}

	// Verify we have some common useful queries
	hasBasicQueries := false
	for _, query := range queries {
		if strings.Contains(query.Query, ".summary") {
			hasBasicQueries = true
			break
		}
	}

	if !hasBasicQueries {
		t.Error("Should have basic summary query in suggestions")
	}
}

func TestFormatGuardrailMessage(t *testing.T) {
	guardrail := MCPLogsGuardrailResponse{
		Message:          "Test message",
		OutputSize:       150000,
		OutputSizeLimit:  MaxMCPLogsOutputSize,
		Schema:           getLogsDataSchema(),
		SuggestedQueries: getSuggestedJqQueries(),
	}

	message := formatGuardrailMessage(guardrail)

	// Verify message contains key components
	if !strings.Contains(message, "Test message") {
		t.Error("Formatted message should contain the original message")
	}

	if !strings.Contains(message, "Output Schema") {
		t.Error("Formatted message should contain schema section")
	}

	if !strings.Contains(message, "Suggested jq Queries") {
		t.Error("Formatted message should contain suggested queries section")
	}

	// Verify it mentions some fields
	if !strings.Contains(message, "summary") {
		t.Error("Formatted message should mention 'summary' field")
	}
}

func TestGuardrailResponseJSON(t *testing.T) {
	// Create a large output to trigger guardrail
	largeOutput := strings.Repeat("x", MaxMCPLogsOutputSize*2)

	result, triggered := checkLogsOutputSize(largeOutput)

	if !triggered {
		t.Fatal("Guardrail should be triggered")
	}

	// Parse the JSON response
	var guardrail MCPLogsGuardrailResponse
	if err := json.Unmarshal([]byte(result), &guardrail); err != nil {
		t.Fatalf("Should return valid JSON: %v", err)
	}

	// Verify the JSON structure is complete and valid
	if guardrail.Message == "" {
		t.Error("JSON should have message field")
	}

	if guardrail.OutputSize == 0 {
		t.Error("JSON should have output_size field")
	}

	if guardrail.OutputSizeLimit == 0 {
		t.Error("JSON should have output_size_limit field")
	}

	if guardrail.Schema.Type == "" {
		t.Error("JSON should have schema.type field")
	}

	if len(guardrail.Schema.Fields) == 0 {
		t.Error("JSON should have schema.fields")
	}

	if len(guardrail.SuggestedQueries) == 0 {
		t.Error("JSON should have suggested_queries")
	}

	// Verify each suggested query has the expected fields
	for i, query := range guardrail.SuggestedQueries {
		if query.Description == "" {
			t.Errorf("Query %d should have description in JSON", i)
		}
		if query.Query == "" {
			t.Errorf("Query %d should have query in JSON", i)
		}
	}
}

func TestMaxMCPLogsOutputSize_Constant(t *testing.T) {
	// Verify the constant is set to expected value (10KB)
	expected := 10 * 1024
	if MaxMCPLogsOutputSize != expected {
		t.Errorf("MaxMCPLogsOutputSize should be %d bytes (10KB), got %d", expected, MaxMCPLogsOutputSize)
	}
}
