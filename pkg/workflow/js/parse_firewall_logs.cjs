// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Parses firewall logs and creates a step summary
 * Firewall log format: timestamp client_ip:port domain dest_ip:port proto method status decision url user_agent
 */

function main() {
  const fs = require("fs");
  const path = require("path");

  try {
    // Get the squid logs directory path from environment or use default
    const workflowName = process.env.GITHUB_WORKFLOW || "workflow";
    const sanitizedName = sanitizeWorkflowName(workflowName);
    const squidLogsDir = `/tmp/gh-aw/squid-logs-${sanitizedName}/`;

    if (!fs.existsSync(squidLogsDir)) {
      core.info(`No firewall logs directory found at: ${squidLogsDir}`);
      return;
    }

    // Find all access.log files
    const files = fs.readdirSync(squidLogsDir).filter(file => file.endsWith(".log"));

    if (files.length === 0) {
      core.info(`No firewall log files found in: ${squidLogsDir}`);
      return;
    }

    core.info(`Found ${files.length} firewall log file(s)`);

    // Parse all log files and aggregate results
    let totalRequests = 0;
    let allowedRequests = 0;
    let deniedRequests = 0;
    const allowedDomains = new Set();
    const deniedDomains = new Set();
    const requestsByDomain = new Map();

    for (const file of files) {
      const filePath = path.join(squidLogsDir, file);
      core.info(`Parsing firewall log: ${file}`);

      const content = fs.readFileSync(filePath, "utf8");
      const lines = content.split("\n").filter(line => line.trim());

      for (const line of lines) {
        const entry = parseFirewallLogLine(line);
        if (!entry) {
          continue;
        }

        totalRequests++;

        // Determine if request was allowed or denied
        const isAllowed = isRequestAllowed(entry.decision, entry.status);

        if (isAllowed) {
          allowedRequests++;
          allowedDomains.add(entry.domain);
        } else {
          deniedRequests++;
          deniedDomains.add(entry.domain);
        }

        // Track request count per domain
        if (!requestsByDomain.has(entry.domain)) {
          requestsByDomain.set(entry.domain, { allowed: 0, denied: 0 });
        }
        const domainStats = requestsByDomain.get(entry.domain);
        if (isAllowed) {
          domainStats.allowed++;
        } else {
          domainStats.denied++;
        }
      }
    }

    // Generate step summary
    const summary = generateFirewallSummary({
      totalRequests,
      allowedRequests,
      deniedRequests,
      allowedDomains: Array.from(allowedDomains).sort(),
      deniedDomains: Array.from(deniedDomains).sort(),
      requestsByDomain,
    });

    core.summary.addRaw(summary).write();
    core.info("Firewall log summary generated successfully");
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

/**
 * Parses a single firewall log line
 * Format: timestamp client_ip:port domain dest_ip:port proto method status decision url user_agent
 * @param {string} line - Log line to parse
 * @returns {object|null} Parsed entry or null if invalid
 */
function parseFirewallLogLine(line) {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith("#")) {
    return null;
  }

  // Split by whitespace but preserve quoted strings
  const fields = trimmed.match(/(?:[^\s"]+|"[^"]*")+/g);

  if (!fields || fields.length < 10) {
    return null;
  }

  return {
    timestamp: fields[0],
    clientIpPort: fields[1],
    domain: fields[2],
    destIpPort: fields[3],
    proto: fields[4],
    method: fields[5],
    status: fields[6],
    decision: fields[7],
    url: fields[8],
    userAgent: fields[9] ? fields[9].replace(/^"|"$/g, "") : "-",
  };
}

/**
 * Determines if a request was allowed based on decision and status
 * @param {string} decision - Decision field (e.g., TCP_TUNNEL:HIER_DIRECT, NONE_NONE:HIER_NONE)
 * @param {string} status - Status code (e.g., 200, 403, 0)
 * @returns {boolean} True if request was allowed
 */
function isRequestAllowed(decision, status) {
  // Check status code first
  const statusCode = parseInt(status, 10);
  if (statusCode === 200 || statusCode === 206 || statusCode === 304) {
    return true;
  }

  // Check decision field
  if (decision.includes("TCP_TUNNEL") || decision.includes("TCP_HIT") || decision.includes("TCP_MISS")) {
    return true;
  }

  if (decision.includes("NONE_NONE") || decision.includes("TCP_DENIED") || statusCode === 403 || statusCode === 407) {
    return false;
  }

  // Default to denied for safety
  return false;
}

/**
 * Generates markdown summary from firewall log analysis
 * @param {object} analysis - Analysis results
 * @returns {string} Markdown formatted summary
 */
function generateFirewallSummary(analysis) {
  const { totalRequests, allowedRequests, deniedRequests, allowedDomains, deniedDomains, requestsByDomain } = analysis;

  let summary = "# 🔥 Firewall Activity Summary\n\n";

  // Overview statistics
  summary += "## 📊 Overview\n\n";
  summary += `- **Total Requests**: ${totalRequests}\n`;
  summary += `- **Allowed**: ${allowedRequests} (${totalRequests > 0 ? Math.round((allowedRequests / totalRequests) * 100) : 0}%)\n`;
  summary += `- **Denied**: ${deniedRequests} (${totalRequests > 0 ? Math.round((deniedRequests / totalRequests) * 100) : 0}%)\n`;
  summary += `- **Unique Allowed Domains**: ${allowedDomains.length}\n`;
  summary += `- **Unique Denied Domains**: ${deniedDomains.length}\n\n`;

  // Denied domains section (most important for debugging)
  if (deniedDomains.length > 0) {
    summary += "## 🚫 Denied Domains\n\n";
    summary += "The following domains were blocked by the firewall:\n\n";
    summary += "| Domain | Denied Requests |\n";
    summary += "|--------|----------------|\n";

    for (const domain of deniedDomains) {
      const stats = requestsByDomain.get(domain);
      summary += `| ${domain} | ${stats.denied} |\n`;
    }
    summary += "\n";
  }

  // Allowed domains section
  if (allowedDomains.length > 0) {
    summary += "## ✅ Allowed Domains\n\n";
    summary += "The following domains were allowed through the firewall:\n\n";

    // Only show table if there are more than 10 domains, otherwise show as list
    if (allowedDomains.length > 10) {
      summary += "| Domain | Allowed Requests |\n";
      summary += "|--------|------------------|\n";

      for (const domain of allowedDomains) {
        const stats = requestsByDomain.get(domain);
        summary += `| ${domain} | ${stats.allowed} |\n`;
      }
    } else {
      for (const domain of allowedDomains) {
        const stats = requestsByDomain.get(domain);
        summary += `- **${domain}** (${stats.allowed} request${stats.allowed !== 1 ? "s" : ""})\n`;
      }
    }
    summary += "\n";
  }

  // No requests case
  if (totalRequests === 0) {
    summary += "No firewall activity detected.\n\n";
  }

  return summary;
}

/**
 * Sanitizes a workflow name for use in file paths
 * @param {string} name - Workflow name to sanitize
 * @returns {string} Sanitized name
 */
function sanitizeWorkflowName(name) {
  return name
    .toLowerCase()
    .replace(/[:\\/\s]/g, "-")
    .replace(/[^a-z0-9._-]/g, "-");
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseFirewallLogLine,
    isRequestAllowed,
    generateFirewallSummary,
    sanitizeWorkflowName,
    main,
  };
}

// Only run main if not being imported
if (require.main === module) {
  main();
}
