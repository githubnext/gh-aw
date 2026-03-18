import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import crypto from "crypto";
import { getMimeType, getBackendIdsFromToken, buildArtifactUrl } from "./upload_assets.cjs";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(void 0) },
};

describe("upload_assets.cjs", () => {
  let tempFilePath;

  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.ACTIONS_RESULTS_URL;
    delete process.env.ACTIONS_RUNTIME_TOKEN;
    global.core = mockCore;
  });

  afterEach(() => {
    if (tempFilePath && fs.existsSync(tempFilePath)) fs.unlinkSync(tempFilePath);
  });

  describe("getMimeType", () => {
    it("returns image/png for .png files", () => {
      expect(getMimeType("chart.png")).toBe("image/png");
    });
    it("returns image/jpeg for .jpg files", () => {
      expect(getMimeType("photo.jpg")).toBe("image/jpeg");
    });
    it("returns image/jpeg for .jpeg files", () => {
      expect(getMimeType("photo.jpeg")).toBe("image/jpeg");
    });
    it("returns image/gif for .gif files", () => {
      expect(getMimeType("anim.gif")).toBe("image/gif");
    });
    it("returns image/webp for .webp files", () => {
      expect(getMimeType("image.webp")).toBe("image/webp");
    });
    it("returns image/svg+xml for .svg files", () => {
      expect(getMimeType("diagram.svg")).toBe("image/svg+xml");
    });
    it("returns application/octet-stream for unknown extensions", () => {
      expect(getMimeType("data.bin")).toBe("application/octet-stream");
    });
  });

  describe("getBackendIdsFromToken", () => {
    it("parses backend IDs from a valid JWT scp claim", () => {
      // Build a minimal JWT with scp = "Actions.Results:run123:job456"
      const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url");
      const payload = Buffer.from(JSON.stringify({ scp: "Actions.Results:run123:job456" })).toString("base64url");
      const token = `${header}.${payload}.signature`;
      const ids = getBackendIdsFromToken(token);
      expect(ids.workflowRunBackendId).toBe("run123");
      expect(ids.workflowJobRunBackendId).toBe("job456");
    });

    it("throws if scp claim is missing", () => {
      const header = Buffer.from(JSON.stringify({ alg: "HS256" })).toString("base64url");
      const payload = Buffer.from(JSON.stringify({ sub: "user" })).toString("base64url");
      const token = `${header}.${payload}.sig`;
      expect(() => getBackendIdsFromToken(token)).toThrow("missing scp claim");
    });

    it("throws if scp does not start with expected prefix", () => {
      const header = Buffer.from(JSON.stringify({})).toString("base64url");
      const payload = Buffer.from(JSON.stringify({ scp: "OtherScope:abc:def" })).toString("base64url");
      const token = `${header}.${payload}.sig`;
      expect(() => getBackendIdsFromToken(token)).toThrow("expected prefix");
    });
  });

  describe("buildArtifactUrl", () => {
    it("builds correct URL for github.com", () => {
      process.env.GITHUB_SERVER_URL = "https://github.com";
      process.env.GITHUB_REPOSITORY = "owner/repo";
      process.env.GITHUB_RUN_ID = "42";
      const url = buildArtifactUrl("99");
      expect(url).toBe("https://github.com/owner/repo/actions/runs/42/artifacts/99");
      delete process.env.GITHUB_SERVER_URL;
      delete process.env.GITHUB_REPOSITORY;
      delete process.env.GITHUB_RUN_ID;
    });

    it("builds correct URL for GitHub Enterprise Server", () => {
      process.env.GITHUB_SERVER_URL = "https://ghe.example.com";
      process.env.GITHUB_REPOSITORY = "myorg/myrepo";
      process.env.GITHUB_RUN_ID = "100";
      const url = buildArtifactUrl("7");
      expect(url).toBe("https://ghe.example.com/myorg/myrepo/actions/runs/100/artifacts/7");
      delete process.env.GITHUB_SERVER_URL;
      delete process.env.GITHUB_REPOSITORY;
      delete process.env.GITHUB_RUN_ID;
    });
  });

  describe("main - no upload items", () => {
    it("outputs empty asset_url_map when no upload_asset items are present", async () => {
      setAgentOutput({ items: [{ type: "create_issue", title: "Hello" }] });
      const { main } = await import("./upload_assets.cjs");
      await main();
      const call = mockCore.setOutput.mock.calls.find(c => c[0] === "asset_url_map");
      expect(call).toBeDefined();
      expect(call[1]).toBe("{}");
    });

    it("outputs upload_count=0 when there are no items", async () => {
      setAgentOutput({ items: [] });
      const { main } = await import("./upload_assets.cjs");
      await main();
      const call = mockCore.setOutput.mock.calls.find(c => c[0] === "upload_count");
      expect(call).toBeDefined();
      expect(call[1]).toBe("0");
    });
  });

  describe("main - missing ACTIONS_RESULTS_URL", () => {
    it("fails when artifact env vars are missing and there are upload items", async () => {
      const assetDir = "/tmp/gh-aw/safeoutputs/assets";
      if (!fs.existsSync(assetDir)) fs.mkdirSync(assetDir, { recursive: true });
      const assetPath = path.join(assetDir, "test-missing-env.png");
      fs.writeFileSync(assetPath, "fake png data");
      const fileContent = fs.readFileSync(assetPath);
      const sha = crypto.createHash("sha256").update(fileContent).digest("hex");
      setAgentOutput({
        items: [
          {
            type: "upload_asset",
            fileName: "test-missing-env.png",
            sha,
            size: fileContent.length,
            temporaryId: "aw_testenv1",
          },
        ],
      });
      // ACTIONS_RESULTS_URL is NOT set
      const { main } = await import("./upload_assets.cjs");
      await main();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("ACTIONS_RESULTS_URL"));
      fs.existsSync(assetPath) && fs.unlinkSync(assetPath);
    });
  });

  describe("main - SHA mismatch", () => {
    it("fails when file SHA does not match the recorded SHA", async () => {
      const assetDir = "/tmp/gh-aw/safeoutputs/assets";
      if (!fs.existsSync(assetDir)) fs.mkdirSync(assetDir, { recursive: true });
      const assetPath = path.join(assetDir, "sha-mismatch.png");
      fs.writeFileSync(assetPath, "original data");
      setAgentOutput({
        items: [
          {
            type: "upload_asset",
            fileName: "sha-mismatch.png",
            sha: "aaaaaaaabbbbbbbbccccccccddddddddeeeeeeeeffffffffaaaaaaaabbbbbbbb",
            size: 13,
            temporaryId: "aw_shamis1",
          },
        ],
      });
      process.env.ACTIONS_RESULTS_URL = "https://results.actions.githubusercontent.com/";
      process.env.ACTIONS_RUNTIME_TOKEN = "fake-token";
      const { main } = await import("./upload_assets.cjs");
      await main();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("SHA mismatch"));
      delete process.env.ACTIONS_RESULTS_URL;
      delete process.env.ACTIONS_RUNTIME_TOKEN;
      fs.existsSync(assetPath) && fs.unlinkSync(assetPath);
    });
  });

  describe("main - staged mode", () => {
    it("skips actual artifact upload in staged mode and returns placeholder URLs", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      const assetDir = "/tmp/gh-aw/safeoutputs/assets";
      if (!fs.existsSync(assetDir)) fs.mkdirSync(assetDir, { recursive: true });
      const assetPath = path.join(assetDir, "staged-asset.png");
      fs.writeFileSync(assetPath, "staged png data");
      const fileContent = fs.readFileSync(assetPath);
      const sha = crypto.createHash("sha256").update(fileContent).digest("hex");
      setAgentOutput({
        items: [
          {
            type: "upload_asset",
            fileName: "staged-asset.png",
            sha,
            size: fileContent.length,
            temporaryId: "aw_staged1",
          },
        ],
      });
      // No ACTIONS_RESULTS_URL needed in staged mode
      const { main } = await import("./upload_assets.cjs");
      await main();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      const countCall = mockCore.setOutput.mock.calls.find(c => c[0] === "upload_count");
      expect(countCall[1]).toBe("1");
      const mapCall = mockCore.setOutput.mock.calls.find(c => c[0] === "asset_url_map");
      const urlMap = JSON.parse(mapCall[1]);
      expect(urlMap["aw_staged1"]).toContain("aw://staged/");
      delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
      fs.existsSync(assetPath) && fs.unlinkSync(assetPath);
    });
  });
});
