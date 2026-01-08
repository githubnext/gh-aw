import { describe, it, expect, beforeEach } from "vitest";

describe("missing_messages_helper.cjs", () => {
  let helper;

  beforeEach(async () => {
    helper = await import("./missing_messages_helper.cjs");
    // Reset state between tests
    helper.setCollectedMissings(null);
  });

  describe("setCollectedMissings and getCollectedMissings", () => {
    it("should store and retrieve missing messages", () => {
      const { setCollectedMissings, getCollectedMissings } = helper;
      const missings = {
        missingTools: [{ tool: "docker", reason: "Need containers" }],
        missingData: [{ data_type: "api_key", reason: "No credentials" }],
      };

      setCollectedMissings(missings);
      const retrieved = getCollectedMissings();

      expect(retrieved).toEqual(missings);
    });

    it("should return null when no missings have been set", () => {
      const { getCollectedMissings } = helper;
      const result = getCollectedMissings();
      expect(result).toBeNull();
    });

    it("should allow overwriting previous missings", () => {
      const { setCollectedMissings, getCollectedMissings } = helper;
      const firstMissings = {
        missingTools: [{ tool: "docker", reason: "First" }],
        missingData: [],
      };
      const secondMissings = {
        missingTools: [],
        missingData: [{ data_type: "api_key", reason: "Second" }],
      };

      setCollectedMissings(firstMissings);
      setCollectedMissings(secondMissings);
      const retrieved = getCollectedMissings();

      expect(retrieved).toEqual(secondMissings);
      expect(retrieved).not.toEqual(firstMissings);
    });
  });

  describe("getMissingInfoSections", () => {
    it("should return empty string when no missings are set", () => {
      const { getMissingInfoSections } = helper;
      const result = getMissingInfoSections();
      expect(result).toBe("");
    });

    it("should generate HTML sections when missings are set", () => {
      const { setCollectedMissings, getMissingInfoSections } = helper;
      const missings = {
        missingTools: [{ tool: "docker", reason: "Need containers" }],
        missingData: [{ data_type: "api_key", reason: "No credentials" }],
      };

      setCollectedMissings(missings);
      const result = getMissingInfoSections();

      expect(result).toContain("<details>");
      expect(result).toContain("Missing Tools");
      expect(result).toContain("Missing Data");
      expect(result).toContain("docker");
      expect(result).toContain("api\\_key"); // Escaped underscore
    });

    it("should return empty string when missings are empty arrays", () => {
      const { setCollectedMissings, getMissingInfoSections } = helper;
      const missings = {
        missingTools: [],
        missingData: [],
      };

      setCollectedMissings(missings);
      const result = getMissingInfoSections();

      expect(result).toBe("");
    });

    it("should handle only missing tools", () => {
      const { setCollectedMissings, getMissingInfoSections } = helper;
      const missings = {
        missingTools: [{ tool: "kubectl", reason: "K8s management" }],
        missingData: [],
      };

      setCollectedMissings(missings);
      const result = getMissingInfoSections();

      expect(result).toContain("Missing Tools");
      expect(result).not.toContain("Missing Data");
      expect(result).toContain("kubectl");
    });

    it("should handle only missing data", () => {
      const { setCollectedMissings, getMissingInfoSections } = helper;
      const missings = {
        missingTools: [],
        missingData: [{ data_type: "config", reason: "Config not found" }],
      };

      setCollectedMissings(missings);
      const result = getMissingInfoSections();

      expect(result).not.toContain("Missing Tools");
      expect(result).toContain("Missing Data");
      expect(result).toContain("config");
    });
  });
});
