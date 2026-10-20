package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/errorutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// HealthCollection describes a bounded sample, including any quota-related truncation.
type HealthCollection struct {
	Partial             bool                     `json:"partial"`
	Reason              string                   `json:"reason,omitempty"`
	Warning             string                   `json:"warning,omitempty"`
	Host                string                   `json:"host"`
	APIRateLimit        *GitHubAPIRateLimitState `json:"api_rate_limit,omitempty"`
	RevisionLookups     int                      `json:"revision_lookups"`
	APIRequests         int                      `json:"api_requests"`
	MinimumRuns         int                      `json:"minimum_runs"`
	MaximumDays         int                      `json:"maximum_days"`
	StartDate           string                   `json:"start_date,omitempty"`
	EndDate             string                   `json:"end_date,omitempty"`
	FailurePassComplete bool                     `json:"failure_pass_complete"`
	workflows           []GitHubWorkflow
	revisions           map[string]*workflow.LockMetadata
	headRevisions       map[string]*workflow.LockMetadata
	sourceRevisions     map[string]string
	checkedRequests     int
	repoOverride        string
	stopped             bool
	visited             map[string]struct{}
	jobPresence         map[healthRunAttempt]bool
}

const healthMinimumRuns = 5
const healthMaximumAPIRequests = 1000

var errHealthAPIBudget = errors.New("health API request budget reached")
var healthRunGH = workflow.RunGHCombinedContext

func checkHealthAPIBudget(collection *HealthCollection) error {
	if collection.APIRequests >= healthMaximumAPIRequests {
		return errHealthAPIBudget
	}
	if collection.APIRateLimit != nil && collection.APIRequests-collection.checkedRequests < 20 &&
		collection.APIRateLimit.Remaining-(collection.APIRequests-collection.checkedRequests) > RateLimitThreshold {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := fetchRateLimitForHostFunc(ctx, collection.Host)
	if err != nil {
		healthLog.Printf("Health API budget unavailable: %v", err)
		collection.Warning = "GitHub API quota could not be checked: " + err.Error()
		return fmt.Errorf("cannot check health API budget: %w", err)
	}
	collection.APIRateLimit = &state
	collection.checkedRequests = collection.APIRequests
	if state.Remaining <= RateLimitThreshold {
		return fmt.Errorf("%w: %d core requests remain on %s; resets at %s",
			errLogsAPIRateLimitReached, state.Remaining, collection.Host,
			time.Unix(state.Reset, 0).UTC().Format(time.RFC3339))
	}
	return nil
}

func stopHealthCollection(collection *HealthCollection, err error) {
	collection.Partial = true
	collection.stopped = true
	collection.Reason = "api_error"
	if errorutil.IsRateLimitError(err.Error()) || errors.Is(err, errLogsAPIRateLimitReached) {
		collection.Reason = "rate_limit"
	} else if errors.Is(err, errHealthAPIBudget) {
		collection.Reason = "request_budget"
	}
	collection.Warning = err.Error()
	healthLog.Printf("Health collection stopped: %s", collection.Warning)
}

func healthAPI(collection *HealthCollection, endpoint string, extra ...string) ([]byte, error) {
	if err := checkHealthAPIBudget(collection); err != nil {
		stopHealthCollection(collection, err)
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{"api", "--hostname", collection.Host, endpoint}
	args = append(args, extra...)
	collection.APIRequests++
	out, err := healthRunGH(ctx, "Fetching health evidence...", args...)
	if err != nil {
		err = fmt.Errorf("health API request failed: %w: %s", err, strings.TrimSpace(string(out)))
		if errorutil.IsRateLimitError(err.Error()) {
			stopHealthCollection(collection, err)
		}
	}
	return out, err
}

func (config HealthConfig) collectionWorkflows() []GitHubWorkflow {
	if config.collection == nil {
		return nil
	}
	return config.collection.workflows
}

func renderHealthCollection(w io.Writer, collection *HealthCollection) error {
	if collection == nil || (!collection.Partial && collection.Warning == "") {
		return nil
	}
	_, err := fmt.Fprintf(w, "\n[health_collection]\npartial=%t reason=%s warning=%q\n",
		collection.Partial, collection.Reason, collection.Warning)
	return err
}
