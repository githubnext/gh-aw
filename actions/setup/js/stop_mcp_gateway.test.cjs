import { describe, expect, it, vi } from "vitest";
import { printGatewayStderr } from "./stop_mcp_gateway.cjs";

describe("stop_mcp_gateway", () => {
  it("emits only container stderr through core.info", async () => {
    const execApi = {
      getExecOutput: vi.fn().mockResolvedValue({
        exitCode: 0,
        stdout: '{"apiKey":"must-not-be-logged"}',
        stderr: "gateway diagnostic",
      }),
    };
    const coreApi = { info: vi.fn() };

    await printGatewayStderr(execApi, coreApi);

    expect(execApi.getExecOutput).toHaveBeenCalledWith("docker", ["logs", "awmg-mcpg"], {
      ignoreReturnCode: true,
      silent: true,
    });
    expect(coreApi.info).toHaveBeenCalledWith("MCP Gateway stderr:\ngateway diagnostic");
    expect(coreApi.info).not.toHaveBeenCalledWith(expect.stringContaining("must-not-be-logged"));
  });

  it("reports when no stderr was produced", async () => {
    const execApi = {
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
    };
    const coreApi = { info: vi.fn() };

    await printGatewayStderr(execApi, coreApi);

    expect(coreApi.info).toHaveBeenCalledWith("MCP Gateway produced no stderr output.");
  });
});
