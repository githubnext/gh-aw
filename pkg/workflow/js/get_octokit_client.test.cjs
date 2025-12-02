import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock __original_require__ globally before importing
const mockGetOctokit = vi.fn();
globalThis.__original_require__ = vi.fn().mockReturnValue({
  getOctokit: mockGetOctokit,
});

const { getOctokitClient, setGetOctokitFactory } = await import("./get_octokit_client.cjs");

describe("get_octokit_client.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setGetOctokitFactory(null);
  });

  afterEach(() => {
    setGetOctokitFactory(null);
  });

  describe("getOctokitClient", () => {
    it("should use factory function when set", () => {
      const mockOctokit = { graphql: vi.fn() };
      const factory = vi.fn().mockReturnValue(mockOctokit);
      setGetOctokitFactory(factory);

      const result = getOctokitClient("test-token");

      expect(factory).toHaveBeenCalledWith("test-token");
      expect(result).toBe(mockOctokit);
      expect(mockGetOctokit).not.toHaveBeenCalled();
    });

    it("should use __original_require__ to get @actions/github when no factory is set", () => {
      const mockOctokit = { graphql: vi.fn() };
      mockGetOctokit.mockReturnValue(mockOctokit);

      const result = getOctokitClient("my-token");

      expect(globalThis.__original_require__).toHaveBeenCalledWith("@actions/github");
      expect(mockGetOctokit).toHaveBeenCalledWith("my-token");
      expect(result).toBe(mockOctokit);
    });
  });

  describe("setGetOctokitFactory", () => {
    it("should allow setting a custom factory", () => {
      const mockOctokit1 = { graphql: vi.fn() };
      const mockOctokit2 = { graphql: vi.fn() };

      const factory1 = vi.fn().mockReturnValue(mockOctokit1);
      const factory2 = vi.fn().mockReturnValue(mockOctokit2);

      setGetOctokitFactory(factory1);
      expect(getOctokitClient("token1")).toBe(mockOctokit1);

      setGetOctokitFactory(factory2);
      expect(getOctokitClient("token2")).toBe(mockOctokit2);
    });

    it("should reset to using __original_require__ when set to null", () => {
      const mockOctokit = { graphql: vi.fn() };
      const factory = vi.fn().mockReturnValue(mockOctokit);

      setGetOctokitFactory(factory);
      getOctokitClient("token1");
      expect(factory).toHaveBeenCalled();
      expect(mockGetOctokit).not.toHaveBeenCalled();

      vi.clearAllMocks();
      const mockOctokitFromRequire = { graphql: vi.fn() };
      mockGetOctokit.mockReturnValue(mockOctokitFromRequire);

      setGetOctokitFactory(null);
      const result = getOctokitClient("token2");

      expect(factory).not.toHaveBeenCalled();
      expect(mockGetOctokit).toHaveBeenCalledWith("token2");
      expect(result).toBe(mockOctokitFromRequire);
    });
  });
});
