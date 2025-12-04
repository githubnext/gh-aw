import { describe, it, expect } from "vitest";

describe("safe_inputs_tool_factory.cjs", () => {
  describe("createToolConfig", () => {
    it("should create a tool configuration with all parameters", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createToolConfig(
        "my_tool",
        "My tool description",
        { type: "object", properties: { input: { type: "string" } } },
        "my_tool.handler"
      );

      expect(config.name).toBe("my_tool");
      expect(config.description).toBe("My tool description");
      expect(config.inputSchema).toEqual({ type: "object", properties: { input: { type: "string" } } });
      expect(config.handler).toBe("my_tool.handler");
    });

    it("should create configuration with minimal parameters", async () => {
      const { createToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createToolConfig("tool", "desc", {}, "handler");

      expect(config.name).toBe("tool");
      expect(config.description).toBe("desc");
      expect(config.inputSchema).toEqual({});
      expect(config.handler).toBe("handler");
    });
  });

  describe("createJsToolConfig", () => {
    it("should create a tool configuration for JavaScript handler", async () => {
      const { createJsToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createJsToolConfig(
        "my_tool",
        "My tool description",
        { type: "object", properties: { input: { type: "string" } } },
        "my_tool.cjs"
      );

      expect(config.name).toBe("my_tool");
      expect(config.description).toBe("My tool description");
      expect(config.inputSchema).toEqual({ type: "object", properties: { input: { type: "string" } } });
      expect(config.handler).toBe("my_tool.cjs");
    });

    it("should create configuration with minimal parameters", async () => {
      const { createJsToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createJsToolConfig("tool", "desc", {}, "handler.cjs");

      expect(config.name).toBe("tool");
      expect(config.description).toBe("desc");
      expect(config.inputSchema).toEqual({});
      expect(config.handler).toBe("handler.cjs");
    });

    it("should handle complex input schemas", async () => {
      const { createJsToolConfig } = await import("./safe_inputs_tool_factory.cjs");

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

      const config = createJsToolConfig("complex_tool", "Complex tool", complexSchema, "complex.cjs");

      expect(config.inputSchema).toEqual(complexSchema);
    });
  });

  describe("createShellToolConfig", () => {
    it("should create a tool configuration for shell script handler", async () => {
      const { createShellToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createShellToolConfig(
        "shell_tool",
        "Shell tool description",
        { type: "object", properties: { message: { type: "string" } } },
        "shell_tool.sh"
      );

      expect(config.name).toBe("shell_tool");
      expect(config.description).toBe("Shell tool description");
      expect(config.inputSchema).toEqual({ type: "object", properties: { message: { type: "string" } } });
      expect(config.handler).toBe("shell_tool.sh");
    });

    it("should create shell configuration with empty schema", async () => {
      const { createShellToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createShellToolConfig("script", "Simple script", {}, "script.sh");

      expect(config.name).toBe("script");
      expect(config.handler).toBe("script.sh");
    });
  });

  describe("createPythonToolConfig", () => {
    it("should create a tool configuration for Python script handler", async () => {
      const { createPythonToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const config = createPythonToolConfig(
        "python_tool",
        "Python tool description",
        { type: "object", properties: { data: { type: "string" } } },
        "python_tool.py"
      );

      expect(config.name).toBe("python_tool");
      expect(config.description).toBe("Python tool description");
      expect(config.inputSchema).toEqual({ type: "object", properties: { data: { type: "string" } } });
      expect(config.handler).toBe("python_tool.py");
    });

    it("should create Python configuration with complex types", async () => {
      const { createPythonToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const schema = {
        type: "object",
        properties: {
          numbers: {
            type: "array",
            items: { type: "number" },
            description: "List of numbers",
          },
          operation: {
            type: "string",
            enum: ["sum", "average", "min", "max"],
          },
        },
      };

      const config = createPythonToolConfig("analyzer", "Number analyzer", schema, "analyzer.py");

      expect(config.handler).toBe("analyzer.py");
      expect(config.inputSchema.properties.numbers.type).toBe("array");
      expect(config.inputSchema.properties.operation.enum).toContain("sum");
    });
  });

  describe("All factory functions", () => {
    it("should create configurations with consistent structure", async () => {
      const { createJsToolConfig, createShellToolConfig, createPythonToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const jsConfig = createJsToolConfig("js", "JS", {}, "js.cjs");
      const shellConfig = createShellToolConfig("sh", "Shell", {}, "sh.sh");
      const pyConfig = createPythonToolConfig("py", "Python", {}, "py.py");

      // All should have the same structure
      expect(Object.keys(jsConfig).sort()).toEqual(["name", "description", "inputSchema", "handler"].sort());
      expect(Object.keys(shellConfig).sort()).toEqual(["name", "description", "inputSchema", "handler"].sort());
      expect(Object.keys(pyConfig).sort()).toEqual(["name", "description", "inputSchema", "handler"].sort());
    });

    it("should handle handler paths with different extensions", async () => {
      const { createJsToolConfig, createShellToolConfig, createPythonToolConfig } = await import("./safe_inputs_tool_factory.cjs");

      const jsConfig = createJsToolConfig("t1", "Tool", {}, "tools/handler.cjs");
      const shellConfig = createShellToolConfig("t2", "Tool", {}, "../scripts/handler.sh");
      const pyConfig = createPythonToolConfig("t3", "Tool", {}, "/absolute/path/handler.py");

      expect(jsConfig.handler).toBe("tools/handler.cjs");
      expect(shellConfig.handler).toBe("../scripts/handler.sh");
      expect(pyConfig.handler).toBe("/absolute/path/handler.py");
    });
  });
});
