package testutil

import "testing"

func TestTestMessage(t *testing.T) {
	result := TestMessage()
	expected := "Spec-kit workflow is working!"
	
	if result != expected {
		t.Errorf("TestMessage() = %q, want %q", result, expected)
	}
}

func TestTestMessageNotEmpty(t *testing.T) {
	result := TestMessage()
	
	if result == "" {
		t.Error("TestMessage() returned empty string, want non-empty string")
	}
}
