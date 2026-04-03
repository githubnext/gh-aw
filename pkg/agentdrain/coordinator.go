package agentdrain

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Coordinator manages one Miner per agent pipeline stage.
type Coordinator struct {
	miners map[string]*Miner
	cfg    Config
	mu     sync.RWMutex
}

// NewCoordinator creates a Coordinator with one Miner for each provided stage name.
func NewCoordinator(cfg Config, stages []string) (*Coordinator, error) {
	miners := make(map[string]*Miner, len(stages))
	for _, stage := range stages {
		m, err := NewMiner(cfg)
		if err != nil {
			return nil, fmt.Errorf("agentdrain: NewCoordinator: stage %q: %w", stage, err)
		}
		miners[stage] = m
	}
	return &Coordinator{miners: miners, cfg: cfg}, nil
}

// TrainEvent routes the event to the miner responsible for evt.Stage.
// Returns an error when the stage has no associated miner.
func (c *Coordinator) TrainEvent(evt AgentEvent) (*MatchResult, error) {
	m, err := c.minerFor(evt.Stage)
	if err != nil {
		return nil, err
	}
	return m.TrainEvent(evt)
}

// AnalyzeEvent routes the event to the correct stage miner and returns both
// the match result and an anomaly report.
func (c *Coordinator) AnalyzeEvent(evt AgentEvent) (*MatchResult, *AnomalyReport, error) {
	m, err := c.minerFor(evt.Stage)
	if err != nil {
		return nil, nil, err
	}
	return m.AnalyzeEvent(evt)
}

// Stages returns the list of stage names managed by this Coordinator.
func (c *Coordinator) Stages() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stages := make([]string, 0, len(c.miners))
	for s := range c.miners {
		stages = append(stages, s)
	}
	return stages
}

// MinerForStage returns the Miner for the given stage, or false if not found.
func (c *Coordinator) MinerForStage(stage string) (*Miner, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.miners[stage]
	return m, ok
}

// AllClusters returns a map from stage name to the list of clusters in that miner.
func (c *Coordinator) AllClusters() map[string][]Cluster {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string][]Cluster, len(c.miners))
	for stage, m := range c.miners {
		result[stage] = m.Clusters()
	}
	return result
}

// SaveSnapshots serializes each stage miner's state and returns a map from
// stage name to JSON bytes.
func (c *Coordinator) SaveSnapshots() (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string][]byte, len(c.miners))
	for stage, m := range c.miners {
		data, err := m.SaveJSON()
		if err != nil {
			return nil, fmt.Errorf("agentdrain: SaveSnapshots: stage %q: %w", stage, err)
		}
		out[stage] = data
	}
	return out, nil
}

// LoadSnapshots restores each stage miner from the provided JSON bytes map.
// Stages that are not present in snapshots retain their current state.
func (c *Coordinator) LoadSnapshots(snapshots map[string][]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for stage, data := range snapshots {
		m, ok := c.miners[stage]
		if !ok {
			// Create a new miner for previously unknown stages.
			var err error
			m, err = NewMiner(c.cfg)
			if err != nil {
				return fmt.Errorf("agentdrain: LoadSnapshots: stage %q: %w", stage, err)
			}
			c.miners[stage] = m
		}
		if err := m.LoadJSON(data); err != nil {
			return fmt.Errorf("agentdrain: LoadSnapshots: stage %q: %w", stage, err)
		}
	}
	return nil
}

// minerFor retrieves the miner for the given stage, returning an error if missing.
func (c *Coordinator) minerFor(stage string) (*Miner, error) {
	c.mu.RLock()
	m, ok := c.miners[stage]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("agentdrain: no miner registered for stage %q", stage)
	}
	return m, nil
}

// SaveWeightsJSON serializes all stage snapshots into a single combined JSON blob.
// The result can be written to pkg/agentdrain/data/default_weights.json and
// committed to embed it as the default starting weights for future runs.
func (c *Coordinator) SaveWeightsJSON() ([]byte, error) {
	snapshots, err := c.SaveSnapshots()
	if err != nil {
		return nil, err
	}
	combined := make(map[string]json.RawMessage, len(snapshots))
	for stage, data := range snapshots {
		combined[stage] = json.RawMessage(data)
	}
	return json.Marshal(combined)
}

// LoadWeightsJSON restores all stage miners from a combined JSON blob produced
// by SaveWeightsJSON.
func (c *Coordinator) LoadWeightsJSON(data []byte) error {
	var combined map[string]json.RawMessage
	if err := json.Unmarshal(data, &combined); err != nil {
		return fmt.Errorf("agentdrain: LoadWeightsJSON: %w", err)
	}
	snapshots := make(map[string][]byte, len(combined))
	for stage, raw := range combined {
		snapshots[stage] = []byte(raw)
	}
	return c.LoadSnapshots(snapshots)
}

// StageSequence converts a slice of AgentEvents into a space-separated string
// of their stage names, e.g. "plan tool_call tool_result finish".
func StageSequence(events []AgentEvent) string {
	stages := make([]string, 0, len(events))
	for _, e := range events {
		stages = append(stages, e.Stage)
	}
	return strings.Join(stages, " ")
}
