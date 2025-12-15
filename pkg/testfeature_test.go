package pkg

import (
	"testing"
)

func TestNewTestFeature(t *testing.T) {
	name := "test-feature"
	feature := NewTestFeature(name)

	if feature == nil {
		t.Fatal("NewTestFeature returned nil")
	}

	if feature.Name != name {
		t.Errorf("Expected name %q, got %q", name, feature.Name)
	}

	if !feature.Enabled {
		t.Error("Expected feature to be enabled by default")
	}
}

func TestIsEnabled(t *testing.T) {
	feature := NewTestFeature("test")
	
	if !feature.IsEnabled() {
		t.Error("Expected IsEnabled to return true")
	}

	feature.Enabled = false
	if feature.IsEnabled() {
		t.Error("Expected IsEnabled to return false after disabling")
	}
}

func TestGetName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"feature1", "feature1"},
		{"test-feature", "test-feature"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := NewTestFeature(tt.name)
			if got := feature.GetName(); got != tt.want {
				t.Errorf("GetName() = %q, want %q", got, tt.want)
			}
		})
	}
}
