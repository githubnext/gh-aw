package cli

// getDeleteSchemaFileCodemod creates a codemod for deleting deprecated schema files
func getDeleteSchemaFileCodemod() Codemod {
	return Codemod{
		ID:           "delete-schema-file",
		Name:         "Delete deprecated schema file",
		Description:  "Deletes .github/aw/schemas/agentic-workflow.json which is no longer written by init command",
		IntroducedIn: "0.6.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// This codemod is handled by the fix command itself (see runFixCommand)
			// It doesn't modify workflow files, so we just return content unchanged
			return content, false, nil
		},
	}
}
