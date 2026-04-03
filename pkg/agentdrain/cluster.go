package agentdrain

// clusterStore manages the set of known log template clusters.
type clusterStore struct {
	clusters map[int]*Cluster
	nextID   int
}

func newClusterStore() *clusterStore {
	return &clusterStore{
		clusters: make(map[int]*Cluster),
		nextID:   1,
	}
}

// add creates a new Cluster for the given template and returns a pointer to it.
func (s *clusterStore) add(template []string, stage string) *Cluster {
	id := s.nextID
	s.nextID++
	tmpl := make([]string, len(template))
	copy(tmpl, template)
	c := &Cluster{
		ID:       id,
		Template: tmpl,
		Size:     1,
		Stage:    stage,
	}
	s.clusters[id] = c
	return c
}

// get retrieves a cluster by ID.
func (s *clusterStore) get(id int) (*Cluster, bool) {
	c, ok := s.clusters[id]
	return c, ok
}

// all returns a snapshot of all clusters as a value slice.
func (s *clusterStore) all() []Cluster {
	out := make([]Cluster, 0, len(s.clusters))
	for _, c := range s.clusters {
		out = append(out, *c)
	}
	return out
}

// computeSimilarity returns the fraction of positions where tokens a and b
// match exactly, considering only positions that are not paramToken in a.
// Returns 0 when the slices have different lengths.
func computeSimilarity(a, b []string, paramToken string) float64 {
	if len(a) != len(b) {
		return 0
	}
	nonParam := 0
	matches := 0
	for i, tok := range a {
		if tok == paramToken {
			continue
		}
		nonParam++
		if tok == b[i] {
			matches++
		}
	}
	if nonParam == 0 {
		// All positions are wildcards – treat as a perfect structural match.
		return 1.0
	}
	return float64(matches) / float64(nonParam)
}

// mergeTemplate produces a new template by replacing positions where the two
// token slices differ with paramToken. Positions where either token already is
// paramToken also become paramToken.
func mergeTemplate(existing, incoming []string, paramToken string) []string {
	if len(existing) != len(incoming) {
		return existing
	}
	merged := make([]string, len(existing))
	for i, tok := range existing {
		if tok == paramToken || incoming[i] == paramToken || tok != incoming[i] {
			merged[i] = paramToken
		} else {
			merged[i] = tok
		}
	}
	return merged
}
