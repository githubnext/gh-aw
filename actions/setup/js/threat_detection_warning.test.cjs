import { describe, it, expect } from "vitest";
import { normalizeThreatKinds, getThreatDetectedMarker, getThreatDetectedMarkerTemplate, getDetectionReasonText } from "./threat_detection_warning.cjs";

describe("threat_detection_warning", () => {
  describe("normalizeThreatKinds", () => {
    it("returns unknown for empty input", () => {
      expect(normalizeThreatKinds(undefined)).toBe("unknown");
      expect(normalizeThreatKinds(null)).toBe("unknown");
      expect(normalizeThreatKinds("   ")).toBe("unknown");
    });

    it("normalizes comma/space-delimited values, strips invalid chars, and de-duplicates", () => {
      expect(normalizeThreatKinds("THREAT_DETECTED, parse.error threat_detected parse-error")).toBe("threat_detected,parseerror,parse-error");
    });
  });

  describe("marker helpers", () => {
    it("emits the normative threat marker", () => {
      expect(getThreatDetectedMarker("threat_detected,parse_error")).toBe("<!-- agentic threat detected -->");
      expect(getThreatDetectedMarkerTemplate()).toBe("<!-- agentic threat detected -->");
    });
  });

  describe("getDetectionReasonText", () => {
    it("returns mapped description for known reason", () => {
      expect(getDetectionReasonText("threat_detected")).toContain("Potential security threats were detected");
    });

    it("returns fallback description for unknown reason", () => {
      expect(getDetectionReasonText("new_reason")).toBe("The threat detection analysis could not be completed.");
    });
  });
});
