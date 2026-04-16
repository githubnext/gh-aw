package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/stringutil"
)

// renderJobsTable renders the jobs as a table using console.RenderTable
func renderJobsTable(jobs []JobData) {
	auditReportLog.Printf("Rendering jobs table with %d jobs", len(jobs))
	config := console.TableConfig{
		Headers: []string{"Name", "Status", "Conclusion", "Duration"},
		Rows:    make([][]string, 0, len(jobs)),
	}

	for _, job := range jobs {
		conclusion := job.Conclusion
		if conclusion == "" {
			conclusion = "-"
		}
		duration := job.Duration
		if duration == "" {
			duration = "-"
		}

		row := []string{
			stringutil.Truncate(job.Name, 40),
			job.Status,
			conclusion,
			duration,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderTable(config))
}
