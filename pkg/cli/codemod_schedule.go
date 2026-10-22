package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

func getScheduleAtToAroundCodemod() Codemod {
	return Codemod{
		ID:           "schedule-at-to-around-migration",
		Name:         "Migrate schedule 'at' syntax to 'around' syntax",
		Description:  "Converts deprecated 'daily at TIME', 'weekly on DAY at TIME', and 'monthly on N at TIME' to fuzzy schedules or standard cron",
		IntroducedIn: "0.5.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Edit(func(document *aw.Document) error {
					if value, ok := document.Frontmatter["schedule"].(string); ok {
						document.Frontmatter["schedule"] = migrateScheduleAtValue(value)
					}
					on, ok := document.Frontmatter["on"].(map[string]any)
					if !ok {
						return nil
					}
					switch schedule := on["schedule"].(type) {
					case string:
						on["schedule"] = migrateScheduleAtValue(schedule)
					case []any:
						for i, entry := range schedule {
							switch entry := entry.(type) {
							case string:
								schedule[i] = migrateScheduleAtValue(entry)
							case map[string]any:
								for _, key := range []string{"cron", "schedule"} {
									if value, ok := entry[key].(string); ok {
										entry[key] = migrateScheduleAtValue(value)
									}
								}
							}
						}
					}
					return nil
				})
			})
		},
	}
}

func migrateScheduleAtValue(value string) string {
	if strings.HasPrefix(value, "daily at") && !strings.Contains(value, "around") && !strings.Contains(value, "between") {
		return strings.Replace(value, "daily at", "daily around", 1)
	}
	if strings.Contains(value, "weekly on") && strings.Contains(value, " at ") && !strings.Contains(value, "around") {
		return strings.Replace(value, " at ", " around ", 1)
	}
	if strings.HasPrefix(value, "monthly on") {
		afterOn := false
		for part := range strings.FieldsSeq(value) {
			if afterOn {
				hour := 0
				if strings.Contains(value, " at ") {
					hour = 9
				}
				return fmt.Sprintf("0 %d %s * *", hour, part)
			}
			afterOn = part == "on"
		}
	}
	return value
}
