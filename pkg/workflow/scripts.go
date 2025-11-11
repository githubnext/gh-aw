package workflow

import (
	_ "embed"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var scriptsLog = logger.New("workflow:scripts")

// Source scripts that may contain local requires
//
//go:embed js/collect_ndjson_output.cjs
var collectJSONLOutputScriptSource string

//go:embed js/compute_text.cjs
var computeTextScriptSource string

//go:embed js/sanitize_output.cjs
var sanitizeOutputScriptSource string

//go:embed js/create_issue.cjs
var createIssueScriptSource string

//go:embed js/add_labels.cjs
var addLabelsScriptSource string

//go:embed js/commit_status.cjs
var commitStatusScriptSource string

//go:embed js/create_discussion.cjs
var createDiscussionScriptSource string

//go:embed js/update_issue.cjs
var updateIssueScriptSource string

//go:embed js/create_code_scanning_alert.cjs
var createCodeScanningAlertScriptSource string

//go:embed js/create_pr_review_comment.cjs
var createPRReviewCommentScriptSource string

//go:embed js/add_comment.cjs
var addCommentScriptSource string

//go:embed js/upload_assets.cjs
var uploadAssetsScriptSource string

//go:embed js/parse_firewall_logs.cjs
var parseFirewallLogsScriptSource string

//go:embed js/push_to_pull_request_branch.cjs
var pushToPullRequestBranchScriptSource string

//go:embed js/create_pull_request.cjs
var createPullRequestScriptSource string

// Log parser source scripts
//
//go:embed js/parse_claude_log.cjs
var parseClaudeLogScriptSource string

//go:embed js/parse_codex_log.cjs
var parseCodexLogScriptSource string

//go:embed js/parse_copilot_log.cjs
var parseCopilotLogScriptSource string

// Bundled scripts (lazily bundled on-demand and cached)
var (
	collectJSONLOutputScript     string
	collectJSONLOutputScriptOnce sync.Once

	computeTextScript     string
	computeTextScriptOnce sync.Once

	sanitizeOutputScript     string
	sanitizeOutputScriptOnce sync.Once

	createIssueScript     string
	createIssueScriptOnce sync.Once

	addLabelsScript     string
	addLabelsScriptOnce sync.Once

	commitStatusScript     string
	commitStatusScriptOnce sync.Once

	createDiscussionScript     string
	createDiscussionScriptOnce sync.Once

	updateIssueScript     string
	updateIssueScriptOnce sync.Once

	createCodeScanningAlertScript     string
	createCodeScanningAlertScriptOnce sync.Once

	createPRReviewCommentScript     string
	createPRReviewCommentScriptOnce sync.Once

	addCommentScript     string
	addCommentScriptOnce sync.Once

	uploadAssetsScript     string
	uploadAssetsScriptOnce sync.Once

	parseFirewallLogsScript     string
	parseFirewallLogsScriptOnce sync.Once

	pushToPullRequestBranchScript     string
	pushToPullRequestBranchScriptOnce sync.Once

	createPullRequestScript     string
	createPullRequestScriptOnce sync.Once

	interpolatePromptBundled     string
	interpolatePromptBundledOnce sync.Once

	parseClaudeLogBundled     string
	parseClaudeLogBundledOnce sync.Once

	parseCodexLogBundled     string
	parseCodexLogBundledOnce sync.Once

	parseCopilotLogBundled     string
	parseCopilotLogBundledOnce sync.Once
)

// getCollectJSONLOutputScript returns the bundled collect_ndjson_output script
// Bundling is performed on first access and cached for subsequent calls
func getCollectJSONLOutputScript() string {
	collectJSONLOutputScriptOnce.Do(func() {
		scriptsLog.Print("Bundling collect_ndjson_output script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(collectJSONLOutputScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for collect_ndjson_output, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			collectJSONLOutputScript = collectJSONLOutputScriptSource
		} else {
			scriptsLog.Printf("Successfully bundled collect_ndjson_output script: %d bytes", len(bundled))
			collectJSONLOutputScript = bundled
		}
	})
	return collectJSONLOutputScript
}

// getComputeTextScript returns the bundled compute_text script
// Bundling is performed on first access and cached for subsequent calls
func getComputeTextScript() string {
	computeTextScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(computeTextScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for compute_text, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			computeTextScript = computeTextScriptSource
		} else {
			computeTextScript = bundled
		}
	})
	return computeTextScript
}

// getSanitizeOutputScript returns the bundled sanitize_output script
// Bundling is performed on first access and cached for subsequent calls
func getSanitizeOutputScript() string {
	sanitizeOutputScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(sanitizeOutputScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for sanitize_output, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			sanitizeOutputScript = sanitizeOutputScriptSource
		} else {
			sanitizeOutputScript = bundled
		}
	})
	return sanitizeOutputScript
}

// getCreateIssueScript returns the bundled create_issue script
// Bundling is performed on first access and cached for subsequent calls
func getCreateIssueScript() string {
	createIssueScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createIssueScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for create_issue, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			createIssueScript = createIssueScriptSource
		} else {
			createIssueScript = bundled
		}
	})
	return createIssueScript
}

// getAddLabelsScript returns the bundled add_labels script
// Bundling is performed on first access and cached for subsequent calls
func getAddLabelsScript() string {
	addLabelsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(addLabelsScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for add_labels, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			addLabelsScript = addLabelsScriptSource
		} else {
			addLabelsScript = bundled
		}
	})
	return addLabelsScript
}

// getCommitStatusScript returns the bundled commit_status script
// Bundling is performed on first access and cached for subsequent calls
func getCommitStatusScript() string {
	commitStatusScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(commitStatusScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for commit_status, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			commitStatusScript = commitStatusScriptSource
		} else {
			commitStatusScript = bundled
		}
	})
	return commitStatusScript
}

// getParseFirewallLogsScript returns the bundled parse_firewall_logs script
// Bundling is performed on first access and cached for subsequent calls
func getParseFirewallLogsScript() string {
	parseFirewallLogsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(parseFirewallLogsScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for parse_firewall_logs, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			parseFirewallLogsScript = parseFirewallLogsScriptSource
		} else {
			parseFirewallLogsScript = bundled
		}
	})
	return parseFirewallLogsScript
}

// getCreateDiscussionScript returns the bundled create_discussion script
// Bundling is performed on first access and cached for subsequent calls
func getCreateDiscussionScript() string {
	createDiscussionScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createDiscussionScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for create_discussion, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			createDiscussionScript = createDiscussionScriptSource
		} else {
			createDiscussionScript = bundled
		}
	})
	return createDiscussionScript
}

// getUpdateIssueScript returns the bundled update_issue script
// Bundling is performed on first access and cached for subsequent calls
func getUpdateIssueScript() string {
	updateIssueScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(updateIssueScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for update_issue, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			updateIssueScript = updateIssueScriptSource
		} else {
			updateIssueScript = bundled
		}
	})
	return updateIssueScript
}

// getCreateCodeScanningAlertScript returns the bundled create_code_scanning_alert script
// Bundling is performed on first access and cached for subsequent calls
func getCreateCodeScanningAlertScript() string {
	createCodeScanningAlertScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createCodeScanningAlertScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for create_code_scanning_alert, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			createCodeScanningAlertScript = createCodeScanningAlertScriptSource
		} else {
			createCodeScanningAlertScript = bundled
		}
	})
	return createCodeScanningAlertScript
}

// getCreatePRReviewCommentScript returns the bundled create_pr_review_comment script
// Bundling is performed on first access and cached for subsequent calls
func getCreatePRReviewCommentScript() string {
	createPRReviewCommentScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createPRReviewCommentScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for create_pr_review_comment, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			createPRReviewCommentScript = createPRReviewCommentScriptSource
		} else {
			createPRReviewCommentScript = bundled
		}
	})
	return createPRReviewCommentScript
}

// getAddCommentScript returns the bundled add_comment script
// Bundling is performed on first access and cached for subsequent calls
func getAddCommentScript() string {
	addCommentScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(addCommentScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for add_comment, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			addCommentScript = addCommentScriptSource
		} else {
			addCommentScript = bundled
		}
	})
	return addCommentScript
}

// getUploadAssetsScript returns the bundled upload_assets script
// Bundling is performed on first access and cached for subsequent calls
func getUploadAssetsScript() string {
	uploadAssetsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(uploadAssetsScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for upload_assets, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			uploadAssetsScript = uploadAssetsScriptSource
		} else {
			uploadAssetsScript = bundled
		}
	})
	return uploadAssetsScript
}

// getPushToPullRequestBranchScript returns the bundled push_to_pull_request_branch script
// Bundling is performed on first access and cached for subsequent calls
func getPushToPullRequestBranchScript() string {
	pushToPullRequestBranchScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(pushToPullRequestBranchScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for push_to_pull_request_branch, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			pushToPullRequestBranchScript = pushToPullRequestBranchScriptSource
		} else {
			pushToPullRequestBranchScript = bundled
		}
	})
	return pushToPullRequestBranchScript
}

// getCreatePullRequestScript returns the bundled create_pull_request script
// Bundling is performed on first access and cached for subsequent calls
func getCreatePullRequestScript() string {
	createPullRequestScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createPullRequestScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for create_pull_request, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			createPullRequestScript = createPullRequestScriptSource
		} else {
			createPullRequestScript = bundled
		}
	})
	return createPullRequestScript
}

// getInterpolatePromptScript returns the bundled interpolate_prompt script
// Bundling is performed on first access and cached for subsequent calls
// This bundles is_truthy.cjs inline to avoid require() issues in GitHub Actions
func getInterpolatePromptScript() string {
	interpolatePromptBundledOnce.Do(func() {
		scriptsLog.Print("Bundling interpolate_prompt script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(interpolatePromptScript, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for interpolate_prompt, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			interpolatePromptBundled = interpolatePromptScript
		} else {
			scriptsLog.Printf("Successfully bundled interpolate_prompt script: %d bytes", len(bundled))
			interpolatePromptBundled = bundled
		}
	})
	return interpolatePromptBundled
}

// getParseClaudeLogScript returns the bundled parse_claude_log script
// Bundling is performed on first access and cached for subsequent calls
// This bundles log_parser_bootstrap.cjs inline to avoid require() issues in GitHub Actions
func getParseClaudeLogScript() string {
	parseClaudeLogBundledOnce.Do(func() {
		scriptsLog.Print("Bundling parse_claude_log script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(parseClaudeLogScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for parse_claude_log, using source as-is: %v", err)
			parseClaudeLogBundled = parseClaudeLogScriptSource
		} else {
			scriptsLog.Printf("Successfully bundled parse_claude_log script: %d bytes", len(bundled))
			parseClaudeLogBundled = bundled
		}
	})
	return parseClaudeLogBundled
}

// getParseCodexLogScript returns the bundled parse_codex_log script
// Bundling is performed on first access and cached for subsequent calls
// This bundles log_parser_bootstrap.cjs inline to avoid require() issues in GitHub Actions
func getParseCodexLogScript() string {
	parseCodexLogBundledOnce.Do(func() {
		scriptsLog.Print("Bundling parse_codex_log script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(parseCodexLogScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for parse_codex_log, using source as-is: %v", err)
			parseCodexLogBundled = parseCodexLogScriptSource
		} else {
			scriptsLog.Printf("Successfully bundled parse_codex_log script: %d bytes", len(bundled))
			parseCodexLogBundled = bundled
		}
	})
	return parseCodexLogBundled
}

// getParseCopilotLogScript returns the bundled parse_copilot_log script
// Bundling is performed on first access and cached for subsequent calls
// This bundles log_parser_bootstrap.cjs inline to avoid require() issues in GitHub Actions
func getParseCopilotLogScript() string {
	parseCopilotLogBundledOnce.Do(func() {
		scriptsLog.Print("Bundling parse_copilot_log script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(parseCopilotLogScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for parse_copilot_log, using source as-is: %v", err)
			parseCopilotLogBundled = parseCopilotLogScriptSource
		} else {
			scriptsLog.Printf("Successfully bundled parse_copilot_log script: %d bytes", len(bundled))
			parseCopilotLogBundled = bundled
		}
	})
	return parseCopilotLogBundled
}
