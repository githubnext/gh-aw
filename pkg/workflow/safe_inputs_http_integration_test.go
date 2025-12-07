package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeInputsHTTPServer_Integration(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: copilot
safe-inputs:
  echo-tool:
    description: Echo a message
    inputs:
      message:
        type: string
        description: Message to echo
        required: true
    script: |
      return { content: [{ type: 'text', text: ` + "`Echo: ${message}`" + ` }] };
---

Test safe-inputs HTTP server
`

	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify that the HTTP server configuration steps are generated
	expectedSteps := []string{
		"Generate Safe Inputs MCP Server Config",
		"Create Docker Network for Safe Inputs",
		"Start Safe Inputs MCP HTTP Server",
		"Setup MCPs",
	}

	for _, stepName := range expectedSteps {
		if !strings.Contains(yamlStr, stepName) {
			t.Errorf("Expected step not found in workflow: %q", stepName)
		}
	}

	// Verify docker network creation
	dockerNetworkChecks := []string{
		"docker network create gh-aw-network",
		"Docker network created: gh-aw-network",
	}

	for _, check := range dockerNetworkChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected docker network creation content not found: %q", check)
		}
	}

	// Verify API key generation step uses github-script
	apiKeyGenChecks := []string{
		"uses: actions/github-script@",
		"generateSafeInputsConfig",
		"crypto.randomBytes",
	}

	for _, check := range apiKeyGenChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected API key generation content not found: %q", check)
		}
	}

	// Verify Docker container startup
	serverStartupChecks := []string{
		"docker run -d --rm --init",
		"--name safeinputs",
		"--network gh-aw-network",
		"-e GH_AW_SAFE_INPUTS_PORT=${{ steps.safe-inputs-config.outputs.safe_inputs_port }}",
		"-e GH_AW_SAFE_INPUTS_API_KEY=${{ steps.safe-inputs-config.outputs.safe_inputs_api_key }}",
		"-v /tmp/gh-aw/safe-inputs:/tmp/gh-aw/safe-inputs",
		"-w /tmp/gh-aw/safe-inputs",
		"node:24",
		"node mcp-server.cjs",
	}

	for _, check := range serverStartupChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected server startup content not found: %q", check)
		}
	}

	// Verify health check using docker run with curl on the network
	healthCheckItems := []string{
		"docker run --rm --network gh-aw-network curlimages/curl:latest",
		"curl -s -f http://safeinputs:${{ steps.safe-inputs-config.outputs.safe_inputs_port }}/health",
		"Safe Inputs MCP server is ready",
		"ERROR: Safe Inputs MCP server failed to start",
		"docker logs safeinputs",
	}

	for _, check := range healthCheckItems {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected health check content not found: %q", check)
		}
	}

	// Verify HTTP MCP configuration uses container name
	expectedMCPChecks := []string{
		`"safeinputs": {`,
		`"type": "http"`,
		`"url": "http://safeinputs:\${GH_AW_SAFE_INPUTS_PORT}"`,
		`"headers": {`,
		`"Authorization": "Bearer \${GH_AW_SAFE_INPUTS_API_KEY}"`,
		`"tools": ["*"]`,
		`"env": {`,
		`"GH_AW_SAFE_INPUTS_PORT": "\${GH_AW_SAFE_INPUTS_PORT}"`,
		`"GH_AW_SAFE_INPUTS_API_KEY": "\${GH_AW_SAFE_INPUTS_API_KEY}"`,
	}

	for _, expected := range expectedMCPChecks {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected MCP config content not found: %q", expected)
		}
	}

	// Verify env variables are set in Setup MCPs step
	setupMCPsEnvChecks := []string{
		"GH_AW_SAFE_INPUTS_PORT: ${{ steps.safe-inputs-start.outputs.port }}",
		"GH_AW_SAFE_INPUTS_API_KEY: ${{ steps.safe-inputs-start.outputs.api_key }}",
	}

	for _, check := range setupMCPsEnvChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected env var in Setup MCPs not found: %q", check)
		}
	}
}

func TestSafeInputsHTTPWithSecrets_Integration(t *testing.T) {
	// Create a temporary workflow file with safe-inputs that use secrets
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: copilot
safe-inputs:
  api-call:
    description: Make an API call
    inputs:
      url:
        type: string
        description: API URL
        required: true
    env:
      API_KEY: ${{ secrets.API_KEY }}
      API_SECRET: ${{ secrets.API_SECRET }}
    script: |
      return fetch(url);
---

Test safe-inputs with secrets
`

	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify that tool-specific env vars are passed to the Docker container
	serverEnvVarChecks := []string{
		`-e API_KEY="${API_KEY}"`,
		`-e API_SECRET="${API_SECRET}"`,
	}

	for _, check := range serverEnvVarChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected server env var not found: %q", check)
		}
	}

	// Verify that tool env vars are included in the MCP config env block
	expectedEnvVars := []string{
		`"API_KEY": "\${API_KEY}"`,
		`"API_SECRET": "\${API_SECRET}"`,
	}

	for _, expected := range expectedEnvVars {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected env var in MCP config not found: %q", expected)
		}
	}

	// Verify that secret expressions are set in Setup MCPs env block
	setupMCPsSecretChecks := []string{
		"API_KEY: ${{ secrets.API_KEY }}",
		"API_SECRET: ${{ secrets.API_SECRET }}",
	}

	for _, check := range setupMCPsSecretChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected secret expression in Setup MCPs not found: %q", check)
		}
	}
}

func TestSafeInputsHTTPEntryPointScript_Integration(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: copilot
safe-inputs:
  test-tool:
    description: Test tool
    script: return 'test';
---

Test entry point script
`

	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify that the entry point script uses HTTP server module
	entryPointChecks := []string{
		"safe_inputs_mcp_server_http.cjs",
		"startHttpServer",
		"SAFE_INPUTS_PORT",
		"SAFE_INPUTS_API_KEY",
	}

	for _, check := range entryPointChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected entry point script content not found: %q", check)
		}
	}

	// Verify that stdio server is NOT used
	stdiChecks := []string{
		"startSafeInputsServer",
	}

	for _, check := range stdiChecks {
		if strings.Contains(yamlStr, check) && !strings.Contains(yamlStr, "startHttpServer") {
			t.Errorf("Unexpected stdio server content found: %q", check)
		}
	}
}

func TestSafeInputsHTTPServerReadinessCheck_Integration(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: copilot
safe-inputs:
  test-tool:
    description: Test tool
    script: return 'test';
---

Test readiness check
`

	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify readiness check loop using docker with container name on network
	readinessChecks := []string{
		"for i in {1..10}; do",
		"docker run --rm --network gh-aw-network curlimages/curl:latest",
		"curl -s -f http://safeinputs:${{ steps.safe-inputs-config.outputs.safe_inputs_port }}/health",
		"Safe Inputs MCP server is ready",
		"break",
		"if [ $i -eq 10 ]; then",
		"ERROR: Safe Inputs MCP server failed to start after 10 seconds",
		"docker logs safeinputs",
		"exit 1",
		"sleep 1",
	}

	for _, check := range readinessChecks {
		if !strings.Contains(yamlStr, check) {
			t.Errorf("Expected readiness check content not found: %q", check)
		}
	}
}
