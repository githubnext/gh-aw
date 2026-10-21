package cli

import (
	"slices"

	"github.com/github/gh-aw/pkg/workflow/codemod"
)

func getSafeJobRunnerCodemod() Codemod {
	return Codemod{
		ID:           "safe-job-runner-to-runs-on",
		Name:         "Rename safe-outputs.jobs runner to runs-on",
		Description:  "Renames deprecated safe-outputs.jobs.<job>.runner fields to runs-on.",
		IntroducedIn: "1.5.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				value, _ := f.Lookup("safe-outputs", "jobs")
				jobs, ok := value.(map[string]any)
				if !ok {
					return nil
				}
				names := make([]string, 0, len(jobs))
				for name := range jobs {
					names = append(names, name)
				}
				slices.Sort(names)
				for _, name := range names {
					if err := f.Rename([]string{"safe-outputs", "jobs", name, "runner"}, "runs-on"); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}
