package cli

import (
	"cmp"
	"slices"
	"strings"
)

const defaultFailureInvestigationBudget = 5

// FailureInvestigation is an initial, bounded audit plan, not a diagnosis.
// CoveredRuns measures shared-symptom reach, not confirmed root-cause coverage.
type FailureInvestigation struct {
	UniqueFailedRuns       int                          `json:"unique_failed_runs"`
	ObservedEvidenceRuns   int                          `json:"observed_evidence_runs"`
	MetadataOnlyRuns       int                          `json:"metadata_only_runs"`
	NonExecutionRuns       int                          `json:"non_execution_runs"`
	IntentionalFailureRuns int                          `json:"intentional_failure_runs"`
	AuditBudget            int                          `json:"audit_budget"`
	EligibleClusters       int                          `json:"eligible_clusters"`
	CoveredClusters        int                          `json:"covered_clusters"`
	RemainingClusters      int                          `json:"remaining_clusters"`
	CoveredRuns            int                          `json:"covered_runs"`
	SelectedRuns           []FailureInvestigationTarget `json:"selected_runs"`
}

type FailureInvestigationTarget struct {
	FailureClusterTarget
	ClusterIDs []string `json:"cluster_ids"`
}

type failureInvestigationCandidate struct {
	run        *RunData
	clusterIDs []string
	signature  string
}

type failureInvestigationGain struct {
	rank     int
	clusters int
	runs     int
}

func buildFailureInvestigation(runs []RunData, clusters []FailureCluster) *FailureInvestigation {
	if len(clusters) == 0 {
		return nil
	}
	plan := &FailureInvestigation{AuditBudget: defaultFailureInvestigationBudget, SelectedRuns: []FailureInvestigationTarget{}}
	eligible := make(map[string]FailureCluster)
	for _, cluster := range clusters {
		if !cluster.IntentionalFailure && cluster.Evidence != "confirmed_no_jobs" {
			eligible[cluster.ID] = cluster
		}
	}
	candidates := failureInvestigationCandidates(runs, eligible, plan)
	plan.EligibleClusters = len(eligible)
	covered := make(map[string]struct{})
	for len(plan.SelectedRuns) < plan.AuditBudget {
		best := selectFailureInvestigationCandidate(candidates, eligible, covered)
		if best == nil {
			break
		}
		plan.SelectedRuns = append(plan.SelectedRuns, FailureInvestigationTarget{
			FailureClusterTarget: failureTarget(best.run), ClusterIDs: best.clusterIDs,
		})
		for _, id := range best.clusterIDs {
			covered[id] = struct{}{}
		}
	}
	reached := make(map[FailureRunIdentity]struct{})
	for id := range covered {
		for _, identity := range eligible[id].RunIdentities {
			reached[identity] = struct{}{}
		}
	}
	plan.CoveredClusters = len(covered)
	plan.RemainingClusters = len(eligible) - len(covered)
	plan.CoveredRuns = len(reached)
	return plan
}

func failureInvestigationCandidates(runs []RunData, eligible map[string]FailureCluster, plan *FailureInvestigation) []failureInvestigationCandidate {
	byID := make(map[FailureRunIdentity]*RunData)
	for i := range runs {
		run := &runs[i]
		if !isClusterFailureConclusion(run.Conclusion) || isObsoletePRRun(*run) {
			continue
		}
		identity := failureRunIdentityFor(*run)
		if prior, exists := byID[identity]; !exists || compareFailureRuns(run, prior) < 0 {
			byID[identity] = run
		}
	}
	candidates := make([]failureInvestigationCandidate, 0, len(byID))
	for _, run := range byID {
		plan.UniqueFailedRuns++
		if run.JobCount != nil && *run.JobCount == 0 && len(run.FailureSignals) == 0 {
			plan.NonExecutionRuns++
			continue
		}
		if len(run.FailureSignals) == 0 {
			plan.MetadataOnlyRuns++
		} else {
			plan.ObservedEvidenceRuns++
		}
		if run.IntentionalFailure {
			plan.IntentionalFailureRuns++
			continue
		}
		candidate := failureInvestigationCandidate{run: run}
		for _, id := range run.FailureClusterIDs {
			if _, ok := eligible[id]; ok {
				candidate.clusterIDs = append(candidate.clusterIDs, id)
			}
		}
		slices.Sort(candidate.clusterIDs)
		candidate.signature = strings.Join(candidate.clusterIDs, ",")
		candidates = append(candidates, candidate)
	}
	return candidates
}

func failureCandidateGain(candidate *failureInvestigationCandidate, eligible map[string]FailureCluster, covered map[string]struct{}, reachedByRole map[int]map[FailureRunIdentity]struct{}, scratch map[FailureRunIdentity]struct{}) failureInvestigationGain {
	gain := failureInvestigationGain{rank: 4}
	clear(scratch)
	for _, id := range candidate.clusterIDs {
		if _, seen := covered[id]; seen {
			continue
		}
		cluster := eligible[id]
		rank := failureSignalRoleRank(cluster.SignalRole)
		if rank > gain.rank {
			continue
		}
		if rank < gain.rank {
			gain.rank, gain.clusters = rank, 0
			clear(scratch)
		}
		gain.clusters++
		for _, identity := range cluster.RunIdentities {
			if _, reached := reachedByRole[rank][identity]; !reached {
				scratch[identity] = struct{}{}
			}
		}
	}
	gain.runs = len(scratch)
	return gain
}

func selectFailureInvestigationCandidate(candidates []failureInvestigationCandidate, eligible map[string]FailureCluster, covered map[string]struct{}) *failureInvestigationCandidate {
	var best *failureInvestigationCandidate
	var bestGain failureInvestigationGain
	// Identical symptom memberships have identical gains, independent of run context.
	gains := make(map[string]failureInvestigationGain)
	reachedByRole := coveredFailureRunsByRole(eligible, covered)
	scratch := make(map[FailureRunIdentity]struct{})
	for i := range candidates {
		candidate := &candidates[i]
		gain, cached := gains[candidate.signature]
		if !cached {
			gain = failureCandidateGain(candidate, eligible, covered, reachedByRole, scratch)
			gains[candidate.signature] = gain
		}
		if gain.clusters == 0 {
			continue
		}

		if best == nil || cmp.Or(
			cmp.Compare(gain.rank, bestGain.rank),
			cmp.Compare(bestGain.clusters, gain.clusters),
			cmp.Compare(bestGain.runs, gain.runs),
			compareFailureRuns(candidate.run, best.run),
		) < 0 {
			best, bestGain = candidate, gain
		}
	}
	return best
}

func coveredFailureRunsByRole(eligible map[string]FailureCluster, covered map[string]struct{}) map[int]map[FailureRunIdentity]struct{} {
	reached := make(map[int]map[FailureRunIdentity]struct{})
	for id := range covered {
		cluster := eligible[id]
		rank := failureSignalRoleRank(cluster.SignalRole)
		if reached[rank] == nil {
			reached[rank] = make(map[FailureRunIdentity]struct{})
		}
		for _, identity := range cluster.RunIdentities {
			reached[rank][identity] = struct{}{}
		}
	}
	return reached
}
