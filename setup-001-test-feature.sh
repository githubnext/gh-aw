#!/bin/bash
# Script to complete the setup for 001-test-feature implementation
# This script creates the directory structure and copies implementation files

set -e

echo "Creating pkg/test directory..."
mkdir -p pkg/test

echo "Creating test_feature.go..."
cat > pkg/test/test_feature.go <<'EOF'
package test

// TestFeature is a simple proof-of-concept feature to validate the spec-kit-execute workflow
type TestFeature struct {
	Name string
}

// NewTestFeature creates a new TestFeature instance
func NewTestFeature(name string) *TestFeature {
	return &TestFeature{Name: name}
}

// GetMessage returns a greeting message
func (tf *TestFeature) GetMessage() string {
	if tf.Name == "" {
		return "Hello from test feature"
	}
	return "Hello from " + tf.Name
}
EOF

echo "Creating test_feature_test.go..."
cat > pkg/test/test_feature_test.go <<'EOF'
package test

import (
	"testing"
)

func TestNewTestFeature(t *testing.T) {
	tf := NewTestFeature("SpecKit")
	if tf == nil {
		t.Fatal("NewTestFeature returned nil")
	}
	if tf.Name != "SpecKit" {
		t.Errorf("Expected Name to be 'SpecKit', got '%s'", tf.Name)
	}
}

func TestGetMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with name",
			input:    "SpecKit",
			expected: "Hello from SpecKit",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "Hello from test feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := NewTestFeature(tt.input)
			got := tf.GetMessage()
			if got != tt.expected {
				t.Errorf("GetMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}
EOF

echo "Running validation..."
make fmt
make lint  
make test-unit

echo "✓ Implementation complete! All files created and tests passing."
