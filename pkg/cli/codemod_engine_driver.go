package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getEngineCopilotSDKDriverToDriverCodemod renames engine.copilot-sdk-driver to driver.
func getEngineCopilotSDKDriverToDriverCodemod() Codemod {
	return Codemod{
		ID:           "engine-copilot-sdk-driver-to-driver",
		Name:         "Rename 'engine.copilot-sdk-driver' to 'engine.driver'",
		Description:  "Renames the deprecated 'copilot-sdk-driver:' field to 'driver:' inside the engine configuration block.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"engine", "copilot-sdk-driver"}, "driver")
			})
		},
	}
}
