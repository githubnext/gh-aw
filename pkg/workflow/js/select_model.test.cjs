/**
 * @file select_model.test.cjs
 * @description Tests for model selection with wildcard pattern matching
 */

import { describe, it, expect } from "vitest";
const fs = require("fs");
const path = require("path");

// Extract the helper functions for testing
function patternToRegex(pattern) {
  const escaped = pattern.replace(/[.+?^${}()|[\]\\]/g, "\\$&");
  const regex = escaped.replace(/\*/g, ".*");
  return new RegExp(`^${regex}$`, "i");
}

function matchesPattern(model, pattern) {
  if (!pattern.includes("*")) {
    return model.toLowerCase() === pattern.toLowerCase();
  }
  const regex = patternToRegex(pattern);
  return regex.test(model);
}

function selectModel(requestedModels, availableModels) {
  for (const pattern of requestedModels) {
    for (const model of availableModels) {
      if (matchesPattern(model, pattern)) {
        return { selectedModel: model, matchedPattern: pattern };
      }
    }
  }
  return null;
}

describe("select_model", () => {
  describe("patternToRegex", () => {
    it("should convert simple wildcard to regex", () => {
      const regex = patternToRegex("gpt-*");
      expect(regex.test("gpt-4")).toBe(true);
      expect(regex.test("gpt-3.5-turbo")).toBe(true);
      expect(regex.test("claude-3")).toBe(false);
    });

    it("should handle multiple wildcards", () => {
      const regex = patternToRegex("gpt-*-*");
      expect(regex.test("gpt-4-turbo")).toBe(true);
      expect(regex.test("gpt-3.5-turbo")).toBe(true);
      expect(regex.test("gpt-4")).toBe(false);
    });

    it("should escape special regex characters", () => {
      const regex = patternToRegex("model.v1*");
      expect(regex.test("model.v1.0")).toBe(true);
      expect(regex.test("modelXv1.0")).toBe(false);
    });
  });

  describe("matchesPattern", () => {
    it("should match exact strings (case-insensitive)", () => {
      expect(matchesPattern("gpt-4", "gpt-4")).toBe(true);
      expect(matchesPattern("GPT-4", "gpt-4")).toBe(true);
      expect(matchesPattern("gpt-3.5", "gpt-4")).toBe(false);
    });

    it("should match wildcard patterns", () => {
      expect(matchesPattern("claude-3-sonnet-20240229", "*sonnet*")).toBe(true);
      expect(matchesPattern("claude-3-opus-20240229", "*sonnet*")).toBe(false);
    });

    it("should match gpt-*-mini pattern", () => {
      expect(matchesPattern("gpt-4-mini", "gpt-*-mini")).toBe(true);
      expect(matchesPattern("gpt-3.5-mini", "gpt-*-mini")).toBe(true);
      expect(matchesPattern("gpt-4-turbo", "gpt-*-mini")).toBe(false);
    });

    it("should match prefix wildcards", () => {
      expect(matchesPattern("claude-3-sonnet-20240229", "claude-*")).toBe(true);
      expect(matchesPattern("claude-3-opus-20240229", "claude-*")).toBe(true);
      expect(matchesPattern("gpt-4", "claude-*")).toBe(false);
    });

    it("should match suffix wildcards", () => {
      expect(matchesPattern("claude-3-sonnet-20240229", "*-sonnet-*")).toBe(true);
      expect(matchesPattern("claude-2-sonnet-20231107", "*-sonnet-*")).toBe(true);
      expect(matchesPattern("claude-3-opus-20240229", "*-sonnet-*")).toBe(false);
    });
  });

  describe("selectModel", () => {
    it("should select exact match", () => {
      const requested = ["gpt-4"];
      const available = ["gpt-3.5-turbo", "gpt-4", "claude-3-sonnet"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "gpt-4",
        matchedPattern: "gpt-4",
      });
    });

    it("should select first matching wildcard", () => {
      const requested = ["gpt-*"];
      const available = ["claude-3-sonnet", "gpt-4", "gpt-3.5-turbo"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "gpt-4",
        matchedPattern: "gpt-*",
      });
    });

    it("should try patterns in order", () => {
      const requested = ["nonexistent-model", "gpt-4", "claude-*"];
      const available = ["gpt-4", "claude-3-sonnet"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "gpt-4",
        matchedPattern: "gpt-4",
      });
    });

    it("should fall back to later patterns", () => {
      const requested = ["gpt-5", "claude-*"];
      const available = ["gpt-4", "claude-3-sonnet"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "claude-3-sonnet",
        matchedPattern: "claude-*",
      });
    });

    it("should return null if no match found", () => {
      const requested = ["gpt-5", "nonexistent"];
      const available = ["gpt-4", "claude-3-sonnet"];
      const result = selectModel(requested, available);

      expect(result).toBe(null);
    });

    it("should handle case-insensitive matching", () => {
      const requested = ["GPT-4"];
      const available = ["gpt-4", "claude-3-sonnet"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "gpt-4",
        matchedPattern: "GPT-4",
      });
    });

    it("should handle sonnet* wildcard", () => {
      const requested = ["*sonnet*"];
      const available = ["claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "claude-3-sonnet-20240229",
        matchedPattern: "*sonnet*",
      });
    });

    it("should handle gpt-*-mini wildcard", () => {
      const requested = ["gpt-*-mini"];
      const available = ["gpt-4-turbo", "gpt-4-mini", "gpt-3.5-turbo"];
      const result = selectModel(requested, available);

      expect(result).toEqual({
        selectedModel: "gpt-4-mini",
        matchedPattern: "gpt-*-mini",
      });
    });

    it("should handle complex priority list", () => {
      const requested = ["claude-3-opus-20240229", "claude-3-sonnet-*", "gpt-4", "gpt-*-mini"];
      const available = ["gpt-3.5-turbo", "gpt-4-mini", "claude-3-sonnet-20240229"];
      const result = selectModel(requested, available);

      // Should match claude-3-sonnet-* pattern
      expect(result).toEqual({
        selectedModel: "claude-3-sonnet-20240229",
        matchedPattern: "claude-3-sonnet-*",
      });
    });
  });
});
