package cli

import (
	"io"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

func renderFailureClustersToWriter(w io.Writer, clusters []FailureCluster, prefix string) error {
	if len(clusters) == 0 {
		return nil
	}
	out := &compactLogsWriter{writer: w}
	out.printf("%s[failure_clusters] groups=%d overlapping=true; symptoms are not proven causes; audit representative runs first\n", prefix, len(clusters))
	for _, c := range clusters {
		ids := make([]string, 0, len(c.RunIDs))
		for _, id := range c.RunIDs {
			ids = append(ids, strconv.FormatInt(id, 10))
		}
		out.printf("%s%s count=%d signal=%s value=%q conclusion=%s kind=%q evidence=%s intentional=%t runs=%s workflows=%q engines=%q\n",
			prefix, c.ID, c.Count, c.Signal.Kind, c.Signal.Value, c.Conclusion, c.FailureKind,
			c.Evidence, c.IntentionalFailure, strings.Join(ids, ","), strings.Join(c.Workflows, ","), strings.Join(c.Engines, ","))
		for _, target := range c.RepresentativeRuns {
			out.printf("%s  representative=%d repository=%q audit=%q logs=%q command=%q\n",
				prefix, target.RunID, target.Repository, target.AuditPath, target.LogsPath, failureClusterAuditCommand(target))
		}
	}
	return out.err
}

func failureClusterAuditCommand(target FailureClusterTarget) string {
	quote := func(value string) string {
		return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
	}
	if target.URL != "" {
		return "gh aw audit " + quote(target.URL)
	}
	command := "gh aw audit " + strconv.FormatInt(target.RunID, 10)
	if target.Repository != "" {
		command += " --repo " + quote(target.Repository)
	}
	return command
}

func renderMarkdownFailureClustersToWriter(w io.Writer, clusters []FailureCluster) {
	if len(clusters) == 0 {
		return
	}
	out := &compactLogsWriter{writer: w}
	out.println("## Failure Investigation Clusters")
	out.println("\nClusters overlap and describe observed symptoms, not proven root causes. Audit representative runs first.")
	out.println("\n| Cluster | Runs | Symptom | Conclusion | Evidence | Intentional | Representatives |")
	out.println("|---|---:|---|---|---|---|---|")
	for _, c := range clusters {
		ids := make([]string, 0, len(c.RepresentativeRuns))
		for _, target := range c.RepresentativeRuns {
			ids = append(ids, strconv.FormatInt(target.RunID, 10))
		}
		out.printf("| %s | %d | %s: %s | %s | %s | %t | %s |\n",
			c.ID, c.Count, modelRoutingMarkdownCell(c.Signal.Kind), modelRoutingMarkdownCell(c.Signal.Value),
			c.Conclusion, c.Evidence, c.IntentionalFailure, strings.Join(ids, ", "))
	}
	out.println()
	// Preserve full membership and exact audit selectors in the saved report.
	out.println("```text")
	if err := renderFailureClustersToWriter(out, clusters, ""); err != nil {
		console.PrintErrorMessage("Cannot render failure clusters: " + err.Error())
		return
	}
	out.println("```")
	out.println()
	if out.err != nil {
		console.PrintErrorMessage("Cannot render failure clusters: " + out.err.Error())
	}
}
