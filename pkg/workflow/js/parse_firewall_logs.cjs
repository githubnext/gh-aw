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

  // Validate timestamp format (should be numeric with optional decimal point)
  const timestamp = fields[0];
  if (!/^\d+(\.\d+)?$/.test(timestamp)) {
    return null;
  }

  // Validate client IP:port format (should be IP:port or "-")
  const clientIpPort = fields[1];
  if (clientIpPort !== "-" && !/^[\d.]+:\d+$/.test(clientIpPort)) {
    return null;
  }

  // Validate domain format (should be domain:port or "-")
  const domain = fields[2];
  if (domain !== "-" && !/^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*:\d+$/.test(domain)) {
    return null;
  }

  // Validate dest IP:port format (should be IP:port or "-")
  const destIpPort = fields[3];
  if (destIpPort !== "-" && !/^[\d.]+:\d+$/.test(destIpPort)) {
    return null;
  }

  // Validate status code (should be numeric or "-")
  const status = fields[6];
  if (status !== "-" && !/^\d+$/.test(status)) {
    return null;
  }

  // Validate decision format (should contain ":" or be "-")
  const decision = fields[7];
  if (decision !== "-" && !decision.includes(":")) {
    return null;
  }

  return {
    timestamp: timestamp,
    clientIpPort: clientIpPort,
    domain: domain,
    destIpPort: destIpPort,
    proto: fields[4],
    method: fields[5],
    status: status,
    decision: decision,
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
 * Focuses on blocked requests only for debugging purposes
 * @param {object} analysis - Analysis results
 * @returns {string} Markdown formatted summary
 */
function generateFirewallSummary(analysis) {
  const { totalRequests, deniedRequests, deniedDomains, requestsByDomain } = analysis;

  let summary = "### 🔥 Firewall Blocked Requests\n\n";

  // Filter out invalid domains (placeholder "-" values)
  const validDeniedDomains = deniedDomains.filter(domain => domain !== "-");

  // Calculate denied requests from valid domains only
  const validDeniedRequests = validDeniedDomains.reduce((sum, domain) => sum + (requestsByDomain.get(domain)?.denied || 0), 0);

  // Show blocked requests if any exist
  if (validDeniedRequests > 0) {
    summary += `**${validDeniedRequests}** request${validDeniedRequests !== 1 ? "s" : ""} blocked across **${validDeniedDomains.length}** unique domain${validDeniedDomains.length !== 1 ? "s" : ""}`;
    summary += ` (${totalRequests > 0 ? Math.round((validDeniedRequests / totalRequests) * 100) : 0}% of total traffic)\n\n`;

    summary += "<details>\n";
    summary += "<summary>🚫 Blocked Domains (click to expand)</summary>\n\n";
    summary += "| Domain | Blocked Requests |\n";
    summary += "|--------|------------------|\n";

    for (const domain of validDeniedDomains) {
      const stats = requestsByDomain.get(domain);
      summary += `| ${domain} | ${stats.denied} |\n`;
    }
    summary += "\n</details>\n\n";
  } else {
    summary += "✅ **No blocked requests detected**\n\n";
    if (totalRequests > 0) {
      summary += `All ${totalRequests} request${totalRequests !== 1 ? "s" : ""} were allowed through the firewall.\n\n`;
    } else {
      summary += "No firewall activity detected.\n\n";
    }
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

// Run main when executed directly (not when imported as a module)
const isDirectExecution =
  typeof module === "undefined" || (typeof require !== "undefined" && typeof require.main !== "undefined" && require.main === module);
if (isDirectExecution) {
  main();
}
