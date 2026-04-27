// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
};

const mockGithub = {
  rest: {
    actions: {
      getWorkflowRun: vi.fn(),
      listWorkflowRuns: vi.fn(),
      listWorkflowRunArtifacts: vi.fn(),
    },
  },
};

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
  runId: 99999,
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("find_circuit_breaker_artifact.cjs", () => {
  let findArtifact;

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.resetModules();
    findArtifact = await import("./find_circuit_breaker_artifact.cjs");

    // Default: current run is workflow 42
    mockGithub.rest.actions.getWorkflowRun.mockResolvedValue({
      data: { workflow_id: 42 },
    });
  });

  it("returns empty when no completed runs exist", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: { workflow_runs: [] },
    });

    await findArtifact.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No previous circuit-breaker-state artifact found"));
  });

  it("skips the current run ID", async () => {
    // The only completed run is the current run itself
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: {
        workflow_runs: [{ id: 99999 }],
      },
    });

    await findArtifact.main();

    expect(mockGithub.rest.actions.listWorkflowRunArtifacts).not.toHaveBeenCalled();
    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "");
  });

  it("returns the run ID of the most recent run with the artifact", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: {
        workflow_runs: [{ id: 100 }, { id: 200 }],
      },
    });

    // Run 100 has the artifact
    mockGithub.rest.actions.listWorkflowRunArtifacts.mockImplementation(({ run_id }) => {
      if (run_id === 100) {
        return Promise.resolve({
          data: {
            artifacts: [{ name: "circuit-breaker-state", expired: false }],
          },
        });
      }
      return Promise.resolve({ data: { artifacts: [] } });
    });

    await findArtifact.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "100");
  });

  it("skips expired artifacts and continues to next run", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: {
        workflow_runs: [{ id: 100 }, { id: 200 }],
      },
    });

    // Run 100 has the artifact but it's expired; run 200 has it fresh
    mockGithub.rest.actions.listWorkflowRunArtifacts.mockImplementation(({ run_id }) => {
      if (run_id === 100) {
        return Promise.resolve({
          data: {
            artifacts: [{ name: "circuit-breaker-state", expired: true }],
          },
        });
      }
      if (run_id === 200) {
        return Promise.resolve({
          data: {
            artifacts: [{ name: "circuit-breaker-state", expired: false }],
          },
        });
      }
      return Promise.resolve({ data: { artifacts: [] } });
    });

    await findArtifact.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "200");
  });

  it("continues past runs where artifact listing fails", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: {
        workflow_runs: [{ id: 100 }, { id: 200 }],
      },
    });

    // Run 100 throws an error; run 200 has the artifact
    mockGithub.rest.actions.listWorkflowRunArtifacts.mockImplementation(({ run_id }) => {
      if (run_id === 100) {
        return Promise.reject(new Error("403 Forbidden"));
      }
      return Promise.resolve({
        data: {
          artifacts: [{ name: "circuit-breaker-state", expired: false }],
        },
      });
    });

    await findArtifact.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "200");
    expect(mockCore.debug).toHaveBeenCalledWith(expect.stringContaining("Could not list artifacts"));
  });

  it("returns empty when no run has the named artifact", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: {
        workflow_runs: [{ id: 100 }, { id: 200 }],
      },
    });

    // Neither run has the circuit-breaker-state artifact
    mockGithub.rest.actions.listWorkflowRunArtifacts.mockResolvedValue({
      data: { artifacts: [{ name: "some-other-artifact", expired: false }] },
    });

    await findArtifact.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "");
  });

  it("handles getWorkflowRun API failure gracefully", async () => {
    mockGithub.rest.actions.getWorkflowRun.mockRejectedValue(new Error("API unavailable"));

    await findArtifact.main();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not search for previous circuit breaker state"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "");
  });

  it("handles listWorkflowRuns API failure gracefully", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockRejectedValue(new Error("Rate limited"));

    await findArtifact.main();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not search for previous circuit breaker state"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("previous_run_id", "");
  });
});
