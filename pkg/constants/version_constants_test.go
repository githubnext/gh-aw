//go:build !integration

package constants

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDefaultPlaywrightCLIVersionOutsideCooldownWindow(t *testing.T) {
	const (
		expectedVersion    Version = "0.1.17"
		publishedAtRFC3339         = "2026-07-09T19:11:00Z"
		minReleaseAge              = 72 * time.Hour
	)

	if DefaultPlaywrightCLIVersion != expectedVersion {
		t.Fatalf("DefaultPlaywrightCLIVersion = %q, want %q; update this test metadata when changing the pinned default", DefaultPlaywrightCLIVersion, expectedVersion)
	}

	publishedAt, err := time.Parse(time.RFC3339Nano, publishedAtRFC3339)
	if err != nil {
		t.Fatalf("parse publishedAtRFC3339: %v", err)
	}

	age := time.Since(publishedAt)
	if age < minReleaseAge {
		t.Fatalf("@playwright/cli@%s is only %s old, but Playwright CLI installs enforce a %s npm release-age cooldown", DefaultPlaywrightCLIVersion, age.Round(time.Second), minReleaseAge)
	}
}

// TestDefaultCopilotVersionWithinCompatWindow asserts that DefaultCopilotVersion falls
// within the declared compat.json window so the runner toolcache can satisfy installs
// without a network download.  Failures here indicate that either DefaultCopilotVersion
// or the compat.json max-agent needs to be updated.
func TestDefaultCopilotVersionWithinCompatWindow(t *testing.T) {
	// Locate compat.json relative to this test file (three directories up from
	// pkg/constants/ → repo root → .github/aw/compat.json).
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	compatPath := filepath.Join(filepath.Dir(testFile), "..", "..", ".github", "aw", "compat.json")
	compatPath = filepath.Clean(compatPath)

	data, err := os.ReadFile(compatPath)
	if err != nil {
		t.Fatalf("read %s: %v", compatPath, err)
	}

	var compat struct {
		AgentCompatV1 struct {
			Copilot []struct {
				MinGhAw  string `json:"min-gh-aw"`
				MaxGhAw  string `json:"max-gh-aw"`
				MinAgent string `json:"min-agent"`
				MaxAgent string `json:"max-agent"`
				Open     bool   `json:"open"`
			} `json:"copilot"`
		} `json:"agent-compat-v1"`
	}
	if err := json.Unmarshal(data, &compat); err != nil {
		t.Fatalf("parse %s: %v", compatPath, err)
	}

	version := string(DefaultCopilotVersion)

	// Find the first open row (open: true means it covers the current gh-aw release).
	for _, row := range compat.AgentCompatV1.Copilot {
		if !row.Open {
			continue
		}
		if row.MinAgent == "" || row.MaxAgent == "" {
			t.Fatalf("compat row missing min-agent or max-agent: %+v", row)
		}
		if cmp, err := semverCmp(version, row.MinAgent); err != nil {
			t.Fatalf("semverCmp(%q, %q): %v", version, row.MinAgent, err)
		} else if cmp < 0 {
			t.Fatalf("DefaultCopilotVersion %q is below compat min-agent %q; bump min-agent or lower DefaultCopilotVersion", version, row.MinAgent)
		}
		if cmp, err := semverCmp(version, row.MaxAgent); err != nil {
			t.Fatalf("semverCmp(%q, %q): %v", version, row.MaxAgent, err)
		} else if cmp > 0 {
			t.Fatalf("DefaultCopilotVersion %q exceeds compat max-agent %q; update .github/aw/compat.json max-agent or lower DefaultCopilotVersion to prevent toolcache bypass", version, row.MaxAgent)
		}
		return // found and validated
	}
	t.Fatalf("no open compat row found in %s; add an open row for the current gh-aw release", compatPath)
}

// semverCmp compares two semver strings (without leading "v") and returns -1, 0, or 1.
func semverCmp(a, b string) (int, error) {
	pa, err := parseSemver(a)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", a, err)
	}
	pb, err := parseSemver(b)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", b, err)
	}
	for i := range pa {
		if pa[i] < pb[i] {
			return -1, nil
		}
		if pa[i] > pb[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseSemver(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("expected MAJOR.MINOR.PATCH, got %q", v)
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, fmt.Errorf("part %d of %q is not a number: %w", i, v, err)
		}
		out[i] = n
	}
	return out, nil
}
