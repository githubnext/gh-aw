package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

func getDiscussionFlagRemovalCodemod() Codemod {
	return Codemod{
		ID:           "add-comment-discussion-removal",
		Name:         "Remove deprecated add-comment.discussion field",
		Description:  "Removes the deprecated 'safe-outputs.add-comment.discussion' field. Discussion targeting is now automatic based on context. Use 'discussions: false' to opt out of discussions:write permission.",
		IntroducedIn: "0.3.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Remove("safe-outputs", "add-comment", "discussion")
			})
		},
	}
}
