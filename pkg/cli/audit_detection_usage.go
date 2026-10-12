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
// or nil when the usage artifact has no detection token usage.
func readDetectionModelUsage(runDir string) []DetectionModelUsage {
	if runDir == "" {
		return nil
	}
	path := filepath.Join(runDir, constants.UsageArtifactName.String(), "detection", "token_usage.jsonl")
	if !fileutil.FileExists(path) {
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

func renderConsoleDetectionModelUsage(usage []DetectionModelUsage) {
	for _, entry := range usage {
		fmt.Fprintf(os.Stderr, "  detection_model_usage: model=%s requests=%d\n", safeModelRoutingText(entry.Model), entry.Requests)
	}
}
