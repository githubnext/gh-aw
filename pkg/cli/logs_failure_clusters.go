package cli

import (
	"cmp"
	"crypto/sha256"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

// FailureSignal records an observed symptom, not a proven root cause.
type FailureSignal struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Job   string `json:"job,omitempty"`
	Step  string `json:"step,omitempty"`
}

// FailureCluster provides an investigation subset for runs sharing a symptom.
// Clusters overlap; their run counts must not be summed to count failed runs.
type FailureCluster struct {
	ID                 string                   `json:"id"`
	Signal             FailureSignal            `json:"signal"`
	Conclusion         string                   `json:"conclusion"`
	FailureKind        string                   `json:"failure_kind,omitempty"`
	IntentionalFailure bool                     `json:"intentional_failure,omitempty"`
	Evidence           string                   `json:"evidence"`
	SignalRole         string                   `json:"signal_role"`
	Count              int                      `json:"count"`
	RunIDs             []int64                  `json:"run_ids"`
	RunIdentities      []FailureRunIdentity     `json:"run_identities,omitempty"`
	Workflows          []string                 `json:"workflows"`
	Repositories       []string                 `json:"repositories,omitempty"`
	Engines            []string                 `json:"engines,omitempty"`
	Models             []string                 `json:"models,omitempty"`
	FirstSeen          time.Time                `json:"first_seen"`
	LastSeen           time.Time                `json:"last_seen"`
	RepresentativeRuns []FailureClusterTarget   `json:"representative_runs"`
	Statistics         FailureClusterStatistics `json:"statistics"`
}

// FailureRunIdentity uniquely identifies a run across GitHub hosts and repositories.
type FailureRunIdentity struct {
	Host       string `json:"host,omitempty"`
	Repository string `json:"repository,omitempty"`
	RunID      int64  `json:"run_id"`
}

type FailureClusterTarget struct {
	RunID        int64     `json:"run_id"`
	Repository   string    `json:"repository,omitempty"`
	WorkflowName string    `json:"workflow_name,omitempty"`
	WorkflowPath string    `json:"workflow_path,omitempty"`
	Engine       string    `json:"engine,omitempty"`
	Model        string    `json:"model,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitzero"`
	URL          string    `json:"url,omitempty"`
	LogsPath     string    `json:"logs_path,omitempty"`
	AuditPath    string    `json:"audit_path,omitempty"`
}

func failureRunIdentityFor(run RunData) FailureRunIdentity {
	identity := FailureRunIdentity{
		Repository: strings.ToLower(strings.TrimSpace(run.Repository)),
		RunID:      run.RunID,
	}
	parsedURL, err := url.Parse(run.URL)
	if err != nil {
		return identity
	}
	identity.Host = strings.ToLower(parsedURL.Host)
	if identity.Repository == "" {
		if repoPath, _, found := strings.Cut(parsedURL.Path, "/actions/runs/"); found {
			repoPath = strings.TrimRight(repoPath, "/")
			ownerPath := path.Dir(repoPath)
			if repoPath != "" && ownerPath != "." && ownerPath != "/" {
				identity.Repository = strings.ToLower(path.Join(path.Base(ownerPath), path.Base(repoPath)))
			}
		}
	}
	return identity
}

func (identity FailureRunIdentity) key() string {
	return failureClusterKey(identity.Host, identity.Repository, strconv.FormatInt(identity.RunID, 10))
}

func failureSignalsForRun(pr ProcessedRun) []FailureSignal {
	if !isClusterFailureConclusion(pr.Run.Conclusion) {
		return nil
	}
	var signals []FailureSignal
	add := func(kind, value string) {
		if value = strings.TrimSpace(value); value != "" {
			signals = append(signals, enrichFailureSignal(FailureSignal{Kind: kind, Value: value}))
		}
	}
	for _, job := range pr.JobDetails {
		if !isFailureConclusion(job.Conclusion) {
			continue
		}
		foundStep := false
		for _, step := range job.Steps {
			if isFailureConclusion(step.Conclusion) {
				// Encode the pair so names containing separators cannot collide.
				add("failed_step", failureClusterKey(normalizeJobName(job.Name), step.Name))
				foundStep = true
			}
		}
		if !foundStep {
			add("failed_job", normalizeJobName(job.Name))
		}
	}
	for _, failure := range pr.MCPFailures {
		add("mcp_failure", failure.ServerName)
	}
	for _, tool := range pr.MissingTools {
		add("missing_tool", tool.Tool)
	}
	for _, data := range pr.MissingData {
		add("missing_data", data.DataType)
	}
	if pr.FirewallAnalysis != nil {
		for domain, stats := range pr.FirewallAnalysis.RequestsByDomain {
			if stats.Blocked > 0 {
				add("blocked_domain", domain)
			}
		}
	}
	slices.SortFunc(signals, compareFailureSignals)
	return slices.Compact(signals)
}

func compareFailureSignals(a, b FailureSignal) int {
	return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Value, b.Value))
}

func isClusterFailureConclusion(conclusion string) bool {
	return isFailureConclusion(conclusion) || conclusion == "startup_failure"
}

type failureClusterGroup struct {
	cluster FailureCluster
	members []*RunData
}

func failureClusterKey(parts ...string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, strconv.Quote(part))
	}
	return strings.Join(quoted, "/")
}

func failureClusterSignals(run RunData) ([]FailureSignal, string) {
	signals := slices.Clone(run.FailureSignals)
	if len(signals) == 0 {
		// Unknown failures are workflow-scoped, not a presumed common cause.
		identity := failureRunIdentityFor(run)
		scope := failureClusterKey(identity.Host, identity.Repository, run.WorkflowPath, run.WorkflowName)
		if run.JobCount != nil && *run.JobCount == 0 {
			return []FailureSignal{{Kind: "no_jobs", Value: scope}}, "confirmed_no_jobs"
		}
		return []FailureSignal{{Kind: "workflow_failure", Value: scope}}, "metadata_only"
	}
	for i := range signals {
		signals[i] = enrichFailureSignal(signals[i])
	}
	slices.SortFunc(signals, compareFailureSignals)
	return slices.Compact(signals), "observed_symptom"
}

// annotateFailureClusters only uses already extracted metadata, so it works
// without additional downloads, audit generation, or model calls.
func annotateFailureClusters(runs []RunData) []FailureCluster {
	groups := make(map[string]*failureClusterGroup)
	for i := range runs {
		run := &runs[i]
		// Membership is query-scoped, never reused from a cached report.
		run.FailureClusterIDs = nil
		if !isClusterFailureConclusion(run.Conclusion) || isObsoletePRRun(*run) {
			continue
		}
		signals, evidence := failureClusterSignals(*run)
		if len(run.FailureSignals) > 0 {
			run.FailureSignals = signals
		}
		for _, signal := range signals {
			key := failureClusterKey(signal.Kind, signal.Value, run.Conclusion, run.FailureKind, strconv.FormatBool(run.IntentionalFailure))
			digest := sha256.Sum256([]byte(key))
			id := fmt.Sprintf("failure-%x", digest[:8])
			g := groups[id]
			if g == nil {
				g = &failureClusterGroup{cluster: FailureCluster{
					ID: id, Signal: signal, Conclusion: run.Conclusion,
					FailureKind: run.FailureKind, IntentionalFailure: run.IntentionalFailure,
					Evidence: evidence, SignalRole: failureSignalRole(signal),
				}}
				groups[id] = g
			}
			g.members = append(g.members, run)
			run.FailureClusterIDs = append(run.FailureClusterIDs, id)
		}
		slices.Sort(run.FailureClusterIDs)
	}
	clusters := make([]FailureCluster, 0, len(groups))
	for _, g := range groups {
		clusters = append(clusters, finalizeFailureCluster(g))
	}
	annotateFailureClusterStatistics(runs, clusters)
	slices.SortFunc(clusters, func(a, b FailureCluster) int {
		return cmp.Or(
			cmp.Compare(strconv.FormatBool(a.IntentionalFailure), strconv.FormatBool(b.IntentionalFailure)),
			cmp.Compare(failureSignalRoleRank(a.SignalRole), failureSignalRoleRank(b.SignalRole)),
			cmp.Compare(b.Count, a.Count), cmp.Compare(a.ID, b.ID),
		)
	})
	return clusters
}

func finalizeFailureCluster(g *failureClusterGroup) FailureCluster {
	c := g.cluster
	// Prefer inspectable runs, then the newest, with deterministic tie breaks.
	slices.SortFunc(g.members, compareFailureRuns)
	seen := make(map[FailureRunIdentity]struct{})
	for _, run := range g.members {
		identity := failureRunIdentityFor(*run)
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		c.Count++
		c.RunIDs = append(c.RunIDs, run.RunID)
		c.RunIdentities = append(c.RunIdentities, identity)
		c.Workflows = append(c.Workflows, workflowIDFromRun(run.WorkflowPath, run.WorkflowName))
		c.Repositories = append(c.Repositories, run.Repository)
		c.Engines = append(c.Engines, run.EngineID)
		c.Models = append(c.Models, run.Model)
		if c.FirstSeen.IsZero() || (!run.CreatedAt.IsZero() && run.CreatedAt.Before(c.FirstSeen)) {
			c.FirstSeen = run.CreatedAt
		}
		if run.CreatedAt.After(c.LastSeen) {
			c.LastSeen = run.CreatedAt
		}
	}
	c.RepresentativeRuns = selectFailureRepresentatives(g.members)
	slices.Sort(c.RunIDs)
	slices.SortFunc(c.RunIdentities, func(a, b FailureRunIdentity) int {
		return cmp.Or(
			cmp.Compare(a.Host, b.Host),
			cmp.Compare(a.Repository, b.Repository),
			cmp.Compare(a.RunID, b.RunID),
		)
	})
	c.Workflows = sortedFailureFacets(c.Workflows)
	c.Repositories = sortedFailureFacets(c.Repositories)
	c.Engines = sortedFailureFacets(c.Engines)
	c.Models = sortedFailureFacets(c.Models)
	return c
}

func compareFailureRuns(x, y *RunData) int {
	xIdentity, yIdentity := failureRunIdentityFor(*x), failureRunIdentityFor(*y)
	return cmp.Or(
		cmp.Compare(failureRunDetailScore(*y), failureRunDetailScore(*x)),
		y.CreatedAt.Compare(x.CreatedAt),
		cmp.Compare(y.RunID, x.RunID),
		cmp.Compare(xIdentity.Host, yIdentity.Host),
		cmp.Compare(xIdentity.Repository, yIdentity.Repository),
	)
}

func failureRunDetailScore(run RunData) int {
	score := 0
	if run.LogsPath != "" {
		score++
	}
	if run.AuditPath != "" {
		score += 2
	}
	return score
}

func sortedFailureFacets(values []string) []string {
	slices.Sort(values)
	values = slices.Compact(values)
	return slices.DeleteFunc(values, func(value string) bool { return value == "" })
}
