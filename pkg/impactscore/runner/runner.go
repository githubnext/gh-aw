// Package runner runs impact score analysis for a GitHub repository and writes
// JSON artifacts plus a selected report format.
package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/impactscore"
	"github.com/github/gh-aw/pkg/workflow"
)

const githubAPI = "https://api.github.com"

const defaultCostRunCount = 200

const maxIssueCommentPages = 2
const maxWorkflowRunLookups = 50
const normalizeProgressEvery = 10
const workItemFetchConcurrency = 8
const impactPolicyPath = workflow.RepoConfigFileName

var githubAPIRetryDelays = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond}
var waitBeforeGitHubAPIRetry = sleepBeforeGitHubAPIRetry
var createTempOutputDir = os.MkdirTemp
var createOutputParentDir = os.MkdirAll

var currentRepository = repository.Current

type config struct {
	Repo         string
	OutDir       string
	MaxItems     int
	State        string
	Token        string
	ReportFormat string
}

// Config configures one impact score run.
type Config = config

// DefaultConfig returns the command defaults for impact score analysis.
func DefaultConfig() Config {
	return Config{MaxItems: 200, State: "all", ReportFormat: "text"}
}

type githubClient struct {
	client *http.Client
	token  string
}

type githubIssue struct {
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	State       string           `json:"state"`
	StateReason string           `json:"state_reason"`
	Comments    int              `json:"comments"`
	Labels      []githubLabel    `json:"labels"`
	PullRequest *json.RawMessage `json:"pull_request"`
	Milestone   *githubMilestone `json:"milestone"`
}

type githubLabel struct {
	Name string `json:"name"`
}

type githubComment struct {
	Body string `json:"body"`
}

type githubMilestone struct {
	Title string `json:"title"`
}

type githubPullRequest struct {
	ChangedFiles   int    `json:"changed_files"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	Commits        int    `json:"commits"`
	ReviewComments int    `json:"review_comments"`
	Merged         bool   `json:"merged"`
	MergedAt       string `json:"merged_at"`
	Draft          bool   `json:"draft"`
}

type githubPRFile struct {
	Filename string `json:"filename"`
}

type githubContentEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type githubWorkflowRun struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	HTMLURL string `json:"html_url"`
}

type workflowRunReference struct {
	Name       string
	SourcePath string
	RunURL     string
}

type workflowMatcher struct {
	aliases   map[string]string
	workflows []workflowMatchCandidate
}

type workflowMatchCandidate struct {
	Name          string
	TitlePrefix   string
	BracketPrefix string
}

type githubAPIError struct {
	StatusCode int
	Status     string
	Endpoint   string
}

func (e githubAPIError) Error() string {
	return fmt.Sprintf("github request failed: %s %s", e.Status, e.Endpoint)
}

type output struct {
	Repo          string                           `json:"repo"`
	GeneratedAt   string                           `json:"generated_at"`
	Items         []impactscore.WorkItem           `json:"items"`
	Workflows     []impactscore.WorkflowDefinition `json:"workflows"`
	CostRuns      []impactscore.WorkflowCostRun    `json:"cost_runs"`
	ItemRanks     []impactscore.ItemRank           `json:"item_ranks"`
	WorkflowRanks []impactscore.WorkflowRank       `json:"workflow_ranks"`
	GraphNodes    []impactscore.Node               `json:"graph_nodes"`
	GraphEdges    []impactscore.Edge               `json:"graph_edges"`
	Features      []impactscore.ItemFeatures       `json:"features"`
}

// Output is the complete result written to impact score artifacts.
type Output = output

type sourceData struct {
	Items     []impactscore.WorkItem
	Workflows []impactscore.WorkflowDefinition
	CostRuns  []impactscore.WorkflowCostRun
}

type issueCommentCache struct {
	mu     sync.Mutex
	values map[int][]githubComment
}

type workflowRunCache struct {
	mu     sync.Mutex
	values map[string]workflowRunReference
}

func newIssueCommentCache() *issueCommentCache {
	return &issueCommentCache{values: map[int][]githubComment{}}
}

func (c *issueCommentCache) get(number int) ([]githubComment, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	comments, ok := c.values[number]
	return comments, ok
}

func (c *issueCommentCache) set(number int, comments []githubComment) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[number] = comments
}

func newWorkflowRunCache() *workflowRunCache {
	return &workflowRunCache{values: map[string]workflowRunReference{}}
}

func (c *workflowRunCache) reserve(runID, runURL string) (workflowRunReference, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if reference, ok := c.values[runID]; ok {
		return reference, false
	}
	reference := workflowRunReference{RunURL: runURL}
	if len(c.values) >= maxWorkflowRunLookups {
		return reference, false
	}
	c.values[runID] = reference
	return reference, true
}

func (c *workflowRunCache) set(runID string, reference workflowRunReference) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[runID] = reference
}

// Run prepares configuration, fetches or loads source data, scores items, ranks
// workflows, and writes artifacts.
func Run(ctx context.Context, cfg Config) (Output, error) {
	cfg = configWithDefaults(cfg)
	if err := validateConfig(cfg); err != nil {
		return output{}, err
	}
	if err := prepareConfig(&cfg); err != nil {
		return output{}, err
	}
	return runOnce(ctx, cfg)
}

func configWithDefaults(cfg config) config {
	defaults := DefaultConfig()
	if cfg.MaxItems == 0 {
		cfg.MaxItems = defaults.MaxItems
	}
	if cfg.State == "" {
		cfg.State = defaults.State
	}
	if cfg.ReportFormat == "" {
		cfg.ReportFormat = defaults.ReportFormat
	}
	return cfg
}

func validateConfig(cfg config) error {
	if cfg.MaxItems < 0 {
		return fmt.Errorf("invalid --max-items %d: use a non-negative value", cfg.MaxItems)
	}
	switch cfg.State {
	case "open", "closed", "all":
	default:
		return fmt.Errorf("invalid --state %q: use open, closed, or all", cfg.State)
	}
	switch cfg.ReportFormat {
	case "text", "html":
	default:
		return fmt.Errorf("invalid --report-format %q: use text or html", cfg.ReportFormat)
	}
	return nil
}

func prepareConfig(cfg *config) error {
	cfg.Repo = strings.TrimSpace(cfg.Repo)
	if strings.TrimSpace(cfg.Repo) == "" {
		repo, err := currentRepoSlug()
		if err != nil {
			return fmt.Errorf("current repository could not be determined: %w", err)
		}
		cfg.Repo = repo
	}
	if cfg.OutDir == "" {
		outDir, err := defaultOutputDir(cfg.Repo)
		if err != nil {
			return err
		}
		cfg.OutDir = outDir
	}
	if cfg.ReportFormat == "" {
		cfg.ReportFormat = "text"
	}
	return nil
}

func defaultOutputDir(repo string) (string, error) {
	parent := filepath.Join(os.TempDir(), "gh-aw", "impact-score")
	if err := createOutputParentDir(parent, constants.DirPermPublic); err != nil {
		return "", fmt.Errorf("create temporary output parent directory: %w", err)
	}
	dir, err := createTempOutputDir(parent, repoPathSlug(repo)+"-*")
	if err != nil {
		return "", fmt.Errorf("create temporary output directory: %w", err)
	}
	return dir, nil
}

func repoPathSlug(repo string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-")
	return replacer.Replace(repo)
}

func hasImpactPolicy(path string) (bool, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}
	_, ok := doc["impact"]
	return ok, nil
}

func currentRepoSlug() (string, error) {
	repo, err := currentRepository()
	if err != nil {
		return "", err
	}
	if repo.Owner == "" || repo.Name == "" {
		return "", fmt.Errorf("repository owner or name is empty (owner: %q, name: %q)", repo.Owner, repo.Name)
	}
	return repo.Owner + "/" + repo.Name, nil
}

func runOnce(ctx context.Context, cfg config) (output, error) {
	owner, name, err := splitRepo(cfg.Repo)
	if err != nil {
		return output{}, err
	}
	source, reused, err := loadOrFetchSourceData(ctx, cfg, owner, name)
	if err != nil {
		return output{}, err
	}
	source = canonicalizeSourceDataWorkflows(source)
	if reused {
		fmt.Fprintf(os.Stderr, "reused impact-score source artifacts from %s\n", cfg.OutDir)
	}
	hasPolicy, err := hasImpactPolicy(impactPolicyPath)
	if err != nil {
		return output{}, err
	}
	if !hasPolicy {
		if err := writeImpactPolicy(impactPolicyPath, generateHistoryImpactPolicy(cfg.Repo, source)); err != nil {
			return output{}, err
		}
		fmt.Fprintf(os.Stderr, "wrote impact policy to %s\n", impactPolicyPath)
	} else {
		fmt.Fprintf(os.Stderr, "impact policy already exists at %s\n", impactPolicyPath)
	}

	options := rankOptions()
	graph := impactscore.BuildGraph(source.Items, source.Workflows)
	policy, err := loadImpactPolicy(impactPolicyPath)
	if err != nil {
		return output{}, err
	}
	options = applyImpactPolicy(options, policy, impactPolicyPath)
	itemRanks, features := impactscore.RankItemsAndFeaturesWithOptions(graph, options)
	workflowRanks := impactscore.RankWorkflows(itemRanks, source.CostRuns)
	result := output{
		Repo:          cfg.Repo,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Items:         source.Items,
		Workflows:     source.Workflows,
		CostRuns:      source.CostRuns,
		ItemRanks:     itemRanks,
		WorkflowRanks: workflowRanks,
		GraphNodes:    sortedNodes(graph),
		GraphEdges:    graph.Edges,
		Features:      features,
	}
	if err := writeArtifacts(cfg.OutDir, result, cfg.ReportFormat); err != nil {
		return output{}, err
	}
	fmt.Fprintf(os.Stderr, "wrote impact-score artifacts to %s\n", cfg.OutDir)
	return result, nil
}

func loadOrFetchSourceData(ctx context.Context, cfg config, owner, name string) (sourceData, bool, error) {
	if sourceDataArtifactsExist(cfg.OutDir) {
		data, err := readSourceData(cfg.OutDir)
		if err != nil {
			return sourceData{}, false, fmt.Errorf("load cached source artifacts from %s: %w", cfg.OutDir, err)
		}
		if err := validateSourceDataRepo(cfg.Repo, data); err != nil {
			return sourceData{}, false, fmt.Errorf("cached source artifacts in %s do not match current repository: %w", cfg.OutDir, err)
		}
		return data, true, nil
	}
	data, err := fetchSourceData(ctx, cfg, owner, name)
	return data, false, err
}

func fetchSourceData(ctx context.Context, cfg config, owner, name string) (sourceData, error) {
	token := cfg.Token
	if token == "" {
		token = resolveToken(ctx)
	}
	client := githubClient{client: &http.Client{Timeout: 30 * time.Second}, token: token}
	issues, err := client.fetchIssues(ctx, owner, name, cfg.State, cfg.MaxItems)
	if err != nil {
		return sourceData{}, err
	}
	fmt.Fprintf(os.Stderr, "fetched %d issues/PRs from %s\n", len(issues), cfg.Repo)
	workflows, err := client.fetchWorkflows(ctx, owner, name)
	if err != nil {
		return sourceData{}, err
	}
	fmt.Fprintf(os.Stderr, "fetched %d workflow definitions from %s\n", len(workflows), cfg.Repo)
	commentCache := newIssueCommentCache()
	items, err := normalizeItems(ctx, client, owner, name, issues, workflows, commentCache)
	if err != nil {
		return sourceData{}, err
	}
	fmt.Fprintf(os.Stderr, "normalized %d work items\n", len(items))
	costRuns, err := fetchCostRunsFromGHAWLogs(ctx, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load gh aw logs cost data: %v\n", err)
	}
	issueCostRuns, err := fetchIssueAICCostRuns(ctx, client, owner, name, issues, sourceWorkflowsByIssue(items), commentCache, newWorkflowMatcher(workflows))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load issue/PR AIC cost data: %v\n", err)
	}
	costRuns = append(costRuns, issueCostRuns...)
	fmt.Fprintf(os.Stderr, "loaded %d cost observations\n", len(costRuns))
	return sourceData{Items: items, Workflows: workflows, CostRuns: costRuns}, nil
}

func sourceDataArtifactsExist(outDir string) bool {
	for _, name := range []string{"items.json", "workflows.json", "cost_runs.json"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			return false
		}
	}
	return true
}

func readSourceData(outDir string) (sourceData, error) {
	var data sourceData
	if err := readJSONFile(filepath.Join(outDir, "items.json"), &data.Items); err != nil {
		return data, err
	}
	if err := readJSONFile(filepath.Join(outDir, "workflows.json"), &data.Workflows); err != nil {
		return data, err
	}
	if err := readJSONFile(filepath.Join(outDir, "cost_runs.json"), &data.CostRuns); err != nil {
		return data, err
	}
	return data, nil
}

func readJSONFile(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(target)
}

func validateSourceDataRepo(repo string, data sourceData) error {
	for _, item := range data.Items {
		if item.Repo != "" && item.Repo != repo {
			return fmt.Errorf("items.json contains repo %q", item.Repo)
		}
	}
	return nil
}

func rankOptions() impactscore.RankOptions {
	return impactscore.DefaultRankOptions()
}

func (c githubClient) fetchIssues(ctx context.Context, owner, repo, state string, maxItems int) ([]githubIssue, error) {
	issues := []githubIssue{}
	for page := 1; len(issues) < maxItems; page++ {
		endpoint := fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&sort=updated&direction=desc&per_page=100&page=%d", githubAPI, owner, repo, state, page)
		var pageIssues []githubIssue
		if err := c.getJSON(ctx, endpoint, &pageIssues); err != nil {
			return nil, err
		}
		if len(pageIssues) == 0 {
			break
		}
		remaining := maxItems - len(issues)
		if len(pageIssues) > remaining {
			pageIssues = pageIssues[:remaining]
		}
		issues = append(issues, pageIssues...)
		if len(pageIssues) < 100 {
			break
		}
	}
	return issues, nil
}

func (c githubClient) fetchPR(ctx context.Context, owner, repo string, number int) (githubPullRequest, error) {
	var pr githubPullRequest
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", githubAPI, owner, repo, number)
	return pr, c.getJSON(ctx, endpoint, &pr)
}

func (c githubClient) fetchPRFiles(ctx context.Context, owner, repo string, number int) ([]githubPRFile, error) {
	files := []githubPRFile{}
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=100&page=%d", githubAPI, owner, repo, number, page)
		var pageFiles []githubPRFile
		if err := c.getJSON(ctx, endpoint, &pageFiles); err != nil {
			return nil, err
		}
		if len(pageFiles) == 0 {
			break
		}
		files = append(files, pageFiles...)
		if len(pageFiles) < 100 {
			break
		}
	}
	return files, nil
}

func (c githubClient) fetchRecentIssueComments(ctx context.Context, owner, repo string, number, commentCount int) ([]githubComment, error) {
	if commentCount <= 0 {
		return nil, nil
	}
	lastPage := (commentCount + 99) / 100
	firstPage := max(lastPage-maxIssueCommentPages+1, 1)
	comments := []githubComment{}
	for page := firstPage; page <= lastPage; page++ {
		endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=100&page=%d", githubAPI, owner, repo, number, page)
		var pageComments []githubComment
		if err := c.getJSON(ctx, endpoint, &pageComments); err != nil {
			return nil, err
		}
		comments = append(comments, pageComments...)
	}
	return comments, nil
}

func (c githubClient) fetchWorkflowRun(ctx context.Context, owner, repo, runID string) (githubWorkflowRun, error) {
	var run githubWorkflowRun
	endpoint := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%s", githubAPI, owner, repo, runID)
	return run, c.getJSON(ctx, endpoint, &run)
}

func (c githubClient) getJSON(ctx context.Context, endpoint string, target any) error {
	var lastErr error
	for attempt := 0; attempt <= len(githubAPIRetryDelays); attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		res, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < len(githubAPIRetryDelays) {
				if waitErr := waitBeforeGitHubAPIRetry(ctx, githubAPIRetryDelays[attempt]); waitErr != nil {
					return waitErr
				}
				continue
			}
			return err
		}

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			defer res.Body.Close()
			return json.NewDecoder(res.Body).Decode(target)
		}

		lastErr = githubAPIError{StatusCode: res.StatusCode, Status: res.Status, Endpoint: endpoint}
		res.Body.Close()
		if !isTransientGitHubAPIStatus(res.StatusCode) || attempt == len(githubAPIRetryDelays) {
			return lastErr
		}
		if waitErr := waitBeforeGitHubAPIRetry(ctx, githubAPIRetryDelays[attempt]); waitErr != nil {
			return waitErr
		}
	}
	return lastErr
}

func isTransientGitHubAPIStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func sleepBeforeGitHubAPIRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizeItems(ctx context.Context, client githubClient, owner, repo string, issues []githubIssue, workflows []impactscore.WorkflowDefinition, commentCache *issueCommentCache) ([]impactscore.WorkItem, error) {
	items := make([]impactscore.WorkItem, len(issues))
	if len(issues) == 0 {
		return items, nil
	}
	runCache := newWorkflowRunCache()
	matcher := newWorkflowMatcher(workflows)
	workerCount := min(workItemFetchConcurrency, len(issues))
	fmt.Fprintf(os.Stderr, "normalizing %d work items with %d workers...\n", len(issues), workerCount)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	completed := 0

	reportProgress := func() {
		progressMu.Lock()
		defer progressMu.Unlock()
		completed++
		if completed%normalizeProgressEvery == 0 || completed == len(issues) {
			fmt.Fprintf(os.Stderr, "normalized %d/%d work items\n", completed, len(issues))
		}
	}

	for range workerCount {
		wg.Go(func() {
			for index := range jobs {
				item, err := normalizeItem(ctx, client, owner, repo, issues[index], workflows, matcher, commentCache, runCache)
				if err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
				items[index] = item
				reportProgress()
			}
		})
	}

sendLoop:
	for index := range issues {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func normalizeItem(ctx context.Context, client githubClient, owner, repo string, issue githubIssue, workflows []impactscore.WorkflowDefinition, matcher workflowMatcher, commentCache *issueCommentCache, runCache *workflowRunCache) (impactscore.WorkItem, error) {
	comments, err := issueCommentsForAttribution(ctx, client, owner, repo, issue, commentCache)
	if err != nil {
		return impactscore.WorkItem{}, err
	}
	provenanceText := issueProvenanceText(issue.Body, comments)
	runReferences := sourceWorkflowRunReferences(ctx, client, owner, repo, provenanceText, workflows, runCache)
	sourceWorkflowNames := matcher.sourceWorkflows(issue.Title, provenanceText)
	sourceWorkflowPaths := []string{}
	sourceWorkflowRuns := []string{}
	for _, reference := range runReferences {
		if reference.Name != "" {
			sourceWorkflowNames = append(sourceWorkflowNames, canonicalWorkflowName(reference.Name, workflows))
		}
		if reference.SourcePath != "" {
			sourceWorkflowPaths = append(sourceWorkflowPaths, reference.SourcePath)
		}
		if reference.RunURL != "" {
			sourceWorkflowRuns = append(sourceWorkflowRuns, reference.RunURL)
		}
	}
	labels := labelNames(issue.Labels)
	item := impactscore.WorkItem{
		Repo:                owner + "/" + repo,
		Number:              issue.Number,
		Type:                "issue",
		Title:               issue.Title,
		State:               issue.State,
		Labels:              labels,
		Measures:            map[string]float64{impactscore.MeasureComments: float64(issue.Comments)},
		Dimensions:          map[string][]string{},
		SourceWorkflows:     uniqueSorted(sourceWorkflowNames),
		SourceWorkflowPaths: uniqueSorted(sourceWorkflowPaths),
		SourceWorkflowRuns:  uniqueSorted(sourceWorkflowRuns),
		ContextSignals:      contextSignals(strings.Join([]string{issue.Title, provenanceText, strings.Join(labels, " ")}, "\n")),
		ContextEvidence:     compactEvidence(issue.Title, issue.Body),
	}
	if len(item.SourceWorkflowPaths) > 0 {
		item.Dimensions["source_workflow_path"] = item.SourceWorkflowPaths
	}
	if len(item.SourceWorkflowRuns) > 0 {
		item.Dimensions["source_workflow_run"] = item.SourceWorkflowRuns
	}
	if issue.Milestone != nil && issue.Milestone.Title != "" {
		item.Dimensions["milestone"] = []string{issue.Milestone.Title}
	}
	item.ChangeType = inferChangeType(issue.Title, item.Labels, item.ContextSignals)

	if issue.PullRequest != nil {
		item.Type = "pr"
		pr, files, err := fetchPREnrichment(ctx, client, owner, repo, issue.Number)
		if err != nil {
			return impactscore.WorkItem{}, err
		}
		setWorkItemStateReason(&item, pullRequestStateReason(issue.State, pr))
		enrichPRItem(&item, pr, files)
	} else {
		setWorkItemStateReason(&item, issue.StateReason)
	}
	return item, nil
}

func fetchPREnrichment(ctx context.Context, client githubClient, owner, repo string, number int) (githubPullRequest, []githubPRFile, error) {
	var pr githubPullRequest
	var files []githubPRFile
	var prErr error
	var filesErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pr, prErr = client.fetchPR(ctx, owner, repo, number)
	}()
	go func() {
		defer wg.Done()
		files, filesErr = client.fetchPRFiles(ctx, owner, repo, number)
	}()
	wg.Wait()
	if prErr != nil {
		return githubPullRequest{}, nil, prErr
	}
	if filesErr != nil {
		return githubPullRequest{}, nil, filesErr
	}
	return pr, files, nil
}

func enrichPRItem(item *impactscore.WorkItem, pr githubPullRequest, files []githubPRFile) {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Filename)
	}
	areas := map[string]bool{}
	components := map[string]bool{}
	for _, path := range paths {
		areas[topLevelArea(path)] = true
		components[componentForPath(path)] = true
		if isSensitivePath(path) {
			item.SensitivePathCount++
		}
	}
	item.Areas = sortedSet(areas)
	item.Components = sortedSet(components)
	item.ComponentCount = len(item.Components)
	item.ChangedFiles = pr.ChangedFiles
	item.Measures[impactscore.MeasureChangedFiles] = float64(pr.ChangedFiles)
	item.Measures[impactscore.MeasureAdditions] = float64(pr.Additions)
	item.Measures[impactscore.MeasureDeletions] = float64(pr.Deletions)
	item.Measures[impactscore.MeasureCommits] = float64(pr.Commits)
	item.Measures[impactscore.MeasureReviewComments] = float64(pr.ReviewComments)
	item.Measures[impactscore.MeasureTopLevelAreaCount] = float64(len(item.Areas))
	item.Measures[impactscore.MeasureComponentCount] = float64(item.ComponentCount)
	item.Measures[impactscore.MeasureSensitivePathCount] = float64(item.SensitivePathCount)
	item.Measures[impactscore.MeasureRuntimeFileCount] = float64(countPaths(paths, isRuntimePath))
	item.Measures[impactscore.MeasureTestFileCount] = float64(countPaths(paths, isTestPath))
	item.Measures[impactscore.MeasureDocsFileCount] = float64(countPaths(paths, isDocsPath))
	item.Measures[impactscore.MeasureWorkflowFileCount] = float64(countPaths(paths, isAgenticWorkflowPath))
}

func setWorkItemStateReason(item *impactscore.WorkItem, reason string) {
	reason = normalizeLifecycleValue(reason)
	if reason == "" {
		return
	}
	item.StateReason = reason
	if item.Dimensions == nil {
		item.Dimensions = map[string][]string{}
	}
	item.Dimensions[impactscore.DimensionStateReason] = uniqueSorted(append(item.Dimensions[impactscore.DimensionStateReason], reason))
}

func pullRequestStateReason(issueState string, pr githubPullRequest) string {
	if pr.Merged || strings.TrimSpace(pr.MergedAt) != "" {
		return "merged"
	}
	if normalizeLifecycleValue(issueState) == "closed" {
		return "closed_unmerged"
	}
	if pr.Draft {
		return "draft"
	}
	return ""
}

func normalizeLifecycleValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (c githubClient) fetchWorkflows(ctx context.Context, owner, repo string) ([]impactscore.WorkflowDefinition, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/contents/.github/workflows", githubAPI, owner, repo)
	var entries []githubContentEntry
	if err := c.getJSON(ctx, endpoint, &entries); err != nil {
		var apiErr githubAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	availablePaths := map[string]bool{}
	for _, entry := range entries {
		availablePaths[entry.Path] = true
	}
	workflows := []impactscore.WorkflowDefinition{}
	for _, entry := range entries {
		if entry.Type != "file" || !isAgenticWorkflowLockFile(entry.Name) {
			continue
		}
		sourcePath := workflowSourcePath(entry.Path, availablePaths)
		text, err := c.fetchContentText(ctx, entry)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, workflowDefinitionFromText(entry.Name, entry.Path, sourcePath, text))
	}
	sort.SliceStable(workflows, func(i, j int) bool { return workflows[i].Name < workflows[j].Name })
	return workflows, nil
}

func (c githubClient) fetchContentText(ctx context.Context, entry githubContentEntry) (string, error) {
	var file githubContentEntry
	if err := c.getJSON(ctx, entry.URL, &file); err != nil {
		return "", err
	}
	if file.Encoding == "base64" {
		content := strings.ReplaceAll(file.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return "", fmt.Errorf("decode %s: %w", entry.Path, err)
		}
		return string(decoded), nil
	}
	return file.Content, nil
}

func workflowDefinitionFromText(name, path, sourcePath, text string) impactscore.WorkflowDefinition {
	workflowName := extractScalar(text, "name")
	if workflowName == "" {
		workflowName = workflowNameFromPath(firstNonEmpty(path, name))
	}
	return impactscore.WorkflowDefinition{
		Name:        workflowName,
		Aliases:     workflowDefinitionAliases(name, path, sourcePath, text),
		Path:        path,
		SourcePath:  firstNonEmpty(sourcePath, path),
		TitlePrefix: firstNonEmpty(extractScalar(text, "title-prefix"), extractScalar(text, "title_prefix")),
	}
}

func workflowDefinitionAliases(name, path, sourcePath, text string) []string {
	aliases := []string{name, workflowNameFromPath(path), workflowNameFromPath(sourcePath), extractScalar(text, "tracker-id"), extractScalar(text, "tracker_id")}
	return uniqueSorted(aliases)
}

func isAgenticWorkflowLockFile(name string) bool {
	return strings.HasSuffix(name, ".lock.yml") || strings.HasSuffix(name, ".lock.yaml")
}

func fetchCostRunsFromGHAWLogs(ctx context.Context, token string) ([]impactscore.WorkflowCostRun, error) {
	cmd := exec.CommandContext(ctx, "gh", "aw", "logs", "--json", "-c", strconv.Itoa(defaultCostRunCount))
	if token != "" {
		cmd.Env = append(os.Environ(), "GH_TOKEN="+token, "GITHUB_TOKEN="+token)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if len(output) == 0 {
		if err != nil {
			return nil, fmt.Errorf("gh aw logs: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, nil
	}

	runs, parseErr := parseGHAWLogsCostRuns(output)
	if parseErr != nil {
		return nil, parseErr
	}
	if err != nil && len(runs) == 0 {
		return nil, fmt.Errorf("gh aw logs: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return runs, nil
}

func parseGHAWLogsCostRuns(data []byte) ([]impactscore.WorkflowCostRun, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse gh aw logs JSON: %w", err)
	}
	runs := []impactscore.WorkflowCostRun{}
	for _, row := range anySlice(payload["runs"]) {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		workflow := firstString(m, "workflow_name", "workflow", "name")
		if workflow == "" {
			continue
		}
		runs = append(runs, impactscore.WorkflowCostRun{
			Workflow:      workflow,
			RunID:         firstString(m, "run_id", "database_id", "id", "number"),
			RunURL:        firstString(m, "run_url", "html_url", "url"),
			AICCost:       firstNumber(m, "aic", "total_aic", "ai_credits_total", "estimated_cost", "cost_usd"),
			TokenUsage:    firstNumber(m, "token_usage", "total_tokens"),
			Turns:         firstNumber(m, "turns"),
			ActionMinutes: firstNumber(m, "action_minutes"),
			Errors:        firstNumber(m, "error_count", "errors"),
			Source:        "gh aw logs",
		})
	}
	return runs, nil
}

var (
	aicMentionPattern          = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*AIC\b`)
	actionRunURLPattern        = regexp.MustCompile(`https://github\.com/[^\s)]+/[^\s)]+/actions/runs/([0-9]+)`)
	agenticWorkflowPattern     = regexp.MustCompile(`(?i)gh-aw-agentic-workflow:\s*([^,\n]+)`)
	generatedByWorkflowPattern = regexp.MustCompile(`(?i)Generated (?:by|from) \[([^\]]+)\]`)
	generatedWorkflowPattern   = regexp.MustCompile(`(?i)Generated from \[([^\]]+)\]`)
	linkedWorkflowPattern      = regexp.MustCompile(`(?i)\*\*Workflow:\*\*\s*\[([^\]]+)\]`)
)

func fetchIssueAICCostRuns(ctx context.Context, client githubClient, owner, repo string, issues []githubIssue, itemWorkflows map[int][]string, commentCache *issueCommentCache, matcher workflowMatcher) ([]impactscore.WorkflowCostRun, error) {
	runGroups := make([][]impactscore.WorkflowCostRun, len(issues))
	if len(issues) == 0 {
		return nil, nil
	}
	workerCount := min(workItemFetchConcurrency, len(issues))
	fmt.Fprintf(os.Stderr, "scanning recent issue/PR comments for AIC cost observations with %d workers...\n", workerCount)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	completed := 0

	reportProgress := func() {
		progressMu.Lock()
		defer progressMu.Unlock()
		completed++
		if completed%normalizeProgressEvery == 0 || completed == len(issues) {
			fmt.Fprintf(os.Stderr, "scanned %d/%d work items for AIC comments\n", completed, len(issues))
		}
	}

	for range workerCount {
		wg.Go(func() {
			for index := range jobs {
				runs, err := issueAICCostRuns(ctx, client, owner, repo, issues[index], itemWorkflows[issues[index].Number], commentCache, matcher)
				if err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
				runGroups[index] = runs
				reportProgress()
			}
		})
	}

sendLoop:
	for index := range issues {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runs := []impactscore.WorkflowCostRun{}
	for _, group := range runGroups {
		runs = append(runs, group...)
	}
	return runs, nil
}

func issueAICCostRuns(ctx context.Context, client githubClient, owner, repo string, issue githubIssue, issueWorkflows []string, commentCache *issueCommentCache, matcher workflowMatcher) ([]impactscore.WorkflowCostRun, error) {
	if len(issueWorkflows) == 0 {
		issueWorkflows = matcher.sourceWorkflows(issue.Title, issue.Body)
	}
	runs := parseIssueTextAICCostRuns(issue.Body, issueWorkflows, "issue body AIC")
	if issue.Comments == 0 {
		return runs, nil
	}
	comments, err := cachedIssueComments(ctx, client, owner, repo, issue, commentCache)
	if err != nil {
		return runs, err
	}
	for _, comment := range comments {
		runs = append(runs, parseIssueTextAICCostRuns(comment.Body, issueWorkflows, "issue comment AIC")...)
	}
	return runs, nil
}

func parseIssueTextAICCostRuns(text string, sourceWorkflows []string, source string) []impactscore.WorkflowCostRun {
	runs := []impactscore.WorkflowCostRun{}
	for line := range strings.SplitSeq(text, "\n") {
		matches := aicMentionPattern.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		workflow := firstNonEmpty(workflowNameFromAICLine(line), singleWorkflow(sourceWorkflows))
		if workflow == "" {
			continue
		}
		runURL, runID := actionRunFromText(line)
		for _, match := range matches {
			value, err := strconv.ParseFloat(match[1], 64)
			if err != nil || value <= 0 {
				continue
			}
			runs = append(runs, impactscore.WorkflowCostRun{Workflow: workflow, RunID: runID, RunURL: runURL, AICCost: value, Source: source})
		}
	}
	return runs
}

func workflowNameFromAICLine(line string) string {
	for _, pattern := range []*regexp.Regexp{generatedWorkflowPattern, linkedWorkflowPattern} {
		match := pattern.FindStringSubmatch(line)
		if len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

func actionRunFromText(text string) (string, string) {
	match := actionRunURLPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return "", ""
	}
	return match[0], match[1]
}

func issueCommentsForAttribution(ctx context.Context, client githubClient, owner, repo string, issue githubIssue, commentCache *issueCommentCache) ([]githubComment, error) {
	if hasAttributionFooterSignals(issue.Body) {
		return nil, nil
	}
	return cachedIssueComments(ctx, client, owner, repo, issue, commentCache)
}

func cachedIssueComments(ctx context.Context, client githubClient, owner, repo string, issue githubIssue, commentCache *issueCommentCache) ([]githubComment, error) {
	if issue.Comments == 0 {
		return nil, nil
	}
	if comments, ok := commentCache.get(issue.Number); ok {
		return comments, nil
	}
	comments, err := client.fetchRecentIssueComments(ctx, owner, repo, issue.Number, issue.Comments)
	if err != nil {
		return nil, err
	}
	commentCache.set(issue.Number, comments)
	return comments, nil
}

func issueProvenanceText(body string, comments []githubComment) string {
	parts := []string{body}
	for _, comment := range comments {
		parts = append(parts, comment.Body)
	}
	return strings.Join(parts, "\n")
}

func hasAttributionFooterSignals(text string) bool {
	return actionRunURLPattern.MatchString(text) || generatedWorkflowPattern.MatchString(text) || linkedWorkflowPattern.MatchString(text) || strings.Contains(strings.ToLower(text), "gh-aw-agentic-workflow:")
}

func sourceWorkflowRunReferences(ctx context.Context, client githubClient, owner, repo, text string, workflows []impactscore.WorkflowDefinition, cache *workflowRunCache) []workflowRunReference {
	references := []workflowRunReference{}
	seen := map[string]bool{}
	for _, match := range actionRunURLPattern.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		reference, shouldFetch := cache.reserve(match[1], match[0])
		if shouldFetch {
			run, err := client.fetchWorkflowRun(ctx, owner, repo, match[1])
			if err == nil {
				reference = workflowRunReferenceFromRun(run, match[0], workflows)
				cache.set(match[1], reference)
			}
		}
		references = append(references, reference)
	}
	return references
}

func workflowRunReferenceFromRun(run githubWorkflowRun, fallbackRunURL string, workflows []impactscore.WorkflowDefinition) workflowRunReference {
	sourcePath := workflowSourcePathForRun(run.Path, workflows)
	return workflowRunReference{
		Name:       workflowNameForRun(run, sourcePath, workflows),
		SourcePath: sourcePath,
		RunURL:     firstNonEmpty(run.HTMLURL, fallbackRunURL),
	}
}

func workflowNameForRun(run githubWorkflowRun, sourcePath string, workflows []impactscore.WorkflowDefinition) string {
	if run.Name != "" {
		return run.Name
	}
	for _, workflow := range workflows {
		if sameWorkflowPath(run.Path, workflow.Path) || sameWorkflowPath(sourcePath, workflow.SourcePath) || sameWorkflowPath(run.Path, workflow.SourcePath) {
			return workflow.Name
		}
	}
	return workflowNameFromPath(firstNonEmpty(sourcePath, run.Path))
}

func workflowSourcePathForRun(runPath string, workflows []impactscore.WorkflowDefinition) string {
	availablePaths := map[string]bool{}
	for _, workflow := range workflows {
		if workflow.Path != "" {
			availablePaths[workflow.Path] = true
		}
		if workflow.SourcePath != "" {
			availablePaths[workflow.SourcePath] = true
		}
	}
	return workflowSourcePath(runPath, availablePaths)
}

func workflowSourcePath(path string, availablePaths map[string]bool) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	for _, candidate := range workflowSourcePathCandidates(path) {
		if availablePaths[candidate] {
			return candidate
		}
	}
	return path
}

func workflowSourcePathCandidates(path string) []string {
	for _, suffix := range []string{".lock.yml", ".lock.yaml"} {
		if source, ok := strings.CutSuffix(path, suffix); ok {
			return []string{source + ".md"}
		}
	}
	return nil
}

func isGeneratedWorkflowLock(path string) bool {
	return strings.HasSuffix(path, ".lock.yml") || strings.HasSuffix(path, ".lock.yaml")
}

func sameWorkflowPath(left, right string) bool {
	return strings.TrimSpace(left) != "" && strings.TrimSpace(left) == strings.TrimSpace(right)
}

func workflowNameFromPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	name := filepath.Base(path)
	name = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name, ".lock.yml"), ".lock.yaml"), ".yaml"), ".yml")
	return strings.TrimSuffix(name, ".md")
}

func sourceWorkflowsByIssue(items []impactscore.WorkItem) map[int][]string {
	lookup := map[int][]string{}
	for _, item := range items {
		lookup[item.Number] = item.SourceWorkflows
	}
	return lookup
}

func singleWorkflow(workflows []string) string {
	if len(workflows) == 1 {
		return workflows[0]
	}
	return ""
}

func writeArtifacts(outDir string, result output, reportFormat string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	artifacts := map[string]any{
		"summary.json":        map[string]any{"repo": result.Repo, "generated_at": result.GeneratedAt, "items": len(result.Items), "workflows": len(result.Workflows), "workflow_ranks": len(result.WorkflowRanks)},
		"items.json":          result.Items,
		"workflows.json":      result.Workflows,
		"cost_runs.json":      result.CostRuns,
		"item_ranks.json":     result.ItemRanks,
		"features.json":       result.Features,
		"workflow_ranks.json": result.WorkflowRanks,
		"graph_nodes.json":    result.GraphNodes,
		"graph_edges.json":    result.GraphEdges,
	}
	for name, value := range artifacts {
		if err := writeJSON(filepath.Join(outDir, name), value); err != nil {
			return err
		}
	}
	reportFormat = normalizedReportFormat(reportFormat)
	if reportFormat == "" {
		return errors.New("invalid report format")
	}
	if err := removeUnselectedReportArtifacts(outDir, reportFormat); err != nil {
		return err
	}
	if reportFormat == "text" {
		if err := os.WriteFile(filepath.Join(outDir, "impact_score_report.txt"), renderTextReport(result), 0o644); err != nil {
			return err
		}
	}
	if reportFormat == "html" {
		if err := writeUIArtifact(outDir, result); err != nil {
			return err
		}
	}
	if err := writeCSVArtifacts(outDir, result); err != nil {
		return err
	}
	return nil
}

func normalizedReportFormat(reportFormat string) string {
	if reportFormat == "" {
		return "text"
	}
	switch reportFormat {
	case "text", "html":
		return reportFormat
	default:
		return ""
	}
}

func removeUnselectedReportArtifacts(outDir, reportFormat string) error {
	if reportFormat != "text" {
		if err := removeArtifactIfExists(filepath.Join(outDir, "impact_score_report.txt")); err != nil {
			return err
		}
	}
	if reportFormat != "html" {
		if err := removeArtifactIfExists(filepath.Join(outDir, "impact_score_dashboard.html")); err != nil {
			return err
		}
	}
	return nil
}

func removeArtifactIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func renderTextReport(result output) []byte {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Impact Score Report\n")
	fmt.Fprintf(&builder, "Repo: %s\n", result.Repo)
	fmt.Fprintf(&builder, "Generated: %s\n", result.GeneratedAt)
	fmt.Fprintf(&builder, "Items: %d\n", len(result.Items))
	fmt.Fprintf(&builder, "Workflows: %d\n\n", len(result.Workflows))

	builder.WriteString("Workflow Ranking\n")
	workflowRanks := append([]impactscore.WorkflowRank{}, result.WorkflowRanks...)
	if len(workflowRanks) == 0 {
		builder.WriteString("No workflow impact data available.\n")
	} else {
		if len(workflowRanks) > 20 {
			workflowRanks = workflowRanks[:20]
		}
		for index, rank := range workflowRanks {
			fmt.Fprintf(&builder, "%2d. %s | zone=%s | impact=%.1f | AIC=%.1f | linked=%d\n", index+1, rank.Workflow, rank.ActionZone, rank.AttributedImpactScore, rank.TotalAICCost, rank.LinkedItems)
		}
	}

	builder.WriteString("\nHigh Impact Work Items\n")
	itemRanks := append([]impactscore.ItemRank{}, result.ItemRanks...)
	sort.SliceStable(itemRanks, func(i, j int) bool {
		if itemRanks[i].ImpactScore != itemRanks[j].ImpactScore {
			return itemRanks[i].ImpactScore > itemRanks[j].ImpactScore
		}
		if itemRanks[i].ItemType != itemRanks[j].ItemType {
			return itemRanks[i].ItemType < itemRanks[j].ItemType
		}
		return itemRanks[i].Number < itemRanks[j].Number
	})
	if len(itemRanks) == 0 {
		builder.WriteString("No work item impact data available.\n")
	} else {
		if len(itemRanks) > 20 {
			itemRanks = itemRanks[:20]
		}
		for index, rank := range itemRanks {
			explanation := formatScoreExplanation(rank.ScoreExplanation)
			if explanation != "" {
				explanation = " | " + explanation
			}
			fmt.Fprintf(&builder, "%2d. %s#%d %.1f | %s%s | %s | %s\n", index+1, rank.ItemType, rank.Number, rank.ImpactScore, rank.ScoreSource, explanation, formatSourceWorkflows(rank.SourceWorkflows), rank.Title)
		}
	}
	return []byte(builder.String())
}

func formatScoreExplanation(explanation impactscore.ScoreExplanation) string {
	parts := []string{}
	if explanation.PolicyPath != "" {
		policy := "policy=" + explanation.PolicyPath
		if explanation.PolicyVersion != 0 {
			policy += "@v" + strconv.Itoa(explanation.PolicyVersion)
		}
		if explanation.PolicySHA256 != "" {
			policy += "#" + shortPolicySHA(explanation.PolicySHA256)
		}
		parts = append(parts, policy)
	}
	if len(explanation.MatchedRules) > 0 {
		parts = append(parts, "rules="+strings.Join(explanation.MatchedRules, ";"))
	}
	return strings.Join(parts, " ")
}

func shortPolicySHA(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func formatSourceWorkflows(sourceWorkflows []string) string {
	if len(sourceWorkflows) == 0 {
		return "no linked agentic workflow"
	}
	return "agentic workflow: " + strings.Join(sourceWorkflows, ";")
}

func writeJSON(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeCSVArtifacts(outDir string, result output) error {
	files := map[string][][]string{
		"items.csv":          itemCSVRows(result),
		"workflow_ranks.csv": workflowRankCSVRows(result.WorkflowRanks),
		"cost_runs.csv":      costRunCSVRows(result.CostRuns),
	}
	for name, rows := range files {
		if err := writeCSV(filepath.Join(outDir, name), rows); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV(path string, rows [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.WriteAll(rows); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func itemCSVRows(result output) [][]string {
	rows := [][]string{{"repo", "number", "type", "state", "state_reason", "title", "url", "impact_score", "score_source", "score_policy_path", "score_policy_version", "score_policy_sha256", "matched_rules", "source_workflows", "labels", "components", "areas", "context_signals", "changed_files", "sensitive_path_count", "component_count", "dimensions_json", "measures_json"}}
	ranks := itemRankByKey(result.ItemRanks)
	features := result.Features
	if len(features) == 0 {
		features = featuresFromItems(result.Items)
	}
	for _, feature := range features {
		item := feature.Item
		rank := ranks[workItemKey(item.Type, item.Number)]
		rows = append(rows, []string{
			item.Repo,
			strconv.Itoa(item.Number),
			item.Type,
			item.State,
			item.StateReason,
			item.Title,
			githubWorkItemURL(item.Repo, item.Type, item.Number),
			formatCSVFloat(rank.ImpactScore),
			rank.ScoreSource,
			rank.ScoreExplanation.PolicyPath,
			formatOptionalInt(rank.ScoreExplanation.PolicyVersion),
			rank.ScoreExplanation.PolicySHA256,
			strings.Join(rank.ScoreExplanation.MatchedRules, ";"),
			strings.Join(rank.SourceWorkflows, ";"),
			strings.Join(item.Labels, ";"),
			strings.Join(item.Components, ";"),
			strings.Join(item.Areas, ";"),
			strings.Join(item.ContextSignals, ";"),
			strconv.Itoa(item.ChangedFiles),
			strconv.Itoa(item.SensitivePathCount),
			strconv.Itoa(item.ComponentCount),
			jsonCSVString(feature.Dimensions),
			jsonCSVString(feature.Measures),
		})
	}
	return rows
}

func formatOptionalInt(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func workflowRankCSVRows(ranks []impactscore.WorkflowRank) [][]string {
	rows := [][]string{{"workflow", "action_zone", "attributed_impact_score", "linked_items", "open_items", "released_items", "run_count", "costed_run_count", "total_aic_cost", "average_aic_cost_per_run", "impact_per_aic", "impact_per_thousand_aic", "aic_per_impact", "total_tokens", "total_turns", "action_minutes", "errors", "cost_sources"}}
	for _, rank := range ranks {
		rows = append(rows, []string{
			rank.Workflow,
			rank.ActionZone,
			formatCSVFloat(rank.AttributedImpactScore),
			strconv.Itoa(rank.LinkedItems),
			strconv.Itoa(rank.OpenItems),
			strconv.Itoa(rank.ReleasedItems),
			strconv.Itoa(rank.RunCount),
			strconv.Itoa(rank.CostedRunCount),
			formatCSVFloat(rank.TotalAICCost),
			formatCSVFloat(rank.AverageAICCostPerRun),
			formatCSVFloat(rank.ImpactPerAIC),
			formatCSVFloat(rank.ImpactPerThousandAIC),
			formatCSVFloat(rank.AICPerImpact),
			formatCSVFloat(rank.TotalTokens),
			formatCSVFloat(rank.TotalTurns),
			formatCSVFloat(rank.ActionMinutes),
			formatCSVFloat(rank.Errors),
			strings.Join(rank.CostSources, ";"),
		})
	}
	return rows
}

func costRunCSVRows(costRuns []impactscore.WorkflowCostRun) [][]string {
	rows := [][]string{{"workflow", "run_id", "run_url", "aic_cost", "token_usage", "turns", "action_minutes", "errors", "source"}}
	for _, run := range costRuns {
		rows = append(rows, []string{run.Workflow, run.RunID, run.RunURL, formatCSVFloat(run.AICCost), formatCSVFloat(run.TokenUsage), formatCSVFloat(run.Turns), formatCSVFloat(run.ActionMinutes), formatCSVFloat(run.Errors), run.Source})
	}
	return rows
}

func itemRankByKey(ranks []impactscore.ItemRank) map[string]impactscore.ItemRank {
	lookup := map[string]impactscore.ItemRank{}
	for _, rank := range ranks {
		lookup[workItemKey(rank.ItemType, rank.Number)] = rank
	}
	return lookup
}

func featuresFromItems(items []impactscore.WorkItem) []impactscore.ItemFeatures {
	features := make([]impactscore.ItemFeatures, 0, len(items))
	for _, item := range items {
		features = append(features, impactscore.ItemFeatures{Item: item, Dimensions: item.Dimensions, Measures: item.Measures})
	}
	return features
}

func workItemKey(itemType string, number int) string {
	return itemType + "#" + strconv.Itoa(number)
}

func jsonCSVString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func formatCSVFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func sortedNodes(graph impactscore.Graph) []impactscore.Node {
	nodes := make([]impactscore.Node, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes = append(nodes, node)
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be owner/name, got %q", repo)
	}
	return parts[0], parts[1], nil
}

func resolveToken(ctx context.Context) string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func labelNames(labels []githubLabel) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		names = append(names, label.Name)
	}
	sort.Strings(names)
	return names
}

func contextSignals(text string) []string {
	signals := []string{}
	signalTerms := map[string][]string{
		"security":            {"security", "vulnerab", "cve", "credential", "secret", "sandbox"},
		"failure":             {"fail", "broken", "error", "crash", "regression"},
		"feature_request":     {"support", "feature", "enable", "add"},
		"performance":         {"performance", "latency", "faster", "speed"},
		"test_coverage":       {"test", "coverage", "assertion", "fixture"},
		"maintainability":     {"duplicate", "deduplicate", "refactor", "cleanup"},
		"workflow_automation": {"workflow", "github actions", "agentic", "artifact"},
		"dependency":          {"dependency", "dependencies", "package", "upgrade"},
		"cost":                {"token", "cost", "credits", "usage report"},
		"documentation":       {"docs", "readme", "documentation"},
		"user_impact":         {"user", "developer", "customer", "impact", "blocked"},
	}
	lowered := strings.ToLower(text)
	for signal, terms := range signalTerms {
		for _, term := range terms {
			if strings.Contains(lowered, term) {
				signals = append(signals, signal)
				break
			}
		}
	}
	sort.Strings(signals)
	return signals
}

func compactEvidence(title, body string) []string {
	evidence := []string{}
	if strings.TrimSpace(title) != "" {
		evidence = append(evidence, "title: "+shortText(title, 180))
	}
	if strings.TrimSpace(body) != "" {
		evidence = append(evidence, "body: "+shortText(body, 240))
	}
	return evidence
}

func newWorkflowMatcher(workflows []impactscore.WorkflowDefinition) workflowMatcher {
	matcher := workflowMatcher{aliases: workflowAliasMapForDefinitions(workflows), workflows: make([]workflowMatchCandidate, 0, len(workflows))}
	for _, workflow := range workflows {
		matcher.workflows = append(matcher.workflows, workflowMatchCandidate{
			Name:          workflow.Name,
			TitlePrefix:   workflow.TitlePrefix,
			BracketPrefix: strings.ToLower("[" + workflow.Name + "]"),
		})
	}
	return matcher
}

func (matcher workflowMatcher) sourceWorkflows(title, body string) []string {
	sources := []string{}
	for _, match := range agenticWorkflowPattern.FindAllStringSubmatch(body, -1) {
		sources = append(sources, strings.TrimSpace(match[1]))
	}
	for _, match := range generatedByWorkflowPattern.FindAllStringSubmatch(body, -1) {
		sources = append(sources, strings.TrimSpace(match[1]))
	}
	loweredTitle := strings.ToLower(title)
	for _, workflow := range matcher.workflows {
		if workflow.TitlePrefix != "" && strings.HasPrefix(title, workflow.TitlePrefix) {
			sources = append(sources, workflow.Name)
		}
		if strings.HasPrefix(loweredTitle, workflow.BracketPrefix) {
			sources = append(sources, workflow.Name)
		}
	}
	return canonicalizeWorkflowNamesWithAliases(sources, matcher.aliases)
}

func sourceWorkflows(title, body string, workflows []impactscore.WorkflowDefinition) []string {
	return newWorkflowMatcher(workflows).sourceWorkflows(title, body)
}

func canonicalizeSourceDataWorkflows(data sourceData) sourceData {
	aliases := workflowAliasMapForSourceData(data)
	for index := range data.Workflows {
		originalName := data.Workflows[index].Name
		data.Workflows[index].Name = canonicalWorkflowNameWithAliases(originalName, aliases)
		data.Workflows[index].Aliases = uniqueSorted(append(data.Workflows[index].Aliases, originalName))
	}
	for index := range data.Items {
		data.Items[index].SourceWorkflows = canonicalizeWorkflowNamesWithAliases(data.Items[index].SourceWorkflows, aliases)
	}
	data.CostRuns = canonicalizeCostRunWorkflows(data.CostRuns, aliases)
	return data
}

func canonicalizeWorkflowNamesWithAliases(names []string, aliases map[string]string) []string {
	canonical := make([]string, 0, len(names))
	for _, name := range names {
		canonical = append(canonical, canonicalWorkflowNameWithAliases(name, aliases))
	}
	return uniqueSorted(canonical)
}

func canonicalizeCostRunWorkflows(runs []impactscore.WorkflowCostRun, aliases map[string]string) []impactscore.WorkflowCostRun {
	for index := range runs {
		runs[index].Workflow = canonicalWorkflowNameWithAliases(runs[index].Workflow, aliases)
	}
	return runs
}

func canonicalWorkflowName(name string, workflows []impactscore.WorkflowDefinition) string {
	return canonicalWorkflowNameWithAliases(name, workflowAliasMapForDefinitions(workflows))
}

func canonicalWorkflowNameWithAliases(name string, aliases map[string]string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if canonical := aliases[strings.ToLower(trimmed)]; canonical != "" {
		return canonical
	}
	return trimmed
}

func workflowAliasMapForSourceData(data sourceData) map[string]string {
	aliases := workflowAliasMapForDefinitions(data.Workflows)
	for _, run := range data.CostRuns {
		addWorkflowAlias(aliases, workflowSlug(run.Workflow), run.Workflow)
	}
	addCooccurringWorkflowAliases(aliases, data.Items)
	return aliases
}

func addCooccurringWorkflowAliases(aliases map[string]string, items []impactscore.WorkItem) {
	type candidateCount struct {
		name  string
		count int
	}
	slugTotals := map[string]int{}
	candidates := map[string]map[string]int{}
	for _, item := range items {
		workflows := uniqueSorted(item.SourceWorkflows)
		for _, workflow := range workflows {
			if !isSlugLikeWorkflowName(workflow) {
				continue
			}
			slugKey := strings.ToLower(workflow)
			slugTotals[slugKey]++
			for _, candidate := range workflows {
				if workflowCanonicalScore(candidate) <= workflowCanonicalScore(workflow) {
					continue
				}
				if candidates[slugKey] == nil {
					candidates[slugKey] = map[string]int{}
				}
				candidates[slugKey][candidate]++
			}
		}
	}
	for slug, counts := range candidates {
		best := candidateCount{}
		for candidate, count := range counts {
			if count > best.count || (count == best.count && workflowCanonicalScore(candidate) > workflowCanonicalScore(best.name)) {
				best = candidateCount{name: candidate, count: count}
			}
		}
		if best.count >= 2 && best.count == slugTotals[slug] {
			addWorkflowAlias(aliases, slug, best.name)
		}
	}
}

func isSlugLikeWorkflowName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return trimmed != "" && trimmed == workflowSlug(trimmed) && strings.Contains(trimmed, "-")
}

func workflowAliasMapForDefinitions(workflows []impactscore.WorkflowDefinition) map[string]string {
	aliases := map[string]string{}
	for _, workflow := range workflows {
		for _, alias := range append([]string{workflow.Name, workflowNameFromPath(workflow.Path), workflowNameFromPath(workflow.SourcePath)}, workflow.Aliases...) {
			addWorkflowAlias(aliases, alias, workflow.Name)
		}
	}
	return aliases
}

func addWorkflowAlias(aliases map[string]string, alias, canonical string) {
	alias = strings.TrimSpace(alias)
	canonical = strings.TrimSpace(canonical)
	if alias == "" || canonical == "" {
		return
	}
	key := strings.ToLower(alias)
	if existing := aliases[key]; existing != "" && workflowCanonicalScore(existing) >= workflowCanonicalScore(canonical) {
		return
	}
	aliases[key] = canonical
}

func workflowCanonicalScore(name string) int {
	score := 0
	if strings.Contains(name, " ") {
		score++
	}
	for _, char := range name {
		if char >= 'A' && char <= 'Z' {
			score++
			break
		}
	}
	return score
}

func workflowSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var out strings.Builder
	lastDash := false
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			out.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func inferChangeType(title string, labels []string, signals []string) string {
	text := strings.ToLower(strings.Join(append(append([]string{title}, labels...), signals...), " "))
	switch {
	case strings.Contains(text, "security"):
		return "security"
	case strings.Contains(text, "performance") || strings.Contains(text, "latency"):
		return "performance"
	case strings.Contains(text, "fix") || strings.Contains(text, "bug") || strings.Contains(text, "failure"):
		return "fix"
	case strings.Contains(text, "feature") || strings.Contains(text, "enhancement"):
		return "feature"
	case strings.Contains(text, "docs") || strings.Contains(text, "documentation"):
		return "docs"
	case strings.Contains(text, "test") || strings.Contains(text, "coverage"):
		return "test"
	default:
		return ""
	}
}

func topLevelArea(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func componentForPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		switch parts[0] {
		case "src", "containers", "scripts", "tests", "docs", "docs-site", ".github", "pkg", "cmd", "internal":
			return parts[0] + "/" + parts[1]
		}
	}
	return topLevelArea(path)
}

func isSensitivePath(path string) bool {
	lowered := strings.ToLower(path)
	terms := []string{"auth", "token", "secret", "credential", "security", "proxy", "sandbox", "release", "workflow", "schema", "permission"}
	for _, term := range terms {
		if strings.Contains(lowered, term) {
			return true
		}
	}
	return false
}

func isRuntimePath(path string) bool {
	return strings.HasPrefix(path, "src/") || strings.HasPrefix(path, "pkg/") || strings.HasPrefix(path, "cmd/") || strings.HasPrefix(path, "internal/") || strings.HasPrefix(path, "containers/")
}

func isTestPath(path string) bool {
	lowered := strings.ToLower(path)
	return strings.Contains(lowered, "test") || strings.Contains(lowered, ".spec.")
}

func isDocsPath(path string) bool {
	lowered := strings.ToLower(path)
	return strings.HasPrefix(lowered, "docs/") || strings.HasSuffix(lowered, ".md") || strings.HasSuffix(lowered, ".mdx")
}

func isAgenticWorkflowPath(path string) bool {
	return strings.HasPrefix(path, ".github/workflows/") && isAgenticWorkflowLockFile(filepath.Base(path))
}

func countPaths(paths []string, predicate func(string) bool) int {
	count := 0
	for _, path := range paths {
		if predicate(path) {
			count++
		}
	}
	return count
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func sortedSet(values map[string]bool) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		if value != "" {
			items = append(items, value)
		}
	}
	sort.Strings(items)
	return items
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func extractScalar(text, key string) string {
	pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*:\s*["']?([^"'\n#]+)`)
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		case float64:
			return strconv.FormatInt(int64(typed), 10)
		}
	}
	return ""
}

func firstNumber(values map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case int:
			return float64(typed)
		case string:
			parsed, err := strconv.ParseFloat(strings.ReplaceAll(typed, ",", ""), 64)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func anySlice(value any) []any {
	if values, ok := value.([]any); ok {
		return values
	}
	return nil
}

func shortText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit-3]) + "..."
}
