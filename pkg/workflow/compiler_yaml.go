package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
)

var compilerYamlLog = logger.New("workflow:compiler_yaml")

// effectiveStrictMode computes the effective strict mode for a workflow.
// Priority: CLI flag (c.strictMode) > frontmatter strict field > default (true).
// This should be used when emitting metadata/env vars to correctly reflect the
// workflow's strictness as inferred from the source (frontmatter).
func (c *Compiler) effectiveStrictMode(frontmatter map[string]any) bool {
	if c.strictMode {
		// CLI flag takes precedence
		return true
	}
	if strictVal, exists := frontmatter["strict"]; exists {
		if strictBool, ok := strictVal.(bool); ok {
			return strictBool
		}
	}
	// Default: strict mode is on when no explicit setting
	return true
}

// effectiveSafeUpdate returns true when safe update mode should be enforced for
// the given workflow. Safe update mode is equivalent to strict mode: it is
// enabled whenever strict mode is active (CLI --strict flag, frontmatter
// strict: true, or the default). It can be disabled via the CLI --approve flag
// to approve all changes.
func (c *Compiler) effectiveSafeUpdate(data *WorkflowData) bool {
	if c.approve {
		return false
	}
	return c.effectiveStrictMode(data.RawFrontmatter)
}

// buildJobsAndValidate builds all workflow jobs and validates their dependencies.
// It resets the job manager, builds jobs from the workflow data, and performs
// dependency and duplicate step validation.
func (c *Compiler) buildJobsAndValidate(data *WorkflowData, markdownPath string) error {
	compilerYamlLog.Printf("Building and validating jobs for workflow: %s", data.Name)

	// Reset job manager for this compilation
	c.jobManager = NewJobManager()

	// Build all jobs
	if err := c.buildJobs(data, markdownPath); err != nil {
		compilerYamlLog.Printf("Failed to build jobs: %v", err)
		return fmt.Errorf("failed to build jobs: %w", err)
	}

	compilerYamlLog.Printf("Built %d jobs successfully", len(c.jobManager.GetAllJobs()))

	// Validate job dependencies
	if err := c.jobManager.ValidateDependencies(); err != nil {
		return fmt.Errorf("job dependency validation failed: %w", err)
	}

	// Validate no duplicate steps within jobs (compiler bug detection)
	if err := c.jobManager.ValidateDuplicateSteps(); err != nil {
		return fmt.Errorf("duplicate step validation failed: %w", err)
	}

	return nil
}

// generateWorkflowHeader generates the YAML header section including comments
// for description, source, imports/includes, frontmatter-hash, stop-time, and manual-approval.
// All ANSI escape codes are stripped from the output.
// The gh-aw-metadata line is placed first for easy machine parsing.
func (c *Compiler) generateWorkflowHeader(yaml *strings.Builder, data *WorkflowData, frontmatterHash string, secrets []string, actions []string) {
	// Skip the ASCII art banner in wasm/editor mode — it takes up too much space
	if c.skipHeader {
		return
	}

	// Add lock metadata as the very first line for easy machine parsing.
	// Single-line JSON format to minimize merge conflicts.
	if frontmatterHash != "" {
		agentInfo := AgentMetadataInfo{}
		// Agent ID: prefer EngineConfig.ID, fall back to legacy AI field
		if data.EngineConfig != nil && data.EngineConfig.ID != "" {
			agentInfo.AgentID = data.EngineConfig.ID
		} else if data.AI != "" {
			agentInfo.AgentID = data.AI
		}
		// Agent model: only include if statically configured
		if data.EngineConfig != nil && data.EngineConfig.Model != "" {
			agentInfo.AgentModel = data.EngineConfig.Model
		}
		// Detection agent info: only if threat detection has its own engine config
		if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.EngineConfig != nil {
			agentInfo.DetectionAgentID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
			agentInfo.DetectionAgentModel = data.SafeOutputs.ThreatDetection.EngineConfig.Model
		}
		metadata := GenerateLockMetadata(frontmatterHash, data.StopTime, c.effectiveStrictMode(data.RawFrontmatter), agentInfo)
		metadataJSON, err := metadata.ToJSON()
		if err != nil {
			// Fallback to legacy format if JSON serialization fails
			fmt.Fprintf(yaml, "# frontmatter-hash: %s\n", frontmatterHash)
		} else {
			fmt.Fprintf(yaml, "# gh-aw-metadata: %s\n", metadataJSON)
		}
	}

	// Embed the gh-aw-manifest immediately after gh-aw-metadata for easy machine parsing.
	// The manifest records all secrets, external actions, and container images detected at
	// compile time so that subsequent compilations can perform safe update enforcement.
	manifest := NewGHAWManifest(secrets, actions, data.DockerImagePins)
	if manifestJSON, err := manifest.ToJSON(); err == nil {
		fmt.Fprintf(yaml, "# gh-aw-manifest: %s\n", manifestJSON)
	} else {
		compilerYamlLog.Printf("Failed to serialize gh-aw-manifest: %v. Safe update mode will not be available for future compilations of this workflow.", err)
	}

	// Add workflow header with logo and instructions
	sourceFile := "the corresponding .md file"
	if data.Source != "" {
		sourceFile = data.Source
	}
	header := GenerateWorkflowHeader(sourceFile, "gh-aw", "")
	yaml.WriteString(header)

	// Add description comment if provided
	if data.Description != "" {
		cleanDescription := stringutil.StripANSI(data.Description)
		// Split description into lines and prefix each with "# "
		descriptionLines := strings.SplitSeq(strings.TrimSpace(cleanDescription), "\n")
		for line := range descriptionLines {
			fmt.Fprintf(yaml, "# %s\n", strings.TrimSpace(line))
		}
	}

	// Add source comment if provided
	if data.Source != "" {
		yaml.WriteString("#\n")
		cleanSource := stringutil.StripANSI(data.Source)
		// Normalize to Unix paths (forward slashes) for cross-platform compatibility
		cleanSource = filepath.ToSlash(cleanSource)
		fmt.Fprintf(yaml, "# Source: %s\n", cleanSource)
	}

	// Add manifest of imported/included files if any exist
	// Build a user-visible imports list by filtering out internal builtin engine paths
	// (e.g. "@builtin:engines/copilot.md") which are implementation details.
	var visibleImports []string
	for _, file := range data.ImportedFiles {
		if !strings.HasPrefix(file, parser.BuiltinPathPrefix) {
			visibleImports = append(visibleImports, file)
		}
	}

	if len(visibleImports) > 0 || len(data.IncludedFiles) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Resolved workflow manifest:\n")

		if len(visibleImports) > 0 {
			yaml.WriteString("#   Imports:\n")
			for _, file := range visibleImports {
				cleanFile := stringutil.StripANSI(file)
				// Normalize to Unix paths (forward slashes) for cross-platform compatibility
				cleanFile = filepath.ToSlash(cleanFile)
				fmt.Fprintf(yaml, "#     - %s\n", cleanFile)
			}
		}

		if len(data.IncludedFiles) > 0 {
			yaml.WriteString("#   Includes:\n")
			for _, file := range data.IncludedFiles {
				cleanFile := stringutil.StripANSI(file)
				// Normalize to Unix paths (forward slashes) for cross-platform compatibility
				cleanFile = filepath.ToSlash(cleanFile)
				fmt.Fprintf(yaml, "#     - %s\n", cleanFile)
			}
		}
	}

	// Add inlined-imports comment to indicate the field was used at compile time
	if data.InlinedImports {
		yaml.WriteString("#\n")
		yaml.WriteString("# inlined-imports: true\n")
	}

	// Add frontmatter-declared env vars with source attribution.
	// Note: programmatically injected env vars (e.g. OTEL_* from OTLP config) are not listed here.
	if len(data.EnvSources) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Frontmatter env variables:\n")
		// Sort keys for deterministic output
		keys := make([]string, 0, len(data.EnvSources))
		for k := range data.EnvSources {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(yaml, "#   - %s: %s\n", k, data.EnvSources[k])
		}
	}

	// Add list of secrets referenced in the workflow
	if len(secrets) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Secrets used:\n")
		for _, s := range secrets {
			fmt.Fprintf(yaml, "#   - %s\n", s)
		}
	}

	// Add list of external custom actions referenced in the workflow
	if len(actions) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Custom actions used:\n")
		for _, a := range actions {
			fmt.Fprintf(yaml, "#   - %s\n", a)
		}
	}

	// Add list of container images used in the workflow
	if len(data.DockerImages) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Container images used:\n")
		for _, img := range data.DockerImages {
			fmt.Fprintf(yaml, "#   - %s\n", img)
		}
	}

	// Add stop-time comment if configured
	if data.StopTime != "" {
		yaml.WriteString("#\n")
		cleanStopTime := stringutil.StripANSI(data.StopTime)
		fmt.Fprintf(yaml, "# Effective stop-time: %s\n", cleanStopTime)
	}

	// Add manual-approval comment if configured
	if data.ManualApproval != "" {
		yaml.WriteString("#\n")
		cleanManualApproval := stringutil.StripANSI(data.ManualApproval)
		fmt.Fprintf(yaml, "# Manual approval required: environment '%s'\n", cleanManualApproval)
	}

	yaml.WriteString("\n")
}

// generateWorkflowBody generates the main workflow structure including name, triggers,
// permissions, concurrency, run-name, environment variables, cache comments, and jobs.
func (c *Compiler) generateWorkflowBody(yaml *strings.Builder, data *WorkflowData) {
	// Write basic workflow structure
	fmt.Fprintf(yaml, "name: \"%s\"\n", data.Name)

	// Inject on.workflow_call.outputs when workflow_call is configured and safe-outputs are present
	onSection := data.On
	if data.SafeOutputs != nil {
		onSection = c.injectWorkflowCallOutputs(onSection, data.SafeOutputs)
	}
	// Inject aw_context input into workflow_dispatch triggers so dispatched workflows
	// can receive caller metadata (repo, run_id, actor, etc.) from dispatch_workflow.
	// String-based injection preserves existing YAML comments and formatting.
	onSection = injectAwContextIntoOnYAML(onSection)
	yaml.WriteString(onSection + "\n\n")

	// Note: GitHub Actions doesn't support workflow-level if conditions
	// The workflow_run safety check is added to individual jobs instead

	// Always write empty permissions at the top level
	// Agent permissions are applied only to the agent job
	yaml.WriteString("permissions: {}\n\n")

	yaml.WriteString(data.Concurrency + "\n\n")
	yaml.WriteString(data.RunName + "\n\n")

	// Add env section if present
	if data.Env != "" {
		yaml.WriteString(data.Env + "\n\n")
	}

	// Add cache comment if cache configuration was provided
	if data.Cache != "" {
		yaml.WriteString("# Cache configuration from frontmatter was processed and added to the main job steps\n\n")
	}

	// Generate jobs section using JobManager
	yaml.WriteString(c.jobManager.RenderToYAML())
}

func (c *Compiler) generateYAML(data *WorkflowData, markdownPath string) (string, []string, []string, error) {
	compilerYamlLog.Printf("Generating YAML for workflow: %s", data.Name)

	// Compute frontmatter hash BEFORE building jobs so that the stable hash is
	// available to heredoc-delimiter generation throughout job construction.
	// Using the hex-encoded SHA-256 frontmatter hash string as an HMAC key keeps
	// the compiled lock file identical across repeated compilations of the same workflow.
	var frontmatterHash string
	if markdownPath != "" {
		baseDir := filepath.Dir(markdownPath)
		cache := parser.NewImportCache(baseDir)
		hash, err := parser.ComputeFrontmatterHashFromFileWithParsedFrontmatter(markdownPath, data.RawFrontmatter, cache, parser.DefaultFileReader)
		if err != nil {
			compilerYamlLog.Printf("Warning: failed to compute frontmatter hash: %v", err)
			// Continue without hash - non-fatal error
		} else {
			frontmatterHash = hash
			compilerYamlLog.Printf("Computed frontmatter hash: %s", hash)
		}
	}
	// Store hash on WorkflowData so job-building helpers (MCP renderers, prompt
	// step generators, etc.) can derive stable heredoc delimiters from it.
	data.FrontmatterHash = frontmatterHash

	// Build all jobs and validate dependencies
	if err := c.buildJobsAndValidate(data, markdownPath); err != nil {
		return "", nil, nil, fmt.Errorf("failed to build and validate jobs: %w", err)
	}

	// Pre-allocate builder capacity based on estimated workflow size.
	// Most workflows are in the 32–160 KB range; 64 KB avoids the first reallocation
	// in the common case while keeping memory waste low for small workflows.
	const initialBuilderCapacity = 64 * 1024
	var yaml strings.Builder
	yaml.Grow(initialBuilderCapacity)

	// Generate workflow body first so we can collect secrets and custom actions
	// for inclusion in the header comment.
	var body strings.Builder
	body.Grow(initialBuilderCapacity)
	c.generateWorkflowBody(&body, data)
	bodyContent := body.String()

	// Collect secrets and external action references from the generated body.
	// These are returned to the caller so they can be used for safe update enforcement
	// without requiring a second scan of the full YAML content.
	secrets := CollectSecretReferences(bodyContent)
	actions := CollectActionReferences(bodyContent)

	// Generate workflow header comments (including metadata as first line, plus secrets/actions lists)
	c.generateWorkflowHeader(&yaml, data, frontmatterHash, secrets, actions)

	// Append the workflow body
	yaml.WriteString(bodyContent)

	yamlContent := yaml.String()

	// If we're in non-cloning trial mode and this workflow has issue triggers,
	// replace github.event.issue.number with inputs.issue_number
	if c.trialMode && c.hasIssueTrigger(data.On) {
		compilerYamlLog.Print("Trial mode enabled, replacing issue number references")
		yamlContent = c.replaceIssueNumberReferences(yamlContent)
	}

	compilerYamlLog.Printf("Successfully generated YAML for workflow: %s (%d bytes)", data.Name, len(yamlContent))
	return yamlContent, secrets, actions, nil
}

func splitContentIntoChunks(content string) []string {
	const maxChunkSize = 20900        // 21000 - 100 character buffer
	const indentSpaces = "          " // 10 spaces added to each line

	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk []string
	currentSize := 0

	for _, line := range lines {
		lineSize := len(indentSpaces) + len(line) + 1 // +1 for newline

		// If adding this line would exceed the limit, start a new chunk
		if currentSize+lineSize > maxChunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))
			currentChunk = []string{line}
			currentSize = lineSize
		} else {
			currentChunk = append(currentChunk, line)
			currentSize += lineSize
		}
	}

	// Add the last chunk if there's content
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}

	return chunks
}

func (c *Compiler) generatePrompt(yaml *strings.Builder, data *WorkflowData, preActivationJobCreated bool, beforeActivationJobs []string) {
	compilerYamlLog.Printf("Generating prompt for workflow: %s (markdown size: %d bytes)", data.Name, len(data.MarkdownContent))

	// Collect built-in prompt sections (these should be prepended to user prompt)
	builtinSections := c.collectPromptSections(data)
	compilerYamlLog.Printf("Collected %d built-in prompt sections", len(builtinSections))

	// NEW APPROACH: Use runtime-import macros for imports without inputs
	// - Imported markdown without inputs uses runtime-import macros (loaded at runtime)
	// - Imported markdown with inputs is still inlined (compile-time substitution required)
	// - Main workflow markdown body uses runtime-import to allow editing without recompilation
	// This ensures consistency for most imports while maintaining import inputs functionality
	//
	// NOTE: When an engine does not support native agent-file handling
	// (SupportsNativeAgentFile() == false), the agent file content is already present in the
	// prompt via the standard mechanisms below — no special Step 0 is needed:
	//   - Agent files WITHOUT inputs: path is in data.ImportPaths → included by Step 1b.
	//   - Agent files WITH inputs: content is in data.ImportedMarkdown → included by Step 1a.
	//   - inlined-imports mode: data.AgentFile is cleared; content is in data.ImportPaths.
	// All current engines (Claude, Codex, Gemini, Copilot) use this mechanism: SupportsNativeAgentFile()
	// returns false and they read the fully-assembled prompt.txt in GetExecutionSteps.

	var userPromptChunks []string
	var expressionMappings []*ExpressionMapping

	// Step 1a: Process and inline imported markdown with inputs (if any)
	// Imports with inputs MUST be inlined because substitution happens at compile time
	if data.ImportedMarkdown != "" {
		compilerYamlLog.Printf("Processing imported markdown (%d bytes)", len(data.ImportedMarkdown))

		// Clean, substitute, and post-process imported markdown
		cleaned := removeXMLComments(data.ImportedMarkdown)
		if len(data.ImportInputs) > 0 {
			compilerYamlLog.Printf("Substituting %d import input values", len(data.ImportInputs))
			cleaned = SubstituteImportInputs(cleaned, data.ImportInputs)
		}
		chunks, exprMaps := processMarkdownBody(cleaned)
		userPromptChunks = append(userPromptChunks, chunks...)
		expressionMappings = exprMaps
		compilerYamlLog.Printf("Inlined imported markdown with inputs in %d chunks", len(chunks))
	}

	// Step 1b: For imports without inputs:
	// - inlinedImports mode (inlined-imports: true frontmatter): read and inline content at compile time
	// - normal mode: generate runtime-import macros (loaded at runtime)
	if len(data.ImportPaths) > 0 {
		if data.InlinedImports && c.markdownPath != "" {
			// inlinedImports mode: read import file content from disk and embed directly
			compilerYamlLog.Printf("Inlining %d imports without inputs at compile time", len(data.ImportPaths))
			workspaceRoot := resolveWorkspaceRoot(c.markdownPath)
			for _, importPath := range data.ImportPaths {
				importPath = filepath.ToSlash(importPath)
				rawContent, err := os.ReadFile(filepath.Join(workspaceRoot, importPath))
				if err != nil {
					// Fall back to runtime-import macro if file cannot be read
					compilerYamlLog.Printf("Warning: failed to read import file %s (%v), falling back to runtime-import", importPath, err)
					userPromptChunks = append(userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
					continue
				}
				importedBody, extractErr := parser.ExtractMarkdownContent(string(rawContent))
				if extractErr != nil {
					importedBody = string(rawContent)
				}
				chunks, exprMaps := processMarkdownBody(importedBody)
				userPromptChunks = append(userPromptChunks, chunks...)
				expressionMappings = append(expressionMappings, exprMaps...)
				compilerYamlLog.Printf("Inlined import without inputs: %s", importPath)
			}
		} else {
			// Normal mode: generate runtime-import macros (loaded at workflow runtime)
			compilerYamlLog.Printf("Generating runtime-import macros for %d imports without inputs", len(data.ImportPaths))
			for _, importPath := range data.ImportPaths {
				importPath = filepath.ToSlash(importPath)
				userPromptChunks = append(userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
				compilerYamlLog.Printf("Added runtime-import macro for: %s", importPath)
			}
		}
	}

	// Step 1.5: Extract expressions from main workflow markdown (not imported content)
	// This is needed for needs.* expressions and other compile-time expressions
	// The main workflow markdown uses runtime-import, but expressions like needs.* must be
	// available at compile time for the substitute placeholders step
	// Use MainWorkflowMarkdown (not MarkdownContent) to avoid extracting from imported content
	// Skip this step when inlinePrompt is true because expression extraction happens in Step 2
	if !c.inlinePrompt && !data.InlinedImports && data.MainWorkflowMarkdown != "" {
		compilerYamlLog.Printf("Extracting expressions from main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))

		// Create a new extractor for main workflow markdown
		mainExtractor := NewExpressionExtractor()
		mainExprMappings, err := mainExtractor.ExtractExpressions(data.MainWorkflowMarkdown)
		if err == nil && len(mainExprMappings) > 0 {
			compilerYamlLog.Printf("Extracted %d expressions from main workflow markdown", len(mainExprMappings))
			// Merge with imported expressions (append to existing mappings)
			expressionMappings = append(expressionMappings, mainExprMappings...)
		}
	}

	// Filter out expression mappings referencing custom jobs that run AFTER activation.
	// These jobs (which explicitly depend on activation) cannot have outputs available when
	// the activation job builds and substitutes the prompt. Keeping them would cause actionlint
	// errors because the jobs are not in activation's needs, yet their outputs would be
	// referenced in activation's step env vars.
	expressionMappings = filterExpressionsForActivation(expressionMappings, data.Jobs, beforeActivationJobs)

	// Step 2: Add main workflow markdown content to the prompt
	if c.inlinePrompt || data.InlinedImports {
		// Inline mode (Wasm/browser): embed the markdown content directly in the YAML
		// since runtime-import macros cannot resolve without filesystem access
		if data.MainWorkflowMarkdown != "" {
			compilerYamlLog.Printf("Inlining main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))

			inlinedMarkdown := removeXMLComments(data.MainWorkflowMarkdown)
			inlinedMarkdown = wrapExpressionsInTemplateConditionals(inlinedMarkdown)

			// Extract expressions and replace with env var references
			inlineExtractor := NewExpressionExtractor()
			inlineExprMappings, err := inlineExtractor.ExtractExpressions(inlinedMarkdown)
			if err == nil && len(inlineExprMappings) > 0 {
				inlinedMarkdown = inlineExtractor.ReplaceExpressionsWithEnvVars(inlinedMarkdown)
				expressionMappings = append(expressionMappings, inlineExprMappings...)
			}

			inlinedChunks := splitContentIntoChunks(inlinedMarkdown)
			userPromptChunks = append(userPromptChunks, inlinedChunks...)
			compilerYamlLog.Printf("Inlined main workflow markdown in %d chunks", len(inlinedChunks))
		}
	} else {
		// Normal mode: use runtime-import macro so users can edit without recompilation
		workflowBasename := filepath.Base(c.markdownPath)

		// Determine the directory path relative to workspace root
		// For a workflow at ".github/workflows/test.md", the runtime-import path should be ".github/workflows/test.md"
		// This makes the path explicit and matches the actual file location in the repository
		var workflowFilePath string

		// Normalize path separators first to handle both Unix and Windows paths consistently
		normalizedPath := filepath.ToSlash(c.markdownPath)

		// Look for "/.github/" as a directory (not just substring in repo name like "username.github.io")
		// We need to match the directory component, not arbitrary substrings.
		// Use LastIndex so that when the repo itself is named ".github" (path like
		// "/root/.github/.github/workflows/file.md"), we find the actual .github
		// workflows directory rather than the repo root directory.
		githubDirPattern := "/.github/"
		githubIndex := strings.LastIndex(normalizedPath, githubDirPattern)

		if githubIndex != -1 {
			// Extract everything from ".github/" onwards (inclusive)
			// +1 to skip the leading slash, so we get ".github/workflows/..." not "/.github/workflows/..."
			workflowFilePath = normalizedPath[githubIndex+1:]
		} else if strings.HasPrefix(normalizedPath, ".github/") {
			// Relative path already starting with ".github/" — use as-is.
			// This can happen when the compiler is invoked with a relative markdown path
			// (e.g. ".github/workflows/test.md") rather than an absolute one.
			workflowFilePath = normalizedPath
		} else {
			// For non-standard paths (like /tmp/test.md), just use the basename
			workflowFilePath = workflowBasename
		}

		// Create a runtime-import macro for the main workflow markdown
		// The runtime_import.cjs helper will extract and process the markdown body at runtime
		// The path uses .github/ prefix for clarity (e.g., .github/workflows/test.md)
		runtimeImportMacro := fmt.Sprintf("{{#runtime-import %s}}", workflowFilePath)
		compilerYamlLog.Printf("Using runtime-import for main workflow markdown: %s", workflowFilePath)

		// Append runtime-import macro after imported chunks
		userPromptChunks = append(userPromptChunks, runtimeImportMacro)
	}

	// Enhance entity number expressions with || inputs.item_number fallback when the
	// workflow has a workflow_dispatch trigger with item_number (generated by the label
	// trigger shorthand). This is applied after all expression mappings (including inline
	// mode ones) have been collected so that every entity number reference gets the fallback.
	applyWorkflowDispatchFallbacks(expressionMappings, data.HasDispatchItemNumber)

	// Generate a single unified prompt creation step WITHOUT known needs expressions
	// Known needs expressions are added later for the substitution step only
	// This returns the combined expression mappings for use in the substitution step
	allExpressionMappings := c.generateUnifiedPromptCreationStep(yaml, builtinSections, userPromptChunks, expressionMappings, data)

	// Step 1.6: Add all known needs.* expressions for the substitution step ONLY
	// Since the markdown may change without recompilation (via runtime-import), we need to
	// ensure all known needs.* variables are available for interpolation in the substitution step.
	// These are NOT added to the prompt creation step because they're not needed there.
	knownNeedsExpressions := generateKnownNeedsExpressions(data, preActivationJobCreated)
	if len(knownNeedsExpressions) > 0 {
		compilerYamlLog.Printf("Adding %d known needs.* expressions for substitution step only", len(knownNeedsExpressions))
		// Merge known needs expressions with the returned expression mappings for substitution
		// We use a map to avoid duplicates (expressions from markdown take precedence)
		expressionMap := make(map[string]*ExpressionMapping)
		// First add known needs expressions (these have lower priority)
		for _, mapping := range knownNeedsExpressions {
			expressionMap[mapping.EnvVar] = mapping
		}
		// Then add/override with expressions from allExpressionMappings (these have higher priority)
		for _, mapping := range allExpressionMappings {
			expressionMap[mapping.EnvVar] = mapping
		}
		// Convert back to slice in sorted order (by environment variable name) for deterministic output
		allExpressionMappings = make([]*ExpressionMapping, 0, len(expressionMap))
		// Get all keys and sort them
		envVarNames := make([]string, 0, len(expressionMap))
		for envVar := range expressionMap {
			envVarNames = append(envVarNames, envVar)
		}
		sort.Strings(envVarNames)
		// Add mappings in sorted order
		for _, envVar := range envVarNames {
			allExpressionMappings = append(allExpressionMappings, expressionMap[envVar])
		}
	}

	// Add combined interpolation and template rendering step
	// This step processes runtime-import macros, so it must run BEFORE placeholder substitution
	c.generateInterpolationAndTemplateStep(yaml, expressionMappings, data)

	// Generate JavaScript-based placeholder substitution step
	// This MUST run AFTER interpolation because placeholders in runtime-imported files
	// (like changeset.md) need to be substituted after the file is imported
	// Now includes the known needs.* expressions
	if len(allExpressionMappings) > 0 {
		generatePlaceholderSubstitutionStep(yaml, allExpressionMappings, "      ", data)
	}

	// Validate that all placeholders have been substituted
	writePromptBashStep(yaml, "Validate prompt placeholders", "validate_prompt_placeholders.sh")

	// Print prompt (merged into prompt generation)
	writePromptBashStep(yaml, "Print prompt", "print_prompt_summary.sh")
}

// writePromptBashStep writes a YAML step that runs a bash script from the gh-aw actions directory
// with the GH_AW_PROMPT env var set. The poutine:ignore suppression is included to address
// untrusted_checkout_exec findings for scripts executed from RUNNER_TEMP.
func writePromptBashStep(yaml *strings.Builder, name, script string) {
	fmt.Fprintf(yaml, "      - name: %s\n", name)
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        # poutine:ignore untrusted_checkout_exec\n")
	fmt.Fprintf(yaml, "        run: bash \"${RUNNER_TEMP}/gh-aw/actions/%s\"\n", script)
}

func (c *Compiler) generatePreSteps(yaml *strings.Builder, data *WorkflowData) {
	writeStepsSection(yaml, data.PreSteps)
}

func (c *Compiler) generatePostSteps(yaml *strings.Builder, data *WorkflowData) {
	writeStepsSection(yaml, data.PostSteps)
}

// writeStepsSection writes a steps section (pre-steps or post-steps) to the YAML builder,
// stripping the header line and normalising indentation to match the agent job step format:
// top-level items get 6-space indent (      - name:) and nested properties get 8-space indent (        run:).
func writeStepsSection(yaml *strings.Builder, stepsYAML string) {
	if stepsYAML == "" {
		return
	}
	lines := strings.Split(stepsYAML, "\n")
	for _, line := range lines[1:] { // skip the "pre-steps:" / "post-steps:" header line
		trimmed := strings.TrimRight(line, " ")
		if strings.TrimSpace(trimmed) == "" {
			yaml.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "  ") {
			yaml.WriteString("        " + line[2:] + "\n")
		} else {
			yaml.WriteString("      " + line + "\n")
		}
	}
}

func (c *Compiler) generateCreateAwInfo(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	// Engine ID (prefer EngineConfig.ID, fallback to AI field for backwards compatibility)
	engineID := engine.GetID()
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}

	// Model - explicit config or runtime env var via vars context
	modelConfigured := data.EngineConfig != nil && data.EngineConfig.Model != ""
	var modelEnvVar string
	if !modelConfigured {
		switch engineID {
		case "copilot":
			modelEnvVar = constants.EnvVarModelAgentCopilot
		case "claude":
			modelEnvVar = constants.EnvVarModelAgentClaude
		case "codex":
			modelEnvVar = constants.EnvVarModelAgentCodex
		case "custom":
			modelEnvVar = constants.EnvVarModelAgentCustom
		default:
			modelEnvVar = constants.EnvVarModelAgentCustom
		}
	}

	// Agent version - use the actual installation version (includes defaults)
	agentVersion := getInstallationVersion(data, engine)

	// Version: prefer explicit engine config version, fall back to the installation version
	// so the run details always show the version being used rather than "(none)".
	version := agentVersion
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		version = data.EngineConfig.Version
	}

	// Staged value from safe-outputs configuration
	stagedValue := "false"
	if data.SafeOutputs != nil && data.SafeOutputs.Staged {
		stagedValue = "true"
	}

	// Network configuration
	var allowedDomains []string
	firewallEnabled := false
	firewallVersion := ""
	if data.NetworkPermissions != nil {
		allowedDomains = data.NetworkPermissions.Allowed
		if data.NetworkPermissions.Firewall != nil {
			firewallEnabled = data.NetworkPermissions.Firewall.Enabled
			firewallVersion = data.NetworkPermissions.Firewall.Version
			if firewallEnabled && firewallVersion == "" {
				firewallVersion = string(constants.DefaultFirewallVersion)
			}
		}
	}

	// Allowed domains as JSON array string
	domainsJSON := "[]"
	if len(allowedDomains) > 0 {
		b, _ := json.Marshal(allowedDomains)
		domainsJSON = string(b)
	}

	// MCP Gateway version
	mcpGatewayVersion := ""
	if data.SandboxConfig != nil && data.SandboxConfig.MCP != nil && data.SandboxConfig.MCP.Version != "" {
		mcpGatewayVersion = data.SandboxConfig.MCP.Version
	}

	// Firewall type
	firewallType := ""
	if isFirewallEnabled(data) {
		firewallType = "squid"
	}

	yaml.WriteString("      - name: Generate agentic run info\n")
	yaml.WriteString("        id: generate_aw_info\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_AW_INFO_ENGINE_ID: \"%s\"\n", engineID)
	fmt.Fprintf(yaml, "          GH_AW_INFO_ENGINE_NAME: \"%s\"\n", engine.GetDisplayName())
	if modelConfigured {
		fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: \"%s\"\n", data.EngineConfig.Model)
	} else {
		// Use the engine's default model as fallback when neither explicit model nor
		// model variable is configured, so the run details show "auto" rather than "(none)".
		defaultModel := getDefaultAgentModel(engineID)
		if defaultModel != "" {
			fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: ${{ vars.%s || '%s' }}\n", modelEnvVar, defaultModel)
		} else {
			fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: ${{ vars.%s || '' }}\n", modelEnvVar)
		}
	}
	fmt.Fprintf(yaml, "          GH_AW_INFO_VERSION: \"%s\"\n", version)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AGENT_VERSION: \"%s\"\n", agentVersion)
	// CLI version only for released builds
	if IsReleasedVersion(c.version) {
		fmt.Fprintf(yaml, "          GH_AW_INFO_CLI_VERSION: \"%s\"\n", c.version)
	}
	fmt.Fprintf(yaml, "          GH_AW_INFO_WORKFLOW_NAME: \"%s\"\n", data.Name)
	fmt.Fprintf(yaml, "          GH_AW_INFO_EXPERIMENTAL: \"%t\"\n", engine.IsExperimental())
	fmt.Fprintf(yaml, "          GH_AW_INFO_SUPPORTS_TOOLS_ALLOWLIST: \"%t\"\n", engine.SupportsToolsAllowlist())
	fmt.Fprintf(yaml, "          GH_AW_INFO_STAGED: \"%s\"\n", stagedValue)
	fmt.Fprintf(yaml, "          GH_AW_INFO_ALLOWED_DOMAINS: '%s'\n", domainsJSON)
	fmt.Fprintf(yaml, "          GH_AW_INFO_FIREWALL_ENABLED: \"%t\"\n", firewallEnabled)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AWF_VERSION: \"%s\"\n", firewallVersion)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AWMG_VERSION: \"%s\"\n", mcpGatewayVersion)
	fmt.Fprintf(yaml, "          GH_AW_INFO_FIREWALL_TYPE: \"%s\"\n", firewallType)
	// Always include strict mode flag for lockdown validation.
	// validateLockdownRequirements uses this to enforce strict: true for public repositories.
	// Use effectiveStrictMode to infer strictness from the source (frontmatter), not just the CLI flag.
	fmt.Fprintf(yaml, "          GH_AW_COMPILED_STRICT: \"%t\"\n", c.effectiveStrictMode(data.RawFrontmatter))
	// When a workflow_call trigger is present, pass the target_repo resolved by the
	// resolve-host-repo step so it can be stored in aw_info.json for observability.
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		yaml.WriteString("          GH_AW_INFO_TARGET_REPO: ${{ steps.resolve-host-repo.outputs.target_repo }}\n")
	}
	// Include lockdown validation env vars when lockdown is explicitly enabled.
	// validateLockdownRequirements is called from generate_aw_info.cjs and uses these vars.
	githubTool, hasGitHub := data.Tools["github"]
	if hasGitHub && githubTool != false && hasGitHubLockdownExplicitlySet(githubTool) && getGitHubLockdown(githubTool) {
		yaml.WriteString("          GITHUB_MCP_LOCKDOWN_EXPLICIT: \"true\"\n")
		yaml.WriteString("          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}\n")
		yaml.WriteString("          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}\n")
		if customToken := getGitHubToken(githubTool); customToken != "" {
			fmt.Fprintf(yaml, "          CUSTOM_GITHUB_TOKEN: %s\n", customToken)
		}
	}
	// Embed custom token weights when specified in engine.token-weights
	if data.EngineConfig != nil && data.EngineConfig.TokenWeights != nil {
		if tokenWeightsJSON, err := json.Marshal(data.EngineConfig.TokenWeights); err == nil {
			// Escape single quotes for YAML single-quoted scalar safety
			escapedTokenWeightsJSON := strings.ReplaceAll(string(tokenWeightsJSON), "'", "''")
			fmt.Fprintf(yaml, "          GH_AW_INFO_TOKEN_WEIGHTS: '%s'\n", escapedTokenWeightsJSON)
		}
	}
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs');\n")
	yaml.WriteString("            await main(core, context);\n")
}

func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) {
	// Copy the raw safe-output NDJSON to a /tmp/gh-aw/ path so it can be included in the
	// unified agent artifact together with all other /tmp/gh-aw/ outputs.
	yaml.WriteString("      - name: Copy Safe Outputs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("        run: |\n")
	fmt.Fprintf(yaml, "          mkdir -p /tmp/gh-aw\n")
	fmt.Fprintf(yaml, "          cp \"$GH_AW_SAFE_OUTPUTS\" /tmp/gh-aw/%s 2>/dev/null || true\n", constants.SafeOutputsFilename)

	yaml.WriteString("      - name: Ingest agent output\n")
	yaml.WriteString("        id: collect_output\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))

	// Add environment variables for JSONL validation
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")

	// Config is written to file, not passed as env var

	// Add allowed domains configuration for sanitization
	// Use manually configured domains if available, otherwise compute from network configuration
	var domainsStr string
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		// allowed-domains: additional domains unioned with engine/network base set; supports ecosystem identifiers
		domainsStr = c.computeExpandedAllowedDomainsForSanitization(data)
	} else {
		// Fall back to computing from network configuration (same as firewall)
		domainsStr = c.computeAllowedDomainsForSanitization(data)
	}
	if domainsStr != "" {
		fmt.Fprintf(yaml, "          GH_AW_ALLOWED_DOMAINS: %q\n", domainsStr)
	}

	// Add allowed GitHub references configuration for reference escaping
	if data.SafeOutputs != nil && data.SafeOutputs.AllowGitHubReferences != nil {
		refsStr := strings.Join(data.SafeOutputs.AllowGitHubReferences, ",")
		fmt.Fprintf(yaml, "          GH_AW_ALLOWED_GITHUB_REFS: %q\n", refsStr)
	}

	// Add GitHub server URL and API URL for dynamic domain extraction
	// This allows the sanitization code to permit GitHub domains that vary by deployment
	yaml.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	yaml.WriteString("          GITHUB_API_URL: ${{ github.api_url }}\n")

	// Add command name for command trigger prevention in safe outputs
	if len(data.Command) > 0 {
		// Pass first command for backward compatibility
		fmt.Fprintf(yaml, "          GH_AW_COMMAND: %s\n", data.Command[0])
	}

	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Load script from external file using require()
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/collect_ndjson_output.cjs');\n")
	yaml.WriteString("            await main();\n")

}

// processMarkdownBody applies the standard post-processing pipeline to a markdown body:
// XML comment removal, expression wrapping, expression extraction/substitution, and chunking.
// It returns the prompt chunks and expression mappings extracted from the content.
func processMarkdownBody(body string) ([]string, []*ExpressionMapping) {
	body = removeXMLComments(body)
	body = wrapExpressionsInTemplateConditionals(body)
	extractor := NewExpressionExtractor()
	exprMappings, err := extractor.ExtractExpressions(body)
	if err == nil && len(exprMappings) > 0 {
		body = extractor.ReplaceExpressionsWithEnvVars(body)
	} else {
		exprMappings = nil
	}
	return splitContentIntoChunks(body), exprMappings
}

// resolveWorkspaceRoot returns the workspace root directory given the path to a workflow markdown
// file. ImportPaths are relative to the workspace root (e.g. ".github/workflows/shared/foo.md"),
// so the workspace root is the directory that contains ".github/".
func resolveWorkspaceRoot(markdownPath string) string {
	normalized := filepath.ToSlash(markdownPath)
	if before, _, ok := strings.Cut(normalized, "/.github/"); ok {
		// Absolute or non-root-relative path: strip everything from "/.github/" onward.
		return filepath.FromSlash(before)
	}
	if strings.HasPrefix(normalized, ".github/") {
		// Path already starts at the workspace root.
		return "."
	}
	// Fallback: use the directory containing the workflow file.
	return filepath.Dir(markdownPath)
}
