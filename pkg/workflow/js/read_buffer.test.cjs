import { describe, it, expect, beforeEach, vi } from "vitest";

describe("read_buffer.cjs", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  describe("ReadBuffer", () => {
    it("should parse complete JSON messages from buffer", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"jsonrpc":"2.0","id":1,"method":"test"}\n'));

      const message = buffer.readMessage();
      expect(message).toEqual({
        jsonrpc: "2.0",
        id: 1,
        method: "test",
      });
    });

    it("should handle incomplete messages", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"jsonrpc":"2.0"'));
      expect(buffer.readMessage()).toBeNull();

      buffer.append(Buffer.from(',"id":1,"method":"test"}\n'));
      expect(buffer.readMessage()).toEqual({
        jsonrpc: "2.0",
        id: 1,
        method: "test",
      });
    });

    it("should skip empty lines", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('\n\n{"jsonrpc":"2.0","id":1,"method":"test"}\n'));

      const message = buffer.readMessage();
      expect(message).toEqual({
        jsonrpc: "2.0",
        id: 1,
        method: "test",
      });
    });

    it("should throw on invalid JSON", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from("invalid json\n"));

      expect(() => buffer.readMessage()).toThrow("Parse error");
    });

    it("should handle Windows line endings", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"jsonrpc":"2.0","id":1,"method":"test"}\r\n'));

      const message = buffer.readMessage();
      expect(message).toEqual({
        jsonrpc: "2.0",
        id: 1,
        method: "test",
      });
    });

    it("should handle multiple messages in buffer", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"jsonrpc":"2.0","id":1,"method":"first"}\n{"jsonrpc":"2.0","id":2,"method":"second"}\n'));

      const first = buffer.readMessage();
      expect(first).toEqual({
        jsonrpc: "2.0",
        id: 1,
        method: "first",
      });

      const second = buffer.readMessage();
      expect(second).toEqual({
        jsonrpc: "2.0",
        id: 2,
        method: "second",
      });

      // No more messages
      expect(buffer.readMessage()).toBeNull();
    });

    it("should handle messages split across multiple appends", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"json'));
      expect(buffer.readMessage()).toBeNull();

      buffer.append(Buffer.from('rpc":"2.0"'));
      expect(buffer.readMessage()).toBeNull();

      buffer.append(Buffer.from(',"id":1,"method":"test"}\n'));
      expect(buffer.readMessage()).toEqual({
        jsonrpc: "2.0",
        id: 1,
        method: "test",
      });
    });

    it("should return null when buffer is empty", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      expect(buffer.readMessage()).toBeNull();
    });

    it("should return null when no newline in buffer", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"incomplete":"message"}'));
      expect(buffer.readMessage()).toBeNull();
    });

    it("should handle deeply nested JSON", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      const nested = {
        jsonrpc: "2.0",
        id: 1,
        method: "test",
        params: {
          level1: {
            level2: {
              level3: {
                value: "deep",
              },
            },
          },
        },
      };

      buffer.append(Buffer.from(JSON.stringify(nested) + "\n"));

      const message = buffer.readMessage();
      expect(message).toEqual(nested);
    });

    it("should handle messages with unicode characters", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"message":"Hello 世界 🌍"}\n'));

      const message = buffer.readMessage();
      expect(message).toEqual({ message: "Hello 世界 🌍" });
    });

    it("should handle messages with escaped characters", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('{"message":"line1\\nline2\\ttab"}\n'));

      const message = buffer.readMessage();
      expect(message).toEqual({ message: "line1\nline2\ttab" });
    });

    it("should skip multiple consecutive empty lines", async () => {
      const { ReadBuffer } = await import("./read_buffer.cjs");
      const buffer = new ReadBuffer();

      buffer.append(Buffer.from('\n\n\n\n{"jsonrpc":"2.0","id":1}\n\n\n'));

      const message = buffer.readMessage();
      expect(message).toEqual({ jsonrpc: "2.0", id: 1 });

      // Skip remaining empty lines, return null
      expect(buffer.readMessage()).toBeNull();
    });
  });
});
