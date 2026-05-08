// @ts-check
import { describe, it, expect } from "vitest";
const { ERR_VALIDATION, ERR_PERMISSION, ERR_API, ERR_CONFIG, ERR_NOT_FOUND, ERR_PARSE, ERR_SYSTEM, SAFE_OUTPUT_E001, SAFE_OUTPUT_E099 } = require("./error_codes.cjs");

describe("error_codes", () => {
  describe("module exports", () => {
    it("exports all expected error codes", () => {
      expect(ERR_VALIDATION).toBeDefined();
      expect(ERR_PERMISSION).toBeDefined();
      expect(ERR_API).toBeDefined();
      expect(ERR_CONFIG).toBeDefined();
      expect(ERR_NOT_FOUND).toBeDefined();
      expect(ERR_PARSE).toBeDefined();
      expect(ERR_SYSTEM).toBeDefined();
      expect(SAFE_OUTPUT_E001).toBeDefined();
      expect(SAFE_OUTPUT_E099).toBeDefined();
    });

    it("exports string values only", () => {
      const codes = [ERR_VALIDATION, ERR_PERMISSION, ERR_API, ERR_CONFIG, ERR_NOT_FOUND, ERR_PARSE, ERR_SYSTEM, SAFE_OUTPUT_E001, SAFE_OUTPUT_E099];
      for (const code of codes) {
        expect(typeof code).toBe("string");
      }
    });
  });

  describe("primary error codes", () => {
    it("ERR_VALIDATION is 'ERR_VALIDATION'", () => {
      expect(ERR_VALIDATION).toBe("ERR_VALIDATION");
    });

    it("ERR_PERMISSION is 'ERR_PERMISSION'", () => {
      expect(ERR_PERMISSION).toBe("ERR_PERMISSION");
    });

    it("ERR_API is 'ERR_API'", () => {
      expect(ERR_API).toBe("ERR_API");
    });

    it("ERR_CONFIG is 'ERR_CONFIG'", () => {
      expect(ERR_CONFIG).toBe("ERR_CONFIG");
    });

    it("ERR_NOT_FOUND is 'ERR_NOT_FOUND'", () => {
      expect(ERR_NOT_FOUND).toBe("ERR_NOT_FOUND");
    });

    it("ERR_PARSE is 'ERR_PARSE'", () => {
      expect(ERR_PARSE).toBe("ERR_PARSE");
    });

    it("ERR_SYSTEM is 'ERR_SYSTEM'", () => {
      expect(ERR_SYSTEM).toBe("ERR_SYSTEM");
    });
  });

  describe("legacy safe-output codes", () => {
    it("SAFE_OUTPUT_E001 is 'E001'", () => {
      expect(SAFE_OUTPUT_E001).toBe("E001");
    });

    it("SAFE_OUTPUT_E099 is 'E099'", () => {
      expect(SAFE_OUTPUT_E099).toBe("E099");
    });
  });

  describe("usage as error message prefixes", () => {
    it("can be used as a prefix in an Error message", () => {
      const err = new Error(`${ERR_VALIDATION}: Missing required field: title`);
      expect(err.message).toBe("ERR_VALIDATION: Missing required field: title");
    });

    it("can be used as a prefix in a setFailed-style string", () => {
      const msg = `${ERR_CONFIG}: GH_AW_PROMPT environment variable is not set`;
      expect(msg).toBe("ERR_CONFIG: GH_AW_PROMPT environment variable is not set");
    });

    it("all primary codes are distinct from each other", () => {
      const primary = [ERR_VALIDATION, ERR_PERMISSION, ERR_API, ERR_CONFIG, ERR_NOT_FOUND, ERR_PARSE, ERR_SYSTEM];
      const unique = new Set(primary);
      expect(unique.size).toBe(primary.length);
    });

    it("legacy codes are distinct from primary codes", () => {
      const all = [ERR_VALIDATION, ERR_PERMISSION, ERR_API, ERR_CONFIG, ERR_NOT_FOUND, ERR_PARSE, ERR_SYSTEM, SAFE_OUTPUT_E001, SAFE_OUTPUT_E099];
      const unique = new Set(all);
      expect(unique.size).toBe(all.length);
    });
  });
});
