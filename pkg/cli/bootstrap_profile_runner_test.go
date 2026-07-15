//go:build !integration

package cli

import "testing"

func TestLoadBootstrapGitHubAppOverrides(t *testing.T) {
	t.Setenv(bootstrapGitHubAppModeEnv, "create")
	t.Setenv(bootstrapGitHubAppOwnerEnv, "octo-platform")
	t.Setenv(bootstrapGitHubAppNameEnv, "octo-control-plane")
	t.Setenv(bootstrapGitHubAppURLEnv, "https://github.com/octo/platform-ops")
	t.Setenv(bootstrapGitHubAppDescriptionEnv, "Bootstrap app")
	t.Setenv(bootstrapNoOpenBrowserEnv, "true")

	overrides, err := loadBootstrapGitHubAppOverrides()
	if err != nil {
		t.Fatalf("loadBootstrapGitHubAppOverrides returned error: %v", err)
	}
	if overrides.Mode != "create" {
		t.Fatalf("expected create mode, got %q", overrides.Mode)
	}
	if overrides.Owner != "octo-platform" {
		t.Fatalf("expected owner override, got %q", overrides.Owner)
	}
	if overrides.Name != "octo-control-plane" {
		t.Fatalf("expected name override, got %q", overrides.Name)
	}
	if overrides.HomepageURL != "https://github.com/octo/platform-ops" {
		t.Fatalf("expected homepage override, got %q", overrides.HomepageURL)
	}
	if overrides.Description != "Bootstrap app" {
		t.Fatalf("expected description override, got %q", overrides.Description)
	}
	if overrides.OpenBrowser {
		t.Fatal("expected browser opening to be disabled")
	}
}

func TestLoadBootstrapGitHubAppOverrides_RejectsInvalidMode(t *testing.T) {
	t.Setenv(bootstrapGitHubAppModeEnv, "later")

	_, err := loadBootstrapGitHubAppOverrides()
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestParseBootstrapBool(t *testing.T) {
	t.Run("truthy", func(t *testing.T) {
		truthy := []string{"1", "true", "yes", "on"}
		for _, raw := range truthy {
			got, err := parseBootstrapBool(raw)
			if err != nil {
				t.Fatalf("parseBootstrapBool(%q) returned error: %v", raw, err)
			}
			if !got {
				t.Fatalf("expected %q to parse as true", raw)
			}
		}
	})

	t.Run("falsy", func(t *testing.T) {
		falsy := []string{"0", "false", "no", "off"}
		for _, raw := range falsy {
			got, err := parseBootstrapBool(raw)
			if err != nil {
				t.Fatalf("parseBootstrapBool(%q) returned error: %v", raw, err)
			}
			if got {
				t.Fatalf("expected %q to parse as false", raw)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := parseBootstrapBool("maybe"); err == nil {
			t.Fatal("expected invalid boolean error")
		}
	})
}
