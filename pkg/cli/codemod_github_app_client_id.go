package cli

import (
	"fmt"
	"maps"
	"slices"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

// getGitHubAppClientIDCodemod migrates github-app.app-id to github-app.client-id.
func getGitHubAppClientIDCodemod() Codemod {
	return Codemod{
		ID:           "github-app-app-id-to-client-id",
		Name:         "Rename 'github-app.app-id' to 'github-app.client-id'",
		Description:  "Renames deprecated 'app-id:' to 'client-id:' inside github-app blocks.",
		IntroducedIn: "0.68.4",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return migrateGitHubAppClientIDs(f, nil)
			})
		},
	}
}

func migrateGitHubAppClientIDs(f *codemod.Frontmatter, path []string) error {
	value, _ := f.Lookup(path...)
	switch value := value.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			childPath := append(slices.Clone(path), key)
			if key == "github-app" {
				if err := f.Rename(append(slices.Clone(childPath), "app-id"), "client-id"); err != nil {
					return err
				}
			}
			if err := migrateGitHubAppClientIDs(f, childPath); err != nil {
				return err
			}
		}
	case []any:
		// Sequence indices are not mapping paths, so mutate these subtrees through Edit.
		return f.Edit(func(document *aw.Document) error {
			var subtree any = document.Frontmatter
			for _, key := range path {
				mapping, ok := subtree.(map[string]any)
				if !ok {
					return fmt.Errorf("cannot migrate github-app client IDs: path %q is not a mapping", path)
				}
				subtree = mapping[key]
			}
			renameGitHubAppClientIDsInValue(subtree)
			return nil
		})
	}
	return nil
}

func renameGitHubAppClientIDsInValue(value any) {
	switch value := value.(type) {
	case map[string]any:
		if app, ok := value["github-app"].(map[string]any); ok {
			id, exists := app["app-id"]
			if _, conflict := app["client-id"]; exists && !conflict {
				app["client-id"] = id
				delete(app, "app-id")
			}
		}
		for _, child := range value {
			renameGitHubAppClientIDsInValue(child)
		}
	case []any:
		for _, child := range value {
			renameGitHubAppClientIDsInValue(child)
		}
	}
}
