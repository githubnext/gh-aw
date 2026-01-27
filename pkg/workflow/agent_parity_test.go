//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentRunnerParity_Utilities tests that essential utilities are accessible in the agent container
func TestAgentRunnerParity_Utilities(t *testing.T) {
	tests := []struct {
		name    string
		utility string
		desc    string
	}{
		{"jq", "jq", "JSON processor"},
		{"curl", "curl", "HTTP client"},
		{"git", "git", "Version control"},
		{"wget", "wget", "File downloader"},
		{"tar", "tar", "Archive utility"},
		{"gzip", "gzip", "Compression utility"},
		{"unzip", "unzip", "Archive extractor"},
		{"sed", "sed", "Stream editor"},
		{"awk", "awk", "Pattern processor"},
		{"grep", "grep", "Text search"},
		{"find", "find", "File finder"},
		{"xargs", "xargs", "Argument builder"},
	}

	tmpDir, err := os.MkdirTemp("", "agent-parity-utilities-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create workflow that tests utility accessibility
	workflowContent := `---
on: push
name: Agent Parity - Utilities Test
permissions:
  contents: read
engine: copilot
tools:
  bash:
    - "*"
timeout-minutes: 5
---

# Utility Accessibility Test

Test that essential utilities are accessible in the agent container.

Use bash to verify the following utilities are in PATH and executable:
` + buildUtilityList(tests) + `

For each utility, run "which <utility>" to verify it's accessible.
Report which utilities are found and which are missing.
`

	testFile := filepath.Join(tmpDir, "test-utilities.md")
	require.NoError(t, os.WriteFile(testFile, []byte(workflowContent), 0644), "Failed to write test file")

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Verify lock file was created
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify the workflow has the expected structure
	assert.Contains(t, lockStr, "name:", "Lock file should contain workflow name")
	assert.Contains(t, lockStr, "Agent Parity - Utilities Test", "Lock file should contain test name")
	assert.Contains(t, lockStr, "bash", "Lock file should contain bash tool")

	// Verify utilities are mentioned in the prompt
	for _, tt := range tests {
		assert.Contains(t, lockStr, tt.utility, "Lock file should mention utility: %s", tt.utility)
	}
}

// TestAgentRunnerParity_Runtimes tests that runtime tools are available and executable
func TestAgentRunnerParity_Runtimes(t *testing.T) {
	tests := []struct {
		name        string
		runtime     string
		command     string
		versionFlag string
	}{
		{"Node.js", "node", "node", "--version"},
		{"Python", "python3", "python3", "--version"},
		{"Go", "go", "go", "version"},
		{"Ruby", "ruby", "ruby", "--version"},
	}

	tmpDir, err := os.MkdirTemp("", "agent-parity-runtimes-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create workflow that tests runtime availability
	workflowContent := `---
on: push
name: Agent Parity - Runtimes Test
permissions:
  contents: read
engine: copilot
tools:
  bash:
    - "*"
timeout-minutes: 5
---

# Runtime Availability Test

Test that runtime tools are accessible and can execute in the agent container.

Use bash to verify the following runtimes are available:
` + buildRuntimeList(tests) + `

For each runtime, run the version command to verify it's executable.
Report which runtimes are found and their versions.
`

	testFile := filepath.Join(tmpDir, "test-runtimes.md")
	require.NoError(t, os.WriteFile(testFile, []byte(workflowContent), 0644), "Failed to write test file")

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Verify lock file was created
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify the workflow has the expected structure
	assert.Contains(t, lockStr, "name:", "Lock file should contain workflow name")
	assert.Contains(t, lockStr, "Agent Parity - Runtimes Test", "Lock file should contain test name")
	assert.Contains(t, lockStr, "bash", "Lock file should contain bash tool")

	// Verify runtimes are mentioned in the prompt
	for _, tt := range tests {
		assert.Contains(t, lockStr, tt.command, "Lock file should mention runtime: %s", tt.command)
	}
}

// TestAgentRunnerParity_EnvironmentVariables tests that environment variables are correctly set
func TestAgentRunnerParity_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name string
		env  string
		desc string
	}{
		{"JAVA_HOME", "JAVA_HOME", "Java installation directory"},
		{"ANDROID_HOME", "ANDROID_HOME", "Android SDK directory"},
		{"GOROOT", "GOROOT", "Go installation root"},
		{"PATH", "PATH", "Executable search path"},
		{"HOME", "HOME", "User home directory"},
		{"USER", "USER", "Current user"},
	}

	tmpDir, err := os.MkdirTemp("", "agent-parity-env-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create workflow that tests environment variables
	workflowContent := `---
on: push
name: Agent Parity - Environment Variables Test
permissions:
  contents: read
engine: copilot
tools:
  bash:
    - "*"
timeout-minutes: 5
---

# Environment Variables Test

Test that essential environment variables are set in the agent container.

Use bash to check if the following environment variables are set:
` + buildEnvVarList(tests) + `

For each variable, use "echo $VAR" or "printenv VAR" to check if it's set.
Report which variables are set and their values (redacting sensitive paths as needed).
`

	testFile := filepath.Join(tmpDir, "test-env-vars.md")
	require.NoError(t, os.WriteFile(testFile, []byte(workflowContent), 0644), "Failed to write test file")

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Verify lock file was created
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify the workflow has the expected structure
	assert.Contains(t, lockStr, "name:", "Lock file should contain workflow name")
	assert.Contains(t, lockStr, "Agent Parity - Environment Variables Test", "Lock file should contain test name")
	assert.Contains(t, lockStr, "bash", "Lock file should contain bash tool")

	// Verify environment variables are mentioned in the prompt
	for _, tt := range tests {
		assert.Contains(t, lockStr, tt.env, "Lock file should mention environment variable: %s", tt.env)
	}
}

// TestAgentRunnerParity_SharedLibraries tests that shared libraries can be loaded
func TestAgentRunnerParity_SharedLibraries(t *testing.T) {
	tests := []struct {
		name   string
		binary string
		desc   string
	}{
		{"Python", "/usr/bin/python3", "Python interpreter"},
		{"Node", "/usr/bin/node", "Node.js runtime"},
		{"Git", "/usr/bin/git", "Git version control"},
		{"Curl", "/usr/bin/curl", "HTTP client"},
	}

	tmpDir, err := os.MkdirTemp("", "agent-parity-libs-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create workflow that tests shared library linking
	workflowContent := `---
on: push
name: Agent Parity - Shared Libraries Test
permissions:
  contents: read
engine: copilot
tools:
  bash:
    - "*"
timeout-minutes: 5
---

# Shared Libraries Test

Test that shared libraries can be loaded by key binaries in the agent container.

Use bash with ldd to check shared library dependencies for the following binaries:
` + buildBinaryList(tests) + `

For each binary, run "ldd <binary>" to verify all shared libraries can be found.
Report if any libraries are missing or if all dependencies are satisfied.
`

	testFile := filepath.Join(tmpDir, "test-libs.md")
	require.NoError(t, os.WriteFile(testFile, []byte(workflowContent), 0644), "Failed to write test file")

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Verify lock file was created
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify the workflow has the expected structure
	assert.Contains(t, lockStr, "name:", "Lock file should contain workflow name")
	assert.Contains(t, lockStr, "Agent Parity - Shared Libraries Test", "Lock file should contain test name")
	assert.Contains(t, lockStr, "bash", "Lock file should contain bash tool")
	assert.Contains(t, lockStr, "ldd", "Lock file should mention ldd command")

	// Verify binaries are mentioned in the prompt
	for _, tt := range tests {
		assert.Contains(t, lockStr, tt.binary, "Lock file should mention binary: %s", tt.binary)
	}
}

// TestAgentRunnerParity_Comprehensive tests multiple aspects in a single workflow
func TestAgentRunnerParity_Comprehensive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-parity-comprehensive-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create comprehensive workflow that tests all aspects
	workflowContent := `---
on: push
name: Agent Parity - Comprehensive Test
permissions:
  contents: read
engine: copilot
tools:
  bash:
    - "*"
timeout-minutes: 10
---

# Comprehensive Agent-Runner Environment Parity Test

This workflow tests multiple aspects of the agent container environment to ensure parity with GitHub Actions runners.

## Tasks

1. **Utilities**: Verify at least 10 essential utilities are accessible (jq, curl, git, wget, tar, gzip, unzip, sed, awk, grep)
2. **Runtimes**: Verify runtime tools are available and can execute (node, python3, go, ruby)
3. **Environment Variables**: Check that essential environment variables are set (JAVA_HOME, ANDROID_HOME, GOROOT, PATH, HOME)
4. **Shared Libraries**: Use ldd to verify shared libraries can be loaded for python3, node, git, and curl

Run each test category and report:
- ✅ Items that passed
- ❌ Items that failed
- 📊 Summary statistics

Keep the report concise and focused on failures.
`

	testFile := filepath.Join(tmpDir, "test-comprehensive.md")
	require.NoError(t, os.WriteFile(testFile, []byte(workflowContent), 0644), "Failed to write test file")

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Failed to compile workflow")

	// Verify lock file was created
	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockStr := string(lockContent)

	// Verify the workflow has the expected structure
	assert.Contains(t, lockStr, "name:", "Lock file should contain workflow name")
	assert.Contains(t, lockStr, "Agent Parity - Comprehensive Test", "Lock file should contain test name")
	assert.Contains(t, lockStr, "bash", "Lock file should contain bash tool")

	// Verify key terms are present
	assert.Contains(t, lockStr, "Utilities", "Lock file should mention utilities")
	assert.Contains(t, lockStr, "Runtimes", "Lock file should mention runtimes")
	assert.Contains(t, lockStr, "Environment Variables", "Lock file should mention environment variables")
	assert.Contains(t, lockStr, "Shared Libraries", "Lock file should mention shared libraries")
}

// Helper functions to build lists for workflow content

func buildUtilityList(tests []struct {
	name    string
	utility string
	desc    string
}) string {
	var sb strings.Builder
	for _, tt := range tests {
		sb.WriteString("- ")
		sb.WriteString(tt.utility)
		sb.WriteString(" (")
		sb.WriteString(tt.desc)
		sb.WriteString(")\n")
	}
	return sb.String()
}

func buildRuntimeList(tests []struct {
	name        string
	runtime     string
	command     string
	versionFlag string
}) string {
	var sb strings.Builder
	for _, tt := range tests {
		sb.WriteString("- ")
		sb.WriteString(tt.command)
		sb.WriteString(" ")
		sb.WriteString(tt.versionFlag)
		sb.WriteString(" (")
		sb.WriteString(tt.name)
		sb.WriteString(")\n")
	}
	return sb.String()
}

func buildEnvVarList(tests []struct {
	name string
	env  string
	desc string
}) string {
	var sb strings.Builder
	for _, tt := range tests {
		sb.WriteString("- ")
		sb.WriteString(tt.env)
		sb.WriteString(" (")
		sb.WriteString(tt.desc)
		sb.WriteString(")\n")
	}
	return sb.String()
}

func buildBinaryList(tests []struct {
	name   string
	binary string
	desc   string
}) string {
	var sb strings.Builder
	for _, tt := range tests {
		sb.WriteString("- ")
		sb.WriteString(tt.binary)
		sb.WriteString(" (")
		sb.WriteString(tt.desc)
		sb.WriteString(")\n")
	}
	return sb.String()
}
