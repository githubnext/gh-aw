//go:build !integration

package workflow

import "testing"

func TestValidateSafeOutputsURLs(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr bool
	}{
		{name: "nil config", config: nil, wantErr: false},
		{name: "empty policy", config: &SafeOutputsConfig{}, wantErr: false},
		{name: "allowed-only", config: &SafeOutputsConfig{URLs: SafeOutputsURLsPolicyAllowedOnly}, wantErr: false},
		{name: "allowed-or-code-region", config: &SafeOutputsConfig{URLs: SafeOutputsURLsPolicyAllowedOrCodeRegion}, wantErr: false},
		{name: "invalid", config: &SafeOutputsConfig{URLs: "unknown"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsURLs(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSafeOutputsURLs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
