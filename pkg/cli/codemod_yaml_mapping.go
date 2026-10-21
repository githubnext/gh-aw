package cli

import (
	"errors"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

func removeYAMLMappingPath(content string, path []string, keepEmptyMapping bool) (string, bool, error) {
	if len(path) < 2 {
		return content, false, errors.New("YAML mapping path must contain at least two keys")
	}
	return codemod.Apply(content, func(f *codemod.Frontmatter) error {
		if err := f.Remove(path...); err != nil {
			return err
		}
		if !keepEmptyMapping {
			return f.PruneEmpty(path[:len(path)-1]...)
		}
		return nil
	})
}
