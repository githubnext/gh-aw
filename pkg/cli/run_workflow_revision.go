package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/githubapi"
)

func verifyExpectedWorkflowRun(ctx context.Context, lockFile string, info *WorkflowRunInfo, opts RunOptions) error {
	if info == nil || info.DatabaseID <= 0 || opts.RepoOverride == "" {
		return errors.New("exact native run identity and repository are required")
	}
	client, err := api.NewRESTClient(githubapi.ClientOptions("", ""))
	if err != nil {
		return err
	}
	var run struct {
		ID         json.Number `json:"id"`
		HeadSHA    string      `json:"head_sha"`
		Path       string      `json:"path"`
		Event      string      `json:"event"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d", opts.RepoOverride, info.DatabaseID)
	if err := client.DoWithContext(ctx, http.MethodGet, endpoint, nil, &run); err != nil {
		return err
	}
	workflowPath, _, _ := strings.Cut(run.Path, "@")
	if run.ID.String() != strconv.FormatInt(info.DatabaseID, 10) ||
		!strings.EqualFold(run.Repository.FullName, opts.RepoOverride) ||
		run.Event != "workflow_dispatch" || workflowPath != logsWorkflowsPathSegment+lockFile ||
		run.HeadSHA != opts.ExpectedRevision {
		return errors.New("native dispatcher run does not match the verified repository, workflow, and revision")
	}
	return nil
}
