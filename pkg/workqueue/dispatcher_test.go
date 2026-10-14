package workqueue

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

const dispatcherSource = "---\ntools:\n  work-queue: true\nsafe-outputs:\n  dispatch-workflow:\n    workflows: [worker]\n---\nDispatch queued work."
const dispatcherLock = "on:\n  workflow_dispatch:\njobs:\n  agent:\n    env:\n      GH_AW_WORK_QUEUE_ROLE: dispatcher\n"

func TestDispatcherAcceptsESLintFactoryArtifacts(t *testing.T) {
	source, err := os.ReadFile("../../.github/workflows/eslint-factory-dispatcher.md")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := os.ReadFile("../../.github/workflows/eslint-factory-dispatcher.lock.yml")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDispatcherSource(source); err != nil {
		t.Fatal(err)
	}
	if err := validateDispatcherLock(compiled); err != nil {
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
