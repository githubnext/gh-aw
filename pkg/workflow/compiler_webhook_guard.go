package workflow

import (
	"fmt"
	"strings"
)

// Guard every job, including user jobs that precede activation or use always().
// Payload sender fields are never an authorization identity.
func (c *Compiler) applyWebhookActorGuard(data *WorkflowData) {
	if len(data.WebhookSubscriptions) == 0 {
		return
	}
	actors := make([]string, 0, len(data.WebhookActors))
	for _, actor := range data.WebhookActors {
		actors = append(actors, fmt.Sprintf("github.actor == '%s'", actor))
	}
	authorized := "false"
	if len(actors) > 0 {
		authorized = "(" + strings.Join(actors, " || ") + ")"
	}
	guard := &ExpressionNode{Expression: fmt.Sprintf(
		"github.event_name != 'repository_dispatch' || github.event.action != '%s' || %s",
		WebhookDispatchEventType(data.WorkflowID), authorized,
	)}
	for _, job := range c.jobManager.GetAllJobs() {
		if strings.TrimSpace(job.If) == "" {
			job.If = RenderCondition(guard)
		} else {
			job.If = RenderCondition(BuildAnd(&ExpressionNode{Expression: stripExpressionWrapper(job.If)}, guard))
		}
	}
}
