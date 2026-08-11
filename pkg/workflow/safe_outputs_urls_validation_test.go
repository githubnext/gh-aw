//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestValidateSafeOutputsURLs(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr bool
		errText string
	}{
		{name: "nil config", config: nil, wantErr: false, errText: ""},
		{name: "empty policy", config: &SafeOutputsConfig{}, wantErr: false, errText: ""},
		{name: "allowed-only", config: &SafeOutputsConfig{URLs: SafeOutputsURLsPolicyAllowedOnly}, wantErr: false, errText: ""},
		{name: "allowed-or-code-region", config: &SafeOutputsConfig{URLs: SafeOutputsURLsPolicyAllowedOrCodeRegion}, wantErr: false, errText: ""},
		{name: "invalid", config: &SafeOutputsConfig{URLs: "unknown"}, wantErr: true, errText: "Example:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsURLs(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSafeOutputsURLs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errText != "" && (err == nil || !strings.Contains(err.Error(), tt.errText)) {
				t.Fatalf("validateSafeOutputsURLs() error = %v, should contain %q", err, tt.errText)
			}
		})
	}
}
