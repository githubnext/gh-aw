// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);

const mockCore = {
  setSecret: vi.fn(),
  setOutput: vi.fn(),
};
global.core = mockCore;

const { main } = req("./exchange_otlp_workload_identity.cjs");

const STS_URL = "https://sts.googleapis.com/v1/token";

/** @param {any} body @param {{ok?: boolean, status?: number, statusText?: string}} [init] */
function jsonResponse(body, init = {}) {
  return {
    ok: init.ok !== false,
    status: init.status ?? 200,
    statusText: init.statusText ?? "OK",
    json: async () => body,
  };
}

describe("exchange_otlp_workload_identity", () => {
  /** @type {any} */
  let fetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    process.env.GH_AW_OTLP_OIDC_TOKEN = "github-oidc-token";
    process.env.GH_AW_OTLP_WIF_AUDIENCE = "//iam.googleapis.com/projects/1/locations/global/workloadIdentityPools/p/providers/gh";
    delete process.env.GH_AW_OTLP_WIF_SERVICE_ACCOUNT;
    fetchMock = vi.fn();
    global.fetch = fetchMock;
  });

  afterEach(() => {
    delete process.env.GH_AW_OTLP_OIDC_TOKEN;
    delete process.env.GH_AW_OTLP_WIF_AUDIENCE;
    delete process.env.GH_AW_OTLP_WIF_SERVICE_ACCOUNT;
  });

  it("exchanges the OIDC token and sets the access token output", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ access_token: "federated-token" }));

    const token = await main();

    expect(token).toBe("federated-token");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(STS_URL);
    expect(options.method).toBe("POST");
    const params = new URLSearchParams(options.body.toString());
    expect(params.get("subject_token")).toBe("github-oidc-token");
    expect(params.get("audience")).toBe(process.env.GH_AW_OTLP_WIF_AUDIENCE);
    expect(params.get("grant_type")).toBe("urn:ietf:params:oauth:grant-type:token-exchange");
    expect(mockCore.setSecret).toHaveBeenCalledWith("github-oidc-token");
    expect(mockCore.setSecret).toHaveBeenCalledWith("federated-token");
    expect(mockCore.setOutput).toHaveBeenCalledWith("token", "federated-token");
    expect(mockCore.setSecret.mock.invocationCallOrder[1]).toBeLessThan(mockCore.setOutput.mock.invocationCallOrder[0]);
  });

  it("impersonates the service account when configured", async () => {
    process.env.GH_AW_OTLP_WIF_SERVICE_ACCOUNT = "otlp@example.iam.gserviceaccount.com";
    fetchMock.mockResolvedValueOnce(jsonResponse({ access_token: "federated-token" })).mockResolvedValueOnce(jsonResponse({ accessToken: "impersonated-token" }));

    const token = await main();

    expect(token).toBe("impersonated-token");
    expect(fetchMock).toHaveBeenCalledTimes(2);
    const [url, options] = fetchMock.mock.calls[1];
    expect(url).toBe("https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/otlp%40example.iam.gserviceaccount.com:generateAccessToken");
    expect(options.headers.authorization).toContain("federated-token");
    expect(mockCore.setSecret.mock.calls).toEqual([["github-oidc-token"], ["federated-token"], ["impersonated-token"]]);
    expect(mockCore.setOutput).toHaveBeenCalledWith("token", "impersonated-token");
    expect(mockCore.setSecret.mock.invocationCallOrder[1]).toBeLessThan(fetchMock.mock.invocationCallOrder[1]);
    expect(mockCore.setSecret.mock.invocationCallOrder[2]).toBeLessThan(mockCore.setOutput.mock.invocationCallOrder[0]);
  });

  it("does not call the impersonation endpoint when no service account is configured", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ access_token: "federated-token" }));

    await main();

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("throws with the HTTP status when the STS exchange fails", async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, { ok: false, status: 403, statusText: "Forbidden" }));

    await expect(main()).rejects.toThrow(/HTTP 403 Forbidden/);
    await expect(main()).rejects.toThrow(/workload-identity\.audience/);
  });

  it("throws when the STS exchange returns no access token", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({}));

    await expect(main()).rejects.toThrow(/returned no access token/);
  });

  it("throws with the HTTP status when impersonation fails", async () => {
    process.env.GH_AW_OTLP_WIF_SERVICE_ACCOUNT = "otlp@example.iam.gserviceaccount.com";
    fetchMock.mockResolvedValue(jsonResponse({}, { ok: false, status: 404, statusText: "Not Found" })).mockResolvedValueOnce(jsonResponse({ access_token: "federated-token" }));

    await expect(main()).rejects.toThrow(/impersonation failed with HTTP 404 Not Found/);
  });

  it("throws when impersonation returns no access token", async () => {
    process.env.GH_AW_OTLP_WIF_SERVICE_ACCOUNT = "otlp@example.iam.gserviceaccount.com";
    fetchMock.mockResolvedValueOnce(jsonResponse({ access_token: "federated-token" })).mockResolvedValueOnce(jsonResponse({}));

    await expect(main()).rejects.toThrow(/impersonation returned no access token/);
  });

  it("throws when the GitHub OIDC token is missing", async () => {
    delete process.env.GH_AW_OTLP_OIDC_TOKEN;

    await expect(main()).rejects.toThrow(/Missing GitHub OIDC token/);
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
