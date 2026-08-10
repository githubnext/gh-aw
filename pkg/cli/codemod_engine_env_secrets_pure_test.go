package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRemoveUnsafeEngineEnvKeys covers removeUnsafeEngineEnvKeys, a pure function that
// rewrites frontmatter YAML lines to drop unsafe engine.env: keys while leaving
// everything else (including unrelated top-level env: blocks) untouched.
func TestRemoveUnsafeEngineEnvKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		lines        []string
		unsafeKeys   map[string]struct{}
		wantModified bool
		wantLines    []string
	}{
		{
			name:         "no engine key present",
			lines:        []string{"on: push", "permissions: {}"},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: false,
			wantLines:    []string{"on: push", "permissions: {}"},
		},
		{
			name: "no env under engine",
			lines: []string{
				"engine:",
				"  id: copilot",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: false,
			wantLines: []string{
				"engine:",
				"  id: copilot",
			},
		},
		{
			name: "removes single unsafe simple key",
			lines: []string{
				"engine:",
				"  id: copilot",
				"  env:",
				"    FOO: bar",
				"    BAZ: qux",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: true,
			wantLines: []string{
				"engine:",
				"  id: copilot",
				"  env:",
				"    BAZ: qux",
			},
		},
		{
			name: "removes unsafe key with nested/multiline value",
			lines: []string{
				"engine:",
				"  env:",
				"    FOO: |",
				"      ${{ secrets.FOO }}",
				"      more",
				"    BAZ: qux",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: true,
			wantLines: []string{
				"engine:",
				"  env:",
				"    BAZ: qux",
			},
		},
		{
			name: "removes unsafe key followed by comment continuation",
			lines: []string{
				"engine:",
				"  env:",
				"    FOO: bar",
				"      # trailing comment nested under FOO",
				"    BAZ: qux",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: true,
			wantLines: []string{
				"engine:",
				"  env:",
				"    BAZ: qux",
			},
		},
		{
			name: "leaves blank lines inside env untouched when not removing",
			lines: []string{
				"engine:",
				"  env:",
				"    BAZ: qux",
				"",
				"    QUX: zap",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: false,
			wantLines: []string{
				"engine:",
				"  env:",
				"    BAZ: qux",
				"",
				"    QUX: zap",
			},
		},
		{
			name: "removes multiple unsafe keys",
			lines: []string{
				"engine:",
				"  env:",
				"    FOO: bar",
				"    BAZ: qux",
				"    QUX: zap",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}, "QUX": {}},
			wantModified: true,
			wantLines: []string{
				"engine:",
				"  env:",
				"    BAZ: qux",
			},
		},
		{
			name: "stops treating lines as engine.env after exiting engine block",
			lines: []string{
				"engine:",
				"  env:",
				"    FOO: bar",
				"on: push",
				"env:",
				"  FOO: unrelated-top-level",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: true,
			wantLines: []string{
				"engine:",
				"  env:",
				"on: push",
				"env:",
				"  FOO: unrelated-top-level",
			},
		},
		{
			name: "keeps keys not in unsafe set",
			lines: []string{
				"engine:",
				"  env:",
				"    SAFE: value",
			},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: false,
			wantLines: []string{
				"engine:",
				"  env:",
				"    SAFE: value",
			},
		},
		{
			name:         "empty input",
			lines:        []string{},
			unsafeKeys:   map[string]struct{}{"FOO": {}},
			wantModified: false,
			wantLines:    []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotLines, gotModified := removeUnsafeEngineEnvKeys(tt.lines, tt.unsafeKeys)
			assert.Equal(t, tt.wantModified, gotModified, "modified flag mismatch")
			assert.Equal(t, tt.wantLines, gotLines, "resulting lines mismatch")
		})
	}
}

// TestRemoveUnsafeEngineEnvKeysPurity ensures the function does not mutate its
// input slice or map arguments (a hallmark of purity), and is deterministic
// across repeated invocations with identical inputs.
func TestRemoveUnsafeEngineEnvKeysPurity(t *testing.T) {
	t.Parallel()

	original := []string{
		"engine:",
		"  env:",
		"    FOO: bar",
		"    BAZ: qux",
	}
	inputCopy := make([]string, len(original))
	copy(inputCopy, original)

	unsafeKeys := map[string]struct{}{"FOO": {}}

	result1, modified1 := removeUnsafeEngineEnvKeys(inputCopy, unsafeKeys)
	// Input slice must remain unchanged.
	assert.Equal(t, original, inputCopy, "input lines were mutated")

	result2, modified2 := removeUnsafeEngineEnvKeys(inputCopy, unsafeKeys)
	assert.Equal(t, result1, result2, "results differ across repeated calls with identical input")
	assert.Equal(t, modified1, modified2)
}
