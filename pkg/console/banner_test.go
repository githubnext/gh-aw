//go:build !integration

package console

import (
	"strings"
	"testing"
)

func TestFormatBanner(t *testing.T) {
	banner := FormatBanner()

	// Check that the banner contains the expected ASCII art patterns
	// The logo spells out "Agentic" and "Workflows" in ASCII art
	if !strings.Contains(banner, "___") {
		t.Errorf("FormatBanner() should contain ASCII art '___', got: %s", banner)
	}

	if !strings.Contains(banner, "/ _ \\") {
		t.Errorf("FormatBanner() should contain ASCII art '/ _ \\', got: %s", banner)
	}

	// Check that banner is multi-line
	lines := strings.Split(banner, "\n")
	if len(lines) < 10 {
		t.Errorf("FormatBanner() should have at least 10 lines, got %d lines", len(lines))
	}
}

func TestBannerLogoEmbedded(t *testing.T) {
	// Check that the embedded logo is not empty
	if bannerLogo == "" {
		t.Error("bannerLogo should not be empty")
	}

	// Check that it contains the expected ASCII art patterns
	if !strings.Contains(bannerLogo, "___") {
		t.Error("bannerLogo should contain ASCII art pattern '___'")
	}

	if !strings.Contains(bannerLogo, "|") {
		t.Error("bannerLogo should contain ASCII art pipe characters")
	}
}

func TestBannerStyleInitialized(t *testing.T) {
	// Ensure FormatBanner always returns visible banner content.
	banner := FormatBanner()
	if banner == "" {
		t.Error("FormatBanner() should produce non-empty output")
	}

	// In non-TTY mode, FormatBanner should return plain logo text.
	if !strings.Contains(banner, "\x1b[") {
		expected := strings.TrimRight(bannerLogo, "\n")
		if banner != expected {
			t.Errorf("FormatBanner() should return plain logo text in non-TTY mode")
		}
	}
}
