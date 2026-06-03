import { describe, it, expect, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { runWithCopilotSDK } = require("./copilot_sdk_driver.cjs");

describe("copilot_sdk_driver.cjs", () => {
  describe("runWithCopilotSDK", () => {
    it("disconnects session and stops client on success", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      let onEvent = () => {};
      const session = {
        sessionId: "session-success",
        on: handler => {
          onEvent = handler;
        },
        sendAndWait: vi.fn().mockImplementation(async () => {
          onEvent({
            type: "assistant.message",
            ephemeral: false,
            timestamp: new Date().toISOString(),
            data: { content: "hello from sdk" },
          });
          return { data: { content: "hello from sdk" } };
        }),
        disconnect,
      };
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = vi.fn().mockResolvedValue(session);
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll: () => "allow",
        },
      });

      expect(result.exitCode).toBe(0);
      expect(result.hasOutput).toBe(true);
      expect(result.output).toContain("hello from sdk");
      expect(disconnect).toHaveBeenCalledTimes(1);
      expect(stop).toHaveBeenCalledTimes(1);
    });

    it("disconnects session and stops client on send failure", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const session = {
        sessionId: "session-failure",
        on: () => {},
        sendAndWait: vi.fn().mockRejectedValue(new Error("send failed")),
        disconnect,
      };
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = vi.fn().mockResolvedValue(session);
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll: () => "allow",
        },
      });

      expect(result.exitCode).toBe(1);
      expect(result.output).toContain("send failed");
      expect(disconnect).toHaveBeenCalledTimes(1);
      expect(stop).toHaveBeenCalledTimes(1);
    });

    it("passes custom provider and model through to SDK createSession", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const forUri = vi.fn(() => ({}));
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-provider",
        on: () => {},
        sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
        disconnect,
      });
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        model: "gpt-5.4",
        provider: { type: "openai", baseUrl: "http://api-proxy:10002" },
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri },
          approveAll: () => "allow",
        },
      });

      expect(result.exitCode).toBe(0);
      expect(createSession).toHaveBeenCalledWith(
        expect.objectContaining({
          model: "gpt-5.4",
          provider: { type: "openai", baseUrl: "http://api-proxy:10002" },
        })
      );
      expect(forUri).toHaveBeenCalledWith("http://127.0.0.1:3002", {});
    });

    it("passes COPILOT_CONNECTION_TOKEN to RuntimeConnection.forUri", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const connection = { kind: "uri", url: "http://127.0.0.1:3002", connectionToken: "token-123" };
      const forUri = vi.fn(() => connection);
      const constructorSpy = vi.fn();
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-connection-token",
        on: () => {},
        sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
        disconnect,
      });
      class FakeCopilotClient {
        constructor(options) {
          constructorSpy(options);
        }
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        connectionToken: "token-123",
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri },
          approveAll: () => "allow",
        },
      });

      expect(result.exitCode).toBe(0);
      expect(forUri).toHaveBeenCalledWith("http://127.0.0.1:3002", { connectionToken: "token-123" });
      expect(constructorSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          connection,
        })
      );
    });

    it("uses scoped permission handler from SDK permission config", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-permissions",
        on: () => {},
        sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
        disconnect,
      });
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        permissionConfig: {
          allowedTools: ["shell(git:*)", "github(get_file_contents)", "web_fetch", "write"],
        },
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll: () => ({ kind: "approve-once" }),
        },
      });

      expect(result.exitCode).toBe(0);
      const sessionConfig = createSession.mock.calls[0][0];
      const onPermissionRequest = sessionConfig.onPermissionRequest;
      expect(onPermissionRequest({ kind: "shell", commands: [{ identifier: "git" }], fullCommandText: "git status" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "mcp", serverName: "github", toolName: "get_file_contents" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "url", url: "https://example.com" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "write", fileName: "a.txt", diff: "", intention: "" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "read", fileName: "a.txt" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });
      expect(onPermissionRequest({ kind: "shell", commands: [{ identifier: "rm" }], fullCommandText: "rm -rf /tmp/x" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });
    });

    it("allows read requests when read is explicitly allowlisted", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-read-allowed",
        on: () => {},
        sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
        disconnect,
      });
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        permissionConfig: {
          allowedTools: ["read"],
        },
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll: () => ({ kind: "approve-once" }),
        },
      });

      expect(result.exitCode).toBe(0);
      const sessionConfig = createSession.mock.calls[0][0];
      const onPermissionRequest = sessionConfig.onPermissionRequest;
      expect(onPermissionRequest({ kind: "read", fileName: "a.txt" })).toEqual({ kind: "approve-once" });
    });

    it("logs permission-denied SDK requests as core warnings", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-permission-warnings",
        on: () => {},
        sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
        disconnect,
      });
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }
      const coreLogger = {
        info: vi.fn(),
        warning: vi.fn(),
      };

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        permissionConfig: {
          allowedTools: ["shell(git:*)"],
        },
        coreLogger,
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll: () => ({ kind: "approve-once" }),
        },
      });

      expect(result.exitCode).toBe(0);
      const sessionConfig = createSession.mock.calls[0][0];
      const onPermissionRequest = sessionConfig.onPermissionRequest;
      expect(onPermissionRequest({ kind: "shell", commands: [{ identifier: "rm" }], fullCommandText: "rm -rf /tmp/x" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });
      expect(coreLogger.info).toHaveBeenCalledWith(expect.stringContaining("shell(rm -rf /tmp/x)"));
      expect(coreLogger.warning).toHaveBeenCalledWith(expect.stringContaining("shell(rm -rf /tmp/x)"));
    });

    it("uses SDK default permission behavior when no permissionConfig is provided", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-default-permissions",
        on: () => {},
        sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
        disconnect,
      });
      const approveAll = vi.fn(() => ({ kind: "approve-once" }));
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }

      const result = await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll,
        },
      });

      expect(result.exitCode).toBe(0);
      const sessionConfig = createSession.mock.calls[0][0];
      // The SDK's default policy is exercised by omitting onPermissionRequest entirely.
      // This assertion verifies we do not force approve-all in the no-toolset path.
      expect(sessionConfig).not.toHaveProperty("onPermissionRequest");
      expect(approveAll).not.toHaveBeenCalled();
    });
  });
});
