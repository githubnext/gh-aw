package cli

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectShell(t *testing.T) {
	// Save original environment
	originalShell := os.Getenv("SHELL")
	originalBashVersion := os.Getenv("BASH_VERSION")
	originalZshVersion := os.Getenv("ZSH_VERSION")
	originalFishVersion := os.Getenv("FISH_VERSION")

	// Restore environment after test
	defer func() {
		os.Setenv("SHELL", originalShell)
		os.Setenv("BASH_VERSION", originalBashVersion)
		os.Setenv("ZSH_VERSION", originalZshVersion)
		os.Setenv("FISH_VERSION", originalFishVersion)
	}()

	tests := []struct {
		name         string
		shellEnv     string
		bashVersion  string
		zshVersion   string
		fishVersion  string
		expectedType ShellType
	}{
		{
			name:         "detect bash from BASH_VERSION",
			shellEnv:     "/bin/bash",
			bashVersion:  "5.0.0",
			zshVersion:   "",
			fishVersion:  "",
			expectedType: ShellBash,
		},
		{
			name:         "detect zsh from ZSH_VERSION",
			shellEnv:     "/bin/zsh",
			bashVersion:  "",
			zshVersion:   "5.8",
			fishVersion:  "",
			expectedType: ShellZsh,
		},
		{
			name:         "detect fish from FISH_VERSION",
			shellEnv:     "/usr/bin/fish",
			bashVersion:  "",
			zshVersion:   "",
			fishVersion:  "3.1.2",
			expectedType: ShellFish,
		},
		{
			name:         "detect bash from SHELL path",
			shellEnv:     "/bin/bash",
			bashVersion:  "",
			zshVersion:   "",
			fishVersion:  "",
			expectedType: ShellBash,
		},
		{
			name:         "detect zsh from SHELL path",
			shellEnv:     "/usr/local/bin/zsh",
			bashVersion:  "",
			zshVersion:   "",
			fishVersion:  "",
			expectedType: ShellZsh,
		},
		{
			name:         "detect fish from SHELL path",
			shellEnv:     "/usr/bin/fish",
			bashVersion:  "",
			zshVersion:   "",
			fishVersion:  "",
			expectedType: ShellFish,
		},
		{
			name:         "detect powershell from SHELL path",
			shellEnv:     "/usr/bin/pwsh",
			bashVersion:  "",
			zshVersion:   "",
			fishVersion:  "",
			expectedType: ShellPowerShell,
		},
		{
			name:         "unknown shell",
			shellEnv:     "/bin/unknown",
			bashVersion:  "",
			zshVersion:   "",
			fishVersion:  "",
			expectedType: ShellUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment for this test
			os.Setenv("SHELL", tt.shellEnv)
			os.Setenv("BASH_VERSION", tt.bashVersion)
			os.Setenv("ZSH_VERSION", tt.zshVersion)
			os.Setenv("FISH_VERSION", tt.fishVersion)

			// Test detection
			detected := DetectShell()
			assert.Equal(t, tt.expectedType, detected)
		})
	}
}

func TestDetectShellNoShellEnv(t *testing.T) {
	// Save original environment
	originalShell := os.Getenv("SHELL")
	originalBashVersion := os.Getenv("BASH_VERSION")
	originalZshVersion := os.Getenv("ZSH_VERSION")
	originalFishVersion := os.Getenv("FISH_VERSION")

	// Restore environment after test
	defer func() {
		os.Setenv("SHELL", originalShell)
		os.Setenv("BASH_VERSION", originalBashVersion)
		os.Setenv("ZSH_VERSION", originalZshVersion)
		os.Setenv("FISH_VERSION", originalFishVersion)
	}()

	// Clear all shell environment variables
	os.Unsetenv("SHELL")
	os.Unsetenv("BASH_VERSION")
	os.Unsetenv("ZSH_VERSION")
	os.Unsetenv("FISH_VERSION")

	detected := DetectShell()

	// On Windows, should detect PowerShell
	if runtime.GOOS == "windows" {
		assert.Equal(t, ShellPowerShell, detected)
	} else {
		// On Unix-like systems, should be unknown
		assert.Equal(t, ShellUnknown, detected)
	}
}

func TestDetectShellPrioritizesVersionVariable(t *testing.T) {
	// Save original environment
	originalShell := os.Getenv("SHELL")
	originalBashVersion := os.Getenv("BASH_VERSION")
	originalZshVersion := os.Getenv("ZSH_VERSION")

	// Restore environment after test
	defer func() {
		os.Setenv("SHELL", originalShell)
		os.Setenv("BASH_VERSION", originalBashVersion)
		os.Setenv("ZSH_VERSION", originalZshVersion)
	}()

	// Set SHELL to bash but ZSH_VERSION is set (running zsh inside bash)
	os.Setenv("SHELL", "/bin/bash")
	os.Setenv("ZSH_VERSION", "5.8")
	os.Unsetenv("BASH_VERSION")

	detected := DetectShell()

	// Should prioritize ZSH_VERSION over SHELL
	assert.Equal(t, ShellZsh, detected)
}

func TestShellTypeString(t *testing.T) {
	tests := []struct {
		shellType ShellType
		expected  string
	}{
		{ShellBash, "bash"},
		{ShellZsh, "zsh"},
		{ShellFish, "fish"},
		{ShellPowerShell, "powershell"},
		{ShellUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.shellType))
		})
	}
}
