package workqueue

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

const dispatcherSource = "---\ntools:\n  work-queue: true\nsafe-outputs:\n  dispatch-workflow:\n    workflows: [worker]\n---\nDispatch queued work."
const dispatcherControlStep = "      - id: work_queue_controls\n        env:\n          GH_AW_WORK_QUEUE_CONTROL_CONFIG: '{\"work_queue_workflows\":[\"worker\"],\"aw_context_workflows\":[\"worker\"]}'\n"
const dispatcherLock = "on:\n  workflow_dispatch:\njobs:\n  agent:\n    env:\n      GH_AW_WORK_QUEUE_ROLE: dispatcher\n    steps:\n" + dispatcherControlStep

func TestDispatcherAcceptsESLintFactoryArtifacts(t *testing.T) {
	source, err := os.ReadFile("../../.github/workflows/eslint-factory-dispatcher.md")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := os.ReadFile("../../.github/workflows/eslint-factory-dispatcher.lock.yml")
	if err != nil {
		t.Fatal(err)
	}
	workers, err := dispatcherSourceWorkflows(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDispatcherLock(compiled, workers); err != nil {
		t.Fatal(err)
	}
}

func TestDispatcherFromConfig(t *testing.T) {
	branch, mock := newQueueAPI(t)
	mock.sources = map[string]string{"dispatcher": dispatcherSource}
	mock.workerRoutes = map[string]string{".github/workflows/dispatcher.lock.yml": dispatcherLock}
	route, err := branch.DispatcherFromConfig(context.Background(), "dispatcher")
	if err != nil {
		t.Fatal(err)
	}
	if route.Repository != testRepository || route.Workflow != "dispatcher.lock.yml" ||
		route.Ref != "main" || route.Revision != strings.Repeat("f", 40) {
		t.Fatalf("unexpected dispatcher route: %+v", route)
	}
	for _, ref := range mock.configRefs {
		if ref != route.Revision {
			t.Fatalf("source read used a moving ref: %s", ref)
		}
	}
	if mock.workerRef != route.Revision || mock.refWrites != 0 || mock.nativeReads != 0 {
		t.Fatal("dispatcher route inspection mutated the queue or launched a worker")
	}
}

func TestDispatcherFromConfigRejectsInvalidRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, source, lock string
		status             int
	}{
		{name: "missing queue", source: "---\ntools: {}\n---\nOrdinary workflow", lock: dispatcherLock},
		{name: "disabled queue", source: strings.ReplaceAll(dispatcherSource, "work-queue: true", "work-queue: false"), lock: dispatcherLock},
		{name: "worker", source: strings.ReplaceAll(dispatcherSource, "work-queue: true", "work-queue:\n    worker: true"), lock: dispatcherLock},
		{name: "requires assignment", source: strings.ReplaceAll(dispatcherSource, "work-queue: true", "work-queue:\n    require-assignment: true"), lock: dispatcherLock},
		{name: "no allowlist", source: strings.ReplaceAll(dispatcherSource, "[worker]", "[]"), lock: dispatcherLock},
		{name: "invalid allowlist", source: strings.ReplaceAll(dispatcherSource, "[worker]", "[../worker]"), lock: dispatcherLock},
		{name: "no frontmatter", source: "Ordinary workflow", lock: dispatcherLock},
		{name: "not dispatchable", source: dispatcherSource, lock: strings.ReplaceAll(dispatcherLock, "workflow_dispatch:", "push:")},
		{name: "wrong compiled role", source: dispatcherSource, lock: strings.ReplaceAll(dispatcherLock, "ROLE: dispatcher", "ROLE: worker")},
		{name: "assignment input", source: dispatcherSource, lock: strings.ReplaceAll(dispatcherLock, "workflow_dispatch:", "workflow_dispatch:\n    inputs:\n      work_queue_assignment:\n        type: string")},
		{name: "stale broadened lock", source: dispatcherSource, lock: strings.ReplaceAll(dispatcherLock, `["worker"]`, `["worker","other"]`)},
		{name: "stale narrowed lock", source: strings.ReplaceAll(dispatcherSource, "[worker]", "[worker, other]"), lock: dispatcherLock},
		{name: "denied source", source: dispatcherSource, lock: dispatcherLock, status: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch, mock := newQueueAPI(t)
			mock.sources = map[string]string{"dispatcher": tc.source}
			mock.workerRoutes = map[string]string{".github/workflows/dispatcher.lock.yml": tc.lock}
			mock.sourceStatuses = map[string]int{"dispatcher": tc.status}
			if _, err := branch.DispatcherFromConfig(context.Background(), "dispatcher"); err == nil {
				t.Fatal("invalid dispatcher route was accepted")
			}
			if mock.refWrites != 0 {
				t.Fatal("invalid dispatcher route mutated the queue")
			}
		})
	}
	for _, name := range []string{"", "../worker", "worker.md", "worker.lock.yml", "worker/name", "--worker", strings.Repeat("a", 129)} {
		branch, _ := newQueueAPI(t)
		if _, err := branch.DispatcherFromConfig(context.Background(), name); err == nil {
			t.Fatalf("invalid dispatcher name was accepted: %q", name)
		}
	}
}

func TestDispatcherLockAllowlistParity(t *testing.T) {
	configLock := func(config string) string {
		return strings.Replace(dispatcherLock, `{"work_queue_workflows":["worker"],"aw_context_workflows":["worker"]}`, config, 1)
	}
	for _, tc := range []struct {
		name, lock string
	}{
		{"missing controls", strings.Replace(dispatcherLock, dispatcherControlStep, "", 1)},
		{"missing config", strings.Replace(dispatcherLock, dispatcherControlStep, "      - id: work_queue_controls\n", 1)},
		{"wrong control id", strings.Replace(dispatcherLock, "id: work_queue_controls", "id: other", 1)},
		{"multiple control steps", dispatcherLock + dispatcherControlStep},
		{"other step config", dispatcherLock + strings.Replace(dispatcherControlStep, "id: work_queue_controls", "id: other", 1)},
		{"multiple control jobs", dispatcherLock + "  other:\n    steps:\n" + dispatcherControlStep},
		{"workflow env config", "env:\n  GH_AW_WORK_QUEUE_CONTROL_CONFIG: '{}'\n" + dispatcherLock},
		{"job env config", strings.Replace(dispatcherLock, "ROLE: dispatcher", "ROLE: dispatcher\n      GH_AW_WORK_QUEUE_CONTROL_CONFIG: '{}'", 1)},
		{"malformed YAML", dispatcherLock + "      - ["},
		{"duplicate YAML config", strings.Replace(dispatcherLock, "          GH_AW_WORK_QUEUE_CONTROL_CONFIG:", "          GH_AW_WORK_QUEUE_CONTROL_CONFIG: '{}'\n          GH_AW_WORK_QUEUE_CONTROL_CONFIG:", 1)},
		{"malformed JSON", configLock("{")},
		{"trailing JSON", configLock(`{"work_queue_workflows":["worker"],"aw_context_workflows":["worker"]} {}`)},
		{"duplicate JSON key", configLock(`{"work_queue_workflows":["other"],"work_queue_workflows":["worker"],"aw_context_workflows":["worker"]}`)},
		{"nonobject JSON", configLock(`[]`)},
		{"expression config", configLock(`${{ needs.activation.outputs.config }}`)},
		{"YAML object config", strings.Replace(dispatcherLock, `'{"work_queue_workflows":["worker"],"aw_context_workflows":["worker"]}'`, `{work_queue_workflows: [worker], aw_context_workflows: [worker]}`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDispatcherLock([]byte(tc.lock), []string{"worker"}); err == nil {
				t.Fatal("invalid compiled dispatcher configuration was accepted")
			}
		})
	}
	for _, field := range []string{"work_queue_workflows", "aw_context_workflows"} {
		for _, tc := range []struct {
			name, value string
		}{
			{"broadened", `["worker","other"]`},
			{"narrowed", `[]`},
			{"substituted", `["other"]`},
			{"wrong case", `["Worker"]`},
			{"not basenames", `["worker.lock.yml"]`},
			{"path", `["../worker"]`},
			{"nonstring member", `["worker",false]`},
			{"object", `{}`},
			{"string", `"worker"`},
			{"null", `null`},
			{"missing", ""},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				config := `{"work_queue_workflows":["worker"],"aw_context_workflows":["worker"]}`
				if tc.value == "" {
					if field == "work_queue_workflows" {
						config = `{"aw_context_workflows":["worker"]}`
					} else {
						config = `{"work_queue_workflows":["worker"]}`
					}
				} else {
					config = strings.Replace(config, `"`+field+`":["worker"]`, `"`+field+`":`+tc.value, 1)
				}
				if err := validateDispatcherLock([]byte(configLock(config)), []string{"worker"}); err == nil {
					t.Fatal("compiled dispatcher allowlist drift was accepted")
				}
			})
		}
	}
}

func TestDispatcherAllowlistCanonicalEquality(t *testing.T) {
	branch, mock := newQueueAPI(t)
	mock.sources = map[string]string{"dispatcher": strings.Replace(dispatcherSource, "[worker]", "[beta, alpha, beta]", 1)}
	mock.workerRoutes = map[string]string{".github/workflows/dispatcher.lock.yml": strings.Replace(dispatcherLock,
		`{"work_queue_workflows":["worker"],"aw_context_workflows":["worker"]}`,
		`{"work_queue_workflows":["alpha","beta"],"aw_context_workflows":["beta","alpha","alpha"]}`, 1)}
	if _, err := branch.DispatcherFromConfig(context.Background(), "dispatcher"); err != nil {
		t.Fatalf("equivalent allowlist sets were rejected: %v", err)
	}
	if mock.refWrites != 0 || mock.nativeReads != 0 {
		t.Fatal("dispatcher parity validation mutated the queue")
	}
}
