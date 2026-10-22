package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getInferToDisableModelInvocationCodemod migrates infer, inverting its boolean value.
func getInferToDisableModelInvocationCodemod() Codemod {
	return Codemod{
		ID:           "infer-to-disable-model-invocation",
		Name:         "Migrate 'infer' to 'disable-model-invocation'",
		Description:  "Migrates the deprecated top-level 'infer' field to 'disable-model-invocation', inverting the boolean value (e.g. infer: false → disable-model-invocation: true).",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, exists := f.Lookup("infer")
				if !exists {
					return nil
				}
				if _, exists := f.Lookup("disable-model-invocation"); exists {
					return f.Remove("infer")
				}
				infer, ok := value.(bool)
				if !ok {
					return nil
				}
				if err := f.Rename([]string{"infer"}, "disable-model-invocation"); err != nil {
					return err
				}
				return f.Set([]string{"disable-model-invocation"}, !infer)
			})
		},
	}
}
