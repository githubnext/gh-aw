package agentdrain

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var clusterLog = logger.New("agentdrain:cluster")

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
	c := &Cluster{
		ID:       id,
		Template: slices.Clone(template),
		Size:     1,
		Stage:    stage,
	}
	s.clusters[id] = c
	clusterLog.Printf("Created new cluster: id=%d, stage=%s, template_length=%d", id, stage, len(c.Template))
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
