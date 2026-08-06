import { afterEach, describe, expect, it, vi } from "vitest";
import { collectGatewayStderr, removeGatewayContainer } from "./stop_mcp_gateway.cjs";

describe("stop_mcp_gateway", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("collectGatewayStderr", () => {
    it("emits container stderr via core.debug and does not write to a file", async () => {
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({
          exitCode: 0,
          stdout: '{"apiKey":"must-not-be-logged"}',
          stderr: "gateway diagnostic",
        }),
      };
      const coreApi = { info: vi.fn(), debug: vi.fn() };

      await collectGatewayStderr(execApi, coreApi);

      expect(execApi.getExecOutput).toHaveBeenCalledWith("docker", ["logs", "awmg-mcpg"], {
        ignoreReturnCode: true,
        silent: true,
      });
      // Must not emit stdout content
      expect(coreApi.info).not.toHaveBeenCalledWith(expect.stringContaining("must-not-be-logged"));
      expect(coreApi.debug).not.toHaveBeenCalledWith(expect.stringContaining("must-not-be-logged"));
      // Stderr goes to debug
      expect(coreApi.debug).toHaveBeenCalledWith(expect.stringContaining("gateway diagnostic"));
    });

    it("reports unavailable when docker logs fails with no output", async () => {
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" }),
      };
      const coreApi = { info: vi.fn(), debug: vi.fn() };

      await collectGatewayStderr(execApi, coreApi);

      expect(coreApi.info).toHaveBeenCalledWith("MCP Gateway stderr is unavailable.");
      expect(coreApi.debug).not.toHaveBeenCalled();
    });
  });

  describe("removeGatewayContainer", () => {
    it("removes the container after log collection", async () => {
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
      };
      const coreApi = { info: vi.fn(), debug: vi.fn() };

      await removeGatewayContainer(execApi, coreApi);

      expect(execApi.getExecOutput).toHaveBeenCalledWith("docker", ["rm", "awmg-mcpg"], {
        ignoreReturnCode: true,
        silent: true,
      });
    });

    it("logs a message when container is already removed", async () => {
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ exitCode: 1, stdout: "", stderr: "No such container" }),
      };
      const coreApi = { info: vi.fn(), debug: vi.fn() };

      await removeGatewayContainer(execApi, coreApi);

      expect(coreApi.info).toHaveBeenCalledWith("MCP Gateway container already removed or not found.");
    });
  });
});
