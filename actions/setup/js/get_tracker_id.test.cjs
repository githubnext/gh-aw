import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core
const mockCore = {
  info: vi.fn(),
};
global.core = mockCore;

describe("getTrackerID", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_TRACKER_ID;
  });

  it("should return empty string when tracker-id not set", async () => {
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID();

    expect(result).toBe("");
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("should return tracker-id and log when set (no format)", async () => {
    process.env.GH_AW_TRACKER_ID = "test-tracker-123";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID();

    expect(result).toBe("test-tracker-123");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: test-tracker-123");
  });

  it("should return tracker-id and log when set (text format)", async () => {
    process.env.GH_AW_TRACKER_ID = "test-tracker-123";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID("text");

    expect(result).toBe("test-tracker-123");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: test-tracker-123");
  });

  it("should return markdown HTML comment when format is markdown", async () => {
    process.env.GH_AW_TRACKER_ID = "project-alpha-2024";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID("markdown");

    expect(result).toBe("\n\n<!-- gh-aw-tracker-id: project-alpha-2024 -->");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: project-alpha-2024");
  });

  it("should return empty string for markdown format when tracker-id not set", async () => {
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID("markdown");

    expect(result).toBe("");
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("should handle tracker-id with hyphens", async () => {
    process.env.GH_AW_TRACKER_ID = "project-alpha-2024";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID();

    expect(result).toBe("project-alpha-2024");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: project-alpha-2024");
  });

  it("should handle tracker-id with underscores", async () => {
    process.env.GH_AW_TRACKER_ID = "project_alpha_2024";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID();

    expect(result).toBe("project_alpha_2024");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: project_alpha_2024");
  });

  it("should handle mixed alphanumeric tracker-id", async () => {
    process.env.GH_AW_TRACKER_ID = "Test123_Project-v2";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID();

    expect(result).toBe("Test123_Project-v2");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: Test123_Project-v2");
  });

  it("should handle markdown format with hyphens and underscores", async () => {
    process.env.GH_AW_TRACKER_ID = "Test123_Project-v2";
    const { getTrackerID } = await import("./get_tracker_id.cjs");

    const result = getTrackerID("markdown");

    expect(result).toBe("\n\n<!-- gh-aw-tracker-id: Test123_Project-v2 -->");
    expect(mockCore.info).toHaveBeenCalledWith("Tracker ID: Test123_Project-v2");
  });
});
