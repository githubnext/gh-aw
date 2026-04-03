package agentdrain

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Miner is a concurrent Drain-style log template miner.
// Use NewMiner to create an instance.
type Miner struct {
	cfg    Config
	masker *Masker
	tree   *parseTree
	store  *clusterStore
	mu     sync.RWMutex
}

// NewMiner creates a Miner from the given Config.
func NewMiner(cfg Config) (*Miner, error) {
	masker, err := NewMasker(cfg.MaskRules)
	if err != nil {
		return nil, fmt.Errorf("agentdrain: NewMiner: %w", err)
	}
	return &Miner{
		cfg:    cfg,
		masker: masker,
		tree:   newParseTree(),
		store:  newClusterStore(),
	}, nil
}

// Train processes a raw log line, updates the miner state, and returns the
// match result. It is safe to call from multiple goroutines.
func (m *Miner) Train(line string) (*MatchResult, error) {
	masked := m.masker.Mask(line)
	tokens := Tokenize(masked)
	if len(tokens) == 0 {
		return nil, errors.New("agentdrain: Train: empty line after masking")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	result, _ := m.match(tokens)
	if result != nil {
		// Merge and update existing cluster.
		c, _ := m.store.get(result.ClusterID)
		c.Template = mergeTemplate(c.Template, tokens, m.cfg.ParamToken)
		c.Size++
		result.Template = strings.Join(c.Template, " ")
		result.Params = extractParams(tokens, c.Template, m.cfg.ParamToken)
		return result, nil
	}

	// Create new cluster.
	c := m.store.add(tokens, "")
	m.tree.addCluster(tokens, c.ID, m.cfg.Depth, m.cfg.MaxChildren, m.cfg.ParamToken)
	return &MatchResult{
		ClusterID:  c.ID,
		Template:   strings.Join(c.Template, " "),
		Params:     []string{},
		Similarity: 1.0,
		Stage:      c.Stage,
	}, nil
}

// Match performs inference only: it finds the best matching cluster but does
// not mutate any state. Returns (result, true) when a match is found.
// It is safe to call from multiple goroutines.
func (m *Miner) Match(line string) (*MatchResult, bool, error) {
	masked := m.masker.Mask(line)
	tokens := Tokenize(masked)
	if len(tokens) == 0 {
		return nil, false, errors.New("agentdrain: Match: empty line after masking")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result, _ := m.match(tokens)
	if result == nil {
		return nil, false, nil
	}
	return result, true, nil
}

// match is the internal (non-locking) lookup. Must be called with mu held.
func (m *Miner) match(tokens []string) (*MatchResult, bool) {
	candidates := m.tree.search(tokens, m.cfg.Depth, m.cfg.ParamToken)
	bestSim := -1.0
	var best *Cluster
	for _, id := range candidates {
		c, ok := m.store.get(id)
		if !ok {
			continue
		}
		sim := computeSimilarity(c.Template, tokens, m.cfg.ParamToken)
		if sim > bestSim {
			bestSim = sim
			best = c
		}
	}
	if best == nil || bestSim < m.cfg.SimThreshold {
		return nil, false
	}
	params := extractParams(tokens, best.Template, m.cfg.ParamToken)
	return &MatchResult{
		ClusterID:  best.ID,
		Template:   strings.Join(best.Template, " "),
		Params:     params,
		Similarity: bestSim,
		Stage:      best.Stage,
	}, true
}

// TrainEvent flattens the AgentEvent and calls Train.
func (m *Miner) TrainEvent(evt AgentEvent) (*MatchResult, error) {
	line := FlattenEvent(evt, m.cfg.ExcludeFields)
	result, err := m.Train(line)
	if err != nil {
		return nil, err
	}
	result.Stage = evt.Stage
	// Propagate stage to cluster.
	m.mu.Lock()
	if c, ok := m.store.get(result.ClusterID); ok && c.Stage == "" {
		c.Stage = evt.Stage
	}
	m.mu.Unlock()
	return result, nil
}

// AnalyzeEvent performs inference on the event, builds an AnomalyReport, and
// then calls TrainEvent to update the miner. Returns the match result and report.
func (m *Miner) AnalyzeEvent(evt AgentEvent) (*MatchResult, *AnomalyReport, error) {
	line := FlattenEvent(evt, m.cfg.ExcludeFields)
	masked := m.masker.Mask(line)
	tokens := Tokenize(masked)
	if len(tokens) == 0 {
		return nil, nil, errors.New("agentdrain: AnalyzeEvent: empty event after masking")
	}

	m.mu.RLock()
	inferResult, _ := m.match(tokens)
	m.mu.RUnlock()

	isNew := inferResult == nil
	result, err := m.TrainEvent(evt)
	if err != nil {
		return nil, nil, err
	}

	var cluster *Cluster
	m.mu.RLock()
	cluster, _ = m.store.get(result.ClusterID)
	m.mu.RUnlock()

	detector := NewAnomalyDetector(m.cfg.SimThreshold, m.cfg.RareClusterThreshold)
	report := detector.Analyze(result, isNew, cluster)
	return result, report, nil
}

// Clusters returns a snapshot of all known clusters.
func (m *Miner) Clusters() []Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.all()
}

// ClusterCount returns the number of known clusters.
func (m *Miner) ClusterCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.store.clusters)
}

// extractParams returns the token values at positions where the template has paramToken.
func extractParams(tokens []string, template []string, paramToken string) []string {
	params := []string{}
	for i, tok := range template {
		if tok == paramToken && i < len(tokens) {
			params = append(params, tokens[i])
		}
	}
	return params
}
