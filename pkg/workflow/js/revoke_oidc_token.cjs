// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Revoke OIDC token
 * 
 * This script revokes the app token that was obtained via OIDC token exchange.
 * It only runs if a token was obtained via OIDC (not fallback).
 */

/**
 * Main function to revoke OIDC token
 */
async function main() {
	try {
		// Get configuration from environment variables
		const revokeUrl = process.env.GH_AW_OIDC_REVOKE_URL;
		const tokenObtained = process.env.GH_AW_OIDC_TOKEN_OBTAINED;
		const token = process.env.GH_AW_OIDC_TOKEN;

		// Only revoke if we obtained a token via OIDC
		if (tokenObtained !== "true") {
			core.info("No OIDC token to revoke (token from fallback or not obtained)");
			return;
		}

		// If no revoke URL is configured, skip revocation
		if (!revokeUrl) {
			core.info("No token revoke URL configured, skipping revocation");
			return;
		}

		if (!token) {
			core.warning("No token available for revocation");
			return;
		}

		core.info(`Revoking token at: ${revokeUrl}`);
		
		const response = await fetch(revokeUrl, {
			method: "POST",
			headers: {
				Authorization: `Bearer ${token}`,
			},
		});

		if (!response.ok) {
			// Log warning but don't fail the workflow
			core.warning(
				`Token revocation failed: ${response.status} ${response.statusText}`
			);
			return;
		}

		core.info("Token successfully revoked");

	} catch (error) {
		// Log warning but don't fail the workflow for revocation errors
		core.warning(
			`Failed to revoke token: ${error instanceof Error ? error.message : String(error)}`
		);
	}
}

await main();
