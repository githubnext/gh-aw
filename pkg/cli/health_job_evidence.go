package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const healthMaximumJobPages = 3

type healthJobs struct {
	details  []JobInfoWithDuration
	total    *int
	complete bool
}

type HealthJobEvidence struct {
	Source    string    `json:"source"`
	Jobs      int       `json:"jobs"`
	Complete  bool      `json:"complete"`
	Attempt   int       `json:"attempt,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

// Only failure-first representatives expand the inexpensive presence lookup.
func collectHealthRepresentativeJobs(run WorkflowRun, collection *HealthCollection) error {
	key := healthRunAttempt{id: run.DatabaseID, attempt: run.Attempt}
	jobs := collection.jobs[key]
	if jobs == nil {
		return nil
	}
	if collection.JobEvidence == nil {
		collection.JobEvidence = make(map[int64]HealthJobEvidence)
	}
	defer func() {
		collection.JobEvidence[run.DatabaseID] = HealthJobEvidence{
			Source: "jobs_api", Jobs: len(jobs.details), Complete: jobs.complete,
			Attempt: run.Attempt, UpdatedAt: run.UpdatedAt,
		}
	}()
	if jobs.total != nil && *jobs.total > len(jobs.details) {
		endpoint := fmt.Sprintf("%s/actions/runs/%d", healthRepositoryEndpoint(collection.repoOverride), run.DatabaseID)
		if run.Attempt > 0 {
			endpoint += fmt.Sprintf("/attempts/%d", run.Attempt)
		}
		fetched := &healthJobs{}
		for page := 1; page <= healthMaximumJobPages; page++ {
			if err := appendHealthJobPage(endpoint, page, fetched, collection); err != nil {
				return err
			}
			*jobs = *fetched
			if jobs.complete {
				break
			}
		}
	}
	return nil
}

func appendHealthJobPage(endpoint string, page int, jobs *healthJobs, collection *HealthCollection) error {
	before := collection.APIRequests
	out, err := healthAPI(collection, fmt.Sprintf("%s/jobs?per_page=100&page=%d", endpoint, page))
	collection.JobRequests += collection.APIRequests - before
	if err != nil {
		return fmt.Errorf("failed to collect health job evidence: %w", err)
	}
	var data struct {
		Jobs       []JobInfoWithDuration `json:"jobs"`
		TotalCount *int                  `json:"total_count"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return fmt.Errorf("failed to parse health job evidence: %w", err)
	}
	if data.Jobs == nil {
		return errors.New("failed to parse health job evidence: missing jobs array")
	}
	jobs.details = append(jobs.details, data.Jobs...)
	jobs.total = data.TotalCount
	jobs.complete = len(data.Jobs) < 100 || (data.TotalCount != nil && len(jobs.details) >= *data.TotalCount)
	return nil
}
