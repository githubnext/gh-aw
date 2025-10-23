// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Setup OIDC token with PAT fallback
 *
 * This script attempts to:
 * 1. Use an existing token if available in the fallback environment variable
 * 2. Get an OIDC token using core.getIDToken()
 * 3. Exchange the OIDC token for an app token
 * 4. Set the token in the environment for the agentic engine
 *
 * Based on: https://github.com/anthropics/claude-code-action/blob/f30f5eecfce2f34fa72e40fa5f7bcdbdcad12eb8/src/github/token.ts
 */

/**
 * Retry a function with exponential backoff
 * @template T
 * @param {() => Promise<T>} fn - Function to retry
 * @param {number} [maxRetries=3] - Maximum number of retries
 * @param {number} [initialDelay=1000] - Initial delay in milliseconds
 * @returns {Promise<T>}
 */
async function retryWithBackoff(fn, maxRetries = 3, initialDelay = 1000) {
  let lastError;
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      if (i < maxRetries - 1) {
        const delay = initialDelay * Math.pow(2, i);
        core.info(`Retry ${i + 1}/${maxRetries} after ${delay}ms...`);
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
  }
  throw lastError;
}

/**
 * Get OIDC token from GitHub Actions
 * @param {string} audience - OIDC audience identifier
 * @returns {Promise<string>}
 */
async function getOidcToken(audience) {
  try {
    core.info(`Requesting OIDC token with audience: ${audience}`);
    const oidcToken = await core.getIDToken(audience);
    core.info("OIDC token successfully obtained");
    return oidcToken;
  } catch (error) {
    core.error(`Failed to get OIDC token: ${error instanceof Error ? error.message : String(error)}`);
    throw new Error("Could not fetch an OIDC token. Did you remember to add `id-token: write` to your workflow permissions?");
  }
}

/**
 * Exchange OIDC token for app token
 * @param {string} oidcToken - OIDC token from GitHub Actions
 * @param {string} exchangeUrl - Token exchange URL
 * @returns {Promise<string>}
 */
async function exchangeForAppToken(oidcToken, exchangeUrl) {
  core.info(`Exchanging OIDC token at: ${exchangeUrl}`);

  const response = await fetch(exchangeUrl, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${oidcToken}`,
    },
  });

  if (!response.ok) {
    /** @type {{ error?: { message?: string; details?: { error_code?: string } }; type?: string; message?: string }} */
    let responseJson;
    try {
      responseJson = await response.json();
    } catch {
      responseJson = {};
    }

    // Check for specific workflow validation error codes that should skip the action
    const errorCode = responseJson.error?.details?.error_code;

    if (errorCode === "workflow_not_found_on_default_branch") {
      const message = responseJson.message ?? responseJson.error?.message ?? "Workflow validation failed";
      core.warning(`Skipping action due to workflow validation: ${message}`);
      core.info(
        "Action skipped due to workflow validation error. This is expected when adding workflows to new repositories or on PRs with workflow changes. If you're seeing this, your workflow will begin working once you merge your PR."
      );
      core.setOutput("skipped_due_to_workflow_validation_mismatch", "true");
      return;
    }

    const errorMessage = responseJson?.error?.message ?? "Unknown error";
    core.error(`App token exchange failed: ${response.status} ${response.statusText} - ${errorMessage}`);
    throw new Error(errorMessage);
  }

  /** @type {{ token?: string; app_token?: string }} */
  const appTokenData = await response.json();
  const appToken = appTokenData.token || appTokenData.app_token;

  if (!appToken) {
    throw new Error("App token not found in response");
  }

  core.info("App token successfully obtained");
  return appToken;
}

/**
 * Main function to setup OIDC token
 */
async function main() {
  try {
    // Get configuration from environment variables
    const audience = process.env.GH_AW_OIDC_AUDIENCE;
    const exchangeUrl = process.env.GH_AW_OIDC_EXCHANGE_URL;
    const envVarName = process.env.GH_AW_OIDC_ENV_VAR_NAME;
    const fallbackEnvVar = process.env.GH_AW_OIDC_FALLBACK_ENV_VAR;

    if (!audience || !exchangeUrl || !envVarName) {
      core.setFailed("Missing required OIDC configuration (audience, exchange_url, or env_var_name)");
      return;
    }

    // Check if token was provided as fallback
    const fallbackToken = fallbackEnvVar ? process.env[fallbackEnvVar] : null;

    if (fallbackToken) {
      core.info(`Using provided token from ${fallbackEnvVar} for authentication`);
      core.setOutput("token", fallbackToken);
      core.setOutput("token_source", "fallback");
      core.exportVariable(envVarName, fallbackToken);
      return;
    }

    // Get OIDC token with retry
    const oidcToken = await retryWithBackoff(() => getOidcToken(audience));

    // Exchange OIDC token for app token with retry
    const appToken = await retryWithBackoff(() => exchangeForAppToken(oidcToken, exchangeUrl));

    // Set the token in the environment for subsequent steps
    core.info(`Setting token in environment variable: ${envVarName}`);
    core.setOutput("token", appToken);
    core.setOutput("token_source", "oidc");
    core.exportVariable(envVarName, appToken);

    // Also output the token for post-step revocation
    core.setOutput("oidc_token_obtained", "true");
  } catch (error) {
    // Only set failed if we get here - workflow validation errors will return before this
    core.setFailed(
      `Failed to setup token: ${error instanceof Error ? error.message : String(error)}\n\nIf you instead wish to use a custom token, provide it via the fallback environment variable.`
    );
  }
}

await main();
