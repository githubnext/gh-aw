package workflow

import (
	"strings"
	"testing"
)

func TestWebhookActorGuardProtectsAllJobs(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()
	for _, job := range []*Job{
		{Name: "activation"},
		{Name: "custom-before-activation", If: "${{ always() }}"},
		{Name: "agent", If: "${{ needs.activation.result == 'success' }}"},
		{Name: "custom-after-agent", If: "failure() || cancelled()"},
		{Name: "reusable", Uses: "./reusable.yml"},
	} {
		// Job structure is immaterial to this post-build guard.
		compiler.jobManager.jobs[job.Name] = job
	}
	data := &WorkflowData{
		WorkflowID:           "guard-test",
		WebhookActors:        []string{"cao[bot]", "trusted-gateway"},
		WebhookSubscriptions: []WebhookSubscription{{Event: "projects_v2_item"}},
	}
	compiler.applyWebhookActorGuard(data)
	for name, job := range compiler.jobManager.GetAllJobs() {
		for _, required := range []string{
			"github.event_name != 'repository_dispatch'",
			"github.event.action != '" + WebhookDispatchEventType("guard-test") + "'",
			"github.actor == 'cao[bot]'",
			"github.actor == 'trusted-gateway'",
		} {
			if !strings.Contains(job.If, required) {
				t.Errorf("job %s is missing actor guard %q: %s", name, required, job.If)
			}
		}
	}
	if !strings.Contains(compiler.jobManager.jobs["custom-before-activation"].If, "always()") {
		t.Fatal("original status condition was discarded")
	}
}

func TestWebhookActorGuardPreservesNativeJobs(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()
	job := &Job{Name: "native", If: "${{ always() }}"}
	compiler.jobManager.jobs[job.Name] = job
	compiler.applyWebhookActorGuard(&WorkflowData{})
	if job.If != "${{ always() }}" {
		t.Fatal("native workflow job condition changed")
	}
}
