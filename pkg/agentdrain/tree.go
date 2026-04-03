package agentdrain

// treeNode is an internal node in the Drain parse tree.
type treeNode struct {
	// children maps a token string to its subtree node.
	children map[string]*treeNode
	// clusterIDs holds the IDs of clusters stored at this leaf.
	clusterIDs []int
}

func newTreeNode() *treeNode {
	return &treeNode{children: make(map[string]*treeNode)}
}

// parseTree is the two-level prefix tree used by Drain to bucket candidate clusters.
// Level 0 (root) → level 1 keyed by token count → level 2 keyed by first token
// → leaf containing cluster IDs.
type parseTree struct {
	// root maps token-count (as int key via an inner map) to first-token nodes.
	// Structure: tokenCount → firstToken → *treeNode (leaf)
	root map[int]map[string]*treeNode
}

func newParseTree() *parseTree {
	return &parseTree{root: make(map[int]map[string]*treeNode)}
}

// addCluster inserts clusterID into the leaf for the given tokens.
func (t *parseTree) addCluster(tokens []string, clusterID int, depth int, maxChildren int, paramToken string) {
	n := len(tokens)
	if t.root[n] == nil {
		t.root[n] = make(map[string]*treeNode)
	}
	key := t.firstKey(tokens, depth, paramToken)
	leaf := t.root[n][key]
	if leaf == nil {
		leaf = newTreeNode()
		t.root[n][key] = leaf
	}
	leaf.clusterIDs = append(leaf.clusterIDs, clusterID)
}

// search returns candidate cluster IDs for the given tokens.
func (t *parseTree) search(tokens []string, depth int, paramToken string) []int {
	n := len(tokens)
	byCount, ok := t.root[n]
	if !ok {
		return nil
	}
	key := t.firstKey(tokens, depth, paramToken)
	leaf, ok := byCount[key]
	if !ok {
		// Also try the wildcard bucket.
		leaf, ok = byCount[paramToken]
		if !ok {
			return nil
		}
	}
	out := make([]int, len(leaf.clusterIDs))
	copy(out, leaf.clusterIDs)
	return out
}

// firstKey returns the routing key derived from the first meaningful token.
// When depth == 1, all lines with the same length share a single bucket.
func (t *parseTree) firstKey(tokens []string, depth int, paramToken string) string {
	if depth <= 1 || len(tokens) == 0 {
		return "*"
	}
	tok := tokens[0]
	if tok == paramToken {
		return paramToken
	}
	return tok
}
