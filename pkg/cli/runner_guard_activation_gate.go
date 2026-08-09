package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// runnerGuardActivationGateRule is the runner-guard rule that flags comment-triggered
// workflows without an author authorization check.
const runnerGuardActivationGateRule = "RGS-004"

// runnerGuardWorkflowJob is the minimal subset of a compiled workflow job needed to
// determine whether the job is gated behind an author-association check.
type runnerGuardWorkflowJob struct {
	If    string `yaml:"if"`
	Needs any    `yaml:"needs"`
}

// runnerGuardWorkflow is the minimal subset of a compiled workflow needed for gate analysis.
type runnerGuardWorkflow struct {
	Jobs map[string]runnerGuardWorkflowJob `yaml:"jobs"`
}

// filterRunnerGuardFindings removes RGS-004 findings that are false positives for gh-aw
// compiled workflows.
//
// gh-aw compiles comment-triggered workflows so that the pre_activation job carries a static
// github.event.comment.author_association guard in its job-level if: condition, and every
// downstream job (activation, agent, safe_outputs, ...) depends on it transitively via needs:.
// A job whose ancestors are all skipped never runs, so no privileged step can execute for an
// unauthorized commenter. runner-guard evaluates each job in isolation and does not follow
// needs: edges, so it reports every step of every job as unauthenticated.
//
// Findings are dropped only when the flagged job itself, or one of its transitive
// dependencies, has an if: condition referencing author_association. Workflows without such
// a gate keep their findings.
func filterRunnerGuardFindings(findings []runnerGuardFinding, gitRoot string) []runnerGuardFinding {
	gatedJobsByFile := make(map[string]map[string]struct{})
	filtered := make([]runnerGuardFinding, 0, len(findings))

	for _, finding := range findings {
		if finding.RuleID != runnerGuardActivationGateRule || finding.JobID == "" {
			filtered = append(filtered, finding)
			continue
		}

		gatedJobs, ok := gatedJobsByFile[finding.File]
		if !ok {
			gatedJobs = authorAssociationGatedJobs(resolveRunnerGuardFilePath(gitRoot, finding.File))
			gatedJobsByFile[finding.File] = gatedJobs
		}

		if _, isGated := gatedJobs[finding.JobID]; isGated {
			runnerGuardLog.Printf("Suppressing %s finding for gated job %q in %s", finding.RuleID, finding.JobID, finding.File)
			continue
		}
		filtered = append(filtered, finding)
	}

	return filtered
}

// resolveRunnerGuardFilePath resolves a runner-guard finding file path to an absolute path
// within gitRoot. runner-guard reports paths relative to the scanned directory, which may be
// the repository root or the .github/workflows directory, so both locations are tried.
// It returns an empty string when the path escapes gitRoot or cannot be resolved.
func resolveRunnerGuardFilePath(gitRoot string, file string) string {
	if file == "" {
		return ""
	}

	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		return ""
	}

	cleanPath := filepath.Clean(file)
	candidates := []string{cleanPath}
	if !filepath.IsAbs(cleanPath) {
		candidates = []string{
			filepath.Join(absGitRoot, cleanPath),
			filepath.Join(absGitRoot, ".github", "workflows", cleanPath),
		}
	}

	for _, candidate := range candidates {
		absCandidate, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		relPath, err := filepath.Rel(absGitRoot, absCandidate)
		if err != nil || !filepath.IsLocal(relPath) {
			continue
		}
		if _, err := os.Stat(absCandidate); err == nil {
			return absCandidate
		}
	}

	return ""
}

// authorAssociationGatedJobs returns the set of job IDs in the workflow at path that are
// protected by an author_association check, either directly on the job's if: condition or
// transitively through the needs: graph. An empty set is returned when the workflow cannot
// be read or parsed, so that findings are preserved rather than silently dropped.
func authorAssociationGatedJobs(path string) map[string]struct{} {
	gated := make(map[string]struct{})
	if path == "" {
		return gated
	}

	// #nosec G304 -- path is produced by resolveRunnerGuardFilePath, which validates that the
	// resolved path stays within the repository root.
	content, err := os.ReadFile(path)
	if err != nil {
		runnerGuardLog.Printf("Failed to read workflow %s for gate analysis: %v", path, err)
		return gated
	}

	var workflow runnerGuardWorkflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		runnerGuardLog.Printf("Failed to parse workflow %s for gate analysis: %v", path, err)
		return gated
	}

	// Resolve transitively with a bounded number of passes: each pass can only mark
	// additional jobs, so len(jobs) passes are always sufficient to reach a fixed point.
	for range len(workflow.Jobs) {
		changed := false
		for jobID, job := range workflow.Jobs {
			if _, isGated := gated[jobID]; isGated {
				continue
			}
			if hasAuthorAssociationCheck(job.If) || anyJobGated(gated, jobNeeds(job.Needs)) {
				gated[jobID] = struct{}{}
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return gated
}

// hasAuthorAssociationCheck reports whether an if: condition references author_association.
func hasAuthorAssociationCheck(condition string) bool {
	return strings.Contains(condition, "author_association")
}

// anyJobGated reports whether any of the named jobs is in the gated set.
func anyJobGated(gated map[string]struct{}, needs []string) bool {
	for _, need := range needs {
		if _, isGated := gated[need]; isGated {
			return true
		}
	}
	return false
}

// jobNeeds normalizes the needs: field, which GitHub Actions allows to be either a single
// string or a list of strings.
func jobNeeds(needs any) []string {
	switch value := needs.(type) {
	case string:
		return []string{value}
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			if name, ok := item.(string); ok {
				result = append(result, name)
			}
		}
		return result
	case []string:
		return value
	default:
		return nil
	}
}
