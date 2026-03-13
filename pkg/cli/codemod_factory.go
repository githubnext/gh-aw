package cli

import "github.com/github/gh-aw/pkg/logger"

// PostTransformFunc is an optional hook called after the primary field removal.
// It receives the already-modified lines, the full frontmatter, and the removed
// field's value. It returns the (potentially further modified) lines.
type PostTransformFunc func(lines []string, frontmatter map[string]any, fieldValue any) []string

// fieldRemovalCodemodConfig holds the configuration for a field-removal codemod.
type fieldRemovalCodemodConfig struct {
	ID            string
	Name          string
	Description   string
	IntroducedIn  string
	ParentKey     string            // Top-level frontmatter key that contains the field
	FieldKey      string            // Child field to remove from the parent block
	LogMsg        string            // Debug log message emitted when the codemod is applied
	Log           *logger.Logger    // Logger for the codemod
	PostTransform PostTransformFunc // Optional hook for additional transforms after field removal
}

// newFieldRemovalCodemod creates a Codemod that:
//  1. Checks that the parent key is present in the frontmatter and is a map.
//  2. Checks that the child field is present in that map.
//  3. Removes the field (and any nested content) from the YAML block.
//  4. Optionally invokes PostTransform for any additional line-level changes.
func newFieldRemovalCodemod(cfg fieldRemovalCodemodConfig) Codemod {
	return Codemod{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Description:  cfg.Description,
		IntroducedIn: cfg.IntroducedIn,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			parentValue, hasParent := frontmatter[cfg.ParentKey]
			if !hasParent {
				return content, false, nil
			}

			parentMap, ok := parentValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			fieldValue, hasField := parentMap[cfg.FieldKey]
			if !hasField {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, cfg.FieldKey, cfg.ParentKey)
				if !modified {
					return lines, false
				}

				if cfg.PostTransform != nil {
					result = cfg.PostTransform(result, frontmatter, fieldValue)
				}

				return result, true
			})
			if applied {
				cfg.Log.Print(cfg.LogMsg)
			}
			return newContent, applied, err
		},
	}
}
