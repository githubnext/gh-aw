/**
 * @file safe_output_types_validation.test.cjs
 * @description Validates that safe output JSONL item types do not contain github-token field.
 * This test ensures the separation between output items (what agents produce) and
 * output configuration (how outputs are handled at workflow level).
 *
 * Key principle: Individual output items should NOT specify authentication tokens.
 * Tokens belong in the configuration layer, not in the data layer.
 */

import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";

describe("Safe Output Types Validation", () => {
  const typeDefsPath = path.join(__dirname, "types", "safe-outputs.d.ts");
  const configDefsPath = path.join(__dirname, "types", "safe-outputs-config.d.ts");

  it("safe-outputs.d.ts should NOT contain github-token field", () => {
    const content = fs.readFileSync(typeDefsPath, "utf-8");

    // Check for various forms of github-token that might be added accidentally
    expect(content).not.toMatch(/[\s"]github-token["\s:]/);
    expect(content).not.toMatch(/[\s"]GitHub-token["\s:]/);
    expect(content).not.toMatch(/[\s"]githubToken["\s:]/);

    // Verify the file exists and is not empty
    expect(content.length).toBeGreaterThan(0);
  });

  it("safe-outputs-config.d.ts SHOULD contain github-token field", () => {
    const content = fs.readFileSync(configDefsPath, "utf-8");

    // Configuration SHOULD have github-token
    expect(content).toMatch(/"github-token"\?:/);

    // Verify it's in the right places (base config and safe job config)
    const lines = content.split("\n");
    const githubTokenLines = lines.filter(line => line.includes('"github-token"'));

    // Should appear at least twice: once in SafeOutputConfig, once in SafeJobConfig
    expect(githubTokenLines.length).toBeGreaterThanOrEqual(2);
  });

  it("safe-outputs.d.ts should define output item interfaces", () => {
    const content = fs.readFileSync(typeDefsPath, "utf-8");

    // Verify core output types exist
    expect(content).toMatch(/interface CreateIssueItem/);
    expect(content).toMatch(/interface CreatePullRequestItem/);
    expect(content).toMatch(/interface AddCommentItem/);
    expect(content).toMatch(/type: "create_issue"/);
    expect(content).toMatch(/type: "create_pull_request"/);
  });

  it("safe-outputs-config.d.ts should define configuration interfaces", () => {
    const content = fs.readFileSync(configDefsPath, "utf-8");

    // Verify configuration types exist
    expect(content).toMatch(/interface SafeOutputConfig/);
    expect(content).toMatch(/interface CreateIssueConfig/);
    expect(content).toMatch(/interface SafeJobConfig/);

    // Configuration should have max, min, type fields
    expect(content).toMatch(/max\?:/);
    expect(content).toMatch(/type:/);
  });

  it("output items should extend BaseSafeOutputItem, not SafeOutputConfig", () => {
    const content = fs.readFileSync(typeDefsPath, "utf-8");

    // Output items extend BaseSafeOutputItem
    expect(content).toMatch(/interface CreateIssueItem extends BaseSafeOutputItem/);
    expect(content).toMatch(/interface AddCommentItem extends BaseSafeOutputItem/);

    // Should NOT extend SafeOutputConfig
    expect(content).not.toMatch(/extends SafeOutputConfig/);
  });

  it("configuration interfaces should extend SafeOutputConfig", () => {
    const content = fs.readFileSync(configDefsPath, "utf-8");

    // Config interfaces extend SafeOutputConfig
    expect(content).toMatch(/interface CreateIssueConfig extends SafeOutputConfig/);
    expect(content).toMatch(/interface AddCommentConfig extends SafeOutputConfig/);
  });

  it("BaseSafeOutputItem should only have type field", () => {
    const content = fs.readFileSync(typeDefsPath, "utf-8");

    // Extract BaseSafeOutputItem definition
    const baseInterfaceMatch = content.match(/interface BaseSafeOutputItem\s*{([^}]*)}/);
    expect(baseInterfaceMatch).toBeTruthy();

    if (baseInterfaceMatch) {
      const interfaceBody = baseInterfaceMatch[1];

      // Should only contain type field and comments
      expect(interfaceBody).toMatch(/type:\s*string/);

      // Should NOT contain any auth-related fields
      expect(interfaceBody).not.toMatch(/token/i);
      expect(interfaceBody).not.toMatch(/auth/i);
      expect(interfaceBody).not.toMatch(/credential/i);
      expect(interfaceBody).not.toMatch(/secret/i);
    }
  });

  it("SafeOutputConfig should contain only configuration fields", () => {
    const content = fs.readFileSync(configDefsPath, "utf-8");

    // Extract SafeOutputConfig definition
    const baseInterfaceMatch = content.match(/interface SafeOutputConfig\s*{([^}]*)}/);
    expect(baseInterfaceMatch).toBeTruthy();

    if (baseInterfaceMatch) {
      const interfaceBody = baseInterfaceMatch[1];

      // Should contain configuration fields
      expect(interfaceBody).toMatch(/type:\s*string/);
      expect(interfaceBody).toMatch(/max\?:/);
      expect(interfaceBody).toMatch(/min\?:/);
      expect(interfaceBody).toMatch(/"github-token"\?:/);

      // Should NOT contain output-specific fields like title, body, etc.
      expect(interfaceBody).not.toMatch(/title:/);
      expect(interfaceBody).not.toMatch(/body:/);
      expect(interfaceBody).not.toMatch(/message:/);
    }
  });
});
