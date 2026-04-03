package agentdrain

import "strings"

// PreTrainTemplate seeds the miner with a known template string and a
// synthetic observation count. The template is tokenized but not masked,
// so callers should pass already-normalized templates.
func (m *Miner) PreTrainTemplate(template string, count int) {
	tokens := Tokenize(template)
	if len(tokens) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for an existing identical template.
	candidates := m.tree.search(tokens, m.cfg.Depth, m.cfg.ParamToken)
	for _, id := range candidates {
		c, ok := m.store.get(id)
		if !ok {
			continue
		}
		if strings.Join(c.Template, " ") == strings.Join(tokens, " ") {
			c.Size += count
			return
		}
	}

	// Create a new cluster pre-seeded with the desired count.
	c := m.store.add(tokens, "")
	c.Size = count
	m.tree.addCluster(tokens, c.ID, m.cfg.Depth, m.cfg.MaxChildren, m.cfg.ParamToken)
}

// PreTrainTemplates seeds the miner with a slice of template strings, each
// with an initial count of 1.
func (m *Miner) PreTrainTemplates(templates []string) {
	for _, t := range templates {
		m.PreTrainTemplate(t, 1)
	}
}

// PreTrainTemplateCounts seeds the miner with a map of template strings to
// their initial observation counts.
func (m *Miner) PreTrainTemplateCounts(templates map[string]int) {
	for t, count := range templates {
		m.PreTrainTemplate(t, count)
	}
}
