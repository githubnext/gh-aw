package cli

import "testing"

func TestCompileConfig_ShellcheckEnabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		config      CompileConfig
		wantEnabled bool
	}{
		{
			name:        "disabled by default",
			config:      CompileConfig{},
			wantEnabled: false,
		},
		{
			name: "validate does not enable shellcheck",
			config: CompileConfig{
				Validate: true,
			},
			wantEnabled: false,
		},
		{
			name: "shellcheck flag enables shellcheck",
			config: CompileConfig{
				Shellcheck: true,
			},
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.shellcheckEnabled(); got != tt.wantEnabled {
				t.Fatalf("shellcheckEnabled() = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}
