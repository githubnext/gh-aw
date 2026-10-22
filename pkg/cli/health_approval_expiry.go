package cli

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const healthMaximumApprovalRequests = 20
const healthApprovalExpiryMessage = "This workflow run required approval but was not approved before it expired."

type HealthApprovalEvidence struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	URL       string    `json:"url,omitempty"`
	Message   string    `json:"message,omitempty"`
	Attempt   int       `json:"attempt"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Call only after the Jobs API has confirmed that this attempt has no jobs.
func observeHealthApprovalExpiry(run WorkflowRun, collection *HealthCollection) {
	if run.Status != "completed" || run.Conclusion != "failure" ||
		run.DatabaseID <= 0 || run.Attempt <= 0 || run.UpdatedAt.IsZero() {
		return
	}
	if collection.ApprovalExpiryEvidence == nil {
		collection.ApprovalExpiryEvidence = make(map[int64]HealthApprovalEvidence)
	}
	if previous, ok := collection.ApprovalExpiryEvidence[run.DatabaseID]; ok &&
		previous.Attempt == run.Attempt && previous.UpdatedAt.Equal(run.UpdatedAt) {
		return
	}
	repository := collection.repoOverride
	if repository == "" {
		repository = run.Repository
	}
	evidence := HealthApprovalEvidence{
		Status: "unknown", Attempt: run.Attempt, UpdatedAt: run.UpdatedAt,
	}
	if repository != "" {
		evidence.URL = fmt.Sprintf("https://%s/%s/actions/runs/%d/attempts/%d", collection.Host,
			strings.TrimPrefix(healthRepositoryEndpoint(repository), "repos/"), run.DatabaseID, run.Attempt)
	}
	if repository == "" {
		evidence.Reason = "missing_repository"
	} else if collection.ApprovalRequests >= healthMaximumApprovalRequests {
		evidence.Reason = "approval_annotation_request_limit"
	} else {
		// REST run/check-suite payloads and approval history omit this annotation.
		// Use the configured host and repository, never a run-provided URL.
		before := collection.APIRequests
		data, err := healthAPI(collection, evidence.URL, "--header", "Accept: text/html")
		collection.ApprovalRequests += collection.APIRequests - before
		if err == nil {
			var expired bool
			expired, err = healthApprovalExpiredAnnotation(data)
			if err == nil {
				evidence.Status, evidence.Reason = "retained", "no_approval_expiry_annotation"
				if expired {
					evidence.Status, evidence.Reason, evidence.Message = "ignored", "approval_expired", healthApprovalExpiryMessage
				}
			}
		}
		if err != nil {
			evidence.Reason = err.Error()
		}
	}
	collection.ApprovalExpiryEvidence[run.DatabaseID] = evidence
	switch evidence.Status {
	case "ignored":
		excludeHealthApprovalExpiryCandidate(run, collection)
	case "unknown":
		healthLog.Printf("Health approval-expiry evidence unavailable: run=%d reason=%s", run.DatabaseID, evidence.Reason)
	}
}

func healthApprovalExpiredAnnotation(data []byte) (bool, error) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return false, fmt.Errorf("failed to parse workflow run annotations: %w", err)
	}
	found, expired := false, true
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		for _, attr := range node.Attr {
			if attr.Key == "data-target" && slices.Contains(strings.Fields(attr.Val), "annotation-message.annotationContainer") {
				found = true
				var text strings.Builder
				appendHealthAnnotationText(node, &text)
				expired = expired && strings.Join(strings.Fields(text.String()), " ") == healthApprovalExpiryMessage
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	if !found {
		return false, errors.New("workflow run annotations unavailable")
	}
	return expired, nil
}

func appendHealthAnnotationText(node *html.Node, text *strings.Builder) {
	if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "style") {
		return
	}
	if node.Type == html.TextNode {
		text.WriteString(node.Data)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendHealthAnnotationText(child, text)
	}
}

func excludeHealthApprovalExpiryCandidate(run WorkflowRun, collection *HealthCollection) {
	candidate := slices.ContainsFunc(collection.failureInventory, func(candidate WorkflowRun) bool {
		return candidate.DatabaseID == run.DatabaseID && candidate.Attempt == run.Attempt && candidate.UpdatedAt.Equal(run.UpdatedAt)
	})
	if !candidate {
		return
	}
	for i := range collection.PotentialIssues {
		issue := &collection.PotentialIssues[i]
		if issue.WorkflowPath != run.WorkflowPath {
			continue
		}
		issue.FailedRuns--
		collection.FailureCandidates--
		if issue.FailedRuns == 0 {
			collection.PotentialIssues = slices.Delete(collection.PotentialIssues, i, i+1)
			collection.FailureCandidateWorkflows--
			return
		}
		if issue.RunID == run.DatabaseID {
			for _, next := range collection.failureInventory {
				if next.WorkflowPath == run.WorkflowPath && collection.ApprovalExpiryEvidence[next.DatabaseID].Status != "ignored" {
					issue.RunID, issue.URL, issue.RepresentativeStatus = next.DatabaseID, next.URL, "unverified"
					break
				}
			}
		}
		return
	}
}
