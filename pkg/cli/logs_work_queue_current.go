package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workqueue"
)

// Receipts deliberately omit payloads, actor credentials, and raw evidence.
// They describe validated snapshot facts, not a live branch or API recheck.
type WorkQueueCurrentSnapshot struct {
	Role            string                      `json:"role"`
	PolicyInstalled bool                        `json:"policy_installed"`
	Tip             string                      `json:"tip"`
	PolicyEpoch     string                      `json:"policy_epoch"`
	CapturedAt      int64                       `json:"captured_at"`
	CommitCount     int                         `json:"commit_count"`
	Paused          bool                        `json:"admission_paused"`
	GrantsPaused    bool                        `json:"grants_paused"`
	Assignment      *WorkQueueAssignmentReceipt `json:"assignment,omitempty"`
	Diagnostics     *WorkQueueDiagnostics       `json:"diagnostics,omitempty"`
}

type WorkQueueAssignmentReceipt = workqueue.AssignmentReceipt

type WorkQueueClaimReceipt = workqueue.ClaimReceipt

type WorkQueueFinishReceipt struct {
	IntentID string `json:"intent_id"`
	Handle   string `json:"claim_handle"`
	Outcome  string `json:"outcome"`
	KeepOpen bool   `json:"keep_open,omitempty"`
}

func readBoundedWorkQueueArtifact(path string, limit int64) (data []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, errors.New("could not close the work queue artifact; check filesystem access before retrying"))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("work queue artifact exceeds bounded input limit")
	}
	data, err = io.ReadAll(io.LimitReader(file, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("work queue artifact exceeds bounded input limit")
	}
	return data, err
}

func decodeClosedWorkQueueObject(data []byte, required, optional []string, target any) error {
	canonical, err := workqueue.Canonical(data)
	if err != nil {
		return currentWorkQueueDiagnosticError("artifact", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &fields); err != nil {
		return err
	}
	if fields == nil {
		return errors.New("work queue artifact must be an object")
	}
	for _, key := range required {
		if _, present := fields[key]; !present {
			return fmt.Errorf("work queue artifact missing %q", key)
		}
	}
	for key := range fields {
		if !slices.Contains(required, key) && !slices.Contains(optional, key) {
			return errors.New("work queue artifact has an unknown field")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid typed work queue artifact")
	}
	return nil
}

func currentWorkQueueDiagnosticError(stage string, err error) error {
	code, _, _ := strings.Cut(err.Error(), ":")
	if code == "" || len(code) > 64 || strings.ContainsFunc(code, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_'
	}) {
		code = "invalid_protocol"
	}
	// Schema/codec error details may include untrusted payload values.
	return fmt.Errorf("current snapshot %s rejected (%s)", stage, code)
}

type currentWorkQueueEnvelope struct {
	Version        int                   `json:"version"`
	Role           json.RawMessage       `json:"role"`
	SHA            *string               `json:"sha"`
	Worker         *workqueue.Assignment `json:"worker"`
	TransactionLog *string               `json:"transactionLog"`
	CapturedAt     *int64                `json:"captured_at"`
	Origin         *workqueue.Actor      `json:"origin"`
	VisibleWorkIDs json.RawMessage       `json:"visible_work_ids"`
}

func parseCurrentWorkQueueSnapshot(data []byte) (*WorkQueueSnapshot, error) {
	envelope, state, role, err := replayCurrentWorkQueueSnapshot(data)
	if err != nil {
		return nil, err
	}
	current := &WorkQueueCurrentSnapshot{
		Role: role, PolicyInstalled: state.Policy != nil,
		Tip: state.Tip, PolicyEpoch: state.PolicyEpoch, CapturedAt: *envelope.CapturedAt,
		CommitCount: state.Stats.Transactions, Paused: state.AdmissionPaused, GrantsPaused: state.GrantsPaused,
	}
	if envelope.Worker != nil {
		current.Assignment, err = currentWorkQueueAssignmentReceipt(state, *envelope.Worker, *envelope.Origin)
		if err != nil {
			return nil, err
		}
	}
	current.Diagnostics, err = capturedWorkQueueDiagnostics(state, envelope)
	if err != nil {
		return nil, currentWorkQueueDiagnosticError("diagnostics", err)
	}
	return &WorkQueueSnapshot{Version: 3, SHA: envelope.SHA, Current: current}, nil
}

func replayCurrentWorkQueueSnapshot(data []byte) (currentWorkQueueEnvelope, workqueue.Projection, string, error) {
	var envelope currentWorkQueueEnvelope
	var state workqueue.Projection
	if err := decodeClosedWorkQueueObject(data,
		[]string{"version", "sha", "worker", "transactionLog", "captured_at", "origin"},
		[]string{"visible_work_ids", "role"}, &envelope); err != nil {
		return envelope, state, "", err
	}
	if err := validateCurrentWorkQueueEnvelope(envelope); err != nil {
		return envelope, state, "", err
	}
	role, err := currentWorkQueueSnapshotRole(envelope)
	if err != nil {
		return envelope, state, "", err
	}
	if role == "observer" && envelope.SHA == nil && *envelope.TransactionLog == "" {
		return envelope, state, role, nil
	}
	commits, err := workqueue.Parse([]byte(*envelope.TransactionLog))
	if err != nil {
		return envelope, state, role, currentWorkQueueDiagnosticError("ledger", err)
	}
	state, err = workqueue.ReplayAt(commits, *envelope.CapturedAt)
	if err != nil {
		return envelope, state, role, currentWorkQueueDiagnosticError("chain", err)
	}
	if state.Policy == nil {
		return envelope, state, role, errors.New("current snapshot requires an installed policy")
	}
	if role == "observer" && envelope.Origin.Repository != state.Repository {
		return envelope, state, role, errors.New("current observer snapshot repository differs from captured queue")
	}
	return envelope, state, role, nil
}

func validateCurrentWorkQueueEnvelope(envelope currentWorkQueueEnvelope) error {
	if envelope.Version != 3 || envelope.TransactionLog == nil || envelope.CapturedAt == nil ||
		*envelope.CapturedAt < 0 || *envelope.CapturedAt > workqueue.MaxTimestamp ||
		(envelope.SHA != nil && !validWorkQueueReceiptID(*envelope.SHA)) || envelope.Origin == nil ||
		!validWorkQueueNativeID(envelope.Origin.Principal) {
		return errors.New("invalid current work queue snapshot")
	}
	if len(envelope.VisibleWorkIDs) != 0 {
		var ids []*string
		if err := json.Unmarshal(envelope.VisibleWorkIDs, &ids); err != nil ||
			ids == nil || slices.Contains(ids, nil) {
			return errors.New("invalid current work queue snapshot visibility")
		}
	}
	return nil
}

func currentWorkQueueSnapshotRole(envelope currentWorkQueueEnvelope) (string, error) {
	role := "dispatcher"
	if envelope.Worker != nil {
		role = "worker"
	}
	if len(envelope.Role) != 0 {
		var declaredRole string
		if err := json.Unmarshal(envelope.Role, &declaredRole); err != nil ||
			!slices.Contains([]string{"observer", "dispatcher", "worker"}, declaredRole) {
			return "", errors.New("invalid current work queue snapshot role")
		}
		role = declaredRole
	}
	if (role == "worker") != (envelope.Worker != nil) {
		return "", errors.New("current snapshot role conflicts with immutable assignment")
	}
	unassignedOriginRole := "dispatcher"
	if role == "observer" {
		unassignedOriginRole = "producer"
		if !validWorkQueueReceiptID(envelope.Origin.Repository) ||
			envelope.Origin.DispatchID != "" || envelope.Origin.ClaimHandle != "" {
			return "", errors.New("current observer snapshot requires bounded producer provenance")
		}
	}
	if envelope.Worker == nil && (envelope.Origin.Role != unassignedOriginRole ||
		envelope.Origin.RunAttempt < 1 || envelope.Origin.RunAttempt > 4096 ||
		!validWorkQueueNativeID(envelope.Origin.RunID) || !validWorkQueueReceiptID(envelope.Origin.Workflow)) {
		return "", errors.New("current snapshot requires bounded native run and workflow provenance")
	}
	return role, nil
}

func currentWorkQueueAssignmentReceipt(state workqueue.Projection, assignment workqueue.Assignment, origin workqueue.Actor) (*WorkQueueAssignmentReceipt, error) {
	if err := workqueue.ValidateAssignment(state, assignment); err != nil {
		return nil, currentWorkQueueDiagnosticError("assignment", err)
	}
	dispatch := state.Dispatches[assignment.DispatchID]
	if dispatch.Run == nil || dispatch.Run.RunID != origin.RunID ||
		dispatch.Run.RunAttempt != 1 || origin.RunAttempt != 1 ||
		origin.Role != "worker" || origin.DispatchID != dispatch.DispatchID ||
		dispatch.Run.Repository != origin.Repository || dispatch.Run.Workflow != origin.Workflow ||
		dispatch.Run.Principal != origin.Principal {
		return nil, errors.New("current snapshot origin differs from durable assignment binding")
	}
	// The embedded Assignment.ToReceipt projects identity; enrich lifecycle facts from the captured state.
	receipt := dispatch.ToReceipt()
	receipt.State, receipt.Released = dispatch.State, dispatch.Released
	for i, member := range receipt.Claims {
		claim, work := state.Claims[member.ClaimID], state.Works[member.WorkID]
		receipt.Claims[i].State, receipt.Claims[i].Barrier = claim.State, work.Barrier
		if claim.KeepOpen {
			receipt.Claims[i].Barrier = claim.Barrier
		}
	}
	return &receipt, nil
}

func readCurrentWorkQueueFinishIntents(path string, snapshot *WorkQueueCurrentSnapshot) ([]WorkQueueFinishReceipt, error) {
	data, err := readBoundedWorkQueueArtifact(path, 4<<20)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read current work queue intents: %w", err)
	}
	var receipts []WorkQueueFinishReceipt
	ids, outcomes := map[string]string{}, map[string]string{}
	lines := 0
	for index, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		lines++
		if lines > 256 {
			return nil, errors.New("work queue finish intent exceeds 256 entries")
		}
		intent, canonical, err := decodeCurrentWorkQueueFinishIntent(line, snapshot, index+1)
		if err != nil {
			return nil, err
		}
		if previous, ok := ids[intent.IntentID]; ok {
			if previous != string(canonical) {
				return nil, errors.New("conflicting work queue intent identity")
			}
			continue
		}
		choice := fmt.Sprintf("%s:%t", intent.Outcome, intent.KeepOpen)
		if previous, ok := outcomes[intent.Handle]; ok && previous != choice {
			return nil, errors.New("conflicting per-Claim work queue finish intents")
		}
		ids[intent.IntentID], outcomes[intent.Handle] = string(canonical), choice
		receipts = append(receipts, intent)
	}
	return receipts, nil
}

func decodeCurrentWorkQueueFinishIntent(line []byte, snapshot *WorkQueueCurrentSnapshot, lineNumber int) (WorkQueueFinishReceipt, []byte, error) {
	var intent struct {
		Version    int    `json:"version"`
		IntentID   string `json:"intent_id"`
		Kind       string `json:"kind"`
		Handle     string `json:"claim_handle"`
		Parameters struct {
			Outcome  string `json:"outcome"`
			KeepOpen bool   `json:"keep_open"`
		} `json:"parameters"`
	}
	if err := decodeClosedWorkQueueObject(line, []string{"version", "intent_id", "kind", "parameters"}, []string{"claim_handle"}, &intent); err != nil {
		return WorkQueueFinishReceipt{}, nil, fmt.Errorf("finish intent line %d: %w", lineNumber, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return WorkQueueFinishReceipt{}, nil, fmt.Errorf("finish intent fields line %d: %w", lineNumber, err)
	}
	if _, supplied := fields["claim_handle"]; !supplied {
		if snapshot == nil || snapshot.Assignment == nil || len(snapshot.Assignment.Claims) != 1 {
			return WorkQueueFinishReceipt{}, nil, errors.New("finish intent requires a selector for the original multi-Claim assignment")
		}
		for _, claim := range snapshot.Assignment.Claims {
			intent.Handle = claim.Handle
		}
	}
	if err := decodeClosedWorkQueueObject(fields["parameters"], []string{"outcome"}, []string{"keep_open"}, &intent.Parameters); err != nil {
		return WorkQueueFinishReceipt{}, nil, fmt.Errorf("finish parameters line %d: %w", lineNumber, err)
	}
	var parameters map[string]json.RawMessage
	if err := json.Unmarshal(fields["parameters"], &parameters); err != nil {
		return WorkQueueFinishReceipt{}, nil, err
	}
	if keepOpen, present := parameters["keep_open"]; present && string(keepOpen) != "true" && string(keepOpen) != "false" {
		return WorkQueueFinishReceipt{}, nil, errors.New("keep_open requires a boolean")
	}
	if intent.Version != 3 || intent.Kind != "finish" || !validWorkQueueReceiptID(intent.IntentID) ||
		!validWorkQueueReceiptID(intent.Handle) || !slices.Contains([]string{"completed", "cancelled"}, intent.Parameters.Outcome) ||
		intent.Parameters.KeepOpen && intent.Parameters.Outcome != "completed" {
		return WorkQueueFinishReceipt{}, nil, fmt.Errorf("invalid current finish intent at line %d", lineNumber)
	}
	if snapshot == nil || snapshot.Assignment == nil ||
		!slices.ContainsFunc(snapshot.Assignment.Claims, func(c WorkQueueClaimReceipt) bool { return c.Handle == intent.Handle }) {
		return WorkQueueFinishReceipt{}, nil, errors.New("finish intent is foreign to original immutable assignment")
	}
	if !intent.Parameters.KeepOpen {
		normalized, err := normalizeCurrentWorkQueueFinishIntent(fields, parameters)
		if err != nil {
			return WorkQueueFinishReceipt{}, nil, fmt.Errorf("finish intent line %d: %w", lineNumber, err)
		}
		line = normalized
	}
	canonical, err := workqueue.Canonical(line)
	if err != nil {
		return WorkQueueFinishReceipt{}, nil, fmt.Errorf("finish intent line %d: %w", lineNumber, err)
	}
	return WorkQueueFinishReceipt{IntentID: intent.IntentID, Handle: intent.Handle, Outcome: intent.Parameters.Outcome, KeepOpen: intent.Parameters.KeepOpen}, canonical, nil
}

func normalizeCurrentWorkQueueFinishIntent(fields, parameters map[string]json.RawMessage) ([]byte, error) {
	delete(parameters, "keep_open")
	normalized, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	fields["parameters"] = normalized
	return json.Marshal(fields)
}

func validWorkQueueNativeID(value string) bool {
	return value != "" && len(value) <= 256 && !strings.HasPrefix(value, "0") &&
		!strings.ContainsFunc(value, func(r rune) bool { return r < '0' || r > '9' })
}

func validWorkQueueReceiptID(value string) bool {
	return value != "" && len(value) <= 256 && !strings.ContainsFunc(value, func(r rune) bool { return r < 32 || r == 127 })
}

func boundWorkQueueDisplay(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	runes := []rune(value)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes) + "…"
}
