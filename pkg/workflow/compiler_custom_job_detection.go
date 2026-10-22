package workflow

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/rhysd/actionlint"
)

// Run after all job augmentation and typed output-derived dependency lowering.
func (c *Compiler) guardCustomJobDetection(data *WorkflowData) error {
	detection := string(constants.DetectionJobName)
	if _, enabled := c.jobManager.GetJob(detection); !enabled {
		return nil
	}
	graph := make(aw.JobDependencyGraph)
	for id, job := range c.jobManager.GetAllJobs() {
		graph[id] = job.Needs
	}
	for _, id := range sliceutil.SortedKeys(data.Jobs) {
		if isBuiltinJobName(id) {
			continue
		}
		job, exists := c.jobManager.GetJob(id)
		if !exists {
			continue
		}
		if !graph.DependsOn(id, string(constants.AgentJobName)) && !graph.DependsOn(id, detection) {
			if !graph.DependsOn(string(constants.AgentJobName), id) {
				return fmt.Errorf("jobs.%s.needs: custom jobs must run before agent or after detection; add this job to jobs.agent.needs or add detection to this job's needs", id)
			}
			continue
		}
		if graph.DependsOn(detection, id) {
			return fmt.Errorf("jobs.%s.needs: custom jobs must run before agent or after detection, not between them; remove this job from detection's dependencies", id)
		}
		if err := addCustomJobDetectionGuard(job); err != nil {
			return err
		}
	}
	return nil
}

func addCustomJobDetectionGuard(job *Job) error {
	condition := strings.TrimSpace(stripExpressionWrapper(job.If))
	if condition != "" {
		lexer := actionlint.NewExprLexer(condition + " }}")
		if _, err := actionlint.NewExprParser().Parse(lexer); err != nil || lexer.Offset() != len(condition+" }}") {
			return fmt.Errorf("jobs.%s.if: expected one valid GitHub Actions expression; remove interpolated text and use one complete ${{ }} wrapper or omit the wrapper", job.Name)
		}
	}
	detection := string(constants.DetectionJobName)
	if !slices.Contains(job.Needs, detection) {
		job.Needs = append(job.Needs, detection)
	}
	// Warn-mode detections can report job success while flagging tainted output.
	guard := "needs." + detection + ".result == 'success' && needs." + detection + ".outputs.detection_success == 'true'"
	if condition != "" {
		guard += " && (" + condition + ")"
	}
	job.If = "${{ " + guard + " }}"
	job.actionsCondition = true
	return nil
}
