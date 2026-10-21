package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

func getDiscussionTriggerCategoriesLowercaseCodemod() Codemod {
	return Codemod{
		ID:           "discussion-trigger-categories-lowercase",
		Name:         "Lowercase discussion trigger category values",
		Description:  "Lowercases mixed-case category strings in on.discussion.types and on.discussion_comment.types.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, trigger := range []string{"discussion", "discussion_comment"} {
					path := []string{"on", trigger, "types"}
					value, _ := f.Lookup(path...)
					types, ok := value.([]any)
					if !ok {
						continue
					}
					updated := append([]any(nil), types...)
					for i, value := range updated {
						if text, ok := value.(string); ok && !strings.Contains(text, "${{") {
							updated[i] = strings.ToLower(text)
						}
					}
					if err := f.Set(path, updated); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}
