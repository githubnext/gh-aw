package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
)

var auditDetectionUsageLog = logger.New("cli:audit_detection_usage")

// DetectionModelUsage is the threat-detection job's own model traffic, read from
// the detection api-proxy token usage. It is reported separately from
// firewall_token_usage so the detection model is never folded into the agent's
// (possibly routed) model.
type DetectionModelUsage struct {
	Model    string `json:"model"`
	Requests int    `json:"requests"`
}

// readDetectionModelUsage returns per-model request counts for the detection job,
// or nil when the run has no detection token usage.
func readDetectionModelUsage(runDir string) []DetectionModelUsage {
	path := findDetectionTokenUsageFile(runDir)
	if path == "" {
		return nil
	}
	summary, err := parseTokenUsageFile(path)
	if err != nil {
		auditDetectionUsageLog.Printf("Failed to parse detection token usage %s: %v", path, err)
		return nil
	}
	if summary == nil {
		return nil
	}
	usage := make([]DetectionModelUsage, 0, len(summary.ByModel))
	for model, stats := range summary.ByModel {
		usage = append(usage, DetectionModelUsage{Model: model, Requests: stats.Requests})
	}
	slices.SortFunc(usage, func(a, b DetectionModelUsage) int { return strings.Compare(a.Model, b.Model) })
	auditDetectionUsageLog.Printf("Detection model usage: %d model(s)", len(usage))
	return usage
}

// findDetectionTokenUsageFile locates the detection job's api-proxy token usage.
// It prefers the compact usage artifact (exact or workflow_call "{hash}-usage")
// and falls back to the detection artifact's firewall logs, resolved with the
// same discovery as the agent's token usage (current, AWF audit-dir and legacy
// layouts, and either file name).
func findDetectionTokenUsageFile(runDir string) string {
	if runDir == "" {
		return ""
	}
	if usageDir := findArtifactDir(runDir, constants.UsageArtifactName.String(), ""); usageDir != "" {
		path := filepath.Join(usageDir, "detection", "token_usage.jsonl")
		if fileutil.FileExists(path) {
			return path
		}
	}
	if detectionDir := findArtifactDir(runDir, constants.DetectionArtifactName.String(), ""); detectionDir != "" {
		return findTokenUsageFile(detectionDir)
	}
	return ""
}

func renderConsoleDetectionModelUsage(usage []DetectionModelUsage) {
	for _, entry := range usage {
		fmt.Fprintf(os.Stderr, "  detection_model_usage: model=%s requests=%d\n", safeModelRoutingText(entry.Model), entry.Requests)
	}
}
