//go:build !integration

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestSpawnMCPInspector_ContextCancellationWaitsForServerMonitor(t *testing.T) {
	stateDir := testutil.TempDir(t, "mcp-inspector-state-*")
	workflowPath := writeMCPInspectorWorkflow(t, stateDir)
	monitorDone := make(chan string, 1)
	configureMCPInspectorTestHooks(t, stateDir, "wait", monitorDone)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- spawnMCPInspector(ctx, workflowPath, "", false)
	}()

	waitForMarker(t, filepath.Join(stateDir, "server-started"))
	waitForMarker(t, filepath.Join(stateDir, "inspector-started"))

	cancel()

	select {
	case err := <-errCh:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("spawnMCPInspector did not return after context cancellation")
	}

	select {
	case name := <-monitorDone:
		require.Equal(t, "test-server", name)
	default:
		t.Fatal("spawnMCPInspector returned before the server monitor completed after cancellation")
	}
}

func TestSpawnMCPInspector_NormalExitWaitsForServerMonitor(t *testing.T) {
	stateDir := testutil.TempDir(t, "mcp-inspector-state-*")
	workflowPath := writeMCPInspectorWorkflow(t, stateDir)
	monitorDone := make(chan string, 1)
	configureMCPInspectorTestHooks(t, stateDir, "exit", monitorDone)

	err := spawnMCPInspector(context.Background(), workflowPath, "", false)
	require.NoError(t, err)

	select {
	case name := <-monitorDone:
		require.Equal(t, "test-server", name)
	default:
		t.Fatal("spawnMCPInspector returned before the server monitor completed on normal exit")
	}
}

func configureMCPInspectorTestHooks(t *testing.T, stateDir, inspectorBehavior string, monitorDone chan string) {
	t.Helper()

	originalLookPath := mcpInspectorLookPath
	originalCommandContext := mcpInspectorCommandContext
	originalMonitorDone := mcpInspectorMonitorDone

	mcpInspectorLookPath = func(file string) (string, error) {
		switch file {
		case "npx", "fake-server":
			return os.Args[0], nil
		default:
			return originalLookPath(file)
		}
	}

	mcpInspectorCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		switch name {
		case "npx":
			return spawnMCPInspectorHelperCommand(ctx, stateDir, "inspector", inspectorBehavior)
		case "fake-server":
			return spawnMCPInspectorHelperCommand(ctx, stateDir, "server", "")
		default:
			return exec.CommandContext(ctx, name, args...)
		}
	}

	mcpInspectorMonitorDone = func(name string) {
		monitorDone <- name
	}

	t.Cleanup(func() {
		mcpInspectorLookPath = originalLookPath
		mcpInspectorCommandContext = originalCommandContext
		mcpInspectorMonitorDone = originalMonitorDone
	})
}

func spawnMCPInspectorHelperCommand(ctx context.Context, stateDir, role, behavior string) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run=TestSpawnMCPInspectorHelperProcess",
		"--",
		"gh-aw-mcp-helper",
		role,
		stateDir,
		behavior,
	)
}

func TestSpawnMCPInspectorHelperProcess(t *testing.T) {
	sentinel := -1
	for i, arg := range os.Args {
		if arg == "gh-aw-mcp-helper" {
			sentinel = i
			break
		}
	}
	if sentinel == -1 || len(os.Args) < sentinel+3 {
		return
	}

	role := os.Args[sentinel+1]
	stateDir := os.Args[sentinel+2]
	behavior := ""
	if len(os.Args) > sentinel+3 {
		behavior = os.Args[sentinel+3]
	}

	if err := os.WriteFile(filepath.Join(stateDir, role+"-started"), []byte(role), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch role {
	case "inspector":
		if behavior == "exit" {
			return
		}
		fallthrough
	case "server":
		for {
			time.Sleep(10 * time.Millisecond)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown helper role: %s\n", role)
		os.Exit(2)
	}
}

func writeMCPInspectorWorkflow(t *testing.T, dir string) string {
	t.Helper()

	workflowPath := filepath.Join(dir, "test-inspector.md")
	content := `---
on:
  workflow_dispatch:

permissions: read-all

mcp-servers:
  test-server:
    type: stdio
    command: fake-server
---

# Test Inspector
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644))
	return workflowPath
}

func waitForMarker(t *testing.T, path string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for marker %s", path)
}
