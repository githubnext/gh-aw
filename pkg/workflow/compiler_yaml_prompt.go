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

	builtinSections := c.collectPromptSections(data)
	compilerYamlPromptLog.Printf("Collected %d built-in prompt sections", len(builtinSections))

	state := &promptGenerationState{}
	c.addPromptImportChunks(state, data)
	c.addMainWorkflowExpressionMappings(state, data)
	state.expressionMappings = filterExpressionsForActivation(state.expressionMappings, data.Jobs, beforeActivationJobs)
	addExperimentExpressionMappings(state, data)
	c.addMainWorkflowPromptChunks(state, data)
	applyWorkflowDispatchFallbacks(state.expressionMappings, data.HasDispatchItemNumber)

	allExpressionMappings := c.generateUnifiedPromptCreationStep(yaml, builtinSections, state.userPromptChunks, state.expressionMappings, data)
	allExpressionMappings = mergeKnownNeedsExpressionMappings(allExpressionMappings, data, preActivationJobCreated)
	c.generatePromptPostProcessingSteps(yaml, state.expressionMappings, allExpressionMappings, data)
}

type promptGenerationState struct {
	userPromptChunks   []string
	expressionMappings []*ExpressionMapping
}

func (c *Compiler) addPromptImportChunks(state *promptGenerationState, data *WorkflowData) {
	if len(data.PromptImports) > 0 {
		c.addOrderedPromptImportChunks(state, data)
		return
	}
	c.addLegacyPromptImportChunks(state, data)
}

func (c *Compiler) addOrderedPromptImportChunks(state *promptGenerationState, data *WorkflowData) {
	compilerYamlPromptLog.Printf("Processing %d ordered prompt import entries", len(data.PromptImports))
	workspaceRoot := ""
	hasImportInputs := len(data.ImportInputs) > 0
	if data.InlinedImports && c.markdownPath != "" {
		workspaceRoot = resolveWorkspaceRoot(c.markdownPath)
	}
	for _, entry := range data.PromptImports {
		if entry.Markdown != "" {
			addMarkdownPromptChunks(state, entry.Markdown, hasImportInputs, data.ImportInputs)
			continue
		}
		if entry.ImportPath != "" {
			c.addPromptImportPathChunk(state, filepath.ToSlash(entry.ImportPath), workspaceRoot)
		}
	}
}

func addMarkdownPromptChunks(state *promptGenerationState, markdown string, substituteInputs bool, importInputs map[string]any) {
	cleaned := removeXMLComments(markdown)
	if substituteInputs {
		cleaned = SubstituteImportInputs(cleaned, importInputs)
	}
	chunks, exprMaps := extractPromptChunksFromMarkdown(cleaned)
	state.userPromptChunks = append(state.userPromptChunks, chunks...)
	state.expressionMappings = append(state.expressionMappings, exprMaps...)
}

func (c *Compiler) addPromptImportPathChunk(state *promptGenerationState, importPath, workspaceRoot string) bool {
	if workspaceRoot == "" {
		state.userPromptChunks = append(state.userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
		return false
	}
	rawContent, err := os.ReadFile(filepath.Join(workspaceRoot, importPath))
	if err != nil {
		compilerYamlPromptLog.Printf("Warning: failed to read import file %s (%v), falling back to runtime-import", importPath, err)
		state.userPromptChunks = append(state.userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
		return false
	}
	importedBody, extractErr := parser.ExtractMarkdownContent(string(rawContent))
	if extractErr != nil {
		importedBody = string(rawContent)
	}
	chunks, exprMaps := extractPromptChunksFromMarkdown(importedBody)
	state.userPromptChunks = append(state.userPromptChunks, chunks...)
	state.expressionMappings = append(state.expressionMappings, exprMaps...)
	return true
}

func (c *Compiler) addLegacyPromptImportChunks(state *promptGenerationState, data *WorkflowData) {
	if data.ImportedMarkdown != "" {
		compilerYamlPromptLog.Printf("Processing imported markdown (%d bytes)", len(data.ImportedMarkdown))
		if len(data.ImportInputs) > 0 {
			compilerYamlPromptLog.Printf("Substituting %d import input values", len(data.ImportInputs))
		}
		before := len(state.userPromptChunks)
		addMarkdownPromptChunks(state, data.ImportedMarkdown, len(data.ImportInputs) > 0, data.ImportInputs)
		compilerYamlPromptLog.Printf("Inlined imported markdown with inputs in %d chunks", len(state.userPromptChunks)-before)
	}
	if len(data.ImportPaths) > 0 {
		c.addLegacyImportPathChunks(state, data)
	}
}

func (c *Compiler) addLegacyImportPathChunks(state *promptGenerationState, data *WorkflowData) {
	if data.InlinedImports && c.markdownPath != "" {
		compilerYamlPromptLog.Printf("Inlining %d imports without inputs at compile time", len(data.ImportPaths))
		workspaceRoot := resolveWorkspaceRoot(c.markdownPath)
		for _, importPath := range data.ImportPaths {
			importPath = filepath.ToSlash(importPath)
			if c.addPromptImportPathChunk(state, importPath, workspaceRoot) {
				compilerYamlPromptLog.Printf("Inlined import without inputs: %s", importPath)
			}
		}
		return
	}
	compilerYamlPromptLog.Printf("Generating runtime-import macros for %d imports without inputs", len(data.ImportPaths))
	for _, importPath := range data.ImportPaths {
		importPath = filepath.ToSlash(importPath)
		state.userPromptChunks = append(state.userPromptChunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
		compilerYamlPromptLog.Printf("Added runtime-import macro for: %s", importPath)
	}
}

func (c *Compiler) addMainWorkflowExpressionMappings(state *promptGenerationState, data *WorkflowData) {
	if c.inlinePrompt || data.InlinedImports || data.MainWorkflowMarkdown == "" {
		return
	}
	compilerYamlPromptLog.Printf("Extracting expressions from main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))
	mainExtractor := NewExpressionExtractor()
	mainExprMappings, err := mainExtractor.ExtractExpressions(data.MainWorkflowMarkdown)
	if err == nil && len(mainExprMappings) > 0 {
		compilerYamlPromptLog.Printf("Extracted %d expressions from main workflow markdown", len(mainExprMappings))
		state.expressionMappings = append(state.expressionMappings, mainExprMappings...)
	}
}

func addExperimentExpressionMappings(state *promptGenerationState, data *WorkflowData) {
	if len(data.Experiments) == 0 {
		return
	}
	experimentMappings := ExperimentExpressionMappings(data.Experiments)
	compilerYamlPromptLog.Printf("Adding %d experiment expression mapping(s)", len(experimentMappings))
	state.expressionMappings = append(state.expressionMappings, experimentMappings...)
}

func (c *Compiler) addMainWorkflowPromptChunks(state *promptGenerationState, data *WorkflowData) {
	if c.inlinePrompt || data.InlinedImports {
		addInlinedMainWorkflowPromptChunks(state, data)
		return
	}
	workflowFilePath := runtimeImportWorkflowPath(c.markdownPath)
	runtimeImportMacro := fmt.Sprintf("{{#runtime-import %s}}", workflowFilePath)
	compilerYamlPromptLog.Printf("Using runtime-import for main workflow markdown: %s", workflowFilePath)
	state.userPromptChunks = append(state.userPromptChunks, runtimeImportMacro)
}

func addInlinedMainWorkflowPromptChunks(state *promptGenerationState, data *WorkflowData) {
	if data.MainWorkflowMarkdown == "" {
		return
	}
	compilerYamlPromptLog.Printf("Inlining main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))
	inlinedMarkdown := removeXMLComments(data.MainWorkflowMarkdown)
	inlinedMarkdown = wrapExpressionsInTemplateConditionals(inlinedMarkdown)
	inlineExtractor := NewExpressionExtractor()
	inlineExprMappings, err := inlineExtractor.ExtractExpressions(inlinedMarkdown)
	if err == nil && len(inlineExprMappings) > 0 {
		inlinedMarkdown = inlineExtractor.ReplaceExpressionsWithEnvVars(inlinedMarkdown)
		state.expressionMappings = append(state.expressionMappings, inlineExprMappings...)
	}
	inlinedChunks := splitContentIntoChunks(inlinedMarkdown)
	state.userPromptChunks = append(state.userPromptChunks, inlinedChunks...)
	compilerYamlPromptLog.Printf("Inlined main workflow markdown in %d chunks", len(inlinedChunks))
}

func runtimeImportWorkflowPath(markdownPath string) string {
	workflowBasename := filepath.Base(markdownPath)
	normalizedPath := filepath.ToSlash(markdownPath)
	githubDirPattern := "/.github/"
	githubIndex := strings.LastIndex(normalizedPath, githubDirPattern)
	if githubIndex != -1 {
		return normalizedPath[githubIndex+1:]
	}
	if strings.HasPrefix(normalizedPath, constants.GithubDir) {
		return normalizedPath
	}
	return workflowBasename
}

func mergeKnownNeedsExpressionMappings(allExpressionMappings []*ExpressionMapping, data *WorkflowData, preActivationJobCreated bool) []*ExpressionMapping {
	knownNeedsExpressions := generateKnownNeedsExpressions(data, preActivationJobCreated)
	if len(knownNeedsExpressions) == 0 {
		return allExpressionMappings
	}
	compilerYamlPromptLog.Printf("Adding %d known needs.* expressions for substitution step only", len(knownNeedsExpressions))
	expressionMap := make(map[string]*ExpressionMapping)
	for _, mapping := range knownNeedsExpressions {
		expressionMap[mapping.EnvVar] = mapping
	}
	for _, mapping := range allExpressionMappings {
		expressionMap[mapping.EnvVar] = mapping
	}
	merged := make([]*ExpressionMapping, 0, len(expressionMap))
	for _, envVar := range sliceutil.SortedKeys(expressionMap) {
		merged = append(merged, expressionMap[envVar])
	}
	return merged
}

func (c *Compiler) generatePromptPostProcessingSteps(yaml *strings.Builder, expressionMappings, allExpressionMappings []*ExpressionMapping, data *WorkflowData) {
	c.generateInterpolationAndTemplateStep(yaml, expressionMappings, data)
	if len(allExpressionMappings) > 0 {
		generatePlaceholderSubstitutionStep(yaml, allExpressionMappings, "      ", data)
	}
	writePromptBashStep(yaml, "Validate prompt placeholders", "validate_prompt_placeholders.sh")
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
