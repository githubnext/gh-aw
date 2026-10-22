package cli

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

func healthRevisionLookupKey(run WorkflowRun) string {
	return failureClusterKey(run.Repository, run.WorkflowPath, run.HeadSha, run.Event)
}

func resolveHealthRevision(run WorkflowRun, collection *HealthCollection) {
	if !healthRunHasWorkflowRevision(run) {
		if collection.revisions == nil {
			collection.revisions = make(map[string]*workflow.LockMetadata)
		}
		if _, known := collection.revisions[healthRevisionLookupKey(run)]; known {
			return
		}
		collection.revisions[healthRevisionLookupKey(run)] = nil
		healthLog.Printf("Workflow revision unavailable for run %d: event=%s, path=%s", run.DatabaseID, run.Event, run.WorkflowPath)
		return
	}
	enableHealthLocalGit(run, collection)
	// Remote fallback may read the run source, current branch source, and lock metadata; local Git and per-ref caches amortize these reads under the shared API budget.
	resolveHealthSourceRevision(run, run.HeadSha, collection)
	resolveHealthHeadRevision(run, collection)
	if healthSourceChanged(run, collection) {
		return
	}
	if collection.revisions == nil {
		collection.revisions = make(map[string]*workflow.LockMetadata)
	}
	key := healthRevisionLookupKey(run)
	if _, known := collection.revisions[key]; known {
		return
	}
	collection.revisions[key] = nil
	metadata, err := readHealthLockMetadata(run, run.HeadSha, collection)
	if err != nil {
		healthLog.Printf("Workflow revision unavailable for run %d: %v", run.DatabaseID, err)
		collection.Warning = fmt.Sprintf("Workflow revision unavailable for run %d: %v", run.DatabaseID, err)
		return
	}
	collection.revisions[key] = metadata
}

func healthHeadRevisionKey(run WorkflowRun) string {
	return failureClusterKey(run.Repository, run.WorkflowPath, run.HeadBranch)
}

func resolveHealthHeadRevision(run WorkflowRun, collection *HealthCollection) {
	if run.HeadBranch == "" || collection.stopped {
		return
	}
	if collection.headRevisions == nil {
		collection.headRevisions = make(map[string]*workflow.LockMetadata)
	}
	key := healthHeadRevisionKey(run)
	if _, known := collection.headRevisions[key]; known {
		return
	}
	collection.headRevisions[key] = nil
	resolveHealthSourceRevision(run, run.HeadBranch, collection)
	metadata, err := readHealthLockMetadata(run, run.HeadBranch, collection)
	if err != nil {
		healthLog.Printf("Current workflow definition unavailable: %v", err)
		collection.Warning = "Current workflow definition unavailable: " + err.Error()
		return
	}
	collection.headRevisions[key] = metadata
}

func healthSourceRevisionKey(run WorkflowRun, ref string) string {
	return failureClusterKey(run.Repository, run.WorkflowPath, ref)
}

func resolveHealthSourceRevision(run WorkflowRun, ref string, collection *HealthCollection) {
	if ref == "" || collection.stopped {
		return
	}
	if collection.sourceRevisions == nil {
		collection.sourceRevisions = make(map[string]string)
	}
	key := healthSourceRevisionKey(run, ref)
	if _, checked := collection.sourceRevisions[key]; checked {
		return
	}
	collection.sourceRevisions[key] = ""
	sourcePath := strings.TrimSuffix(run.WorkflowPath, ".lock.yml") + ".md"
	endpoint := path.Join(healthRepositoryEndpoint(collection.repoOverride), "contents", sourcePath) +
		"?ref=" + url.QueryEscape(ref)
	var sha string
	var err error
	if useHealthFileSnapshots(collection) {
		sha, err = healthWorkflowFileSHA(sourcePath, ref, collection)
	} else {
		var out []byte
		out, err = healthAPI(collection, endpoint, "--jq", ".sha")
		collection.RevisionLookups++
		sha = strings.TrimSpace(string(out))
	}
	if err == nil {
		if !validHealthBlobSHA(sha) {
			err = errors.New("missing or invalid source blob SHA")
		}
	}
	if err != nil {
		collection.Warning = fmt.Sprintf("Workflow source revision unavailable for %s at %s: %v", sourcePath, ref, err)
		healthLog.Print(collection.Warning)
		return
	}
	collection.sourceRevisions[key] = sha
}

func healthSourceChanged(run WorkflowRun, collection *HealthCollection) bool {
	if collection == nil || !healthRunHasWorkflowRevision(run) {
		return false
	}
	source := collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadSha)]
	head := collection.sourceRevisions[healthSourceRevisionKey(run, run.HeadBranch)]
	return source != "" && head != "" && source != head
}

func healthContentRevisionKey(metadata *workflow.LockMetadata, source string) string {
	if metadata == nil && source != "" {
		return failureClusterKey("source-only", source)
	}
	return failureClusterKey(metadata.FrontmatterHash, metadata.BodyHash, metadata.CompilerVersion, source)
}

func healthRunHasWorkflowRevision(run WorkflowRun) bool {
	switch run.Event {
	case "push", "schedule", "workflow_dispatch", "repository_dispatch", "issues", "issue_comment", "discussion", "discussion_comment":
	default:
		// In particular, workflow_run and pull_request head SHAs need not identify the executing workflow.
		return false
	}
	if !strings.HasPrefix(run.WorkflowPath, logsWorkflowsPathSegment) || path.Clean(run.WorkflowPath) != run.WorkflowPath ||
		!strings.HasSuffix(run.WorkflowPath, ".lock.yml") || (len(run.HeadSha) != 40 && len(run.HeadSha) != 64) {
		return false
	}
	return validHealthBlobSHA(run.HeadSha)
}
