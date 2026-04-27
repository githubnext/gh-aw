// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// messages_core.cjs calls core.warning on parse failures - provide a stub
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};
global.core = mockCore;

const { getStagedTitle, getStagedDescription } = require("./messages_staged.cjs");

const OPERATION = "Create Issues";

describe("messages_staged", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
  });

  describe("getStagedTitle", () => {
    it("returns the default title with operation substituted", () => {
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`## 🔍 Preview: ${OPERATION}`);
    });

    it("substitutes a different operation value", () => {
      const title = getStagedTitle({ operation: "Add Comments" });
      expect(title).toBe("## 🔍 Preview: Add Comments");
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedTitle: "Staging: {operation}" });
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`Staging: ${OPERATION}`);
    });

    it("falls back to default when stagedTitle is absent from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ runSuccess: "other" });
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`## 🔍 Preview: ${OPERATION}`);
    });

    it("falls back to default when stagedTitle is an empty string", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedTitle: "" });
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`## 🔍 Preview: ${OPERATION}`);
    });

    it("falls back to default when stagedTitle is a falsy non-string value", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedTitle: 0 });
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`## 🔍 Preview: ${OPERATION}`);
    });

    it("falls back to default when config is invalid JSON", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = "not-json";
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`## 🔍 Preview: ${OPERATION}`);
    });

    it("supports snake_case placeholders from camelCase context keys", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedTitle: "{operation_name}: {operation}" });
      const title = getStagedTitle({ operation: OPERATION, operationName: "Create Comment" });
      expect(title).toBe(`Create Comment: ${OPERATION}`);
    });

    it("returns an empty-operation title when operation is an empty string", () => {
      const title = getStagedTitle({ operation: "" });
      expect(title).toBe("## 🔍 Preview: ");
    });

    it("leaves unrecognised placeholders unchanged in custom template", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedTitle: "{operation} | {unknown}" });
      const title = getStagedTitle({ operation: OPERATION });
      expect(title).toBe(`${OPERATION} | {unknown}`);
    });
  });

  describe("getStagedDescription", () => {
    it("returns the default description", () => {
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).toBe("📋 The following operations would be performed if staged mode was disabled:");
    });

    it("uses custom description template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedDescription: "Preview of: {operation}" });
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).toBe(`Preview of: ${OPERATION}`);
    });

    it("falls back to default when stagedDescription is absent from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedTitle: "title only" });
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).toBe("📋 The following operations would be performed if staged mode was disabled:");
    });

    it("falls back to default when stagedDescription is an empty string", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedDescription: "" });
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).toBe("📋 The following operations would be performed if staged mode was disabled:");
    });

    it("falls back to default when stagedDescription is a falsy non-string value", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedDescription: false });
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).toBe("📋 The following operations would be performed if staged mode was disabled:");
    });

    it("falls back to default when config is invalid JSON", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = "{bad json";
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).toBe("📋 The following operations would be performed if staged mode was disabled:");
    });

    it("supports custom description with operation placeholder", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ stagedDescription: "Would run: {operation}" });
      const desc = getStagedDescription({ operation: "Close PRs" });
      expect(desc).toBe("Would run: Close PRs");
    });

    it("default description does not contain unfilled placeholders", () => {
      const desc = getStagedDescription({ operation: OPERATION });
      expect(desc).not.toMatch(/\{[^}]+\}/);
    });
  });

  describe("independent config keys", () => {
    it("getStagedTitle and getStagedDescription use their own config keys independently", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        stagedTitle: "Custom title: {operation}",
        stagedDescription: "Custom desc: {operation}",
      });

      const title = getStagedTitle({ operation: OPERATION });
      const desc = getStagedDescription({ operation: OPERATION });

      expect(title).toBe(`Custom title: ${OPERATION}`);
      expect(desc).toBe(`Custom desc: ${OPERATION}`);
    });
  });
});
