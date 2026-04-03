package agentdrain

import (
	"encoding/json"
	"fmt"
)

// Snapshot is the serializable representation of a Miner's state.
type Snapshot struct {
	Config   Config            `json:"config"`
	Clusters []SnapshotCluster `json:"clusters"`
	NextID   int               `json:"next_id"`
}

// SnapshotCluster is the serializable form of a single Cluster.
type SnapshotCluster struct {
	ID       int      `json:"id"`
	Template []string `json:"template"`
	Size     int      `json:"size"`
	Stage    string   `json:"stage"`
}

// SaveJSON serializes the miner's current state to JSON bytes.
func (m *Miner) SaveJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := Snapshot{
		Config: m.cfg,
		NextID: m.store.nextID,
	}
	for _, c := range m.store.clusters {
		tmpl := make([]string, len(c.Template))
		copy(tmpl, c.Template)
		snap.Clusters = append(snap.Clusters, SnapshotCluster{
			ID:       c.ID,
			Template: tmpl,
			Size:     c.Size,
			Stage:    c.Stage,
		})
	}
	return json.Marshal(snap)
}

// LoadJSON restores miner state from JSON bytes produced by SaveJSON.
// The existing state is replaced; the parse tree is rebuilt from the snapshot.
func (m *Miner) LoadJSON(data []byte) error {
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("agentdrain: LoadJSON: %w", err)
	}

	masker, err := NewMasker(snap.Config.MaskRules)
	if err != nil {
		return fmt.Errorf("agentdrain: LoadJSON: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = snap.Config
	m.masker = masker
	m.store = newClusterStore()
	m.tree = newParseTree()
	m.store.nextID = snap.NextID

	for _, sc := range snap.Clusters {
		tmpl := make([]string, len(sc.Template))
		copy(tmpl, sc.Template)
		c := &Cluster{
			ID:       sc.ID,
			Template: tmpl,
			Size:     sc.Size,
			Stage:    sc.Stage,
		}
		m.store.clusters[c.ID] = c
		m.tree.addCluster(c.Template, c.ID, m.cfg.Depth, m.cfg.MaxChildren, m.cfg.ParamToken)
	}
	return nil
}

// LoadMinerJSON creates a new Miner by restoring state from JSON bytes.
func LoadMinerJSON(data []byte) (*Miner, error) {
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("agentdrain: LoadMinerJSON: %w", err)
	}
	m, err := NewMiner(snap.Config)
	if err != nil {
		return nil, err
	}
	if err := m.LoadJSON(data); err != nil {
		return nil, err
	}
	return m, nil
}
