//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSafeOutputsURLPolicy(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr bool
		errText string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "default policy when unset",
			config:  &SafeOutputsConfig{},
			wantErr: false,
		},
		{
			name: "allowlist policy",
			config: &SafeOutputsConfig{
				URLPolicy: "allowlist",
			},
			wantErr: false,
		},
		{
			name: "audit policy",
			config: &SafeOutputsConfig{
				URLPolicy: "audit",
			},
			wantErr: false,
		},
		{
			name: "reputation policy with provider",
			config: &SafeOutputsConfig{
				URLPolicy: "reputation",
				Reputation: &SafeOutputsReputationConfig{
					Provider: "google-safe-browsing",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid policy value",
			config: &SafeOutputsConfig{
				URLPolicy: "permissive",
			},
			wantErr: true,
			errText: "safe-outputs.url-policy: invalid value",
		},
		{
			name: "reputation policy requires provider",
			config: &SafeOutputsConfig{
				URLPolicy: "reputation",
			},
			wantErr: true,
			errText: "safe-outputs.url-policy: reputation mode requires safe-outputs.reputation.provider",
		},
		{
			name: "invalid reputation provider",
			config: &SafeOutputsConfig{
				URLPolicy: "reputation",
				Reputation: &SafeOutputsReputationConfig{
					Provider: "virustotal",
				},
			},
			wantErr: true,
			errText: "safe-outputs.reputation.provider: invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsURLPolicy(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.Contains(t, err.Error(), tt.errText)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}
