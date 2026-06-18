import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import { createRequire } from "module";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";

const require = createRequire(import.meta.url);
const { runWithCopilotSDK, parsePermissionConfigFromServerArgs } = require("./copilot_sdk_driver.cjs");

describe("copilot_sdk_driver.cjs", () => {
  let testSessionStateDir;
  let prevSessionStateDir;
  beforeAll(() => {
    prevSessionStateDir = process.env.GH_AW_SESSION_STATE_BASE_DIR;
    testSessionStateDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-test-session-state-"));
    process.env.GH_AW_SESSION_STATE_BASE_DIR = testSessionStateDir;
  });
  afterAll(() => {
    if (prevSessionStateDir === undefined) delete process.env.GH_AW_SESSION_STATE_BASE_DIR;
    else process.env.GH_AW_SESSION_STATE_BASE_DIR = prevSessionStateDir;
    if (testSessionStateDir) fs.rmSync(testSessionStateDir, { recursive: true, force: true });
  });

  describe("runWithCopilotSDK", () => {
    it("disconnects session and stops client on success", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const stderrWriteSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
      try {
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
        const parsedEvents = stderrWriteSpy.mock.calls
          .map(([message]) => {
            if (typeof message !== "string" || !message.endsWith("\n")) return null;
            try {
              return JSON.parse(message.trimEnd());
            } catch {
              return null;
            }
          })
          .filter(Boolean);
        const parsedEvent = parsedEvents.find(event => event.type === "assistant.message");
        expect(parsedEvent).toMatchObject({
          type: "assistant.message",
          data: { content: "hello from sdk" },
        });
        expect(typeof parsedEvent.timestamp).toBe("string");
      } finally {
        stderrWriteSpy.mockRestore();
      }
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
      expect(onPermissionRequest({ kind: "read", path: "a.txt", intention: "" })).toEqual({
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
      expect(onPermissionRequest({ kind: "read", path: "a.txt", intention: "" })).toEqual({ kind: "approve-once" });
    });

    it("allows read requests when read(path) is explicitly allowlisted", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-read-path-allowed",
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
          allowedTools: ["read(/tmp/gh-aw/agent/*)"],
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
      // Copilot SDK treats any granted read capability as global; read(path) entries are
      // effectively equivalent to read for the onPermissionRequest contract.
      expect(onPermissionRequest({ kind: "read", path: "a.txt", intention: "" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "read", path: "/etc/passwd", intention: "" })).toEqual({ kind: "approve-once" });
    });

    it("allows read requests when shell access is allowlisted", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-read-via-shell",
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
          allowedTools: ["shell"],
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
      expect(onPermissionRequest({ kind: "read", path: "a.txt", intention: "" })).toEqual({ kind: "approve-once" });
    });

    it("allows read requests that match read-only shell path rules", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const createSession = vi.fn().mockResolvedValue({
        sessionId: "session-read-via-shell-paths",
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
          allowedTools: ["shell(cat /tmp/gh-aw/agent/*)", "shell(cat /tmp/gh-aw/agent/**/*.txt)", "shell(xargs -a /tmp/gh-aw/agent/doc-samples.txt cat)", "shell(ls /tmp/gh-aw/repo-memory/default/)"],
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
      expect(onPermissionRequest({ kind: "read", path: "/tmp/gh-aw/agent/doc-samples.txt", intention: "" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "read", path: "/tmp/gh-aw/agent/subdir/nested.txt", intention: "" })).toEqual({
        kind: "approve-once",
      });
      expect(onPermissionRequest({ kind: "read", path: "/tmp/gh-aw/agent/previous-findings.json", intention: "" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "read", path: "/tmp/gh-aw/repo-memory/default", intention: "" })).toEqual({ kind: "approve-once" });
      expect(onPermissionRequest({ kind: "read", path: "/etc/passwd", intention: "" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });
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

    it("always configures onPermissionRequest and defaults to approveAll when permissionConfig is absent", async () => {
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
      expect(sessionConfig).toHaveProperty("onPermissionRequest");
      const decision = sessionConfig.onPermissionRequest({ kind: "read", path: "a.txt", intention: "" });
      expect(decision).toEqual({ kind: "approve-once" });
      expect(approveAll).toHaveBeenCalledTimes(1);
    });

    it("stops session when permission denials reach max-tool-denials threshold", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      const stderrWriteSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
      let sessionConfig;
      const session = {
        sessionId: "session-max-tool-denials",
        on: () => {},
        sendAndWait: vi.fn().mockImplementation(async () => {
          const denyRequest = { kind: "shell", commands: [{ identifier: "rm" }], fullCommandText: "rm -rf /tmp/x" };
          sessionConfig.onPermissionRequest(denyRequest);
          sessionConfig.onPermissionRequest(denyRequest);
          sessionConfig.onPermissionRequest(denyRequest);
          return { data: { content: "should-not-complete" } };
        }),
        disconnect,
      };
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = vi.fn().mockImplementation(async config => {
          sessionConfig = config;
          return session;
        });
        stop = stop;
      }

      const oldMaxToolDenials = process.env.GH_AW_MAX_TOOL_DENIALS;
      process.env.GH_AW_MAX_TOOL_DENIALS = "3";
      try {
        const result = await runWithCopilotSDK({
          sdkUri: "http://127.0.0.1:3002",
          prompt: "test prompt",
          logger: () => {},
          permissionConfig: {
            allowedTools: ["shell(git:*)"],
          },
          sdkModule: {
            CopilotClient: FakeCopilotClient,
            RuntimeConnection: { forUri: vi.fn(() => ({})) },
            approveAll: () => ({ kind: "approve-once" }),
          },
        });

        expect(result.exitCode).toBe(1);
        expect(result.output).toContain("max tool denials threshold reached");
        expect(disconnect).toHaveBeenCalled();
        const parsedEvents = stderrWriteSpy.mock.calls
          .map(([message]) => {
            if (typeof message !== "string" || !message.endsWith("\n")) return null;
            try {
              return JSON.parse(message.trimEnd());
            } catch {
              return null;
            }
          })
          .filter(Boolean);
        const toolDenialsEvent = parsedEvents.find(event => event.type === "guard.tool_denials_exceeded");
        expect(toolDenialsEvent).toMatchObject({
          type: "guard.tool_denials_exceeded",
          data: {
            denialCount: 3,
            threshold: 3,
            reason: expect.stringContaining("permission denied"),
          },
        });
      } finally {
        stderrWriteSpy.mockRestore();
        if (oldMaxToolDenials === undefined) {
          delete process.env.GH_AW_MAX_TOOL_DENIALS;
        } else {
          process.env.GH_AW_MAX_TOOL_DENIALS = oldMaxToolDenials;
        }
      }
    });

    it("falls back to default threshold when GH_AW_MAX_TOOL_DENIALS is malformed", async () => {
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      let sessionConfig;
      const session = {
        sessionId: "session-max-tool-denials-malformed-env",
        on: () => {},
        sendAndWait: vi.fn().mockImplementation(async () => {
          const denyRequest = { kind: "shell", commands: [{ identifier: "rm" }], fullCommandText: "rm -rf /tmp/x" };
          sessionConfig.onPermissionRequest(denyRequest);
          sessionConfig.onPermissionRequest(denyRequest);
          sessionConfig.onPermissionRequest(denyRequest);
          return { data: { content: "completed" } };
        }),
        disconnect,
      };
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = vi.fn().mockImplementation(async config => {
          sessionConfig = config;
          return session;
        });
        stop = stop;
      }

      const oldMaxToolDenials = process.env.GH_AW_MAX_TOOL_DENIALS;
      process.env.GH_AW_MAX_TOOL_DENIALS = "3ms";
      try {
        const result = await runWithCopilotSDK({
          sdkUri: "http://127.0.0.1:3002",
          prompt: "test prompt",
          logger: () => {},
          permissionConfig: {
            allowedTools: ["shell(git:*)"],
          },
          sdkModule: {
            CopilotClient: FakeCopilotClient,
            RuntimeConnection: { forUri: vi.fn(() => ({})) },
            approveAll: () => ({ kind: "approve-once" }),
          },
        });

        expect(result.exitCode).toBe(0);
      } finally {
        if (oldMaxToolDenials === undefined) {
          delete process.env.GH_AW_MAX_TOOL_DENIALS;
        } else {
          process.env.GH_AW_MAX_TOOL_DENIALS = oldMaxToolDenials;
        }
      }
    });

    it("returns threshold error when sendAndWait fails after catastrophic denial disconnect", async () => {
      let sessionConfig;
      let disconnected = false;
      const disconnect = vi.fn().mockImplementation(async () => {
        disconnected = true;
      });
      const stop = vi.fn().mockResolvedValue(undefined);
      const session = {
        sessionId: "session-max-tool-denials-disconnect",
        on: () => {},
        sendAndWait: vi.fn().mockImplementation(async () => {
          const denyRequest = { kind: "shell", commands: [{ identifier: "rm" }], fullCommandText: "rm -rf /tmp/x" };
          sessionConfig.onPermissionRequest(denyRequest);
          sessionConfig.onPermissionRequest(denyRequest);
          if (disconnected) {
            throw new Error("transport disconnected");
          }
          return { data: { content: "unexpected" } };
        }),
        disconnect,
      };
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = vi.fn().mockImplementation(async config => {
          sessionConfig = config;
          return session;
        });
        stop = stop;
      }

      const oldMaxToolDenials = process.env.GH_AW_MAX_TOOL_DENIALS;
      process.env.GH_AW_MAX_TOOL_DENIALS = "2";
      try {
        const result = await runWithCopilotSDK({
          sdkUri: "http://127.0.0.1:3002",
          prompt: "test prompt",
          logger: () => {},
          permissionConfig: {
            allowedTools: ["shell(git:*)"],
          },
          sdkModule: {
            CopilotClient: FakeCopilotClient,
            RuntimeConnection: { forUri: vi.fn(() => ({})) },
            approveAll: () => ({ kind: "approve-once" }),
          },
        });

        expect(result.exitCode).toBe(1);
        expect(result.output).toContain("max tool denials threshold reached");
      } finally {
        if (oldMaxToolDenials === undefined) {
          delete process.env.GH_AW_MAX_TOOL_DENIALS;
        } else {
          process.env.GH_AW_MAX_TOOL_DENIALS = oldMaxToolDenials;
        }
      }
    });
  });

  describe("parsePermissionConfigFromServerArgs", () => {
    it("returns undefined when input is undefined", () => {
      expect(parsePermissionConfigFromServerArgs(undefined)).toBeUndefined();
    });

    it("returns undefined when input is empty string", () => {
      expect(parsePermissionConfigFromServerArgs("")).toBeUndefined();
    });

    it("returns undefined when input is invalid JSON", () => {
      expect(parsePermissionConfigFromServerArgs("not-json")).toBeUndefined();
    });

    it("returns undefined when input is not an array", () => {
      expect(parsePermissionConfigFromServerArgs('{"key":"value"}')).toBeUndefined();
    });

    it("returns undefined when args contain no permission flags", () => {
      const args = JSON.stringify(["--headless", "--no-auto-update", "--port", "3002"]);
      expect(parsePermissionConfigFromServerArgs(args)).toBeUndefined();
    });

    it("returns allowAllTools:true when --allow-all-tools is present", () => {
      const args = JSON.stringify(["--headless", "--allow-all-tools", "--port", "3002"]);
      expect(parsePermissionConfigFromServerArgs(args)).toEqual({ allowAllTools: true });
    });

    it("--allow-all-tools takes precedence over --allow-tool entries", () => {
      const args = JSON.stringify(["--allow-tool", "shell(git:*)", "--allow-all-tools", "--allow-tool", "write"]);
      expect(parsePermissionConfigFromServerArgs(args)).toEqual({ allowAllTools: true });
    });

    it("extracts a single --allow-tool entry", () => {
      const args = JSON.stringify(["--allow-tool", "safeoutputs"]);
      expect(parsePermissionConfigFromServerArgs(args)).toEqual({ allowedTools: ["safeoutputs"] });
    });

    it("extracts multiple --allow-tool entries preserving order", () => {
      const args = JSON.stringify(["--headless", "--no-ask-user", "--allow-tool", "github", "--allow-tool", "safeoutputs", "--allow-tool", "shell(safeoutputs:*)", "--allow-tool", "write"]);
      expect(parsePermissionConfigFromServerArgs(args)).toEqual({
        allowedTools: ["github", "safeoutputs", "shell(safeoutputs:*)", "write"],
      });
    });

    it("extracts shell(safeoutputs:*) from a realistic GH_AW_COPILOT_SDK_SERVER_ARGS value", () => {
      const args = JSON.stringify([
        "--headless",
        "--no-auto-update",
        "--port",
        "3002",
        "--no-ask-user",
        "--allow-tool",
        "github",
        "--allow-tool",
        "safeoutputs",
        "--allow-tool",
        "shell(agenticworkflows:*)",
        "--allow-tool",
        "shell(safeoutputs:*)",
        "--allow-tool",
        "shell(git:*)",
        "--allow-tool",
        "write",
        "--allow-all-paths",
      ]);
      const config = parsePermissionConfigFromServerArgs(args);
      expect(config).not.toBeNull();
      expect(config?.allowedTools).toContain("shell(safeoutputs:*)");
      expect(config?.allowedTools).toContain("safeoutputs");
      expect(config?.allowedTools).toContain("write");
    });

    it("ignores non-string array elements", () => {
      // Mixed arrays should not produce an error; only string entries are valid flags.
      const args = JSON.stringify(["--allow-tool", "write", null, 42, "--allow-tool", "safeoutputs"]);
      const config = parsePermissionConfigFromServerArgs(args);
      // null/42 are not the string "--allow-tool", so only the valid pairs are collected.
      expect(config).toEqual({ allowedTools: ["write", "safeoutputs"] });
    });
  });

  // ─────────────────────────────────────────────────────────────────────────
  // Piped / chained shell command permission tests
  //
  // These tests verify the fallback path in the permission handler that parses
  // fullCommandText when the Copilot SDK does not provide command identifiers.
  // This is the scenario that caused the GEO Optimizer daily audit to fail.
  // ─────────────────────────────────────────────────────────────────────────
  describe("buildCopilotSDKPermissionHandler – piped command support", () => {
    /**
     * Helper: build an onPermissionRequest handler with the given allowed tools
     * and return a function that checks a shell request with no command identifiers.
     */
    function makeHandler(allowedTools) {
      // We need access to buildCopilotSDKPermissionHandler via runWithCopilotSDK.
      // The simplest way is to exercise it through the same flow used in production.
      // We recreate the config here and call parsePermissionConfigFromServerArgs.
      const args = allowedTools.map(t => ["--allow-tool", t]).flat();
      const config = parsePermissionConfigFromServerArgs(JSON.stringify(args));
      // Build a minimal handler directly:
      // Import the internal helper used by runWithCopilotSDK via a round-trip
      // through a test-only re-export of buildCopilotSDKPermissionHandler.
      // Since that function is not exported, we exercise it through runWithCopilotSDK
      // in integration tests below.  Here we just verify config parsing is correct.
      return config;
    }

    it("parsePermissionConfigFromServerArgs round-trips piped-command allowed tools", () => {
      const config = makeHandler(["shell(ls)", "shell(cat)", "shell(echo)", "shell(safeoutputs:*)"]);
      expect(config?.allowedTools).toContain("shell(ls)");
      expect(config?.allowedTools).toContain("shell(cat)");
      expect(config?.allowedTools).toContain("shell(echo)");
      expect(config?.allowedTools).toContain("shell(safeoutputs:*)");
    });

    // Integration: drive permission handler through runWithCopilotSDK to verify
    // that piped commands are allowed when all their segments are in the allow-list.
    async function makePermissionHandlerViaSDK(allowedTools) {
      const { runWithCopilotSDK } = require("./copilot_sdk_driver.cjs");
      const { vi } = await import("vitest");
      const disconnect = vi.fn().mockResolvedValue(undefined);
      const stop = vi.fn().mockResolvedValue(undefined);
      let capturedHandler;
      const createSession = vi.fn().mockImplementation(async config => {
        capturedHandler = config.onPermissionRequest;
        return {
          sessionId: "session-pipe-test",
          on: () => {},
          sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
          disconnect,
        };
      });
      class FakeCopilotClient {
        start = vi.fn().mockResolvedValue(undefined);
        createSession = createSession;
        stop = stop;
      }
      await runWithCopilotSDK({
        sdkUri: "http://127.0.0.1:3002",
        prompt: "test prompt",
        logger: () => {},
        permissionConfig: { allowedTools },
        sdkModule: {
          CopilotClient: FakeCopilotClient,
          RuntimeConnection: { forUri: vi.fn(() => ({})) },
          approveAll: () => ({ kind: "approve-once" }),
        },
      });
      return capturedHandler;
    }

    it("allows a piped command when SDK provides no identifiers but all commands are allowed", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(ls)", "shell(cat)", "shell(echo)"]);
      // Simulate what the Copilot SDK sends for a piped command: commands: [] (empty)
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: 'ls /tmp/dir 2>/dev/null && echo "---" && cat /tmp/file.json 2>/dev/null || echo "not found"',
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("denies a piped command when any stage is not in the allow-list", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(ls)", "shell(echo)"]);
      // cat is NOT in the allow-list
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: "ls /tmp && cat /tmp/file.json && echo done",
      });
      expect(result).toEqual({ kind: "reject", feedback: "Tool invocation is not allowed by workflow tool permissions." });
    });

    it("allows a safeoutputs || echo pipeline when both are allowed", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(safeoutputs:*)", "shell(echo)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: 'safeoutputs missing_data --help 2>/dev/null || echo "unavailable"',
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("allows a pwd && ls && safeoutputs && printf pipeline when all are allowed", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(pwd)", "shell(ls)", "shell(safeoutputs:*)", "shell(printf)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: "pwd && ls -la && safeoutputs --help && printf '%s\\n' done",
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("allows a piped grep/wc command when both are in the allow-list", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(grep)", "shell(wc)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: "grep -r pattern /tmp | wc -l",
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("preserves original single-command behaviour when SDK provides identifiers", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(git:*)"]);
      // SDK provides identifiers (non-piped path)
      expect(handler({ kind: "shell", commands: [{ identifier: "git" }], fullCommandText: "git status" })).toEqual({
        kind: "approve-once",
      });
      expect(handler({ kind: "shell", commands: [{ identifier: "rm" }], fullCommandText: "rm -rf /tmp/x" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });
    });

    it("allows safeoutputs command when SDK identifier includes full command text", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(safeoutputs:*)"]);
      const result = handler({
        kind: "shell",
        commands: [{ identifier: "safeoutputs --help 2>&1" }],
        fullCommandText: "safeoutputs --help 2>&1",
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("allows mcpscripts command when SDK identifier includes full command text", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(mcpscripts:*)"]);
      const result = handler({
        kind: "shell",
        commands: [{ identifier: "mcpscripts gh issue list --repo github/gh-aw" }],
        fullCommandText: "mcpscripts gh issue list --repo github/gh-aw",
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("denies when fullCommandText is empty and no identifiers provided", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(ls)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: "",
      });
      expect(result).toEqual({ kind: "reject", feedback: "Tool invocation is not allowed by workflow tool permissions." });
    });

    it("allows a :* wildcard rule to match pipeline stages with the given prefix", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(gh:*)", "shell(echo)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: "gh issue list && echo done",
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("denies multiline shell command when required tools are missing", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(mkdir)", "shell(git:*)", "shell(printf)", "shell(cat)", "shell(wc)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: `set -euo pipefail
CACHE_DIR='cache/gh-aw/cache-memory/compiler-quality'
ANALYSES_DIR="$CACHE_DIR/analyses"
mkdir -p "$ANALYSES_DIR"
FILES='compiler.go compiler_activation_jobs.go compiler_orchestrator.go compiler_jobs.go compiler_safe_outputs.go compiler_safe_outputs_config.go compiler_safe_outputs_job.go compiler_yaml.go compiler_yaml_main_job.go'
for f in $FILES; do git -C /home/runner/work/gh-aw/gh-aw log -1 --format='%H' -- "pkg/workflow/$f" | sed "s|^|$f |"; done
printf '---ROTATION---\n'
if [ -f "$CACHE_DIR/rotation.json" ]; then cat "$CACHE_DIR/rotation.json"; fi
printf '\n---HASHES---\n'
if [ -f "$CACHE_DIR/file-hashes.json" ]; then cat "$CACHE_DIR/file-hashes.json"; fi
printf '\n---FILES---\n'
for f in $FILES; do wc -l "/home/runner/work/gh-aw/gh-aw/pkg/workflow/$f"; done`,
      });
      expect(result).toEqual({ kind: "reject", feedback: "Tool invocation is not allowed by workflow tool permissions." });
    });

    it("approves multiline shell command when all required tools are permitted", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(set)", "shell(mkdir)", "shell(git:*)", "shell(sed)", "shell(printf)", "shell(cat)", "shell(wc)"]);
      const result = handler({
        kind: "shell",
        commands: [],
        fullCommandText: `set -euo pipefail
CACHE_DIR='cache/gh-aw/cache-memory/compiler-quality'
ANALYSES_DIR="$CACHE_DIR/analyses"
mkdir -p "$ANALYSES_DIR"
FILES='compiler.go compiler_activation_jobs.go compiler_orchestrator.go compiler_jobs.go compiler_safe_outputs.go compiler_safe_outputs_config.go compiler_safe_outputs_job.go compiler_yaml.go compiler_yaml_main_job.go'
for f in $FILES; do git -C /home/runner/work/gh-aw/gh-aw log -1 --format='%H' -- "pkg/workflow/$f" | sed "s|^|$f |"; done
printf '---ROTATION---\n'
if [ -f "$CACHE_DIR/rotation.json" ]; then cat "$CACHE_DIR/rotation.json"; fi
printf '\n---HASHES---\n'
if [ -f "$CACHE_DIR/file-hashes.json" ]; then cat "$CACHE_DIR/file-hashes.json"; fi
printf '\n---FILES---\n'
for f in $FILES; do wc -l "/home/runner/work/gh-aw/gh-aw/pkg/workflow/$f"; done`,
      });
      expect(result).toEqual({ kind: "approve-once" });
    });

    it("requires explicit read permission for AGENTS.md and SKILL.md reads", async () => {
      const denied = await makePermissionHandlerViaSDK(["shell(ls)"]);
      expect(denied({ kind: "read", path: "/home/runner/work/gh-aw/gh-aw/AGENTS.md" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });
      expect(denied({ kind: "read", path: "/home/runner/work/gh-aw/gh-aw/SKILL.md" })).toEqual({
        kind: "reject",
        feedback: "Tool invocation is not allowed by workflow tool permissions.",
      });

      const allowed = await makePermissionHandlerViaSDK(["read"]);
      expect(allowed({ kind: "read", path: "/home/runner/work/gh-aw/gh-aw/AGENTS.md" })).toEqual({
        kind: "approve-once",
      });
      expect(allowed({ kind: "read", path: "/home/runner/work/gh-aw/gh-aw/SKILL.md" })).toEqual({
        kind: "approve-once",
      });
    });

    it("denies issue-37538 commands when workflow only allows jq shell usage", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(jq:*)"]);
      const deniedCommands = [
        "gh pr list --repo github/gh-aw --state open --draft --json number,title,author,createdAt,updatedAt,labels,headRefName --limit 100 2>&1",
        "safeoutputs --help 2>&1 | head -50",
        "git --no-pager status --short && gh pr list --repo github/gh-aw --state open --draft --json number,title,author,createdAt,updatedAt,labels,headRefName,comments,reviews --limit 100",
        'echo "test"',
      ];

      for (const command of deniedCommands) {
        expect(
          handler({
            kind: "shell",
            // Intentional: exercise fullCommandText fallback when SDK omits identifiers.
            commands: [],
            fullCommandText: command,
          })
        ).toEqual({ kind: "reject", feedback: "Tool invocation is not allowed by workflow tool permissions." });
      }
    });

    it("allows issue-37538 commands when corresponding shell permissions are granted", async () => {
      const handler = await makePermissionHandlerViaSDK(["shell(gh:*)", "shell(safeoutputs:*)", "shell(head)", "shell(git:*)", "shell(echo)"]);
      const allowedCommands = [
        "gh pr list --repo github/gh-aw --state open --draft --json number,title,author,createdAt,updatedAt,labels,headRefName --limit 100 2>&1",
        "safeoutputs --help 2>&1 | head -50",
        "git --no-pager status --short && gh pr list --repo github/gh-aw --state open --draft --json number,title,author,createdAt,updatedAt,labels,headRefName,comments,reviews --limit 100",
        'echo "test"',
      ];

      for (const command of allowedCommands) {
        expect(
          handler({
            kind: "shell",
            // Intentional: exercise fullCommandText fallback when SDK omits identifiers.
            commands: [],
            fullCommandText: command,
          })
        ).toEqual({ kind: "approve-once" });
      }
    });
  });
});
