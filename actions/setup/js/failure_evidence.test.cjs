import { describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import { readFailureEvidence } from "./failure_evidence.cjs";

describe("built-in failure evidence", () => {
  it("returns no context when no matching artifact exists", () => {
    const stat = vi.spyOn(fs, "lstatSync").mockReturnValue(undefined);
    try {
      expect(readFailureEvidence("Execute agent")).toBe("");
    } finally {
      stat.mockRestore();
    }
  });

  it("retains failed-step attribution and catalogue validation at the reader boundary", () => {
    const stat = vi.spyOn(fs, "lstatSync").mockReturnValue({
      isFile: () => true,
      isSymbolicLink: () => false,
      size: 200,
    });
    const read = vi.spyOn(fs, "readFileSync").mockReturnValue(
      JSON.stringify({
        kind: "goose",
        failures: ["Goose execution failed"],
        files: ["stdio"],
        extra: "PRIVATE PAYLOAD",
      })
    );
    try {
      expect(readFailureEvidence("Execute agent")).toBe("");
      const context = readFailureEvidence("Assert Goose smoke evidence");
      expect(context).toContain("Goose execution failed");
      expect(context).toContain("agent-stdio.log");
      expect(context).not.toContain("PRIVATE PAYLOAD");
    } finally {
      stat.mockRestore();
      read.mockRestore();
    }
  });
});
