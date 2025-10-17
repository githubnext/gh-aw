import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";

// Mock the GitHub Actions core module
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions (less commonly used but included for completeness)
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

// Mock the context
const mockContext = {
  runId: "12345",
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  payload: {
    repository: {
      html_url: "https://github.com/test-owner/test-repo",
    },
  },
};

// Set up globals
global.core = mockCore;
global.context = mockContext;

// Read the code scanning alert script
const securityReportScript = fs.readFileSync(path.join(import.meta.dirname, "create_code_scanning_alert.cjs"), "utf8");

describe("create_code_scanning_alert.cjs", () => {
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset mocks
    mockCore.setOutput.mockClear();
    mockCore.summary.addRaw.mockClear();
    mockCore.summary.write.mockClear();

    // Set up basic environment
    setAgentOutput("");
    delete process.env.GITHUB_AW_SECURITY_REPORT_MAX;
    delete process.env.GITHUB_AW_SECURITY_REPORT_DRIVER;
    delete process.env.GITHUB_AW_WORKFLOW_FILENAME;
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }

    // Clean up any created files
    try {
      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      if (fs.existsSync(sarifFile)) {
        fs.unlinkSync(sarifFile);
      }
    } catch (e) {
      // Ignore cleanup errors
    }
  });

  describe("main function", () => {
    it("should handle missing environment variable", async () => {
      delete process.env.GITHUB_AW_AGENT_OUTPUT;
      await eval(`(async () => { ${securityReportScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    });

    it("should handle empty agent output", async () => {
      setAgentOutput("");
      await eval(`(async () => { ${securityReportScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    });

    it("should handle invalid JSON", async () => {
      setAgentOutput("invalid json");
      await eval(`(async () => { ${securityReportScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Error parsing agent output JSON:/));
    });

    it("should handle missing items array", async () => {
      setAgentOutput({
        status: "success",
      });
      await eval(`(async () => { ${securityReportScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output");
    });

    it("should handle no code scanning alert items", async () => {
      setAgentOutput({
        items: [{ type: "create_issue", title: "Test Issue" }],
      });
      await eval(`(async () => { ${securityReportScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No create-code-scanning-alert items found in agent output");
    });

    it("should create SARIF file for valid security findings", async () => {
      const securityFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/app.js",
            line: 42,
            severity: "error",
            message: "SQL injection vulnerability detected",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/utils.js",
            line: 15,
            severity: "warning",
            message: "Potential XSS vulnerability",
          },
        ],
      };

      setAgentOutput(securityFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      // Check that SARIF file was created
      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      expect(fs.existsSync(sarifFile)).toBe(true);

      // Check SARIF content
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
      expect(sarifContent.version).toBe("2.1.0");
      expect(sarifContent.runs).toHaveLength(1);
      expect(sarifContent.runs[0].results).toHaveLength(2);

      // Check first finding
      const firstResult = sarifContent.runs[0].results[0];
      expect(firstResult.message.text).toBe("SQL injection vulnerability detected");
      expect(firstResult.level).toBe("error");
      expect(firstResult.locations[0].physicalLocation.artifactLocation.uri).toBe("src/app.js");
      expect(firstResult.locations[0].physicalLocation.region.startLine).toBe(42);

      // Check second finding
      const secondResult = sarifContent.runs[0].results[1];
      expect(secondResult.message.text).toBe("Potential XSS vulnerability");
      expect(secondResult.level).toBe("warning");
      expect(secondResult.locations[0].physicalLocation.artifactLocation.uri).toBe("src/utils.js");
      expect(secondResult.locations[0].physicalLocation.region.startLine).toBe(15);

      // Check outputs were set
      expect(mockCore.setOutput).toHaveBeenCalledWith("sarif_file", sarifFile);
      expect(mockCore.setOutput).toHaveBeenCalledWith("findings_count", 2);

      // Check summary was written
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should respect max findings limit", async () => {
      process.env.GITHUB_AW_SECURITY_REPORT_MAX = "1";

      const securityFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/app.js",
            line: 42,
            severity: "error",
            message: "First finding",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/utils.js",
            line: 15,
            severity: "warning",
            message: "Second finding",
          },
        ],
      };

      setAgentOutput(securityFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      // Check that SARIF file was created with only 1 finding
      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      expect(fs.existsSync(sarifFile)).toBe(true);

      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
      expect(sarifContent.runs[0].results).toHaveLength(1);
      expect(sarifContent.runs[0].results[0].message.text).toBe("First finding");

      // Check output reflects the limit
      expect(mockCore.setOutput).toHaveBeenCalledWith("findings_count", 1);
    });

    it("should validate and filter invalid security findings", async () => {
      const mixedFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/valid.js",
            line: 10,
            severity: "error",
            message: "Valid finding",
          },
          {
            type: "create_code_scanning_alert",
            // Missing file
            line: 20,
            severity: "error",
            message: "Invalid - no file",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid.js",
            // Missing line
            severity: "error",
            message: "Invalid - no line",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid2.js",
            line: "not-a-number",
            severity: "error",
            message: "Invalid - bad line",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid3.js",
            line: 30,
            severity: "invalid-severity",
            message: "Invalid - bad severity",
          },
        ],
      };

      setAgentOutput(mixedFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      // Check that SARIF file was created with only the 1 valid finding
      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      expect(fs.existsSync(sarifFile)).toBe(true);

      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
      expect(sarifContent.runs[0].results).toHaveLength(1);
      expect(sarifContent.runs[0].results[0].message.text).toBe("Valid finding");

      // Check outputs
      expect(mockCore.setOutput).toHaveBeenCalledWith("findings_count", 1);
    });

    it("should use custom driver name when configured", async () => {
      process.env.GITHUB_AW_SECURITY_REPORT_DRIVER = "Custom Security Scanner";
      process.env.GITHUB_AW_WORKFLOW_FILENAME = "security-scan";

      const securityFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/app.js",
            line: 42,
            severity: "error",
            message: "Security issue found",
          },
        ],
      };

      setAgentOutput(securityFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));

      // Check driver name
      expect(sarifContent.runs[0].tool.driver.name).toBe("Custom Security Scanner");

      // Check rule ID includes workflow filename
      expect(sarifContent.runs[0].results[0].ruleId).toBe("security-scan-security-finding-1");
    });

    it("should use default driver name when not configured", async () => {
      const securityFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/app.js",
            line: 42,
            severity: "error",
            message: "Security issue found",
          },
        ],
      };

      setAgentOutput(securityFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));

      // Check default driver name
      expect(sarifContent.runs[0].tool.driver.name).toBe("GitHub Agentic Workflows Security Scanner");

      // Check rule ID includes default workflow filename
      expect(sarifContent.runs[0].results[0].ruleId).toBe("workflow-security-finding-1");
    });

    it("should support optional column specification", async () => {
      const securityFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/app.js",
            line: 42,
            column: 15,
            severity: "error",
            message: "Security issue with column info",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/utils.js",
            line: 25,
            // No column specified - should default to 1
            severity: "warning",
            message: "Security issue without column",
          },
        ],
      };

      setAgentOutput(securityFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));

      // Check first result has custom column
      expect(sarifContent.runs[0].results[0].locations[0].physicalLocation.region.startColumn).toBe(15);

      // Check second result has default column
      expect(sarifContent.runs[0].results[1].locations[0].physicalLocation.region.startColumn).toBe(1);
    });

    it("should validate column numbers", async () => {
      const invalidFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/valid.js",
            line: 10,
            column: 5,
            severity: "error",
            message: "Valid with column",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid1.js",
            line: 20,
            column: "not-a-number",
            severity: "error",
            message: "Invalid column - not a number",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid2.js",
            line: 30,
            column: 0,
            severity: "error",
            message: "Invalid column - zero",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid3.js",
            line: 40,
            column: -1,
            severity: "error",
            message: "Invalid column - negative",
          },
        ],
      };

      setAgentOutput(invalidFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      // Only the first valid finding should be processed
      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
      expect(sarifContent.runs[0].results).toHaveLength(1);
      expect(sarifContent.runs[0].results[0].message.text).toBe("Valid with column");
      expect(sarifContent.runs[0].results[0].locations[0].physicalLocation.region.startColumn).toBe(5);
    });

    it("should support optional ruleIdSuffix specification", async () => {
      process.env.GITHUB_AW_WORKFLOW_FILENAME = "security-scan";

      const securityFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/app.js",
            line: 42,
            severity: "error",
            message: "Custom rule ID finding",
            ruleIdSuffix: "sql-injection",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/utils.js",
            line: 25,
            severity: "warning",
            message: "Another custom rule ID",
            ruleIdSuffix: "xss-vulnerability",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/config.js",
            line: 10,
            severity: "info",
            message: "Standard numbered finding",
            // No ruleIdSuffix - should use default numbering
          },
        ],
      };

      setAgentOutput(securityFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));

      // Check first result has custom rule ID
      expect(sarifContent.runs[0].results[0].ruleId).toBe("security-scan-sql-injection");

      // Check second result has custom rule ID
      expect(sarifContent.runs[0].results[1].ruleId).toBe("security-scan-xss-vulnerability");

      // Check third result uses default numbering
      expect(sarifContent.runs[0].results[2].ruleId).toBe("security-scan-security-finding-3");
    });

    it("should validate ruleIdSuffix values", async () => {
      const invalidFindings = {
        items: [
          {
            type: "create_code_scanning_alert",
            file: "src/valid.js",
            line: 10,
            severity: "error",
            message: "Valid with valid ruleIdSuffix",
            ruleIdSuffix: "valid-rule-id_123",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid1.js",
            line: 20,
            severity: "error",
            message: "Invalid ruleIdSuffix - empty string",
            ruleIdSuffix: "",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid2.js",
            line: 30,
            severity: "error",
            message: "Invalid ruleIdSuffix - whitespace only",
            ruleIdSuffix: "   ",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid3.js",
            line: 40,
            severity: "error",
            message: "Invalid ruleIdSuffix - special characters",
            ruleIdSuffix: "rule@id!",
          },
          {
            type: "create_code_scanning_alert",
            file: "src/invalid4.js",
            line: 50,
            severity: "error",
            message: "Invalid ruleIdSuffix - not a string",
            ruleIdSuffix: 123,
          },
        ],
      };

      setAgentOutput(invalidFindings);
      await eval(`(async () => { ${securityReportScript} })()`);

      // Only the first valid finding should be processed
      const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");
      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
      expect(sarifContent.runs[0].results).toHaveLength(1);
      expect(sarifContent.runs[0].results[0].message.text).toBe("Valid with valid ruleIdSuffix");
      expect(sarifContent.runs[0].results[0].ruleId).toBe("workflow-valid-rule-id_123");
    });
  });
});
