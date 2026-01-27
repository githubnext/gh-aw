// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Security Alert Discovery
 *
 * Campaign-specific discovery script for security alerts.
 * Discovers code scanning, secret scanning, and Dependabot alerts
 * for repositories in scope.
 *
 * This is a specialized discovery script for security-focused campaigns
 * and should be called as a custom pre-compute step in the campaign workflow.
 *
 * Outputs:
 * - Manifest file: ./.gh-aw/security-alerts.json
 *
 * Environment variables:
 * - GH_AW_DISCOVERY_REPOS: Comma-separated list of repos (owner/repo format)
 */

const fs = require("fs");
const path = require("path");

/**
 * Discover security alerts for repositories
 * @param {any} octokit - GitHub API client
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @returns {Promise<any>} Security alerts summary
 */
async function discoverSecurityAlerts(octokit, repos) {
  if (!repos || repos.length === 0) {
    core.warning("No repos specified for security alert discovery");
    return null;
  }

  const alerts = {
    code_scanning: { total: 0, by_severity: {}, by_state: {}, items: /** @type {any[]} */ ([]) },
    secret_scanning: { total: 0, by_state: {}, items: /** @type {any[]} */ ([]) },
    dependabot: { total: 0, by_severity: {}, by_state: {}, items: /** @type {any[]} */ ([]) },
  };

  // Discover alerts for each repository
  for (const repoFullName of repos) {
    const [owner, repo] = repoFullName.split("/");
    if (!owner || !repo) {
      core.warning(`Invalid repo format: ${repoFullName}. Expected owner/repo`);
      continue;
    }

    core.info(`Discovering security alerts for ${repoFullName}...`);

    // Code Scanning Alerts
    try {
      const response = await octokit.rest.codeScanning.listAlertsForRepo({
        owner,
        repo,
        per_page: 100,
        state: "open",
      });

      const codeAlerts = response.data;
      alerts.code_scanning.total += codeAlerts.length;

      for (const alert of codeAlerts) {
        const severity = alert.rule?.severity || "unknown";
        alerts.code_scanning.by_severity[severity] = (alerts.code_scanning.by_severity[severity] || 0) + 1;
        alerts.code_scanning.by_state[alert.state] = (alerts.code_scanning.by_state[alert.state] || 0) + 1;

        alerts.code_scanning.items.push({
          type: "code_scanning",
          number: alert.number,
          url: alert.html_url,
          state: alert.state,
          severity: severity,
          rule_id: alert.rule?.id,
          rule_description: alert.rule?.description,
          created_at: alert.created_at,
          updated_at: alert.updated_at,
          repository: repoFullName,
        });
      }

      core.info(`  Code scanning: ${codeAlerts.length} open alerts`);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      core.warning(`Failed to fetch code scanning alerts for ${repoFullName}: ${err.message}`);
    }

    // Secret Scanning Alerts
    try {
      const response = await octokit.rest.secretScanning.listAlertsForRepo({
        owner,
        repo,
        per_page: 100,
        state: "open",
      });

      const secretAlerts = response.data;
      alerts.secret_scanning.total += secretAlerts.length;

      for (const alert of secretAlerts) {
        alerts.secret_scanning.by_state[alert.state] = (alerts.secret_scanning.by_state[alert.state] || 0) + 1;

        alerts.secret_scanning.items.push({
          type: "secret_scanning",
          number: alert.number,
          url: alert.html_url,
          state: alert.state,
          secret_type: alert.secret_type,
          created_at: alert.created_at,
          updated_at: alert.updated_at,
          repository: repoFullName,
        });
      }

      core.info(`  Secret scanning: ${secretAlerts.length} open alerts`);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      core.warning(`Failed to fetch secret scanning alerts for ${repoFullName}: ${err.message}`);
    }

    // Dependabot Alerts
    try {
      const response = await octokit.rest.dependabot.listAlertsForRepo({
        owner,
        repo,
        per_page: 100,
        state: "open",
      });

      const dependabotAlerts = response.data;
      alerts.dependabot.total += dependabotAlerts.length;

      for (const alert of dependabotAlerts) {
        const severity = alert.security_advisory?.severity || "unknown";
        alerts.dependabot.by_severity[severity] = (alerts.dependabot.by_severity[severity] || 0) + 1;
        alerts.dependabot.by_state[alert.state] = (alerts.dependabot.by_state[alert.state] || 0) + 1;

        alerts.dependabot.items.push({
          type: "dependabot",
          number: alert.number,
          url: alert.html_url,
          state: alert.state,
          severity: severity,
          package_name: alert.security_advisory?.package?.name,
          created_at: alert.created_at,
          updated_at: alert.updated_at,
          repository: repoFullName,
        });
      }

      core.info(`  Dependabot: ${dependabotAlerts.length} open alerts`);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      core.warning(`Failed to fetch Dependabot alerts for ${repoFullName}: ${err.message}`);
    }
  }

  // Log summary
  const totalAlerts = alerts.code_scanning.total + alerts.secret_scanning.total + alerts.dependabot.total;
  core.info(`✓ Security alert discovery complete: ${totalAlerts} total alerts`);
  core.info(`  Code scanning: ${alerts.code_scanning.total}`);
  core.info(`  Secret scanning: ${alerts.secret_scanning.total}`);
  core.info(`  Dependabot: ${alerts.dependabot.total}`);

  return alerts;
}

/**
 * Main entry point
 */
async function main() {
  try {
    // Read configuration from environment variables
    const repos = (process.env.GH_AW_DISCOVERY_REPOS || "")
      .split(",")
      .map(r => r.trim())
      .filter(r => r.length > 0);

    if (!repos.length) {
      throw new Error("GH_AW_DISCOVERY_REPOS environment variable is required");
    }

    core.info(`Starting security alert discovery for: ${repos.join(", ")}`);

    // Discover security alerts
    const alerts = await discoverSecurityAlerts(github, repos);

    if (!alerts) {
      throw new Error("Security alert discovery returned no results");
    }

    // Write manifest to output file
    const outputDir = "./.gh-aw";
    const outputPath = path.join(outputDir, "security-alerts.json");

    if (!fs.existsSync(outputDir)) {
      fs.mkdirSync(outputDir, { recursive: true });
    }

    const manifest = {
      schema_version: "v1",
      generated_at: new Date().toISOString(),
      repos: repos,
      alerts: alerts,
    };

    fs.writeFileSync(outputPath, JSON.stringify(manifest, null, 2));
    core.info(`Security alerts manifest written to ${outputPath}`);

    // Set output for GitHub Actions
    core.setOutput("manifest-path", outputPath);
    core.setOutput("total-alerts", alerts.code_scanning.total + alerts.secret_scanning.total + alerts.dependabot.total);
    core.setOutput("code-scanning-total", alerts.code_scanning.total);
    core.setOutput("secret-scanning-total", alerts.secret_scanning.total);
    core.setOutput("dependabot-total", alerts.dependabot.total);

    // Log summary
    core.info(`✓ Security alert discovery complete`);
    core.info(`  Total alerts: ${alerts.code_scanning.total + alerts.secret_scanning.total + alerts.dependabot.total}`);
    core.info(`  Code scanning: ${alerts.code_scanning.total}`);
    core.info(`  Secret scanning: ${alerts.secret_scanning.total}`);
    core.info(`  Dependabot: ${alerts.dependabot.total}`);
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    core.setFailed(`Security alert discovery failed: ${err.message}`);
    throw err;
  }
}

module.exports = {
  main,
  discoverSecurityAlerts,
};
