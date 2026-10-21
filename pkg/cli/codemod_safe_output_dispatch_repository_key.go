package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var safeOutputDispatchRepositoryKeyCodemodLog = logger.New("cli:codemod_safe_output_dispatch_repository_key")

func getSafeOutputDispatchRepositoryKeyCodemod() Codemod {
	return Codemod{
		ID:           "safe-output-dispatch-repository-key",
		Name:         "Rename safe-outputs.dispatch_repository to dispatch-repository",
		Description:  "Renames deprecated safe-outputs.dispatch_repository to safe-outputs.dispatch-repository.",
		IntroducedIn: "1.0.65",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				_, old := f.Lookup("safe-outputs", "dispatch_repository")
				_, canonical := f.Lookup("safe-outputs", "dispatch-repository")
				if old && canonical {
					safeOutputDispatchRepositoryKeyCodemodLog.Print("WARN: safe-outputs has both dispatch_repository and dispatch-repository; manual review needed, skipping migration")
				}
				return f.Rename([]string{"safe-outputs", "dispatch_repository"}, "dispatch-repository")
			})
		},
	}
}
