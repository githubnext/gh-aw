//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAWFSupportsExcludeEnv verifies that --exclude-env is only enabled for AWF v0.25.3+.
func TestAWFSupportsExcludeEnv(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config (default version) supports --exclude-env",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version (default) supports --exclude-env",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "v0.25.3 supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.25.3"},
			want:           true,
		},
		{
			name:           "v0.26.0 supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.26.0"},
			want:           true,
		},
		{
			name:           "v0.27.0 supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.27.0"},
			want:           true,
		},
		{
			name:           "latest supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.0 does not support --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.25.0"},
			want:           false,
		},
		{
			name:           "v0.1.0 does not support --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.1.0"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsExcludeEnv(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsExcludeEnv result")
		})
	}
}

// TestAWFSupportsCliProxy tests the awfSupportsCliProxy version gate function.
func TestAWFSupportsCliProxy(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.17 supports CLI proxy flags (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.17"},
			want:           true,
		},
		{
			name:           "v0.26.0 supports CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.26.0"},
			want:           true,
		},
		{
			name:           "v0.27.0 supports CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.27.0"},
			want:           true,
		},
		{
			name:           "v0.25.16 does not support CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.25.16"},
			want:           false,
		},
		{
			name:           "v0.25.14 does not support CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.25.14"},
			want:           false,
		},
		{
			name:           "v0.1.0 does not support CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.1.0"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsCliProxy(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsCliProxy result")
		})
	}
}

// TestAWFSupportsAllowHostPorts tests the awfSupportsAllowHostPorts version gate function.
func TestAWFSupportsAllowHostPorts(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.24 supports --allow-host-ports (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.24"},
			want:           true,
		},
		{
			name:           "v0.26.0 supports --allow-host-ports",
			firewallConfig: &FirewallConfig{Version: "v0.26.0"},
			want:           true,
		},
		{
			name:           "v0.25.23 does not support --allow-host-ports",
			firewallConfig: &FirewallConfig{Version: "v0.25.23"},
			want:           false,
		},
		{
			name:           "v0.1.0 does not support --allow-host-ports",
			firewallConfig: &FirewallConfig{Version: "v0.1.0"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsAllowHostPorts(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsAllowHostPorts result")
		})
	}
}

// TestAWFSupportsDockerHostPathPrefix tests the awfSupportsDockerHostPathPrefix version gate.
func TestAWFSupportsDockerHostPathPrefix(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.43 supports --docker-host-path-prefix (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.43"},
			want:           true,
		},
		{
			name:           "v0.25.42 does not support --docker-host-path-prefix",
			firewallConfig: &FirewallConfig{Version: "v0.25.42"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsDockerHostPathPrefix(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsDockerHostPathPrefix result")
		})
	}
}

// TestAWFSupportsTokenSteering tests the awfSupportsTokenSteering version gate.
func TestAWFSupportsTokenSteering(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.44 supports token steering (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.44"},
			want:           true,
		},
		{
			name:           "v0.25.43 does not support token steering",
			firewallConfig: &FirewallConfig{Version: "v0.25.43"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsTokenSteering(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsTokenSteering result")
		})
	}
}

// TestAWFSupportsChrootConfig tests the awfSupportsChrootConfig version gate.
func TestAWFSupportsChrootConfig(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.27.1 supports chroot config (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.27.1"},
			want:           true,
		},
		{
			name:           "v0.27.0 does not support chroot config",
			firewallConfig: &FirewallConfig{Version: "v0.27.0"},
			want:           false,
		},
		{
			name:           "v0.25.44 (old) does not support chroot config",
			firewallConfig: &FirewallConfig{Version: "v0.25.44"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsChrootConfig(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsChrootConfig result")
		})
	}
}

// TestAWFSupportsAPIProxyProviders tests the awfSupportsAPIProxyProviders version gate.
func TestAWFSupportsAPIProxyProviders(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns false (uses default version)",
			firewallConfig: nil,
			want:           false,
		},
		{
			name:           "empty version returns false (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           false,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.27.43 supports apiProxy.providers (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.27.43"},
			want:           true,
		},
		{
			name:           "v0.27.42 does not support apiProxy.providers (schema not present)",
			firewallConfig: &FirewallConfig{Version: "v0.27.42"},
			want:           false,
		},
		{
			name:           "v0.27.41 does not support apiProxy.providers",
			firewallConfig: &FirewallConfig{Version: "v0.27.41"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsAPIProxyProviders(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsAPIProxyProviders result")
		})
	}
}
