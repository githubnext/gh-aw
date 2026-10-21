package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/workflow/codemod"
	"gopkg.in/yaml.v3"
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

func lowercaseDiscussionTriggerTypesInLines(lines []string) ([]string, bool) {
	content := "---\n" + strings.Join(lines, "\n") + "\n---\n"
	updated, changed, err := getDiscussionTriggerCategoriesLowercaseCodemod().Apply(content, nil)
	if err != nil || !changed {
		return lines, false
	}
	return strings.Split(strings.TrimSuffix(strings.TrimPrefix(updated, "---\n"), "\n---\n"), "\n"), true
}

func getBlockMappingKey(line string) (string, bool) {
	var document yaml.Node
	if yaml.Unmarshal([]byte(line), &document) != nil {
		return "", false
	}
	if len(document.Content) < 1 {
		return "", false
	}
	if len(document.Content) > 1 {
		return "", false
	}
	mapping := document.Content[0] //nolint:uncheckedsliceindex // The document content length is verified to be exactly one above.
	if mapping.Kind != yaml.MappingNode {
		return "", false
	}
	if len(mapping.Content) < 2 {
		return "", false
	}
	if len(mapping.Content) > 2 {
		return "", false
	}
	key, value := mapping.Content[0], mapping.Content[1] //nolint:uncheckedsliceindex // The mapping content length is verified to be exactly two above.
	if key.Kind != yaml.ScalarNode || key.Value == "" || value.Tag != "!!null" || value.Value != "" {
		return "", false
	}
	return key.Value, true
}
