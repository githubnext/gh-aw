package workflow

import (
	"strings"
	"testing"
)

// TestGenerateSquidConfig tests that the squid configuration is properly loaded from embedded resource
func TestGenerateSquidConfig(t *testing.T) {
	config := generateSquidConfig()

	// Test that config is not empty
	if config == "" {
		t.Error("Expected squid config to be non-empty")
	}

	// Test that it contains expected directives
	expectedDirectives := []string{
		"# Squid configuration for egress traffic control",
		"http_port 3128",
		"acl allowed_domains dstdomain",
		"access_log /var/log/squid/access.log",
		"cache deny all",
		"dns_nameservers 8.8.8.8 8.8.4.4",
	}

	for _, directive := range expectedDirectives {
		if !strings.Contains(config, directive) {
			t.Errorf("Expected squid config to contain directive: %s", directive)
		}
	}

	// Test that it doesn't contain template literals (indicating the embedded file is used)
	if strings.Contains(config, "`") {
		t.Error("Squid config should not contain template literals, indicating embedded file should be used")
	}
}

// TestSquidConfigContent tests specific content matches the expected configuration
func TestSquidConfigContent(t *testing.T) {
	config := generateSquidConfig()

	// Check that critical security settings are present
	expectedSecurity := []string{
		"http_access deny !allowed_domains",
		"http_access deny !Safe_ports",
		"http_access deny CONNECT !SSL_ports",
		"http_access allow localnet",
		"http_access deny all",
	}

	for _, setting := range expectedSecurity {
		if !strings.Contains(config, setting) {
			t.Errorf("Expected security setting not found in squid config: %s", setting)
		}
	}
}
