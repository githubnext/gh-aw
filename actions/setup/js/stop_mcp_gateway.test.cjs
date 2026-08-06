import fs from "fs";
import { afterEach, describe, expect, it, vi } from "vitest";
import { collectGatewayStderr, removeGatewayContainer } from "./stop_mcp_gateway.cjs";

describe("stop_mcp_gateway", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("collectGatewayStderr", () => {
    it("writes container stderr to log file and does not emit raw content to core.info", async () => {
      vi.spyOn(fs, "mkdirSync").mockReturnValue(undefined);
      vi.spyOn(fs, "writeFileSync").mockReturnValue(undefined);

      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({
          exitCode: 0,
          stdout: '{"apiKey":"must-not-be-logged"}',
          stderr: "gateway diagnostic",
        }),
      };
      const coreApi = { info: vi.fn() };

      await collectGatewayStderr(execApi, coreApi);

      expect(execApi.getExecOutput).toHaveBeenCalledWith("docker", ["logs", "awmg-mcpg"], {
        ignoreReturnCode: true,
        silent: true,
      });
      // Must write to the excluded log file, not emit to the Actions log
      expect(fs.writeFileSync).toHaveBeenCalledWith(expect.stringContaining("stderr.log"), "gateway diagnostic", { mode: 0o600 });
      // Must not emit raw stderr content
      expect(coreApi.info).not.toHaveBeenCalledWith(expect.stringContaining("gateway diagnostic"));
      expect(coreApi.info).not.toHaveBeenCalledWith(expect.stringContaining("must-not-be-logged"));
      // Should report byte count only
      expect(coreApi.info).toHaveBeenCalledWith(expect.stringMatching(/MCP Gateway stderr written to log/));
    });

    it("reports unavailable when docker logs fails with no output", async () => {
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" }),
      };
      const coreApi = { info: vi.fn() };

      await collectGatewayStderr(execApi, coreApi);

      expect(coreApi.info).toHaveBeenCalledWith("MCP Gateway stderr is unavailable.");
    });
  });

  describe("removeGatewayContainer", () => {
    it("removes the container after log collection", async () => {
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
      };
      const coreApi = { info: vi.fn() };

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
      const coreApi = { info: vi.fn() };

      await removeGatewayContainer(execApi, coreApi);

      expect(coreApi.info).toHaveBeenCalledWith("MCP Gateway container already removed or not found.");
    });
  });
});
