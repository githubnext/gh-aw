package testutil

import "testing"

func TestBasicFunctionality(t *testing.T) {
	result := BasicTestFunction()
	expected := "test feature works"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
