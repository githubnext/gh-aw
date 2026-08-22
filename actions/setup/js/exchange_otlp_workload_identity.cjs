// @ts-check
// @safe-outputs-exempt SEC-004 — "body" references are HTTP transport payloads for OAuth token exchange, not GitHub content
/**
 * Exchanges a GitHub OIDC token for a Google Cloud access token using
 * Workload Identity Federation, optionally impersonating a service account.
 *
 * Environment variables:
 * - GH_AW_OTLP_OIDC_TOKEN: the GitHub OIDC token minted for the WIF audience
 * - GH_AW_OTLP_WIF_AUDIENCE: the STS audience (//iam.googleapis.com/<provider>)
 * - GH_AW_OTLP_WIF_SERVICE_ACCOUNT: optional service account email to impersonate
 *
 * Sets the resulting access token as the step output "token" and masks it.
 */

const CLOUD_PLATFORM_SCOPE = "https://www.googleapis.com/auth/cloud-platform";

/**
 * Mask a token discovered by this inline github-script before it can be used
 * or exposed as an output.
 *
 * @param {unknown} value
 */
function maskSecret(value) {
  core.setSecret(String(value));
}

async function main() {
  const oidcToken = process.env.GH_AW_OTLP_OIDC_TOKEN;
  if (!oidcToken) {
    throw new Error("Missing GitHub OIDC token for Google workload identity token exchange");
  }
  maskSecret(oidcToken);

  let response;
  try {
    response = await fetch("https://sts.googleapis.com/v1/token", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
        requested_token_type: "urn:ietf:params:oauth:token-type:access_token",
        subject_token_type: "urn:ietf:params:oauth:token-type:jwt",
        subject_token: oidcToken,
        audience: process.env.GH_AW_OTLP_WIF_AUDIENCE || "",
        scope: CLOUD_PLATFORM_SCOPE,
      }),
      signal: AbortSignal.timeout(30_000),
    });
  } catch (error) {
    throw new Error("Google workload identity token exchange request failed", { cause: error });
  }
  if (!response.ok) {
    throw new Error(
      `Google workload identity token exchange failed with HTTP ${response.status} ${response.statusText}. Verify observability.otlp.workload-identity.audience matches the workload identity provider resource and that the provider trusts this repository`
    );
  }

  /** @type {any} */
  let stsPayload;
  try {
    stsPayload = await response.json();
  } catch (error) {
    throw new Error("Failed to parse Google workload identity token exchange response", { cause: error });
  }
  let accessToken = stsPayload.access_token;
  if (!accessToken) {
    throw new Error("Google workload identity token exchange returned no access token");
  }
  maskSecret(accessToken);

  const serviceAccount = process.env.GH_AW_OTLP_WIF_SERVICE_ACCOUNT;
  if (serviceAccount) {
    let impersonationResponse;
    try {
      impersonationResponse = await fetch(`https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/${encodeURIComponent(serviceAccount)}:generateAccessToken`, {
        method: "POST",
        headers: { authorization: "Bearer " + accessToken, "content-type": "application/json" },
        body: JSON.stringify({ scope: [CLOUD_PLATFORM_SCOPE] }),
        signal: AbortSignal.timeout(30_000),
      });
    } catch (error) {
      throw new Error("Google service account impersonation request failed", { cause: error });
    }
    if (!impersonationResponse.ok) {
      throw new Error(
        `Google service account impersonation failed with HTTP ${impersonationResponse.status} ${impersonationResponse.statusText}. Verify observability.otlp.workload-identity.service-account exists and grants roles/iam.workloadIdentityUser to the federated principal`
      );
    }
    /** @type {any} */
    let impersonationPayload;
    try {
      impersonationPayload = await impersonationResponse.json();
    } catch (error) {
      throw new Error("Failed to parse Google service account impersonation response", { cause: error });
    }
    accessToken = impersonationPayload.accessToken;
    if (!accessToken) {
      throw new Error("Google service account impersonation returned no access token");
    }
    maskSecret(accessToken);
  }

  core.setOutput("token", accessToken);
  return accessToken;
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = { main };
}
