package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// generateInitialAndCheckoutSteps emits the OTLP mask step, pre-steps, all checkout steps
// (default workspace checkout, dev-mode CLI build, additional checkouts), repository import
// checkouts, legacy agent import checkout, and the merge-.github-folder step.
// It returns the CheckoutManager (needed later for token invalidation and dev-mode restore)
// and a flag indicating whether the default workspace checkout was emitted.
func (c *Compiler) generateInitialAndCheckoutSteps(yaml *strings.Builder, data *WorkflowData) (*CheckoutManager, bool, error) {
	writeEarlyMaskSteps(yaml, data)
	c.generatePreSteps(yaml, data)

	needsCheckout := c.shouldAddCheckoutStep(data)
	compilerYamlLog.Printf("Checkout step needed: %t", needsCheckout)
	checkoutMgr := c.prepareCheckoutManager(yaml, data)

	if needsCheckout {
		c.writeDefaultCheckoutAndDevSteps(yaml, data, checkoutMgr)
	}

	writeCheckoutManagerSteps(yaml, checkoutMgr, c.getActionPin)
	c.writeImportCheckoutSteps(yaml, data)

	if err := writeMergeRemoteGithubFolderStep(yaml, data); err != nil {
		return nil, false, err
	}

	return checkoutMgr, needsCheckout, nil
}

func writeEarlyMaskSteps(yaml *strings.Builder, data *WorkflowData) {
	if isOTLPHeadersPresent(data) {
		yaml.WriteString(generateOTLPHeadersMaskStep())
	}
	if isOTLPAttributesPresent(data) {
		yaml.WriteString(generateOTLPAttributesMaskStep())
	}
}

func (c *Compiler) prepareCheckoutManager(yaml *strings.Builder, data *WorkflowData) *CheckoutManager {
	checkoutMgr := NewCheckoutManager(data.CheckoutConfigs)
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		checkoutMgr.SetCrossRepoTargetRepo("${{ needs.activation.outputs.target_repo }}")
	}
	if checkoutMgr.HasAppAuth() {
		compilerYamlLog.Print("Generating checkout app token minting steps in agent job")
		for _, step := range checkoutMgr.GenerateCheckoutAppTokenSteps(c, resolveCheckoutPermissions(data)) {
			yaml.WriteString(step)
		}
	}
	return checkoutMgr
}

func (c *Compiler) writeDefaultCheckoutAndDevSteps(yaml *strings.Builder, data *WorkflowData, checkoutMgr *CheckoutManager) {
	for _, line := range checkoutMgr.GenerateDefaultCheckoutStep(c.trialMode, c.trialLogicalRepoSlug, c.getActionPin) {
		yaml.WriteString(line)
	}
	if !c.actionMode.IsDev() {
		return
	}
	if _, hasAgenticWorkflows := data.Tools["agentic-workflows"]; hasAgenticWorkflows {
		compilerYamlLog.Printf("Generating CLI build steps for dev mode (agentic-workflows tool enabled)")
		c.generateDevModeCLIBuildSteps(yaml)
	} else {
		compilerYamlLog.Printf("Skipping CLI build steps in dev mode (agentic-workflows tool not enabled)")
	}
}

func writeCheckoutManagerSteps(yaml *strings.Builder, checkoutMgr *CheckoutManager, getActionPin func(string) string) {
	for _, line := range checkoutMgr.GenerateAdditionalCheckoutSteps(getActionPin) {
		yaml.WriteString(line)
	}
	for _, line := range checkoutMgr.GenerateCheckoutManifestStep(getActionPin) {
		yaml.WriteString(line)
	}
}

func (c *Compiler) writeImportCheckoutSteps(yaml *strings.Builder, data *WorkflowData) {
	if len(data.RepositoryImports) > 0 {
		compilerYamlLog.Printf("Adding checkout steps for %d repository imports", len(data.RepositoryImports))
		c.generateRepositoryImportCheckouts(yaml, data.RepositoryImports)
	}
	if data.AgentFile != "" && data.AgentImportSpec != "" {
		compilerYamlLog.Printf("Adding checkout step for legacy agent import: %s", data.AgentImportSpec)
		c.generateLegacyAgentImportCheckout(yaml, data.AgentImportSpec)
	}
}

func writeMergeRemoteGithubFolderStep(yaml *strings.Builder, data *WorkflowData) error {
	if len(data.RepositoryImports) == 0 && (data.AgentFile == "" || data.AgentImportSpec == "") {
		return nil
	}
	compilerYamlLog.Printf("Adding merge remote .github folder step")
	yaml.WriteString("      - name: Merge remote .github folder\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        env:\n")
	if len(data.RepositoryImports) > 0 {
		repoImportsJSON, err := json.Marshal(data.RepositoryImports)
		if err != nil {
			return fmt.Errorf("failed to marshal repository imports for merge step: %w", err)
		}
		writeYAMLEnv(yaml, "          ", "GH_AW_REPOSITORY_IMPORTS", string(repoImportsJSON))
	}
	if data.AgentFile != "" && data.AgentImportSpec != "" {
		writeYAMLEnv(yaml, "          ", "GH_AW_AGENT_FILE", data.AgentFile)
		writeYAMLEnv(yaml, "          ", "GH_AW_AGENT_IMPORT_SPEC", data.AgentImportSpec)
	}
	writeMergeRemoteGithubFolderScript(yaml)
	return nil
}

func writeMergeRemoteGithubFolderScript(yaml *strings.Builder) {
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/merge_remote_agent_github_folder.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateRepositoryImportCheckouts generates checkout steps for repository imports
// Each repository is checked out into a temporary folder at .github/aw/imports/<owner>-<repo>-<sanitized-ref>
// relative to GITHUB_WORKSPACE. This allows the merge script to copy files from pre-checked-out folders instead of doing git operations
func (c *Compiler) generateRepositoryImportCheckouts(yaml *strings.Builder, repositoryImports []string) {
	for _, repoImport := range repositoryImports {
		compilerYamlLog.Printf("Generating checkout step for repository import: %s", repoImport)

		// Parse the import spec to extract owner, repo, and ref
		// Format: owner/repo@ref or owner/repo
		owner, repo, ref := parseRepositoryImportSpec(repoImport)
		if owner == "" || repo == "" {
			compilerYamlLog.Printf("Warning: failed to parse repository import: %s", repoImport)
			continue
		}

		// Generate a sanitized directory name for the checkout
		// Use a consistent format: owner-repo-ref
		// NOTE: Path must be relative to GITHUB_WORKSPACE for actions/checkout@v6
		sanitizedRef := sanitizeRefForPath(ref)
		checkoutPath := fmt.Sprintf(".github/aw/imports/%s-%s-%s", owner, repo, sanitizedRef)

		// Generate the checkout step
		fmt.Fprintf(yaml, "      - name: Checkout repository import %s/%s@%s\n", owner, repo, ref)
		fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("actions/checkout"))
		yaml.WriteString("        with:\n")
		fmt.Fprintf(yaml, "          repository: %s/%s\n", owner, repo)
		fmt.Fprintf(yaml, "          ref: %s\n", ref)
		fmt.Fprintf(yaml, "          path: %s\n", checkoutPath)
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            .github/\n")
		yaml.WriteString("          persist-credentials: false\n")

		compilerYamlLog.Printf("Added checkout step: %s/%s@%s -> %s", owner, repo, ref, checkoutPath)
	}
}

// parseRepositoryImportSpec parses a repository import specification
// Format: owner/repo@ref or owner/repo (defaults to "main" if no ref)
// Returns: owner, repo, ref
func parseRepositoryImportSpec(importSpec string) (owner, repo, ref string) {
	// Remove section reference if present (file.md#Section)
	cleanSpec := importSpec
	if before, _, ok := strings.Cut(importSpec, "#"); ok {
		cleanSpec = before
	}

	// Split on @ to get path and ref
	parts := strings.Split(cleanSpec, "@")
	pathPart := parts[0]
	ref = "main" // default ref
	if len(parts) > 1 {
		ref = parts[1]
	}

	// Parse path: owner/repo
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) != 2 {
		return "", "", ""
	}

	owner = slashParts[0]
	repo = slashParts[1]

	return owner, repo, ref
}

// generateLegacyAgentImportCheckout generates a checkout step for legacy agent imports
// Accepted format: owner/repo@ref or owner/repo (defaults to ref "main")
// Specs with extra path segments are rejected by parseRepositoryImportSpec.
// Only the .github/ folder is checked out via sparse-checkout.
func (c *Compiler) generateLegacyAgentImportCheckout(yaml *strings.Builder, agentImportSpec string) {
	compilerYamlLog.Printf("Generating checkout step for legacy agent import: %s", agentImportSpec)

	// Parse the import spec to extract owner, repo, and ref
	owner, repo, ref := parseRepositoryImportSpec(agentImportSpec)
	if owner == "" || repo == "" {
		compilerYamlLog.Printf("Warning: failed to parse legacy agent import spec: %s", agentImportSpec)
		return
	}

	// Generate a sanitized directory name for the checkout
	sanitizedRef := sanitizeRefForPath(ref)
	checkoutPath := fmt.Sprintf("/tmp/gh-aw/repo-imports/%s-%s-%s", owner, repo, sanitizedRef)

	// Generate the checkout step
	fmt.Fprintf(yaml, "      - name: Checkout agent import %s/%s@%s\n", owner, repo, ref)
	fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("actions/checkout"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          repository: %s/%s\n", owner, repo)
	fmt.Fprintf(yaml, "          ref: %s\n", ref)
	fmt.Fprintf(yaml, "          path: %s\n", checkoutPath)
	yaml.WriteString("          sparse-checkout: |\n")
	yaml.WriteString("            .github/\n")
	yaml.WriteString("          persist-credentials: false\n")

	compilerYamlLog.Printf("Added legacy agent checkout step: %s/%s@%s -> %s", owner, repo, ref, checkoutPath)
}

// generateDevModeCLIBuildSteps generates the steps needed to build the gh-aw CLI and Docker image in dev mode
// These steps are injected after checkout in dev mode to create a locally built Docker image that includes
// the gh-aw binary and all dependencies. The agentic-workflows MCP server uses this image instead of alpine:latest.
//
// The build process:
// 1. Setup Go using go.mod version
// 2. Build the gh-aw CLI binary for linux/amd64 (since it runs in a Linux container)
// 3. Setup Docker Buildx for advanced build features
// 4. Build Docker image and tag it as localhost/gh-aw:dev
//
// The built image is used by the agentic-workflows MCP server configuration (see mcp_config_builtin.go)
func (c *Compiler) generateDevModeCLIBuildSteps(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating dev mode CLI build steps")

	// Step 1: Setup Go for building the CLI
	yaml.WriteString("      - name: Setup Go for CLI build\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("actions/setup-go"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          go-version-file: go.mod\n")
	yaml.WriteString("          cache: true\n")

	// Step 2: Build CLI binary for linux/amd64
	// Use the standard build command from CI/Makefile (not release build)
	// CGO_ENABLED=0 for static linking (required for Alpine containers)
	yaml.WriteString("      - name: Build gh-aw CLI\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"Building gh-aw CLI for linux/amd64...\"\n")
	yaml.WriteString("          mkdir -p dist\n")
	yaml.WriteString("          VERSION=$(git describe --tags --always --dirty)\n")
	yaml.WriteString("          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \\\n")
	yaml.WriteString("            -ldflags \"-s -w -X main.version=${VERSION}\" \\\n")
	yaml.WriteString("            -o dist/gh-aw-linux-amd64 \\\n")
	yaml.WriteString("            ./cmd/gh-aw\n")
	yaml.WriteString("          # Copy binary to root for direct execution in user-defined steps\n")
	yaml.WriteString("          cp dist/gh-aw-linux-amd64 ./gh-aw\n")
	yaml.WriteString("          chmod +x ./gh-aw\n")
	yaml.WriteString("          echo \"✓ Built gh-aw CLI successfully\"\n")

	// Step 3: Setup Docker Buildx
	yaml.WriteString("      - name: Setup Docker Buildx\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("docker/setup-buildx-action"))

	// Step 4: Build Docker image
	// Use the Dockerfile at the repository root which expects BINARY build arg
	yaml.WriteString("      - name: Build gh-aw Docker image\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("docker/build-push-action"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          context: .\n")
	yaml.WriteString("          platforms: linux/amd64\n")
	yaml.WriteString("          push: false\n")
	yaml.WriteString("          load: true\n")
	yaml.WriteString("          tags: localhost/gh-aw:dev\n")
	yaml.WriteString("          build-args: |\n")
	yaml.WriteString("            BINARY=dist/gh-aw-linux-amd64\n")
}
