// This file provides command-line interface functionality for gh-aw.
// This file (logs_orchestrator_render.go) contains output-rendering helpers for the
// logs orchestrator: transforming []ProcessedRun into console, JSON, markdown,
// TSV, or "pretty" cross-run audit output.

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
)

// renderLogsOutput finalizes processedRuns and renders them in the appropriate output
// format: JSON, console metrics table, or cross-run audit report (pretty/markdown).
// continuation is optional and only set when a timeout was reached during a paginated download.
func renderLogsOutput(processedRuns []ProcessedRun, opts renderLogsOutputOptions) error {
	// Update MissingToolCount, MissingDataCount, and NoopCount in runs
	for i := range processedRuns {
		processedRuns[i].Run.MissingToolCount = len(processedRuns[i].MissingTools)
		processedRuns[i].Run.MissingDataCount = len(processedRuns[i].MissingData)
		processedRuns[i].Run.NoopCount = len(processedRuns[i].Noops)
	}

	// Build structured logs data
	logsOrchestratorLog.Printf("Building logs data from %d processed runs (continuation=%t)", len(processedRuns), opts.continuation != nil)
	logsData := buildLogsData(processedRuns, opts.outputDir, opts.continuation)

	// When only the usage artifact was downloaded, add a hint so consumers know how
	// to fetch additional artifact sets (agent logs, firewall data, etc.).
	if isUsageOnlyArtifactFilter(opts.artifactFilter) {
		logsData.Message = usageOnlyArtifactHintMessage()
	}

	// Write summary file if requested (default behavior unless disabled with empty string)
	if opts.summaryFile != "" {
		summaryPath := filepath.Join(opts.outputDir, opts.summaryFile)
		if err := writeSummaryFile(summaryPath, logsData, opts.verbose); err != nil {
			return fmt.Errorf("failed to write summary file: %w", err)
		}
	}

	// Train drain3 weights if requested.
	if opts.train {
		if err := TrainDrain3Weights(processedRuns, opts.outputDir, opts.verbose); err != nil {
			return fmt.Errorf("log pattern training: %w", err)
		}
	}

	// Render output based on format preference.
	switch opts.format {
	case "tsv":
		if opts.verbose {
			renderLogsTSVVerbose(logsData)
		} else {
			renderLogsTSV(logsData)
		}
		renderLogsArtifactHint(os.Stderr, logsData.Message)
		return nil

	case "markdown", "pretty":
		inputs := make([]crossRunInput, 0, len(processedRuns))
		for _, pr := range processedRuns {
			inputs = append(inputs, crossRunInput{
				RunID:            pr.Run.DatabaseID,
				WorkflowName:     pr.Run.WorkflowName,
				Conclusion:       pr.Run.Conclusion,
				Duration:         pr.Run.Duration,
				FirewallAnalysis: pr.FirewallAnalysis,
				Metrics: LogMetrics{
					TokenUsage: pr.Run.TokenUsage,
					Turns:      pr.Run.Turns,
				},
				MCPToolUsage: pr.MCPToolUsage,
				MCPFailures:  pr.MCPFailures,
				ErrorCount:   pr.Run.ErrorCount,
			})
		}
		report := buildCrossRunAuditReport(inputs)
		if opts.jsonOutput {
			return renderCrossRunReportJSON(report)
		}
		if opts.format == "pretty" {
			renderCrossRunReportPretty(report)
			renderLogsArtifactHint(os.Stderr, logsData.Message)
			return nil
		}
		if opts.reportFile != "" {
			if err := os.MkdirAll(filepath.Dir(opts.reportFile), constants.DirPermPublic); err != nil {
				return fmt.Errorf("failed to create report file directory: %w", err)
			}
			f, err := os.Create(opts.reportFile)
			if err != nil {
				return fmt.Errorf("failed to create report file: %w", err)
			}
			if err := func() (retErr error) {
				defer func() {
					if cerr := f.Close(); cerr != nil && retErr == nil {
						retErr = cerr
					}
				}()
				renderCrossRunReportMarkdownToWriter(f, report)
				return nil
			}(); err != nil {
				return fmt.Errorf("failed to write report file: %w", err)
			}
		} else {
			renderCrossRunReportMarkdown(report)
		}
		renderLogsArtifactHint(os.Stderr, logsData.Message)
		return nil

	case "console":
		// Explicit console format: decorated tables for human reading
		if opts.jsonOutput {
			if err := renderLogsJSON(logsData, opts.verbose); err != nil {
				return fmt.Errorf("failed to render JSON output: %w", err)
			}
		} else {
			renderLogsConsole(logsData)
			displayAggregatedGatewayMetrics(processedRuns, opts.outputDir, opts.verbose)
			displayUnifiedTimeline(processedRuns, opts.verbose)
			if opts.toolGraph {
				generateToolGraph(processedRuns, opts.verbose)
			}
			renderLogsArtifactHint(os.Stderr, logsData.Message)
		}
		return nil
	}

	// Default: compact format optimised for agentic consumption
	if opts.jsonOutput {
		if err := renderLogsJSON(logsData, opts.verbose); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	} else {
		if opts.verbose {
			renderLogsCompactVerbose(logsData)
		} else {
			renderLogsCompact(logsData)
		}
	}

	return nil
}

// renderLogsArtifactHint writes a [hint] line to w when message is non-empty.
func renderLogsArtifactHint(w *os.File, message string) {
	if message == "" {
		return
	}
	fmt.Fprintf(w, "[hint] %s\n", message)
}
