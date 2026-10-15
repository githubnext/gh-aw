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
		var ids []int64
		for _, target := range c.RepresentativeRuns {
			ids = append(ids, target.RunID)
		}
		out.printf("%s%s count=%d role=%s signal=%s value=%q conclusion=%s kind=%q evidence=%s intentional=%t representatives=%s\n",
			prefix, c.ID, c.Count, c.SignalRole, c.Signal.Kind, failureSignalDisplay(c.Signal),
			c.Conclusion, c.FailureKind, c.Evidence, c.IntentionalFailure, failureRunIDsDisplay(ids))
	}
	return out.err
}

func failureRunIDsDisplay(ids []int64) string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, strconv.FormatInt(id, 10))
	}
	return strings.Join(values, ",")
}

func failureSignalDisplay(signal FailureSignal) string {
	if signal.Job != "" && signal.Step != "" {
		return signal.Job + " / " + signal.Step
	}
	return signal.Value
}

func renderFailureReportToWriter(w io.Writer, clusters []FailureCluster, plan *FailureInvestigation, prefix string, verbose bool) error {
	if err := renderFailureInvestigationToWriter(w, plan, prefix); err != nil {
		return err
	}
	if err := renderFailureClustersToWriter(w, clusters, prefix); err != nil {
		return err
	}
	out := &compactLogsWriter{writer: w}
	if verbose {
		for _, cluster := range clusters {
			out.printf("%smembers cluster=%s runs=%s\n", prefix, cluster.ID, failureRunIDsDisplay(cluster.RunIDs))
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
	out.println("\n| Cluster | Runs | Role | Symptom | Conclusion / Kind | Evidence | Intentional | Representatives |")
	out.println("|---|---:|---|---|---|---|---|---|")
	for _, c := range clusters {
		ids := make([]string, 0, len(c.RepresentativeRuns))
		for _, target := range c.RepresentativeRuns {
			ids = append(ids, strconv.FormatInt(target.RunID, 10))
		}
		out.printf("| %s | %d | %s | %s | %s / %s | %s | %t | %s |\n",
			c.ID, c.Count, c.SignalRole, modelRoutingMarkdownCell(failureSignalDisplay(c.Signal)),
			modelRoutingMarkdownCell(c.Conclusion), modelRoutingMarkdownCell(c.FailureKind),
			c.Evidence, c.IntentionalFailure, strings.Join(ids, ", "))
	}
	out.println()
	out.println("<details><summary>Full cluster membership</summary>")
	out.println()
	out.println("```text")
	for _, cluster := range clusters {
		out.printf("%s runs=%s\n", cluster.ID, failureRunIDsDisplay(cluster.RunIDs))
	}
	out.println("```")
	out.println("\n</details>")
	out.println()
	if out.err != nil {
		console.PrintErrorMessage("Cannot render failure clusters: " + out.err.Error())
	}
}
