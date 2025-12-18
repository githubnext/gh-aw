import { describe, it, expect } from "vitest";

describe("safe_inputs_tool_factory.cjs", () => {
  describe("createToolConfig", () => {
    it("should create a tool configuration with all parameters", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createToolConfig("my_tool", "My tool description", { type: "object", properties: { input: { type: "string" } } }, "my_tool.cjs");

      expect(config.name).toBe("my_tool");
      expect(config.description).toBe("My tool description");
      expect(config.inputSchema).toEqual({ type: "object", properties: { input: { type: "string" } } });
      expect(config.handler).toBe("my_tool.cjs");
    });

    it("should create configuration with minimal parameters", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createToolConfig("tool", "desc", {}, "handler.sh");

      expect(config.name).toBe("tool");
      expect(config.description).toBe("desc");
      expect(config.inputSchema).toEqual({});
      expect(config.handler).toBe("handler.sh");
    });

    it("should handle complex input schemas", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const complexSchema = {
        type: "object",
        properties: {
          name: { type: "string", description: "Name parameter" },
          count: { type: "number", default: 1 },
          options: {
            type: "object",
            properties: {
              verbose: { type: "boolean" },
            },
          },
        },
        required: ["name"],
      };

      const config = createToolConfig("complex_tool", "Complex tool", complexSchema, "complex.py");

      expect(config.inputSchema).toEqual(complexSchema);
    });

    it("should handle handler paths with different extensions", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const jsConfig = createToolConfig("t1", "JS Tool", {}, "tools/handler.cjs");
      const shellConfig = createToolConfig("t2", "Shell Tool", {}, "../scripts/handler.sh");
      const pyConfig = createToolConfig("t3", "Python Tool", {}, "/absolute/path/handler.py");

      expect(jsConfig.handler).toBe("tools/handler.cjs");
      expect(shellConfig.handler).toBe("../scripts/handler.sh");
      expect(pyConfig.handler).toBe("/absolute/path/handler.py");
    });

    it("should create configurations with consistent structure", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createToolConfig("test", "Test tool", {}, "test.cjs");

      expect(Object.keys(config).sort()).toEqual(["name", "description", "inputSchema", "handler"].sort());
    });
  });
});
