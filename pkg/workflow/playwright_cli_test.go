//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePlaywrightCLIInstallSteps_DefaultVersionUsesCooldown(t *testing.T) {
	steps := generatePlaywrightCLIInstallSteps(&WorkflowData{
		Tools: map[string]any{
			"playwright": map[string]any{
				"mode": "cli",
			},
		},
	})

	require.Len(t, steps, 5, "expected npm, three browser, and skills install steps")

	installStep := strings.Join(steps[0], "\n")
	assert.Contains(t, installStep, "npm install -g @playwright/cli@"+string(constants.DefaultPlaywrightCLIVersion))
	assert.Contains(t, installStep, "NPM_CONFIG_MIN_RELEASE_AGE: '3'")
	assert.Contains(t, installStep, "timeout-minutes: 10")

	for i, browser := range []string{"chromium", "firefox", "webkit"} {
		browserStep := strings.Join(steps[i+1], "\n")
		assert.Contains(t, browserStep, "playwright-cli install-browser "+browser)
		assert.Contains(t, browserStep, "timeout-minutes: 10")
	}

	skillsStep := strings.Join(steps[4], "\n")
	assert.Contains(t, skillsStep, "playwright-cli install --skills")
	assert.Contains(t, skillsStep, "PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: '1'")
}

func TestGeneratePlaywrightCLIInstallSteps_ModeOmitted(t *testing.T) {
	steps := generatePlaywrightCLIInstallSteps(&WorkflowData{
		Tools: map[string]any{"playwright": nil},
	})

	require.Len(t, steps, 5)
	assert.Contains(t, strings.Join(steps[0], "\n"), "@playwright/cli@"+string(constants.DefaultPlaywrightCLIVersion))
}
