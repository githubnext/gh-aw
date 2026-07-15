package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var compilerYamlPromptLog = logger.New("workflow:compiler_yaml:prompt")

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
	compilerYamlPromptLog.Printf("Generating prompt for workflow: %s (markdown size: %d bytes)", data.Name, len(data.MarkdownContent))

	// Collect built-in prompt sections (these should be prepended to user prompt)
	builtinSections := c.collectPromptSections(data)
	compilerYamlPromptLog.Printf("Collected %d built-in prompt sections", len(builtinSections))

	// NEW APPROACH: Use runtime-import macros for imports without inputs
	// - Imported markdown without inputs uses runtime-import macros (loaded at runtime)
	// - Imported markdown with inputs is still inlined (compile-time substitution required)
	// - Main workflow markdown body uses runtime-import to allow editing without recompilation
	// This ensures consistency for most imports while maintaining import inputs functionality
	//
	// NOTE: When an engine does not support native agent-file handling
	// (GetCapabilities().NativeAgentFile == false), the agent file content is already present in the
	// prompt via the standard mechanisms below — no special Step 0 is needed:
	//   - Agent files WITHOUT inputs: path is in data.ImportPaths → included by Step 1b.
	//   - Agent files WITH inputs: content is in data.ImportedMarkdown → included by Step 1a.
	//   - inlined-imports mode: data.AgentFile is cleared; content is in data.ImportPaths.
	// All current engines (Claude, Codex, Gemini, Copilot) use this mechanism: NativeAgentFile is false,
	// and they read the fully-assembled prompt.txt in GetExecutionSteps.

	var userPromptChunks []string
	var expressionMappings []*ExpressionMapping

	// Step 1a/1b: Process imports in declaration order, interleaving:
	// - compile-time inlined markdown (imports with inputs)
	// - runtime-import macros (imports without inputs)
	// In older workflow data (without PromptImports), fall back to legacy grouped handling.
	if len(data.PromptImports) > 0 {
		compilerYamlPromptLog.Printf("Processing %d ordered prompt import entries", len(data.PromptImports))
		workspaceRoot := ""
		hasImportInputs := len(data.ImportInputs) > 0
		if data.InlinedImports && c.markdownPath != "" {
			workspaceRoot = resolveWorkspaceRoot(c.markdownPath)
		}
		for _, entry := range data.PromptImports {
			if entry.Markdown != "" {
				cleaned := removeXMLComments(entry.Markdown)
				if hasImportInputs {
					cleaned = SubstituteImportInputs(cleaned, data.ImportInputs)
				}
				chunks, exprMaps := extractPromptChunksFromMarkdown(cleaned)
				userPromptChunks = append(userPromptChunks, chunks...)
				expressionMappings = append(expressionMappings, exprMaps...)
				continue
			}
			if entry.ImportPath == "" {
				continue
			}
			importPath := filepath.ToSlash(entry.ImportPath)
			if workspaceRoot != "" {
				rawContent, err := os.ReadFile(filepath.Join(workspaceRoot, importPath))
				if err != nil {
					compilerYamlPromptLog.Printf("Warning: failed to read import file %s (%v), falling back to runtime-import", importPath, err)
					userPromptChunks = append(userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
					continue
				}
				importedBody, extractErr := parser.ExtractMarkdownContent(string(rawContent))
				if extractErr != nil {
					importedBody = string(rawContent)
				}
				chunks, exprMaps := extractPromptChunksFromMarkdown(importedBody)
				userPromptChunks = append(userPromptChunks, chunks...)
				expressionMappings = append(expressionMappings, exprMaps...)
				continue
			}
			userPromptChunks = append(userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
		}
	} else {
		// Step 1a: Process and inline imported markdown with inputs (if any)
		// Imports with inputs MUST be inlined because substitution happens at compile time
		if data.ImportedMarkdown != "" {
			compilerYamlPromptLog.Printf("Processing imported markdown (%d bytes)", len(data.ImportedMarkdown))

			// Clean, substitute, and post-process imported markdown
			cleaned := removeXMLComments(data.ImportedMarkdown)
			if len(data.ImportInputs) > 0 {
				compilerYamlPromptLog.Printf("Substituting %d import input values", len(data.ImportInputs))
				cleaned = SubstituteImportInputs(cleaned, data.ImportInputs)
			}
			chunks, exprMaps := extractPromptChunksFromMarkdown(cleaned)
			userPromptChunks = append(userPromptChunks, chunks...)
			expressionMappings = append(expressionMappings, exprMaps...)
			compilerYamlPromptLog.Printf("Inlined imported markdown with inputs in %d chunks", len(chunks))
		}

		// Step 1b: For imports without inputs:
		// - inlinedImports mode (inlined-imports: true frontmatter): read and inline content at compile time
		// - normal mode: generate runtime-import macros (loaded at runtime)
		if len(data.ImportPaths) > 0 {
			if data.InlinedImports && c.markdownPath != "" {
				// inlinedImports mode: read import file content from disk and embed directly
				compilerYamlPromptLog.Printf("Inlining %d imports without inputs at compile time", len(data.ImportPaths))
				workspaceRoot := resolveWorkspaceRoot(c.markdownPath)
				for _, importPath := range data.ImportPaths {
					importPath = filepath.ToSlash(importPath)
					rawContent, err := os.ReadFile(filepath.Join(workspaceRoot, importPath))
					if err != nil {
						// Fall back to runtime-import macro if file cannot be read
						compilerYamlPromptLog.Printf("Warning: failed to read import file %s (%v), falling back to runtime-import", importPath, err)
						userPromptChunks = append(userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
						continue
					}
					importedBody, extractErr := parser.ExtractMarkdownContent(string(rawContent))
					if extractErr != nil {
						importedBody = string(rawContent)
					}
					chunks, exprMaps := extractPromptChunksFromMarkdown(importedBody)
					userPromptChunks = append(userPromptChunks, chunks...)
					expressionMappings = append(expressionMappings, exprMaps...)
					compilerYamlPromptLog.Printf("Inlined import without inputs: %s", importPath)
				}
			} else {
				// Normal mode: generate runtime-import macros (loaded at workflow runtime)
				compilerYamlPromptLog.Printf("Generating runtime-import macros for %d imports without inputs", len(data.ImportPaths))
				for _, importPath := range data.ImportPaths {
					importPath = filepath.ToSlash(importPath)
					userPromptChunks = append(userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
					compilerYamlPromptLog.Printf("Added runtime-import macro for: %s", importPath)
				}
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
		compilerYamlPromptLog.Printf("Extracting expressions from main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))

		// Create a new extractor for main workflow markdown
		mainExtractor := NewExpressionExtractor()
		mainExprMappings, err := mainExtractor.ExtractExpressions(data.MainWorkflowMarkdown)
		if err == nil && len(mainExprMappings) > 0 {
			compilerYamlPromptLog.Printf("Extracted %d expressions from main workflow markdown", len(mainExprMappings))
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

	// Add expression mappings for declared experiments.
	// These ensure the interpolation and substitution steps have GH_AW_EXPERIMENTS_* env vars
	// set from pick-experiment step outputs, which is required for:
	//   - Step 2.5 of interpolate_prompt.cjs: substitutes __GH_AW_EXPERIMENTS_*__ placeholders
	//     produced by runtime_import.cjs from {{#if experiments.name}} template conditionals.
	//   - The substitute_placeholders step: replaces any remaining occurrences.
	if len(data.Experiments) > 0 {
		experimentMappings := ExperimentExpressionMappings(data.Experiments)
		compilerYamlPromptLog.Printf("Adding %d experiment expression mapping(s)", len(experimentMappings))
		expressionMappings = append(expressionMappings, experimentMappings...)
	}

	// Step 2: Add main workflow markdown content to the prompt
	if c.inlinePrompt || data.InlinedImports {
		// Inline mode (Wasm/browser): embed the markdown content directly in the YAML
		// since runtime-import macros cannot resolve without filesystem access
		if data.MainWorkflowMarkdown != "" {
			compilerYamlPromptLog.Printf("Inlining main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))

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
			compilerYamlPromptLog.Printf("Inlined main workflow markdown in %d chunks", len(inlinedChunks))
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
		} else if strings.HasPrefix(normalizedPath, constants.GithubDir) {
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
		compilerYamlPromptLog.Printf("Using runtime-import for main workflow markdown: %s", workflowFilePath)

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
		compilerYamlPromptLog.Printf("Adding %d known needs.* expressions for substitution step only", len(knownNeedsExpressions))
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
		envVarNames := sliceutil.SortedKeys(expressionMap)
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

// extractPromptChunksFromMarkdown applies the standard post-processing pipeline to a markdown body:
// XML comment removal, expression wrapping, expression extraction/substitution, and chunking.
// It returns the prompt chunks and expression mappings extracted from the content.
func extractPromptChunksFromMarkdown(body string) ([]string, []*ExpressionMapping) {
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
	if strings.HasPrefix(normalized, constants.GithubDir) {
		// Path already starts at the workspace root.
		return "."
	}
	// Fallback: use the directory containing the workflow file.
	return filepath.Dir(markdownPath)
}
