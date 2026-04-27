//go:build !integration

package workflow

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineRegistry(t *testing.T) {
	t.Run("built-in engines are registered", func(t *testing.T) {
		registry := NewEngineRegistry()
		supportedEngines := registry.GetSupportedEngines()

		expectedEngineIDs := []string{"claude", "codex", "copilot", "gemini", "opencode", "crush"}
		for _, engineID := range expectedEngineIDs {
			assert.True(t, slices.Contains(supportedEngines, engineID), "expected engine %q to be registered", engineID)
		}
	})

	t.Run("GetEngine returns engine by ID", func(t *testing.T) {
		tests := []struct {
			engineID string
		}{
			{engineID: "claude"},
			{engineID: "codex"},
			{engineID: "copilot"},
			{engineID: "gemini"},
			{engineID: "opencode"},
			{engineID: "crush"},
		}

		for _, tt := range tests {
			t.Run(tt.engineID, func(t *testing.T) {
				registry := NewEngineRegistry()
				engine, err := registry.GetEngine(tt.engineID)
				require.NoError(t, err, "GetEngine(%q) should not return an error", tt.engineID)
				assert.Equal(t, tt.engineID, engine.GetID(), "engine ID should match requested ID")
			})
		}
	})

	t.Run("GetEngine returns error for unknown engine", func(t *testing.T) {
		registry := NewEngineRegistry()
		_, err := registry.GetEngine("nonexistent")
		assert.Error(t, err, "GetEngine should return an error for unknown engine ID")
	})

	t.Run("IsValidEngine", func(t *testing.T) {
		registry := NewEngineRegistry()

		validEngines := []string{"claude", "codex", "copilot", "gemini", "opencode", "crush"}
		for _, id := range validEngines {
			assert.True(t, registry.IsValidEngine(id), "IsValidEngine(%q) should return true", id)
		}

		assert.False(t, registry.IsValidEngine("nonexistent"), "IsValidEngine should return false for unknown engine ID")
	})

	t.Run("GetDefaultEngine returns copilot", func(t *testing.T) {
		registry := NewEngineRegistry()
		defaultEngine := registry.GetDefaultEngine()
		require.NotNil(t, defaultEngine, "default engine should not be nil")
		assert.Equal(t, "copilot", defaultEngine.GetID(), "default engine should be copilot")
	})

	t.Run("GetEngineByPrefix matches engine", func(t *testing.T) {
		registry := NewEngineRegistry()
		engine, err := registry.GetEngineByPrefix("codex-experimental")
		require.NoError(t, err, "GetEngineByPrefix should not return an error for a valid prefix")
		assert.Equal(t, "codex", engine.GetID(), "engine matched by prefix 'codex-experimental' should be codex")
	})

	t.Run("GetEngineByPrefix returns error for non-matching prefix", func(t *testing.T) {
		registry := NewEngineRegistry()
		_, err := registry.GetEngineByPrefix("nonexistent-prefix")
		assert.Error(t, err, "GetEngineByPrefix should return an error for a non-matching prefix")
	})
}

func TestEngineRegistry_Register(t *testing.T) {
	t.Run("custom engine can be registered and retrieved", func(t *testing.T) {
		// Use direct struct initialization to start with an empty registry so
		// Register is the sole mechanism populating it in this test.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		customEngine := NewCopilotEngine()

		registry.Register(customEngine)

		engine, err := registry.GetEngine("copilot")
		require.NoError(t, err, "registered custom engine should be retrievable")
		assert.Equal(t, "copilot", engine.GetID(), "retrieved engine ID should match registered engine")
	})

	t.Run("registering an engine makes IsValidEngine return true", func(t *testing.T) {
		// Use direct struct initialization to start with an empty registry so
		// IsValidEngine behaviour before and after Register is clearly observable.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		assert.False(t, registry.IsValidEngine("claude"), "engine should not be valid before registration")

		registry.Register(NewClaudeEngine())
		assert.True(t, registry.IsValidEngine("claude"), "engine should be valid after registration")
	})
}

func TestGetGlobalEngineRegistry(t *testing.T) {
	t.Run("returns non-nil registry", func(t *testing.T) {
		registry := GetGlobalEngineRegistry()
		require.NotNil(t, registry, "global engine registry should not be nil")
	})

	t.Run("returns same singleton on repeated calls", func(t *testing.T) {
		registry1 := GetGlobalEngineRegistry()
		registry2 := GetGlobalEngineRegistry()
		assert.Same(t, registry1, registry2, "GetGlobalEngineRegistry should return the same singleton instance")
	})

	t.Run("singleton contains expected built-in engines", func(t *testing.T) {
		registry := GetGlobalEngineRegistry()
		expectedEngineIDs := []string{"claude", "codex", "copilot", "gemini", "opencode", "crush"}
		supportedEngines := registry.GetSupportedEngines()
		for _, engineID := range expectedEngineIDs {
			assert.True(t, slices.Contains(supportedEngines, engineID), "global registry should contain built-in engine %q", engineID)
		}
	})
}

func TestEngineRegistry_GetAllAgentManifestFolders(t *testing.T) {
	t.Run("always includes .agents platform directory", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		assert.Contains(t, folders, ".agents", "manifest folders should always include the .agents platform directory")
	})

	t.Run("result is sorted", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		for i := 1; i < len(folders); i++ {
			assert.LessOrEqual(t, folders[i-1], folders[i], "manifest folders should be sorted alphabetically")
		}
	})

	t.Run("includes engine-specific config directories", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		// Claude and Copilot engines provide known config directory prefixes
		expectedFolders := []string{".agents", ".claude", ".gemini", ".github"}
		for _, folder := range expectedFolders {
			assert.Contains(t, folders, folder, "manifest folders should include engine config directory %q", folder)
		}
	})

	t.Run("no duplicates in result", func(t *testing.T) {
		registry := NewEngineRegistry()
		folders := registry.GetAllAgentManifestFolders()
		seen := make(map[string]bool)
		for _, folder := range folders {
			assert.False(t, seen[folder], "manifest folders should not contain duplicates, found %q twice", folder)
			seen[folder] = true
		}
	})

	t.Run("empty registry still includes .agents", func(t *testing.T) {
		// Use direct struct initialization so there are no engines; this verifies
		// that .agents is always appended regardless of registered engines.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		folders := registry.GetAllAgentManifestFolders()
		assert.Equal(t, []string{".agents"}, folders, "empty registry should still return .agents")
	})
}

func TestEngineRegistry_GetAllAgentManifestFiles(t *testing.T) {
	t.Run("result is sorted", func(t *testing.T) {
		registry := NewEngineRegistry()
		files := registry.GetAllAgentManifestFiles()
		for i := 1; i < len(files); i++ {
			assert.LessOrEqual(t, files[i-1], files[i], "manifest files should be sorted alphabetically")
		}
	})

	t.Run("includes engine-specific instruction files", func(t *testing.T) {
		registry := NewEngineRegistry()
		files := registry.GetAllAgentManifestFiles()
		// Known instruction files contributed by built-in engines
		expectedFiles := []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}
		for _, file := range expectedFiles {
			assert.Contains(t, files, file, "manifest files should include instruction file %q", file)
		}
	})

	t.Run("no duplicates in result", func(t *testing.T) {
		registry := NewEngineRegistry()
		files := registry.GetAllAgentManifestFiles()
		seen := make(map[string]bool)
		for _, file := range files {
			assert.False(t, seen[file], "manifest files should not contain duplicates, found %q twice", file)
			seen[file] = true
		}
	})

	t.Run("empty registry returns empty slice", func(t *testing.T) {
		// Use direct struct initialization so there are no engines; this verifies
		// the empty-input case without interference from built-in engine files.
		registry := &EngineRegistry{engines: make(map[string]CodingAgentEngine)}
		files := registry.GetAllAgentManifestFiles()
		assert.Empty(t, files, "empty registry should return no manifest files")
	})
}
