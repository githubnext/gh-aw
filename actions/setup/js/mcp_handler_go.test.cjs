// @ts-check

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createGoHandler } from "./mcp_handler_go.cjs";
import fs from "fs";
import path from "path";
import os from "os";

describe("createGoHandler", () => {
  let mockServer;
  let tempDir;
  let testScriptPath;

  beforeEach(() => {
    // Create mock server with debug logging
    mockServer = {
      debug: () => {},
      debugError: () => {},
    };

    // Create temporary directory for test scripts
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "go-handler-test-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("should execute a simple Go script", async () => {
    // Create a simple Go script that echoes input
    testScriptPath = path.join(tempDir, "test.go");
    const goCode = `package main

import (
	"encoding/json"
	"io"
	"os"
)

func main() {
	var inputs map[string]interface{}
	data, _ := io.ReadAll(os.Stdin)
	json.Unmarshal(data, &inputs)
	
	result := map[string]interface{}{
		"message": "Hello from Go",
		"input": inputs,
	}
	json.NewEncoder(os.Stdout).Encode(result)
}`;
    fs.writeFileSync(testScriptPath, goCode);

    const handler = createGoHandler(mockServer, "test-tool", testScriptPath, 60);
    const result = await handler({ name: "World", count: 42 });

    expect(result).toBeDefined();
    expect(result.content).toBeDefined();
    expect(result.content.length).toBe(1);
    expect(result.content[0].type).toBe("text");

    const output = JSON.parse(result.content[0].text);
    expect(output.message).toBe("Hello from Go");
    expect(output.input).toEqual({ name: "World", count: 42 });
  }, 30000); // Increase timeout to allow for Go compilation

  it("should handle Go script with no input", async () => {
    testScriptPath = path.join(tempDir, "no-input.go");
    const goCode = `package main

import (
	"encoding/json"
	"os"
)

func main() {
	result := map[string]interface{}{"status": "ok"}
	json.NewEncoder(os.Stdout).Encode(result)
}`;
    fs.writeFileSync(testScriptPath, goCode);

    const handler = createGoHandler(mockServer, "no-input-tool", testScriptPath);
    const result = await handler({});

    const output = JSON.parse(result.content[0].text);
    expect(output.status).toBe("ok");
  }, 30000); // Increase timeout to allow for Go compilation

  it("should handle non-JSON output", async () => {
    testScriptPath = path.join(tempDir, "text-output.go");
    const goCode = `package main

import "fmt"

func main() {
	fmt.Println("Plain text output")
}`;
    fs.writeFileSync(testScriptPath, goCode);

    const handler = createGoHandler(mockServer, "text-tool", testScriptPath);
    const result = await handler({});

    const output = JSON.parse(result.content[0].text);
    expect(output.stdout).toContain("Plain text output");
  }, 30000); // Increase timeout to allow for Go compilation

  it("should respect timeout setting", async () => {
    testScriptPath = path.join(tempDir, "slow.go");
    const goCode = `package main

import (
	"time"
)

func main() {
	time.Sleep(10 * time.Second)
}`;
    fs.writeFileSync(testScriptPath, goCode);

    const handler = createGoHandler(mockServer, "slow-tool", testScriptPath, 1);

    await expect(handler({})).rejects.toThrow();
  }, 15000); // Increase test timeout to allow for process timeout

  it("should handle Go script errors", async () => {
    testScriptPath = path.join(tempDir, "error.go");
    const goCode = `package main

import "os"

func main() {
	os.Exit(1)
}`;
    fs.writeFileSync(testScriptPath, goCode);

    const handler = createGoHandler(mockServer, "error-tool", testScriptPath);

    await expect(handler({})).rejects.toThrow();
  }, 30000); // Increase timeout to allow for Go compilation

  it("should pass complex input data", async () => {
    testScriptPath = path.join(tempDir, "complex.go");
    const goCode = `package main

import (
	"encoding/json"
	"io"
	"os"
)

func main() {
	var inputs map[string]interface{}
	data, _ := io.ReadAll(os.Stdin)
	json.Unmarshal(data, &inputs)
	
	// Echo back the input
	json.NewEncoder(os.Stdout).Encode(inputs)
}`;
    fs.writeFileSync(testScriptPath, goCode);

    const complexInput = {
      name: "test",
      numbers: [1, 2, 3],
      nested: {
        key: "value",
      },
    };

    const handler = createGoHandler(mockServer, "complex-tool", testScriptPath);
    const result = await handler(complexInput);

    const output = JSON.parse(result.content[0].text);
    expect(output).toEqual(complexInput);
  }, 30000); // Increase timeout to allow for Go compilation

  it("should execute script from GITHUB_WORKSPACE directory", async () => {
    // Save original GITHUB_WORKSPACE
    const originalWorkspace = process.env.GITHUB_WORKSPACE;

    // Set GITHUB_WORKSPACE to tempDir
    process.env.GITHUB_WORKSPACE = tempDir;

    try {
      // Create a Go script that outputs current working directory
      testScriptPath = path.join(tempDir, "test-cwd.go");
      const goCode = `package main

import (
	"encoding/json"
	"os"
)

func main() {
	cwd, _ := os.Getwd()
	result := map[string]interface{}{"cwd": cwd}
	json.NewEncoder(os.Stdout).Encode(result)
}`;
      fs.writeFileSync(testScriptPath, goCode);

      const handler = createGoHandler(mockServer, "cwd-tool", testScriptPath);
      const result = await handler({});

      const output = JSON.parse(result.content[0].text);
      expect(output.cwd).toBe(tempDir);
    } finally {
      // Restore original GITHUB_WORKSPACE
      if (originalWorkspace === undefined) {
        delete process.env.GITHUB_WORKSPACE;
      } else {
        process.env.GITHUB_WORKSPACE = originalWorkspace;
      }
    }
  }, 30000); // Increase timeout to allow for Go compilation
});
