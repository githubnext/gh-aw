import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core
const mockCore = {
  info: vi.fn(),
};
global.core = mockCore;

describe("getCampaign", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_CAMPAIGN;
  });

  it("should return empty string when campaign not set", async () => {
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign();

    expect(result).toBe("");
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("should return campaign and log when set (no format)", async () => {
    process.env.GH_AW_CAMPAIGN = "test-campaign-123";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign();

    expect(result).toBe("test-campaign-123");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: test-campaign-123");
  });

  it("should return campaign and log when set (text format)", async () => {
    process.env.GH_AW_CAMPAIGN = "test-campaign-123";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign("text");

    expect(result).toBe("test-campaign-123");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: test-campaign-123");
  });

  it("should return markdown HTML comment when format is markdown", async () => {
    process.env.GH_AW_CAMPAIGN = "project-alpha-2024";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign("markdown");

    expect(result).toBe("\n\n<!-- campaign: project-alpha-2024 -->");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: project-alpha-2024");
  });

  it("should return empty string for markdown format when campaign not set", async () => {
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign("markdown");

    expect(result).toBe("");
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("should handle campaign with hyphens", async () => {
    process.env.GH_AW_CAMPAIGN = "project-alpha-2024";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign();

    expect(result).toBe("project-alpha-2024");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: project-alpha-2024");
  });

  it("should handle campaign with underscores", async () => {
    process.env.GH_AW_CAMPAIGN = "project_alpha_2024";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign();

    expect(result).toBe("project_alpha_2024");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: project_alpha_2024");
  });

  it("should handle mixed alphanumeric campaign", async () => {
    process.env.GH_AW_CAMPAIGN = "Test123_Project-v2";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign();

    expect(result).toBe("Test123_Project-v2");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: Test123_Project-v2");
  });

  it("should handle markdown format with hyphens and underscores", async () => {
    process.env.GH_AW_CAMPAIGN = "Test123_Project-v2";
    const { getCampaign } = await import("./get_campaign.cjs");

    const result = getCampaign("markdown");

    expect(result).toBe("\n\n<!-- campaign: Test123_Project-v2 -->");
    expect(mockCore.info).toHaveBeenCalledWith("Campaign: Test123_Project-v2");
  });
});
