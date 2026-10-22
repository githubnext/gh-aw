package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var missingReportCodemodLog = logger.New("cli:codemod_safe_output_missing_reports")

func getSafeOutputMissingToolCodemod() Codemod {
	return newMissingReportCodemod("tool", []string{"missing-tool", "missing-tools"})
}

func getSafeOutputMissingDataCodemod() Codemod {
	return newMissingReportCodemod("data", []string{"missing-data"})
}

func newMissingReportCodemod(kind string, aliases []string) Codemod {
	target := "report-missing-" + kind
	return Codemod{
		ID:           "safe-output-report-missing-" + kind,
		Name:         "Rename missing " + kind + " reporting configuration",
		Description:  "Renames safe-outputs." + strings.Join(aliases, " and safe-outputs.") + " to safe-outputs." + target + ".",
		IntroducedIn: "1.5.0",
		Guided:       true,
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				source := ""
				for _, alias := range aliases {
					if _, exists := f.Lookup("safe-outputs", alias); exists {
						if _, conflict := f.Lookup("safe-outputs", target); conflict || source != "" {
							return fmt.Errorf("safe-outputs has conflicting missing %s reporting keys; keep a single %s entry before running gh aw fix", kind, target)
						}
						source = alias
					}
				}
				if source == "" {
					return nil
				}
				if err := f.Rename([]string{"safe-outputs", source}, target); err != nil {
					return err
				}
				missingReportCodemodLog.Printf("Renamed safe-outputs.%s to %s", source, target)
				return nil
			})
		},
	}
}
