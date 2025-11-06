package workflow

import (
	_ "embed"
	"fmt"
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
)

// getCollectJSONLOutputScript returns the bundled collect_ndjson_output script
// Bundling is performed on first access and cached for subsequent calls
func getCollectJSONLOutputScript() string {
	collectJSONLOutputScriptOnce.Do(func() {
		scriptsLog.Print("Bundling collect_ndjson_output script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(collectJSONLOutputScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for collect_ndjson_output: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(collectJSONLOutputScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("collect_ndjson_output contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for compute_text: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(computeTextScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("compute_text contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for sanitize_output: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(sanitizeOutputScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("sanitize_output contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for create_issue: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(createIssueScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("create_issue contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for add_labels: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(addLabelsScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("add_labels contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
			addLabelsScript = addLabelsScriptSource
		} else {
			addLabelsScript = bundled
		}
	})
	return addLabelsScript
}

// getParseFirewallLogsScript returns the bundled parse_firewall_logs script
// Bundling is performed on first access and cached for subsequent calls
func getParseFirewallLogsScript() string {
	parseFirewallLogsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(parseFirewallLogsScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for parse_firewall_logs: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(parseFirewallLogsScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("parse_firewall_logs contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for create_discussion: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(createDiscussionScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("create_discussion contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for update_issue: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(updateIssueScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("update_issue contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for create_code_scanning_alert: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(createCodeScanningAlertScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("create_code_scanning_alert contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for create_pr_review_comment: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(createPRReviewCommentScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("create_pr_review_comment contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for add_comment: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(addCommentScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("add_comment contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for upload_assets: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(uploadAssetsScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("upload_assets contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
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
			scriptsLog.Printf("Bundling failed for push_to_pull_request_branch: %v", err)
			// If bundling fails, validate the source before using it
			if validateErr := validateNoLocalRequires(pushToPullRequestBranchScriptSource); validateErr != nil {
				scriptsLog.Printf("CRITICAL: Source validation failed: %v", validateErr)
				panic(fmt.Sprintf("push_to_pull_request_branch contains unbundled local requires: %v", validateErr))
			}
			scriptsLog.Print("Using source as-is (validation passed)")
			pushToPullRequestBranchScript = pushToPullRequestBranchScriptSource
		} else {
			pushToPullRequestBranchScript = bundled
		}
	})
	return pushToPullRequestBranchScript
}
