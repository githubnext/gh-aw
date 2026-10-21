package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

func getExpiresIntegerToDayStringCodemod() Codemod {
	return Codemod{
		ID:           "expires-integer-to-string",
		Name:         "Convert expires integer to day string",
		Description:  "Converts integer 'expires' values (e.g., 'expires: 7') to day string format (e.g., 'expires: 7d') in safe-outputs types",
		IntroducedIn: "0.13.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Edit(func(document *aw.Document) error {
					outputs, ok := document.Frontmatter["safe-outputs"].(map[string]any)
					if !ok {
						return nil
					}
					for _, value := range outputs {
						handler, ok := value.(map[string]any)
						if !ok {
							continue
						}
						if expires, ok := expiresIntegerDayString(handler["expires"]); ok {
							handler["expires"] = expires
						}
					}
					return nil
				})
			})
		},
	}
}

func convertExpiresIntegerLineToDayString(line string) (string, bool) {
	content := "---\n" + line + "\n---\n"
	updated, changed, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
		value, _ := f.Lookup("expires")
		if expires, ok := expiresIntegerDayString(value); ok {
			return f.Set([]string{"expires"}, expires)
		}
		return nil
	})
	if err != nil || !changed {
		return line, false
	}
	return strings.TrimSuffix(strings.TrimPrefix(updated, "---\n"), "\n---\n"), true
}

func expiresIntegerDayString(value any) (string, bool) {
	switch value := value.(type) {
	case int:
		if value >= 0 {
			return fmt.Sprintf("%dd", value), true
		}
	case int64:
		if value >= 0 {
			return fmt.Sprintf("%dd", value), true
		}
	case uint64:
		return fmt.Sprintf("%dd", value), true
	}
	return "", false
}
