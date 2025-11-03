package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var taintAnalysisLog = logger.New("cli:taint_analysis")

// TaintSource represents a source of tainted data
type TaintSource struct {
	Type         string // "agentic_workflow", "web_search", "web_fetch", "github_issue", "github_pr", "external_api"
	Name         string // Name or identifier of the source
	WorkflowFile string // File path of the workflow containing this source
	Severity     string // "high", "medium", "low"
}

// TaintSink represents a destination where tainted data may flow
type TaintSink struct {
	Type         string // "file_write", "github_api", "external_api", "command_execution"
	Name         string // Name or identifier of the sink
	WorkflowFile string // File path of the workflow containing this sink
	Severity     string // "high", "medium", "low"
}

// TaintPath represents a flow of tainted data from source to sink
type TaintPath struct {
	Source       TaintSource
	Sink         TaintSink
	Intermediary []string // List of intermediate steps in the flow
	IsUnsafe     bool     // Whether this path is considered unsafe
	Reason       string   // Reason why this path is unsafe
}

// TaintAnalysisResult holds the results of taint analysis
type TaintAnalysisResult struct {
	Sources       []TaintSource
	Sinks         []TaintSink
	Paths         []TaintPath
	UnsafePaths   []TaintPath
	WorkflowCount int
}

// WorkflowInfo contains parsed information about a workflow
type WorkflowInfo struct {
	FilePath     string
	Name         string
	Frontmatter  map[string]any
	Markdown     string
	IsAgentic    bool // Whether this workflow contains an agentic engine
	HasSafeOuts  bool // Whether this workflow has safe-outputs configured
	Tools        []string
	MCPServers   []string
	Engine       string
}

// performTaintAnalysis performs the main taint flow analysis
func performTaintAnalysis(workflowsDir string, verbose bool) (*TaintAnalysisResult, error) {
	taintAnalysisLog.Printf("Performing taint analysis on directory: %s", workflowsDir)

	// Load all workflows
	workflows, err := loadWorkflows(workflowsDir, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflows: %w", err)
	}

	if verbose {
		taintAnalysisLog.Printf("Loaded %d workflow files", len(workflows))
	}

	result := &TaintAnalysisResult{
		Sources:       []TaintSource{},
		Sinks:         []TaintSink{},
		Paths:         []TaintPath{},
		UnsafePaths:   []TaintPath{},
		WorkflowCount: len(workflows),
	}

	// Identify taint sources
	for _, wf := range workflows {
		sources := identifyTaintSources(wf, verbose)
		result.Sources = append(result.Sources, sources...)
	}

	// Identify taint sinks
	for _, wf := range workflows {
		sinks := identifyTaintSinks(wf, verbose)
		result.Sinks = append(result.Sinks, sinks...)
	}

	// Analyze taint flows
	result.Paths = analyzeTaintFlows(workflows, result.Sources, result.Sinks, verbose)

	// Filter unsafe paths
	for _, path := range result.Paths {
		if path.IsUnsafe {
			result.UnsafePaths = append(result.UnsafePaths, path)
		}
	}

	if verbose {
		taintAnalysisLog.Printf("Analysis complete: %d sources, %d sinks, %d paths (%d unsafe)",
			len(result.Sources), len(result.Sinks), len(result.Paths), len(result.UnsafePaths))
	}

	return result, nil
}

// loadWorkflows loads and parses all workflow markdown files
func loadWorkflows(workflowsDir string, verbose bool) ([]WorkflowInfo, error) {
	var workflows []WorkflowInfo

	// Find all .md files in the workflows directory
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(workflowsDir, entry.Name())
		wf, err := parseWorkflowFile(filePath, verbose)
		if err != nil {
			taintAnalysisLog.Printf("Warning: failed to parse workflow %s: %v", entry.Name(), err)
			continue
		}

		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// parseWorkflowFile parses a single workflow file
func parseWorkflowFile(filePath string, verbose bool) (WorkflowInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return WorkflowInfo{}, fmt.Errorf("failed to read file: %w", err)
	}

	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return WorkflowInfo{}, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	wf := WorkflowInfo{
		FilePath:    filePath,
		Name:        filepath.Base(filePath),
		Frontmatter: result.Frontmatter,
		Markdown:    result.Markdown,
	}

	// Determine if workflow is agentic (has an engine)
	if engine, ok := result.Frontmatter["engine"]; ok {
		wf.IsAgentic = true
		switch v := engine.(type) {
		case string:
			wf.Engine = v
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				wf.Engine = id
			}
		}
	}

	// Check for safe-outputs
	if _, ok := result.Frontmatter["safe-outputs"]; ok {
		wf.HasSafeOuts = true
	}

	// Extract tools
	if tools, ok := result.Frontmatter["tools"].(map[string]any); ok {
		for toolName := range tools {
			wf.Tools = append(wf.Tools, toolName)
		}
	}

	// Extract MCP servers
	if mcpServers, ok := result.Frontmatter["mcp-servers"].(map[string]any); ok {
		for serverName := range mcpServers {
			wf.MCPServers = append(wf.MCPServers, serverName)
		}
	}

	if verbose {
		taintAnalysisLog.Printf("Parsed workflow: %s (agentic=%v, engine=%s, tools=%d, mcp=%d)",
			wf.Name, wf.IsAgentic, wf.Engine, len(wf.Tools), len(wf.MCPServers))
	}

	return wf, nil
}

// identifyTaintSources identifies taint sources in a workflow
func identifyTaintSources(wf WorkflowInfo, verbose bool) []TaintSource {
	var sources []TaintSource

	// All agentic workflows are taint sources
	if wf.IsAgentic {
		sources = append(sources, TaintSource{
			Type:         "agentic_workflow",
			Name:         wf.Name,
			WorkflowFile: wf.FilePath,
			Severity:     "high",
		})
	}

	// Check for web-search tool (third-party input)
	for _, tool := range wf.Tools {
		if tool == "web-search" {
			sources = append(sources, TaintSource{
				Type:         "web_search",
				Name:         wf.Name + ":web-search",
				WorkflowFile: wf.FilePath,
				Severity:     "high",
			})
		}
		if tool == "web-fetch" {
			sources = append(sources, TaintSource{
				Type:         "web_fetch",
				Name:         wf.Name + ":web-fetch",
				WorkflowFile: wf.FilePath,
				Severity:     "medium",
			})
		}
	}

	// Check for GitHub API access (potential external data source)
	for _, tool := range wf.Tools {
		if tool == "github" {
			sources = append(sources, TaintSource{
				Type:         "github_api",
				Name:         wf.Name + ":github",
				WorkflowFile: wf.FilePath,
				Severity:     "medium",
			})
		}
	}

	// Check markdown content for potential taint sources
	markdownLower := strings.ToLower(wf.Markdown)
	if strings.Contains(markdownLower, "issue") || strings.Contains(markdownLower, "pull request") {
		// Workflow processes issues/PRs which may contain untrusted user input
		sources = append(sources, TaintSource{
			Type:         "user_input",
			Name:         wf.Name + ":user-content",
			WorkflowFile: wf.FilePath,
			Severity:     "high",
		})
	}

	if verbose && len(sources) > 0 {
		taintAnalysisLog.Printf("Found %d taint sources in %s", len(sources), wf.Name)
	}

	return sources
}

// identifyTaintSinks identifies taint sinks in a workflow
func identifyTaintSinks(wf WorkflowInfo, verbose bool) []TaintSink {
	var sinks []TaintSink

	// Check for edit tool (file writes)
	for _, tool := range wf.Tools {
		if tool == "edit" {
			sinks = append(sinks, TaintSink{
				Type:         "file_write",
				Name:         wf.Name + ":edit",
				WorkflowFile: wf.FilePath,
				Severity:     "high",
			})
		}
		if tool == "bash" {
			sinks = append(sinks, TaintSink{
				Type:         "command_execution",
				Name:         wf.Name + ":bash",
				WorkflowFile: wf.FilePath,
				Severity:     "critical",
			})
		}
	}

	// Check for safe-outputs (GitHub API writes)
	if wf.HasSafeOuts {
		sinks = append(sinks, TaintSink{
			Type:         "github_api",
			Name:         wf.Name + ":safe-outputs",
			WorkflowFile: wf.FilePath,
			Severity:     "medium",
		})
	}

	// Check for external MCP servers
	for _, mcpServer := range wf.MCPServers {
		sinks = append(sinks, TaintSink{
			Type:         "external_api",
			Name:         wf.Name + ":mcp:" + mcpServer,
			WorkflowFile: wf.FilePath,
			Severity:     "medium",
		})
	}

	if verbose && len(sinks) > 0 {
		taintAnalysisLog.Printf("Found %d taint sinks in %s", len(sinks), wf.Name)
	}

	return sinks
}

// analyzeTaintFlows analyzes data flows from sources to sinks
func analyzeTaintFlows(workflows []WorkflowInfo, sources []TaintSource, sinks []TaintSink, verbose bool) []TaintPath {
	var paths []TaintPath

	// For each source, check if it flows to any sink
	for _, source := range sources {
		for _, sink := range sinks {
			// Check if source and sink are in the same workflow
			if source.WorkflowFile == sink.WorkflowFile {
				// Direct flow within the same workflow
				path := TaintPath{
					Source:       source,
					Sink:         sink,
					Intermediary: []string{},
				}

				// Determine if path is unsafe
				path.IsUnsafe, path.Reason = evaluatePathSafety(source, sink)

				paths = append(paths, path)
			}
		}
	}

	if verbose {
		taintAnalysisLog.Printf("Identified %d taint flow paths", len(paths))
	}

	return paths
}

// evaluatePathSafety determines if a taint path is unsafe and provides a reason
func evaluatePathSafety(source TaintSource, sink TaintSink) (bool, string) {
	// High-risk combinations
	if source.Type == "web_search" && sink.Type == "command_execution" {
		return true, "Web search results flowing to command execution can lead to code injection"
	}

	if source.Type == "user_input" && sink.Type == "command_execution" {
		return true, "User input flowing to command execution can lead to code injection"
	}

	if source.Type == "agentic_workflow" && sink.Type == "command_execution" {
		return true, "AI-generated content flowing to command execution without sanitization"
	}

	if source.Type == "web_fetch" && sink.Type == "file_write" {
		return true, "External web content flowing to file writes without validation"
	}

	if source.Type == "user_input" && sink.Type == "file_write" {
		return true, "User input flowing to file writes without sanitization"
	}

	if source.Type == "agentic_workflow" && sink.Type == "file_write" {
		return true, "AI-generated content flowing to file writes - verify content validation"
	}

	// Medium-risk combinations
	if source.Severity == "high" && sink.Type == "github_api" {
		return true, "Tainted data flowing to GitHub API without safe-outputs validation"
	}

	if source.Type == "web_search" && sink.Type == "external_api" {
		return true, "Web search results flowing to external API calls"
	}

	// Low-risk or mitigated paths
	if sink.Type == "github_api" && strings.Contains(sink.Name, "safe-outputs") {
		return false, "" // safe-outputs provide validation
	}

	return false, ""
}
