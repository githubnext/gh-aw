import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";

describe("mcp_stdio_transport.cjs", () => {
  beforeEach(() => {
    vi.resetModules();
    // Suppress stderr output during tests
    vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("StdioTransport", () => {
    it("should create a new transport instance", async () => {
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      expect(transport).toBeDefined();
      expect(transport.isRunning()).toBe(false);
    });

    it("should accept debug option", async () => {
      const debugFn = vi.fn();
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport({ onDebug: debugFn });

      expect(transport).toBeDefined();
    });

    it("should set message handler", async () => {
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();
      const handler = vi.fn();

      transport.onMessage(handler);
      expect(transport._messageHandler).toBe(handler);
    });
  });

  describe("createStdioTransport", () => {
    it("should create a transport with factory function", async () => {
      const { createStdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = createStdioTransport();

      expect(transport).toBeDefined();
      expect(transport.isRunning()).toBe(false);
    });

    it("should pass options to transport", async () => {
      const debugFn = vi.fn();
      const { createStdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = createStdioTransport({ onDebug: debugFn });

      expect(transport._debug).toBe(debugFn);
    });
  });

  describe("send", () => {
    it("should write JSON message to stdout", async () => {
      const fs = await import("fs");
      const writeSyncSpy = vi.spyOn(fs.default, "writeSync").mockReturnValue(0);

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      const message = { jsonrpc: "2.0", id: 1, result: { status: "ok" } };
      transport.send(message);

      expect(writeSyncSpy).toHaveBeenCalledWith(1, expect.any(Uint8Array));

      // Decode the Uint8Array to verify the content
      const call = writeSyncSpy.mock.calls[0];
      const decoder = new TextDecoder();
      const written = decoder.decode(call[1]);
      expect(written).toBe(JSON.stringify(message) + "\n");
    });

    it("should call debug with send message", async () => {
      const fs = await import("fs");
      vi.spyOn(fs.default, "writeSync").mockReturnValue(0);

      const debugFn = vi.fn();
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport({ onDebug: debugFn });

      const message = { jsonrpc: "2.0", id: 1, result: {} };
      transport.send(message);

      expect(debugFn).toHaveBeenCalledWith(expect.stringContaining("send:"));
    });
  });

  describe("start", () => {
    it("should mark transport as running", async () => {
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockReturnThis();
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      expect(transport.isRunning()).toBe(false);
      transport.start();
      expect(transport.isRunning()).toBe(true);

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should set up stdin listeners", async () => {
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockReturnThis();
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      transport.start();

      expect(stdinOnSpy).toHaveBeenCalledWith("data", expect.any(Function));
      expect(stdinOnSpy).toHaveBeenCalledWith("error", expect.any(Function));
      expect(stdinResumeSpy).toHaveBeenCalled();

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should not start twice", async () => {
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockReturnThis();
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      transport.start();
      transport.start();

      // Should only set up listeners once
      expect(stdinResumeSpy).toHaveBeenCalledTimes(1);

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should call debug on start", async () => {
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockReturnThis();
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const debugFn = vi.fn();
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport({ onDebug: debugFn });

      transport.start();

      expect(debugFn).toHaveBeenCalledWith(expect.stringContaining("starting"));
      expect(debugFn).toHaveBeenCalledWith(expect.stringContaining("listening"));

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });
  });

  describe("close", () => {
    it("should mark transport as not running", async () => {
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockReturnThis();
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      transport.start();
      expect(transport.isRunning()).toBe(true);

      transport.close();
      expect(transport.isRunning()).toBe(false);

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should clear message handler", async () => {
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockReturnThis();
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      transport.onMessage(() => {});
      transport.start();
      transport.close();

      expect(transport._messageHandler).toBeNull();

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should do nothing if not started", async () => {
      const debugFn = vi.fn();
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport({ onDebug: debugFn });

      transport.close();

      // Should not call debug for closing if not started
      expect(debugFn).not.toHaveBeenCalledWith(expect.stringContaining("closing"));
    });
  });

  describe("message handling", () => {
    it("should call message handler when data is received", async () => {
      let dataHandler = null;
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
        if (event === "data") {
          dataHandler = handler;
        }
        return process.stdin;
      });
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      const messageHandler = vi.fn();
      transport.onMessage(messageHandler);
      transport.start();

      // Simulate receiving data
      const message = { jsonrpc: "2.0", id: 1, method: "test" };
      const data = Buffer.from(JSON.stringify(message) + "\n");
      dataHandler(data);

      expect(messageHandler).toHaveBeenCalledWith(message);

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should handle multiple messages in one chunk", async () => {
      let dataHandler = null;
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
        if (event === "data") {
          dataHandler = handler;
        }
        return process.stdin;
      });
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      const messageHandler = vi.fn();
      transport.onMessage(messageHandler);
      transport.start();

      // Simulate receiving multiple messages
      const msg1 = { jsonrpc: "2.0", id: 1, method: "test1" };
      const msg2 = { jsonrpc: "2.0", id: 2, method: "test2" };
      const data = Buffer.from(JSON.stringify(msg1) + "\n" + JSON.stringify(msg2) + "\n");
      dataHandler(data);

      expect(messageHandler).toHaveBeenCalledTimes(2);
      expect(messageHandler).toHaveBeenCalledWith(msg1);
      expect(messageHandler).toHaveBeenCalledWith(msg2);

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should handle partial messages across chunks", async () => {
      let dataHandler = null;
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
        if (event === "data") {
          dataHandler = handler;
        }
        return process.stdin;
      });
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport();

      const messageHandler = vi.fn();
      transport.onMessage(messageHandler);
      transport.start();

      // Simulate receiving a partial message
      const message = { jsonrpc: "2.0", id: 1, method: "test" };
      const fullJson = JSON.stringify(message) + "\n";
      const part1 = fullJson.substring(0, 10);
      const part2 = fullJson.substring(10);

      dataHandler(Buffer.from(part1));
      expect(messageHandler).not.toHaveBeenCalled();

      dataHandler(Buffer.from(part2));
      expect(messageHandler).toHaveBeenCalledWith(message);

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should handle parse errors gracefully", async () => {
      let dataHandler = null;
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
        if (event === "data") {
          dataHandler = handler;
        }
        return process.stdin;
      });
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const debugFn = vi.fn();
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport({ onDebug: debugFn });

      const messageHandler = vi.fn();
      transport.onMessage(messageHandler);
      transport.start();

      // Simulate receiving invalid JSON
      const data = Buffer.from("invalid json\n");
      dataHandler(data);

      expect(messageHandler).not.toHaveBeenCalled();
      expect(debugFn).toHaveBeenCalledWith(expect.stringContaining("Parse error"));

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });

    it("should call debug with received message", async () => {
      let dataHandler = null;
      const stdinOnSpy = vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
        if (event === "data") {
          dataHandler = handler;
        }
        return process.stdin;
      });
      const stdinResumeSpy = vi.spyOn(process.stdin, "resume").mockReturnThis();

      const debugFn = vi.fn();
      const { StdioTransport } = await import("./mcp_stdio_transport.cjs");
      const transport = new StdioTransport({ onDebug: debugFn });

      transport.onMessage(() => {});
      transport.start();

      const message = { jsonrpc: "2.0", id: 1, method: "test" };
      const data = Buffer.from(JSON.stringify(message) + "\n");
      dataHandler(data);

      expect(debugFn).toHaveBeenCalledWith(expect.stringContaining("recv:"));

      stdinOnSpy.mockRestore();
      stdinResumeSpy.mockRestore();
    });
  });
});
