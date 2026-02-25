//go:build !integration

package cli

import (
	"encoding/json"
	"testing"
)

func TestMcpErrorData_Nil(t *testing.T) {
	result := mcpErrorData(nil)
	if result != nil {
		t.Errorf("mcpErrorData(nil) = %v, want nil", result)
	}
}

func TestMcpErrorData_String(t *testing.T) {
	result := mcpErrorData("hello")
	if result == nil {
		t.Fatal("mcpErrorData(string) = nil, want non-nil")
	}
	var got string
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if got != "hello" {
		t.Errorf("mcpErrorData(\"hello\") = %q, want %q", got, "hello")
	}
}

func TestMcpErrorData_Map(t *testing.T) {
	input := map[string]any{"error": "something went wrong", "code": 42}
	result := mcpErrorData(input)
	if result == nil {
		t.Fatal("mcpErrorData(map) = nil, want non-nil")
	}
	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if got["error"] != "something went wrong" {
		t.Errorf("mcpErrorData map[\"error\"] = %v, want %q", got["error"], "something went wrong")
	}
}

func TestMcpErrorData_UnmarshalableType(t *testing.T) {
	// channels cannot be marshaled to JSON
	ch := make(chan struct{})
	result := mcpErrorData(ch)
	// Should return nil without panicking
	if result != nil {
		t.Errorf("mcpErrorData(channel) = %v, want nil", result)
	}
}

func TestBoolPtr_True(t *testing.T) {
	p := boolPtr(true)
	if p == nil {
		t.Fatal("boolPtr(true) = nil, want non-nil pointer")
	}
	if *p != true {
		t.Errorf("*boolPtr(true) = %v, want true", *p)
	}
}

func TestBoolPtr_False(t *testing.T) {
	p := boolPtr(false)
	if p == nil {
		t.Fatal("boolPtr(false) = nil, want non-nil pointer")
	}
	if *p != false {
		t.Errorf("*boolPtr(false) = %v, want false", *p)
	}
}

func TestBoolPtr_Independence(t *testing.T) {
	// Verify that two calls return independent pointers
	p1 := boolPtr(true)
	p2 := boolPtr(true)
	if p1 == p2 {
		t.Error("boolPtr should return distinct pointers on each call")
	}
}

func TestHasWriteAccess(t *testing.T) {
	tests := []struct {
		permission string
		want       bool
	}{
		{"admin", true},
		{"maintain", true},
		{"write", true},
		{"triage", false},
		{"read", false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.permission, func(t *testing.T) {
			got := hasWriteAccess(tt.permission)
			if got != tt.want {
				t.Errorf("hasWriteAccess(%q) = %v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestValidateWorkflowName_Empty(t *testing.T) {
	// Empty workflow name is always valid (means "all workflows")
	if err := validateWorkflowName(""); err != nil {
		t.Errorf("validateWorkflowName(\"\") returned error: %v", err)
	}
}
