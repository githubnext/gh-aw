//go:build js || wasm

package workflow

type RepositoryFeatures struct {
	HasDiscussions bool
	HasIssues      bool
}

func ClearRepositoryFeaturesCache() {}

func (c *Compiler) validateRepositoryFeatures(workflowData *WorkflowData) error {
	return nil
}
