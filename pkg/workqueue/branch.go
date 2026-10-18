package workqueue

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/githubapi"
)

const maxRetries = 5

var branchPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Branch uses an isolated Git object store for the ledger, independently of the checkout.
type Branch struct {
	Remote              string
	Name                string
	DependencyClients   map[string]*api.RESTClient
	DeliveryVerifier    DeliveryVerifier
	RemediationVerifier RemediationVerifier
	PolicyProposal      *Policy // Host-approved first-submit proposal; never replaces installed Policy.
	deliveryFailures    map[string]string
	client              *api.RESTClient
	ledger              *queueGit
	ledgerRequestFn     func(context.Context, string, string, any, any) error
	publicationNow      func() int64
}

func (b Branch) validate() error {
	if !repoPattern.MatchString(b.Remote) {
		return errors.New("repo must be owner/repo")
	}
	for part := range strings.SplitSeq(b.Remote, "/") {
		if part == "." || part == ".." {
			return errors.New("repo must be owner/repo")
		}
	}
	if !branchPattern.MatchString(b.Name) || strings.Contains(b.Name, "..") ||
		strings.Contains(b.Name, "//") || strings.HasSuffix(b.Name, ".") || strings.HasSuffix(b.Name, "/") {
		return errors.New("invalid queue branch")
	}
	for part := range strings.SplitSeq(b.Name, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return errors.New("invalid queue branch")
		}
	}
	return nil
}

func (b Branch) withClient() (Branch, error) {
	if err := b.validate(); err != nil {
		return b, err
	}
	if b.client == nil {
		client, err := api.NewRESTClient(githubapi.ClientOptions("", ""))
		if err != nil {
			return b, err
		}
		b.client = client
	}
	return b, nil
}

func (b Branch) request(ctx context.Context, method, endpoint string, body, response any) error {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	return b.client.DoWithContext(ctx, method, path.Join("repos", b.Remote, endpoint), bytes.NewReader(data), response)
}

func hasStatus(err error, status int) bool {
	var apiError *api.HTTPError
	return errors.As(err, &apiError) && apiError.StatusCode == status
}

var errInvalidHeadReference = errors.New("invalid branch reference")

type gitReference struct {
	Ref    string `json:"ref"`
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

func (b Branch) resolveHeadRef(ctx context.Context, name string) (string, error) {
	var ref gitReference
	if err := b.request(ctx, http.MethodGet, "git/ref/heads/"+name, nil, &ref); err != nil {
		return "", err
	}
	if ref.Ref != "refs/heads/"+name || !revisionPattern.MatchString(ref.Object.SHA) ||
		strings.Trim(ref.Object.SHA, "0") == "" {
		return "", errInvalidHeadReference
	}
	return ref.Object.SHA, nil
}

type branchSnapshot struct {
	head string
	tree string
	data []byte
}

func (b Branch) read(ctx context.Context) ([]QueueCommit, branchSnapshot, error) {
	var ref gitReference
	err := b.ledgerRequest(ctx, http.MethodGet, "git/ref/heads/"+b.Name, nil, &ref)
	head := ref.Object.SHA
	if err == nil && (ref.Ref != "refs/heads/"+b.Name || !revisionPattern.MatchString(head) || strings.Trim(head, "0") == "") {
		err = errInvalidHeadReference
	}
	if err != nil {
		if hasStatus(err, http.StatusNotFound) {
			if b.ledger != nil {
				return nil, branchSnapshot{}, nil
			}
			// A missing repository or missing read permission is not an empty queue.
			var repository any
			err = b.client.DoWithContext(ctx, http.MethodGet, "repos/"+b.Remote, nil, &repository)
			return nil, branchSnapshot{}, err
		}
		if errors.Is(err, errInvalidHeadReference) {
			err = errors.New("invalid queue branch reference")
		}
		return nil, branchSnapshot{}, err
	}
	return b.readAt(ctx, head, true, true)
}

func (b Branch) readAt(ctx context.Context, head string, verify, recover bool) ([]QueueCommit, branchSnapshot, error) {
	snapshot := branchSnapshot{head: head}
	var commit struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if err := b.ledgerRequest(ctx, http.MethodGet, "git/commits/"+snapshot.head, nil, &commit); err != nil {
		return nil, snapshot, err
	}
	snapshot.tree = commit.Tree.SHA
	if snapshot.tree == "" {
		return nil, snapshot, errors.New("queue commit has no tree")
	}
	transactions, data, err := b.readLog(ctx, snapshot.tree)
	snapshot.data = data
	recovered := false
	var protocolError *ProtocolError
	if recover && errors.As(err, &protocolError) && protocolError.Code == "checkpoint_invalid" {
		transactions, err = b.recoverDeploymentCheckpoint(ctx, head, transactions, err)
		recovered = err == nil
	}
	if err != nil {
		return nil, snapshot, err
	}
	if verify && !recovered {
		if err := b.verifyCheckpointHistory(ctx, snapshot.head, transactions); err != nil {
			return nil, snapshot, err
		}
	}
	return transactions, snapshot, nil
}

func (b Branch) verifyCheckpointHistory(ctx context.Context, head string, commits []QueueCommit) error {
	invalid := func() error {
		return queueError("checkpoint_invalid", "checkpoint does not match its Git parent history")
	}
	visited := map[string]struct{}{}
	for {
		if len(commits) == 0 {
			return nil
		}
		root, err := firstQueueCommit(commits)
		if err != nil {
			return err
		}
		if !isCheckpoint(root) {
			return nil
		}
		checkpoint, err := decodeCheckpointOperation(root)
		if err != nil {
			return invalid()
		}
		parent, previous, err := b.checkpointHistoryParent(ctx, head, root.ID, visited)
		if err != nil {
			return err
		}
		data, encodeErr := Serialize(previous)
		if parent != checkpoint.PriorGitSHA || encodeErr != nil || hashBytes(data) != checkpoint.HistorySHA256 {
			return invalid()
		}
		ordered, err := causalOrder(previous)
		if err != nil {
			return invalid()
		}
		if len(ordered) == 0 {
			return invalid()
		}
		var tip string
		for _, commit := range ordered {
			tip = commit.ID
		}
		if tip != checkpoint.PriorTip {
			return invalid()
		}
		reconstructed, err := reconstructCheckpoint(previous, parent, root.Actor, root.At, checkpoint.StateSHA256)
		if err != nil {
			return invalid()
		}
		reconstructedRoot, err := firstQueueCommit(reconstructed)
		if err != nil {
			return invalid()
		}
		expected, err := decodeCheckpointOperation(reconstructedRoot)
		if err != nil || expected.StateSHA256 != checkpoint.StateSHA256 {
			return invalid()
		}
		head, commits = parent, previous
	}
}

func (b Branch) readLog(ctx context.Context, treeSHA string) ([]QueueCommit, []byte, error) {
	var tree struct {
		Truncated bool `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}
	if err := b.ledgerRequest(ctx, http.MethodGet, "git/trees/"+treeSHA, nil, &tree); err != nil {
		return nil, nil, err
	}
	if tree.Truncated {
		return nil, nil, errors.New("queue tree is truncated")
	}
	for _, entry := range tree.Tree {
		if entry.Path != FileName {
			continue
		}
		if entry.Type != "blob" || (entry.Mode != "100644" && entry.Mode != "100755") || entry.SHA == "" {
			return nil, nil, errors.New("queue log must be a regular file")
		}
		var blob struct {
			Encoding string `json:"encoding"`
			Content  string `json:"content"`
		}
		if err := b.ledgerRequest(ctx, http.MethodGet, "git/blobs/"+entry.SHA, nil, &blob); err != nil {
			return nil, nil, err
		}
		if blob.Encoding != "base64" {
			return nil, nil, errors.New("unsupported queue blob encoding")
		}
		data, err := base64.StdEncoding.DecodeString(blob.Content)
		if err != nil {
			return nil, nil, err
		}
		transactions, err := Parse(data)
		if err != nil {
			return nil, nil, err
		}
		for _, commit := range transactions {
			if !strings.EqualFold(commit.Actor.Repository, b.Remote) {
				return nil, nil, queueError("repository_scope_invalid", "ledger actor belongs to another repository")
			}
		}
		_, err = Replay(transactions)
		return transactions, data, err
	}
	return nil, nil, errors.New("queue branch is missing " + FileName)
}

func (b Branch) Read(ctx context.Context) ([]QueueCommit, error) {
	b, cleanup, err := b.withLedger(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	transactions, snapshot, err := b.read(ctx)
	if err == nil && snapshot.head == "" {
		return nil, queueError("queue_missing", "queue branch %s does not exist", b.Name)
	}
	return transactions, err
}

func (b Branch) publish(ctx context.Context, snapshot branchSnapshot, next []QueueCommit) (bool, error) {
	if _, err := Replay(next); err != nil {
		return false, err
	}
	data, err := Serialize(next)
	if err != nil {
		return false, err
	}
	treeBody := map[string]any{
		"tree": []map[string]string{{"path": FileName, "mode": "100644", "type": "blob", "content": string(data)}},
	}
	parents := []string{}
	if snapshot.head != "" {
		treeBody["base_tree"] = snapshot.tree
		parents = append(parents, snapshot.head)
	}
	var tree struct {
		SHA string `json:"sha"`
	}
	if err := b.ledgerRequest(ctx, http.MethodPost, "git/trees", treeBody, &tree); err != nil {
		return false, err
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := b.ledgerRequest(ctx, http.MethodPost, "git/commits", map[string]any{
		"message": "Update work queue", "tree": tree.SHA, "parents": parents,
	}, &commit); err != nil {
		return false, err
	}
	if snapshot.head == "" {
		err = b.ledgerRequest(ctx, http.MethodPost, "git/refs", map[string]any{
			"ref": "refs/heads/" + b.Name, "sha": commit.SHA,
		}, new(any))
	} else {
		err = b.ledgerRequest(ctx, http.MethodPatch, "git/refs/heads/"+b.Name, map[string]any{
			"sha": commit.SHA, "force": false,
		}, new(any))
	}
	return hasStatus(err, http.StatusConflict) || hasStatus(err, http.StatusUnprocessableEntity), err
}

// Authenticate derives operator identity from the credential, not command flags,
// Git authors, submitted actor fields, or an untrusted Actions environment.
func (b Branch) Authenticate(ctx context.Context, role string) (Actor, error) {
	return b.authenticate(ctx, role, false)
}

// AuthenticateClear authorizes repository maintainers and administrators for
// clear operations, retaining the queue's administrator actor for audit records.
func (b Branch) AuthenticateClear(ctx context.Context) (Actor, error) {
	return b.authenticate(ctx, "administrator", true)
}

func (b Branch) authenticate(ctx context.Context, role string, allowMaintain bool) (Actor, error) {
	b, err := b.withClient()
	if err != nil {
		return Actor{}, err
	}
	if role != "administrator" && role != "producer" && role != "reconciler" {
		return Actor{}, queueError("actor_unauthorized", "native operator publisher cannot impersonate workflow roles")
	}
	var user struct {
		ID json.Number `json:"id"`
	}
	if err := b.client.DoWithContext(ctx, http.MethodGet, "user", nil, &user); err != nil {
		return Actor{}, err
	}
	var repository struct {
		FullName    string `json:"full_name"`
		Permissions struct {
			Push     bool `json:"push"`
			Admin    bool `json:"admin"`
			Maintain bool `json:"maintain"`
		} `json:"permissions"`
	}
	if err := b.client.DoWithContext(ctx, http.MethodGet, "repos/"+b.Remote, nil, &repository); err != nil {
		return Actor{}, err
	}
	canAdminister := repository.Permissions.Admin || allowMaintain && repository.Permissions.Maintain
	if !decimalIdentity(user.ID.String()) || !strings.EqualFold(repository.FullName, b.Remote) ||
		!repository.Permissions.Push || role == "administrator" && !canAdminister {
		if allowMaintain {
			return Actor{}, queueError("actor_unauthorized", "authenticated repository maintain or admin permission is required to clear a queue")
		}
		return Actor{}, queueError("actor_unauthorized", "authenticated repository writer/administrator permission is required")
	}
	return Actor{Role: role, Principal: user.ID.String(), Repository: repository.FullName}, nil
}

type Publication struct {
	Commit   *QueueCommit `json:"commit,omitempty"`
	Decision Decision     `json:"decision"`
	Changed  bool         `json:"changed"`
}

func (b Branch) defaultPolicy(ctx context.Context, _ string) (Policy, error) {
	return b.policyFromConfig(ctx)
}

// PolicyFromConfig resolves scheduling and AW worker approval from this target
// repository's verified immutable default ref, never the current local checkout.
func (b Branch) PolicyFromConfig(ctx context.Context) (Policy, error) {
	b, err := b.withClient()
	if err != nil {
		return Policy{}, err
	}
	return b.policyFromConfig(ctx)
}

func (b Branch) policyFromConfig(ctx context.Context) (Policy, error) {
	repository, revision, err := b.verifiedDefaultReference(ctx)
	if err != nil {
		return Policy{}, err
	}
	b.Remote = repository
	return b.approvedConfigPolicy(ctx, revision)
}

func extending(previous, current []QueueCommit) bool {
	oldOrder, err := causalOrder(previous)
	if err != nil {
		return false
	}
	newOrder, err := causalOrder(current)
	if err != nil {
		return false
	}
	root, err := firstQueueCommit(newOrder)
	if err != nil || len(oldOrder) == 0 {
		return false
	}
	if isCheckpoint(root) {
		op, err := decodeCheckpointOperation(root)
		data, encodeErr := Serialize(oldOrder)
		var tip string
		for _, commit := range oldOrder {
			tip = commit.ID
		}
		if err == nil &&
			encodeErr == nil && op.HistorySHA256 == hashBytes(data) &&
			op.PriorTip == tip {
			reconstructed, err := reconstructCheckpoint(oldOrder, op.PriorGitSHA, root.Actor, root.At, op.StateSHA256)
			if err != nil {
				return false
			}
			reconstructedRoot, err := firstQueueCommit(reconstructed)
			if err != nil {
				return false
			}
			expected, err := decodeCheckpointOperation(reconstructedRoot)
			return err == nil && expected.StateSHA256 == op.StateSHA256
		}
	}
	if len(newOrder) < len(oldOrder) {
		return false
	}
	return slices.EqualFunc(oldOrder, newOrder[:len(oldOrder)], func(old, current QueueCommit) bool {
		return sameJSON(old, current)
	})
}

// Publish retains only authenticated logical intent across CAS retries. It
// discards selections, handles, charges and clocks and recovers uncertain ref
// acknowledgments by stable request identity before constructing another grant.
func (b Branch) Publish(ctx context.Context, actor Actor, request Request) (Publication, error) {
	return b.publishRequest(ctx, actor, request, nil)
}

// PublishFromSnapshot rejects replacement ledgers before applying intent selected
// from an existing snapshot. Same-ledger appends and verified checkpoints remain
// eligible, including across CAS retries and uncertain acknowledgments.
// Snapshot cancellations allow repository maintainers as well as administrators;
// other request kinds retain their existing authorization requirements.
func (b Branch) PublishFromSnapshot(ctx context.Context, actor Actor, request Request, selected []QueueCommit) (Publication, error) {
	if len(selected) == 0 {
		return Publication{}, queueError("queue_missing", "snapshot publication requires an existing queue")
	}
	return b.publishRequest(ctx, actor, request, selected)
}

func (b Branch) publishRequest(ctx context.Context, actor Actor, request Request, previous []QueueCommit) (Publication, error) {
	b, err := b.authenticatePublication(ctx, actor, request, len(previous) > 0 && request.Kind == "cancel_work")
	if err != nil {
		return Publication{}, err
	}
	b, cleanup, err := b.withLedger(ctx)
	if err != nil {
		return Publication{}, err
	}
	defer cleanup()
	last := previous
	var pendingErr error
	for attempt := range maxRetries {
		prefix, err := b.readPublicationPrefix(ctx, actor, request, last)
		if err != nil {
			return Publication{}, err
		}
		candidate, err := b.preparePublicationCandidate(ctx, actor, request, prefix, pendingErr)
		if err != nil {
			return Publication{}, err
		}
		if candidate.settled {
			return candidate.publication, nil
		}
		// Regenerate immediately with fresh time to exclude expired Work from the next
		// candidate; no ref write was attempted here, so contention backoff is unnecessary.
		if expiringGrant(prefix.state, candidate.publication.Commit.Operations, b.publicationTime()) {
			if prefix.snapshot.head != "" {
				last = prefix.commits
			}
			pendingErr = queueError("work_expired", "a selected Work expired before publication; refresh the fair prefix")
			continue
		}
		_, err = b.publish(ctx, prefix.snapshot, candidate.commits)
		if err == nil {
			candidate.publication.Changed = true
			return candidate.publication, nil
		}
		if ctx.Err() != nil {
			return Publication{}, ctx.Err()
		}
		// A lost ref response may be an accepted request, not a failed grant.
		pendingErr = err
		if prefix.snapshot.head != "" {
			last = prefix.commits
		}
		if attempt == maxRetries-1 {
			return b.recoverPublicationAcknowledgment(ctx, actor, request, last, err)
		}
		if err := waitForPublicationRetry(ctx, attempt); err != nil {
			return Publication{}, err
		}
	}
	return Publication{}, errors.New("queue publication exhausted retries")
}

func (b Branch) authenticatePublication(ctx context.Context, actor Actor, request Request, allowMaintain bool) (Branch, error) {
	b, err := b.withClient()
	if err != nil {
		return b, err
	}
	authenticated, err := b.authenticate(ctx, actor.Role, allowMaintain)
	if err != nil || !sameJSON(authenticated, actor) {
		if err != nil {
			return b, err
		}
		return b, queueError("actor_unauthorized", "request actor differs from authenticated operator")
	}
	b.Remote = authenticated.Repository
	if err := validateRequestOrigin(actor, request); err != nil {
		return b, err
	}
	if err := validateRequestRole(actor, request.Kind); err != nil {
		return b, err
	}
	return b, nil
}

type publicationPrefix struct {
	commits  []QueueCommit
	snapshot branchSnapshot
	initial  *QueueCommit
	state    Projection
}

func (b Branch) readPublicationPrefix(ctx context.Context, actor Actor, request Request, previous []QueueCommit) (publicationPrefix, error) {
	commits, snapshot, err := b.read(ctx)
	if err != nil {
		return publicationPrefix{}, err
	}
	if err := b.checkPublicationExtension(ctx, previous, commits); err != nil {
		return publicationPrefix{}, err
	}
	var initial *QueueCommit
	if snapshot.head == "" {
		genesis, err := b.initialSubmissionCommit(ctx, actor, request)
		if err != nil {
			return publicationPrefix{}, err
		}
		commits = []QueueCommit{genesis}
		initial = &genesis
	}
	state, err := Replay(commits)
	if err != nil {
		return publicationPrefix{}, err
	}
	if b.PolicyProposal != nil && snapshot.head != "" && request.Kind == "submit" {
		if _, recovered := state.Requests[request.ID]; !recovered && !sameJSON(*b.PolicyProposal, *state.Policy) {
			return publicationPrefix{}, queueError("policy_proposal_mismatch", "submission proposal differs from installed Policy")
		}
	}
	return publicationPrefix{commits: commits, snapshot: snapshot, initial: initial, state: state}, nil
}

func (b Branch) checkPublicationExtension(ctx context.Context, previous, current []QueueCommit) error {
	if len(previous) == 0 {
		return nil
	}
	for !extending(previous, current) {
		root, err := firstQueueCommit(current)
		if err != nil || !isCheckpoint(root) {
			return queueError("ledger_nonextending", "queue history was deleted or rewritten during publication")
		}
		// The current read verified checkpoint ancestry. Walk its bound history
		// when intervening appends or compactions separate it from the selection.
		checkpoint, err := decodeCheckpointOperation(root)
		if err != nil {
			return err
		}
		current, _, err = b.readAt(ctx, checkpoint.PriorGitSHA, false, false)
		if err != nil {
			return err
		}
	}
	return nil
}

type publicationCandidate struct {
	commits     []QueueCommit
	publication Publication
	settled     bool
}

func (b Branch) publicationTime() int64 {
	if b.publicationNow != nil {
		return b.publicationNow()
	}
	return time.Now().UnixMilli()
}

func (b Branch) preparePublicationCandidate(ctx context.Context, actor Actor, request Request, prefix publicationPrefix, pendingErr error) (publicationCandidate, error) {
	if existing, ok := prefix.state.Requests[request.ID]; ok && prefix.initial == nil {
		if existing.Request.Fingerprint != request.Fingerprint || !sameJSON(existing.Actor, actor) {
			return publicationCandidate{}, queueError("request_reuse", "request identity has different accepted meaning")
		}
		_, commit, decision, err := BuildCandidate(prefix.commits, actor, request, b.publicationTime())
		return publicationCandidate{publication: Publication{Commit: commit, Decision: decision}, settled: true}, err
	}
	if err := validateRequestRole(actor, request.Kind); err != nil {
		return publicationCandidate{}, err
	}
	if pendingErr != nil && !hasStatus(pendingErr, http.StatusConflict) &&
		!hasStatus(pendingErr, http.StatusUnprocessableEntity) &&
		(hasStatus(pendingErr, http.StatusUnauthorized) || hasStatus(pendingErr, http.StatusForbidden)) {
		return publicationCandidate{}, pendingErr
	}
	if prefix.initial == nil {
		var evidenceErr error
		if request.Kind == "deployment" {
			evidenceErr = b.verifyNativeDeploymentEvidence(ctx, prefix.state, request)
		} else {
			evidenceErr = b.verifyRequestEvidence(ctx, prefix.state, actor, request)
		}
		if evidenceErr != nil {
			return publicationCandidate{}, evidenceErr
		}
	}
	var observations []Observation
	if request.Kind == "dispatch_next" {
		var params DispatchParameters
		if err := json.Unmarshal(request.Parameters, &params); err != nil {
			return publicationCandidate{}, err
		}
		var err error
		observations, err = b.refreshForDispatch(ctx, prefix.state, params.Pool, request.ID)
		if err != nil {
			return publicationCandidate{}, err
		}
	}
	var candidate publicationCandidate
	var err error
	if prefix.initial != nil {
		candidate.commits, candidate.publication.Commit = prefix.commits, prefix.initial
		candidate.publication.Decision = Decision{Tip: prefix.state.Tip, Operations: prefix.initial.Operations, Assignments: []Assignment{}}
	} else {
		candidate.commits, candidate.publication.Commit, candidate.publication.Decision, err =
			buildCandidateWithObservations(prefix.commits, actor, request, b.publicationTime(), observations)
	}
	if err != nil {
		return publicationCandidate{}, err
	}
	candidate.settled = candidate.publication.Commit == nil
	return candidate, nil
}

func (b Branch) recoverPublicationAcknowledgment(ctx context.Context, actor Actor, request Request, previous []QueueCommit, pendingErr error) (Publication, error) {
	current, _, readErr := b.read(ctx)
	if readErr == nil {
		if err := b.checkPublicationExtension(ctx, previous, current); err != nil {
			return Publication{}, err
		}
	}
	if readErr == nil && len(current) > 0 {
		projection, replayErr := Replay(current)
		if replayErr == nil {
			if existing, ok := projection.Requests[request.ID]; ok &&
				existing.Request.Fingerprint == request.Fingerprint && sameJSON(existing.Actor, actor) {
				_, committed, decision, err := BuildCandidate(current, actor, request, b.publicationTime())
				return Publication{Commit: committed, Decision: decision}, err
			}
		}
	}
	return Publication{}, fmt.Errorf("queue publication unresolved after %d attempts: %w", maxRetries, pendingErr)
}

func waitForPublicationRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(50*(1<<attempt)+rand.Intn(50)) * time.Millisecond
	timer := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
