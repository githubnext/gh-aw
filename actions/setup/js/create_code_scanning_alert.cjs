// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all create-code-scanning-alert items
  const securityItems = result.items.filter(/** @param {any} item */ item => item.type === "create_code_scanning_alert");
  if (securityItems.length === 0) {
    core.info("No create-code-scanning-alert items found in agent output");
    return;
  }

  core.info(`Found ${securityItems.length} create-code-scanning-alert item(s)`);

  // If in staged mode, emit step summary instead of creating code scanning alerts
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Create Code Scanning Alerts Preview\n\n";
    summaryContent += "The following code scanning alerts would be created if staged mode was disabled:\n\n";

    for (let i = 0; i < securityItems.length; i++) {
      const item = securityItems[i];
      summaryContent += `### Security Finding ${i + 1}\n`;
      summaryContent += `**File:** ${item.file || "No file provided"}\n\n`;
      summaryContent += `**Line:** ${item.line || "No line provided"}\n\n`;
      summaryContent += `**Severity:** ${item.severity || "No severity provided"}\n\n`;
      summaryContent += `**Message:**\n${item.message || "No message provided"}\n\n`;
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Code scanning alert creation preview written to step summary");
    return;
  }

  // Get the max configuration from environment variable
  const maxFindings = process.env.GH_AW_SECURITY_REPORT_MAX ? parseInt(process.env.GH_AW_SECURITY_REPORT_MAX) : 0; // 0 means unlimited
  core.info(`Max findings configuration: ${maxFindings === 0 ? "unlimited" : maxFindings}`);

  // Get the driver configuration from environment variable
  const driverName = process.env.GH_AW_SECURITY_REPORT_DRIVER || "GitHub Agentic Workflows Security Scanner";
  core.info(`Driver name: ${driverName}`);

  // Get the workflow filename for rule ID prefix
  const workflowFilename = process.env.GH_AW_WORKFLOW_FILENAME || "workflow";
  core.info(`Workflow filename for rule ID prefix: ${workflowFilename}`);

  const validFindings = [];

  // Process each security item and validate the findings
  for (let i = 0; i < securityItems.length; i++) {
    const securityItem = securityItems[i];
    core.info(
      `Processing create-code-scanning-alert item ${i + 1}/${securityItems.length}: file=${securityItem.file}, line=${securityItem.line}, severity=${securityItem.severity}, messageLength=${securityItem.message ? securityItem.message.length : "undefined"}, ruleIdSuffix=${securityItem.ruleIdSuffix || "not specified"}`
    );

    // Validate required fields
    if (!securityItem.file) {
      core.info('Missing required field "file" in code scanning alert item');
      continue;
    }

    if (!securityItem.line || (typeof securityItem.line !== "number" && typeof securityItem.line !== "string")) {
      core.info('Missing or invalid required field "line" in code scanning alert item');
      continue;
    }

    if (!securityItem.severity || typeof securityItem.severity !== "string") {
      core.info('Missing or invalid required field "severity" in code scanning alert item');
      continue;
    }

    if (!securityItem.message || typeof securityItem.message !== "string") {
      core.info('Missing or invalid required field "message" in code scanning alert item');
      continue;
    }

    // Parse line number
    const line = parseInt(securityItem.line, 10);
    if (isNaN(line) || line <= 0) {
      core.info(`Invalid line number: ${securityItem.line}`);
      continue;
    }

    // Parse optional column number
    let column = 1; // Default to column 1
    if (securityItem.column !== undefined) {
      if (typeof securityItem.column !== "number" && typeof securityItem.column !== "string") {
        core.info('Invalid field "column" in code scanning alert item (must be number or string)');
        continue;
      }
      const parsedColumn = parseInt(securityItem.column, 10);
      if (isNaN(parsedColumn) || parsedColumn <= 0) {
        core.info(`Invalid column number: ${securityItem.column}`);
        continue;
      }
      column = parsedColumn;
    }

    // Parse optional rule ID suffix
    let ruleIdSuffix = null;
    if (securityItem.ruleIdSuffix !== undefined) {
      if (typeof securityItem.ruleIdSuffix !== "string") {
        core.info('Invalid field "ruleIdSuffix" in code scanning alert item (must be string)');
        continue;
      }
      // Validate that the suffix doesn't contain invalid characters
      const trimmedSuffix = securityItem.ruleIdSuffix.trim();
      if (trimmedSuffix.length === 0) {
        core.info('Invalid field "ruleIdSuffix" in code scanning alert item (cannot be empty)');
        continue;
      }
      // Check for characters that would be problematic in rule IDs
      if (!/^[a-zA-Z0-9_-]+$/.test(trimmedSuffix)) {
        core.info(`Invalid ruleIdSuffix "${trimmedSuffix}" (must contain only alphanumeric characters, hyphens, and underscores)`);
        continue;
      }
      ruleIdSuffix = trimmedSuffix;
    }

    // Validate severity level and map to SARIF level
    /** @type {Record<string, string>} */
    const severityMap = {
      error: "error",
      warning: "warning",
      info: "note",
      note: "note",
    };

    const normalizedSeverity = securityItem.severity.toLowerCase();
    if (!severityMap[normalizedSeverity]) {
      core.info(`Invalid severity level: ${securityItem.severity} (must be error, warning, info, or note)`);
      continue;
    }

    const sarifLevel = severityMap[normalizedSeverity];

    // Create a valid finding object
    validFindings.push({
      file: securityItem.file.trim(),
      line: line,
      column: column,
      severity: normalizedSeverity,
      sarifLevel: sarifLevel,
      message: securityItem.message.trim(),
      ruleIdSuffix: ruleIdSuffix,
    });

    // Check if we've reached the max limit
    if (maxFindings > 0 && validFindings.length >= maxFindings) {
      core.info(`Reached maximum findings limit: ${maxFindings}`);
      break;
    }
  }

  if (validFindings.length === 0) {
    core.info("No valid security findings to report");
    return;
  }

  core.info(`Processing ${validFindings.length} valid security finding(s)`);

  // Generate SARIF file
  const sarifContent = {
    $schema: "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
    version: "2.1.0",
    runs: [
      {
        tool: {
          driver: {
            name: driverName,
            version: "1.0.0",
            informationUri: "https://github.com/githubnext/gh-aw",
          },
        },
        results: validFindings.map((finding, index) => ({
          ruleId: finding.ruleIdSuffix ? `${workflowFilename}-${finding.ruleIdSuffix}` : `${workflowFilename}-security-finding-${index + 1}`,
          message: { text: finding.message },
          level: finding.sarifLevel,
          locations: [
            {
              physicalLocation: {
                artifactLocation: { uri: finding.file },
                region: {
                  startLine: finding.line,
                  startColumn: finding.column,
                },
              },
            },
          ],
        })),
      },
    ],
  };

  // Write SARIF file to filesystem
  const fs = require("fs");
  const path = require("path");
  const sarifFileName = "code-scanning-alert.sarif";
  const sarifFilePath = path.join(process.cwd(), sarifFileName);

  try {
    fs.writeFileSync(sarifFilePath, JSON.stringify(sarifContent, null, 2));
    core.info(`✓ Created SARIF file: ${sarifFilePath}`);
    core.info(`SARIF file size: ${fs.statSync(sarifFilePath).size} bytes`);

    // Set outputs for the GitHub Action
    core.setOutput("sarif_file", sarifFilePath);
    core.setOutput("findings_count", validFindings.length);
    core.setOutput("artifact_uploaded", "pending");
    core.setOutput("codeql_uploaded", "pending");

    // Write summary with findings
    let summaryContent = "\n\n## Code Scanning Alert\n";
    summaryContent += `Found **${validFindings.length}** security finding(s):\n\n`;

    for (const finding of validFindings) {
      const emoji = finding.severity === "error" ? "🔴" : finding.severity === "warning" ? "🟡" : "🔵";
      summaryContent += `${emoji} **${finding.severity.toUpperCase()}** in \`${finding.file}:${finding.line}\`: ${finding.message}\n`;
    }

    summaryContent += `\n📄 SARIF file created: \`${sarifFileName}\`\n`;
    summaryContent += `🔍 Findings will be uploaded to GitHub Code Scanning\n`;

    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    core.error(`✗ Failed to create SARIF file: ${error instanceof Error ? error.message : String(error)}`);
    throw error;
  }

  core.info(`Successfully created code scanning alert with ${validFindings.length} finding(s)`);
  return {
    sarifFile: sarifFilePath,
    findingsCount: validFindings.length,
    findings: validFindings,
  };
}
await main();
