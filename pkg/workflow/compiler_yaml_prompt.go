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
		if currentSize+lineSize > maxChunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))
			currentChunk = []string{line}
			currentSize = lineSize
		} else {
			currentChunk = append(currentChunk, line)
			currentSize += lineSize
		}
	}
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}
	return chunks
}

func (c *Compiler) generatePrompt(yaml *strings.Builder, data *WorkflowData, preActivationJobCreated bool, beforeActivationJobs []string) {
	compilerYamlPromptLog.Printf("Generating prompt for workflow: %s (markdown size: %d bytes)", data.Name, len(data.MarkdownContent))
	builtinSections := c.collectPromptSections(data)
	compilerYamlPromptLog.Printf("Collected %d built-in prompt sections", len(builtinSections))

	userPromptChunks, expressionMappings := buildUserPromptChunks(c, data)
	mainChunks, mainMappings := buildMainMarkdownChunks(c, data)
	userPromptChunks = append(userPromptChunks, mainChunks...)
	expressionMappings = append(expressionMappings, mainMappings...)
	expressionMappings = filterExpressionsForActivation(expressionMappings, data.Jobs, beforeActivationJobs)
	if len(data.Experiments) > 0 {
		experimentMappings := ExperimentExpressionMappings(data.Experiments)
		compilerYamlPromptLog.Printf("Adding %d experiment expression mapping(s)", len(experimentMappings))
		expressionMappings = append(expressionMappings, experimentMappings...)
	}
	applyWorkflowDispatchFallbacks(expressionMappings, data.HasDispatchItemNumber)
	allExpressionMappings := c.generateUnifiedPromptCreationStep(yaml, builtinSections, userPromptChunks, expressionMappings, data)
	allExpressionMappings = mergeKnownNeedsExpressions(allExpressionMappings, generateKnownNeedsExpressions(data, preActivationJobCreated))
	c.generateInterpolationAndTemplateStep(yaml, expressionMappings, data)
	if len(allExpressionMappings) > 0 {
		generatePlaceholderSubstitutionStep(yaml, allExpressionMappings, "      ", data)
	}
	writePromptBashStep(yaml, "Validate prompt placeholders", "validate_prompt_placeholders.sh")
	writePromptBashStep(yaml, "Print prompt", "print_prompt_summary.sh")
}

func buildUserPromptChunks(c *Compiler, data *WorkflowData) ([]string, []*ExpressionMapping) {
	if len(data.PromptImports) > 0 {
		return buildOrderedPromptImportChunks(c, data)
	}
	chunks, mappings := buildLegacyPromptImportChunks(c, data)
	if !c.inlinePrompt && !data.InlinedImports && data.MainWorkflowMarkdown != "" {
		compilerYamlPromptLog.Printf("Extracting expressions from main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))
		extractor := NewExpressionExtractor()
		mainMappings, err := extractor.ExtractExpressions(data.MainWorkflowMarkdown)
		if err == nil && len(mainMappings) > 0 {
			compilerYamlPromptLog.Printf("Extracted %d expressions from main workflow markdown", len(mainMappings))
			mappings = append(mappings, mainMappings...)
		}
	}
	return chunks, mappings
}

func buildOrderedPromptImportChunks(c *Compiler, data *WorkflowData) ([]string, []*ExpressionMapping) {
	compilerYamlPromptLog.Printf("Processing %d ordered prompt import entries", len(data.PromptImports))
	workspaceRoot := ""
	if data.InlinedImports && c.markdownPath != "" {
		workspaceRoot = resolveWorkspaceRoot(c.markdownPath)
	}
	var chunks []string
	var mappings []*ExpressionMapping
	for _, entry := range data.PromptImports {
		entryChunks, entryMappings := buildPromptImportEntryChunks(data, workspaceRoot, entry)
		chunks = append(chunks, entryChunks...)
		mappings = append(mappings, entryMappings...)
	}
	return chunks, mappings
}

func buildPromptImportEntryChunks(data *WorkflowData, workspaceRoot string, entry parser.PromptImportEntry) ([]string, []*ExpressionMapping) {
	if entry.Markdown != "" {
		cleaned := removeXMLComments(entry.Markdown)
		if len(data.ImportInputs) > 0 {
			cleaned = SubstituteImportInputs(cleaned, data.ImportInputs)
		}
		return extractPromptChunksFromMarkdown(cleaned)
	}
	if entry.ImportPath == "" {
		return nil, nil
	}
	importPath := filepath.ToSlash(entry.ImportPath)
	if workspaceRoot != "" {
		return loadPromptImportChunks(workspaceRoot, importPath)
	}
	return []string{fmt.Sprintf("{{#runtime-import %s}}", importPath)}, nil
}

func buildLegacyPromptImportChunks(c *Compiler, data *WorkflowData) ([]string, []*ExpressionMapping) {
	chunks, mappings := buildLegacyImportedMarkdownChunks(data)
	if len(data.ImportPaths) == 0 {
		return chunks, mappings
	}
	if data.InlinedImports && c.markdownPath != "" {
		return appendPromptImportFileChunks(chunks, mappings, resolveWorkspaceRoot(c.markdownPath), data.ImportPaths)
	}
	compilerYamlPromptLog.Printf("Generating runtime-import macros for %d imports without inputs", len(data.ImportPaths))
	for _, importPath := range data.ImportPaths {
		importPath = filepath.ToSlash(importPath)
		chunks = append(chunks, fmt.Sprintf("{{#runtime-import %s}}", importPath))
		compilerYamlPromptLog.Printf("Added runtime-import macro for: %s", importPath)
	}
	return chunks, mappings
}

func buildLegacyImportedMarkdownChunks(data *WorkflowData) ([]string, []*ExpressionMapping) {
	if data.ImportedMarkdown == "" {
		return nil, nil
	}
	compilerYamlPromptLog.Printf("Processing imported markdown (%d bytes)", len(data.ImportedMarkdown))
	cleaned := removeXMLComments(data.ImportedMarkdown)
	if len(data.ImportInputs) > 0 {
		compilerYamlPromptLog.Printf("Substituting %d import input values", len(data.ImportInputs))
		cleaned = SubstituteImportInputs(cleaned, data.ImportInputs)
	}
	chunks, mappings := extractPromptChunksFromMarkdown(cleaned)
	compilerYamlPromptLog.Printf("Inlined imported markdown with inputs in %d chunks", len(chunks))
	return chunks, mappings
}

func appendPromptImportFileChunks(chunks []string, mappings []*ExpressionMapping, workspaceRoot string, importPaths []string) ([]string, []*ExpressionMapping) {
	compilerYamlPromptLog.Printf("Inlining %d imports without inputs at compile time", len(importPaths))
	for _, importPath := range importPaths {
		importPath = filepath.ToSlash(importPath)
		entryChunks, entryMappings := loadPromptImportChunks(workspaceRoot, importPath)
		chunks = append(chunks, entryChunks...)
		mappings = append(mappings, entryMappings...)
	}
	return chunks, mappings
}

func loadPromptImportChunks(workspaceRoot, importPath string) ([]string, []*ExpressionMapping) {
	rawContent, err := os.ReadFile(filepath.Join(workspaceRoot, importPath))
	if err != nil {
		compilerYamlPromptLog.Printf("Warning: failed to read import file %s (%v), falling back to runtime-import", importPath, err)
		return []string{fmt.Sprintf("{{#runtime-import %s}}", importPath)}, nil
	}
	importedBody, extractErr := parser.ExtractMarkdownContent(string(rawContent))
	if extractErr != nil {
		importedBody = string(rawContent)
	}
	compilerYamlPromptLog.Printf("Inlined import without inputs: %s", importPath)
	return extractPromptChunksFromMarkdown(importedBody)
}

func buildMainMarkdownChunks(c *Compiler, data *WorkflowData) ([]string, []*ExpressionMapping) {
	if c.inlinePrompt || data.InlinedImports {
		return buildInlineMainMarkdownChunks(data)
	}
	workflowFilePath := resolvePromptRuntimeImportPath(c.markdownPath)
	compilerYamlPromptLog.Printf("Using runtime-import for main workflow markdown: %s", workflowFilePath)
	return []string{fmt.Sprintf("{{#runtime-import %s}}", workflowFilePath)}, nil
}

func buildInlineMainMarkdownChunks(data *WorkflowData) ([]string, []*ExpressionMapping) {
	if data.MainWorkflowMarkdown == "" {
		return nil, nil
	}
	compilerYamlPromptLog.Printf("Inlining main workflow markdown (%d bytes)", len(data.MainWorkflowMarkdown))
	inlinedMarkdown := wrapExpressionsInTemplateConditionals(removeXMLComments(data.MainWorkflowMarkdown))
	extractor := NewExpressionExtractor()
	exprMappings, err := extractor.ExtractExpressions(inlinedMarkdown)
	if err == nil && len(exprMappings) > 0 {
		inlinedMarkdown = extractor.ReplaceExpressionsWithEnvVars(inlinedMarkdown)
	}
	chunks := splitContentIntoChunks(inlinedMarkdown)
	compilerYamlPromptLog.Printf("Inlined main workflow markdown in %d chunks", len(chunks))
	if err != nil || len(exprMappings) == 0 {
		return chunks, nil
	}
	return chunks, exprMappings
}

func resolvePromptRuntimeImportPath(markdownPath string) string {
	workflowBasename := filepath.Base(markdownPath)
	normalizedPath := filepath.ToSlash(markdownPath)
	githubIndex := strings.LastIndex(normalizedPath, "/.github/")
	if githubIndex != -1 {
		return normalizedPath[githubIndex+1:]
	}
	if strings.HasPrefix(normalizedPath, constants.GithubDir) {
		return normalizedPath
	}
	return workflowBasename
}

func mergeKnownNeedsExpressions(base []*ExpressionMapping, knownNeeds []*ExpressionMapping) []*ExpressionMapping {
	if len(knownNeeds) == 0 {
		return base
	}
	compilerYamlPromptLog.Printf("Adding %d known needs.* expressions for substitution step only", len(knownNeeds))
	expressionMap := make(map[string]*ExpressionMapping, len(base)+len(knownNeeds))
	for _, mapping := range knownNeeds {
		expressionMap[mapping.EnvVar] = mapping
	}
	for _, mapping := range base {
		expressionMap[mapping.EnvVar] = mapping
	}
	merged := make([]*ExpressionMapping, 0, len(expressionMap))
	for _, envVar := range sliceutil.SortedKeys(expressionMap) {
		merged = append(merged, expressionMap[envVar])
	}
	return merged
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
		return filepath.FromSlash(before)
	}
	if strings.HasPrefix(normalized, constants.GithubDir) {
		return "."
	}
	return filepath.Dir(markdownPath)
}
