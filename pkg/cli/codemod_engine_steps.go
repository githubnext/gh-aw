package cli

import (
	"slices"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

func getEngineStepsToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-steps-to-top-level",
		Name:         "Move engine.steps to top-level steps",
		Description:  "Moves the 'steps' field from under 'engine' to the top-level 'steps' field, as 'engine.steps' is no longer supported",
		IntroducedIn: "0.11.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, exists := f.Lookup("engine", "steps")
				if !exists {
					return nil
				}
				top, hasTop := f.Lookup("steps")
				if !hasTop {
					if err := f.Move([]string{"engine", "steps"}, []string{"steps"}); err != nil {
						return err
					}
				} else {
					steps, ok := value.([]any)
					existing, topOK := top.([]any)
					if !ok {
						return nil
					}
					if !topOK {
						existing = nil
					}
					merged := append(slices.Clone(existing), steps...)
					if err := f.Set([]string{"steps"}, merged); err != nil {
						return err
					}
					if err := f.Remove("engine", "steps"); err != nil {
						return err
					}
				}
				return f.PruneEmpty("engine")
			})
		},
	}
}
