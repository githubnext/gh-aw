package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var traceLog = logger.New("workflow:trace")

// TraceManifest contains metadata about the trace capture
type TraceManifest struct {
	RunID        string    `json:"run_id"`
	Workflow     string    `json:"workflow"`
	Engine       string    `json:"engine"`
	RepoSHA      string    `json:"repo_sha"`
	CreatedAt    time.Time `json:"created_at"`
	TraceVersion string    `json:"trace_version"`
}

// CheckpointKind defines the type of checkpoint
type CheckpointKind string

const (
	CheckpointToolCall    CheckpointKind = "tool_call"
	CheckpointPatch       CheckpointKind = "patch"
	CheckpointEval        CheckpointKind = "eval"
	CheckpointSafeOutput  CheckpointKind = "safe_output"
	CheckpointDecision    CheckpointKind = "decision"
	CheckpointRiskGate    CheckpointKind = "risk_gate"
)

// Checkpoint represents a single replayable step in agent execution
type Checkpoint struct {
	ID       string         `json:"id"`
	TS       time.Time      `json:"ts"`
	Kind     CheckpointKind `json:"kind"`
	Name     string         `json:"name"`
	RepoSHA  string         `json:"repo_sha"`
	ReqPath  string         `json:"req_path,omitempty"`
	RespPath string         `json:"resp_path,omitempty"`
	DiffPath string         `json:"diff_path,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToolRequest captures tool invocation request data
type ToolRequest struct {
	Tool      string         `json:"tool"`
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments"`
	Timestamp time.Time      `json:"timestamp"`
}

// ToolResponse captures tool invocation response data
type ToolResponse struct {
	Tool      string         `json:"tool"`
	Operation string         `json:"operation"`
	Success   bool           `json:"success"`
	Data      any            `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration"`
	Timestamp time.Time      `json:"timestamp"`
	// Redacted fields for security
	SensitiveFields []string `json:"sensitive_fields,omitempty"`
}

// DiffDelta represents changes introduced at a checkpoint
type DiffDelta struct {
	FilePath string   `json:"file_path"`
	Before   string   `json:"before_hash"`
	After    string   `json:"after_hash"`
	Lines    DiffStat `json:"lines"`
}

// DiffStat captures line changes
type DiffStat struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Context int `json:"context"`
}

// EvalResult captures test/lint/security scan results
type EvalResult struct {
	Type      string    `json:"type"` // test, lint, security, etc.
	Command   string    `json:"command"`
	ExitCode  int       `json:"exit_code"`
	Success   bool      `json:"success"`
	Duration  time.Duration `json:"duration"`
	Output    string    `json:"output,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SafeOutputRecord captures safe-output operations
type SafeOutputRecord struct {
	Type      string         `json:"type"` // pr, comment, issue, label
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	Success   bool           `json:"success"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// DecisionReceipt captures "why this step happened"
type DecisionReceipt struct {
	CheckpointID string   `json:"checkpoint_id"`
	Rationale    string   `json:"rationale"`
	Evidence     []string `json:"evidence"` // File paths, log sections
	Confidence   string   `json:"confidence"` // high, medium, low
	Alternatives []string `json:"alternatives,omitempty"`
}

// TraceCapture provides methods for capturing agent execution
type TraceCapture struct {
	manifest    TraceManifest
	checkpoints []Checkpoint
	counter     int
}

// NewTraceCapture creates a new trace capture instance
func NewTraceCapture(runID, workflow, engine, repoSHA string) *TraceCapture {
	return &TraceCapture{
		manifest: TraceManifest{
			RunID:        runID,
			Workflow:     workflow,
			Engine:       engine,
			RepoSHA:      repoSHA,
			CreatedAt:    time.Now().UTC(),
			TraceVersion: "v1",
		},
		checkpoints: make([]Checkpoint, 0),
		counter:     0,
	}
}

// RecordToolCall records a tool invocation checkpoint
func (tc *TraceCapture) RecordToolCall(tool, operation string, request *ToolRequest, response *ToolResponse) Checkpoint {
	tc.counter++
	checkpointID := fmt.Sprintf("c%03d", tc.counter)
	
	checkpoint := Checkpoint{
		ID:       checkpointID,
		TS:       time.Now().UTC(),
		Kind:     CheckpointToolCall,
		Name:     fmt.Sprintf("%s.%s", tool, operation),
		RepoSHA:  tc.manifest.RepoSHA,
		ReqPath:  fmt.Sprintf("tools/%s.request.json", checkpointID),
		RespPath: fmt.Sprintf("tools/%s.response.json", checkpointID),
		Metadata: map[string]any{
			"duration": response.Duration.String(),
			"success":  response.Success,
		},
	}
	
	tc.checkpoints = append(tc.checkpoints, checkpoint)
	traceLog.Printf("Recorded tool call checkpoint: %s (%s)", checkpointID, checkpoint.Name)
	
	return checkpoint
}

// RecordPatch records a code change checkpoint
func (tc *TraceCapture) RecordPatch(name string, diff *DiffDelta) Checkpoint {
	tc.counter++
	checkpointID := fmt.Sprintf("c%03d", tc.counter)
	
	checkpoint := Checkpoint{
		ID:       checkpointID,
		TS:       time.Now().UTC(),
		Kind:     CheckpointPatch,
		Name:     name,
		RepoSHA:  tc.manifest.RepoSHA,
		DiffPath: fmt.Sprintf("diffs/%s.diff", checkpointID),
		Metadata: map[string]any{
			"file":     diff.FilePath,
			"added":    diff.Lines.Added,
			"removed":  diff.Lines.Removed,
		},
	}
	
	tc.checkpoints = append(tc.checkpoints, checkpoint)
	traceLog.Printf("Recorded patch checkpoint: %s (%s)", checkpointID, name)
	
	return checkpoint
}

// RecordEval records an evaluation checkpoint
func (tc *TraceCapture) RecordEval(evalType, command string, result *EvalResult) Checkpoint {
	tc.counter++
	checkpointID := fmt.Sprintf("c%03d", tc.counter)
	
	checkpoint := Checkpoint{
		ID:       checkpointID,
		TS:       time.Now().UTC(),
		Kind:     CheckpointEval,
		Name:     fmt.Sprintf("%s: %s", evalType, command),
		RepoSHA:  tc.manifest.RepoSHA,
		RespPath: fmt.Sprintf("tools/%s.response.json", checkpointID),
		Metadata: map[string]any{
			"exit_code": result.ExitCode,
			"success":   result.Success,
			"duration":  result.Duration.String(),
		},
	}
	
	tc.checkpoints = append(tc.checkpoints, checkpoint)
	traceLog.Printf("Recorded eval checkpoint: %s (%s)", checkpointID, checkpoint.Name)
	
	return checkpoint
}

// RecordSafeOutput records a safe-output checkpoint
func (tc *TraceCapture) RecordSafeOutput(outputType, action, target string, record *SafeOutputRecord) Checkpoint {
	tc.counter++
	checkpointID := fmt.Sprintf("c%03d", tc.counter)
	
	checkpoint := Checkpoint{
		ID:      checkpointID,
		TS:      time.Now().UTC(),
		Kind:    CheckpointSafeOutput,
		Name:    fmt.Sprintf("%s: %s", outputType, action),
		RepoSHA: tc.manifest.RepoSHA,
		Metadata: map[string]any{
			"target":  target,
			"success": record.Success,
		},
	}
	
	tc.checkpoints = append(tc.checkpoints, checkpoint)
	traceLog.Printf("Recorded safe-output checkpoint: %s (%s)", checkpointID, checkpoint.Name)
	
	return checkpoint
}

// GetManifest returns the trace manifest
func (tc *TraceCapture) GetManifest() TraceManifest {
	return tc.manifest
}

// GetCheckpoints returns all recorded checkpoints
func (tc *TraceCapture) GetCheckpoints() []Checkpoint {
	return tc.checkpoints
}

// ToJSON serializes the checkpoints to JSONL format
func (tc *TraceCapture) ToJSON() ([]byte, error) {
	var result []byte
	for _, checkpoint := range tc.checkpoints {
		data, err := json.Marshal(checkpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal checkpoint %s: %w", checkpoint.ID, err)
		}
		result = append(result, data...)
		result = append(result, '\n')
	}
	return result, nil
}

// HashContent creates a SHA-256 hash of content
func HashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// RedactSensitiveData removes sensitive information from tool responses
func RedactSensitiveData(data any, sensitiveKeys []string) any {
	switch v := data.(type) {
	case map[string]any:
		redacted := make(map[string]any)
		for k, val := range v {
			// Check if this key should be redacted
			shouldRedact := false
			for _, sensitiveKey := range sensitiveKeys {
				if k == sensitiveKey {
					shouldRedact = true
					break
				}
			}
			
			if shouldRedact {
				redacted[k] = "[REDACTED]"
			} else {
				redacted[k] = RedactSensitiveData(val, sensitiveKeys)
			}
		}
		return redacted
		
	case []any:
		redacted := make([]any, len(v))
		for i, val := range v {
			redacted[i] = RedactSensitiveData(val, sensitiveKeys)
		}
		return redacted
		
	default:
		return v
	}
}
