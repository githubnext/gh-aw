async function main() {
  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    console.log("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  if (outputContent.trim() === "") {
    console.log("Agent output content is empty");
    return;
  }

  console.log("Agent output content length:", outputContent.length);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    console.log(
      "Error parsing agent output JSON:",
      error instanceof Error ? error.message : String(error)
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    console.log("No valid items found in agent output");
    return;
  }

  // Find all create-security-report items
  const securityItems = validatedOutput.items.filter(
    /** @param {any} item */ item => item.type === "create-security-report"
  );
  if (securityItems.length === 0) {
    console.log("No create-security-report items found in agent output");
    return;
  }

  console.log(`Found ${securityItems.length} create-security-report item(s)`);

  // Get the max configuration from environment variable
  const maxFindings = process.env.GITHUB_AW_SECURITY_REPORT_MAX
    ? parseInt(process.env.GITHUB_AW_SECURITY_REPORT_MAX)
    : 0; // 0 means unlimited
  console.log(
    `Max findings configuration: ${maxFindings === 0 ? "unlimited" : maxFindings}`
  );

  // Get the driver configuration from environment variable
  const driverName =
    process.env.GITHUB_AW_SECURITY_REPORT_DRIVER ||
    "GitHub Agentic Workflows Security Scanner";
  console.log(`Driver name: ${driverName}`);

  // Get the workflow filename for rule ID prefix
  const workflowFilename =
    process.env.GITHUB_AW_WORKFLOW_FILENAME || "workflow";
  console.log(`Workflow filename for rule ID prefix: ${workflowFilename}`);

  const validFindings = [];

  // Process each security item and validate the findings
  for (let i = 0; i < securityItems.length; i++) {
    const securityItem = securityItems[i];
    console.log(
      `Processing create-security-report item ${i + 1}/${securityItems.length}:`,
      {
        file: securityItem.file,
        line: securityItem.line,
        severity: securityItem.severity,
        messageLength: securityItem.message
          ? securityItem.message.length
          : "undefined",
        ruleIdSuffix: securityItem.ruleIdSuffix || "not specified",
      }
    );

    // Validate required fields
    if (!securityItem.file) {
      console.log('Missing required field "file" in security report item');
      continue;
    }

    if (
      !securityItem.line ||
      (typeof securityItem.line !== "number" &&
        typeof securityItem.line !== "string")
    ) {
      console.log(
        'Missing or invalid required field "line" in security report item'
      );
      continue;
    }

    if (!securityItem.severity || typeof securityItem.severity !== "string") {
      console.log(
        'Missing or invalid required field "severity" in security report item'
      );
      continue;
    }

    if (!securityItem.message || typeof securityItem.message !== "string") {
      console.log(
        'Missing or invalid required field "message" in security report item'
      );
      continue;
    }

    // Parse line number
    const line = parseInt(securityItem.line, 10);
    if (isNaN(line) || line <= 0) {
      console.log(`Invalid line number: ${securityItem.line}`);
      continue;
    }

    // Parse optional column number
    let column = 1; // Default to column 1
    if (securityItem.column !== undefined) {
      if (
        typeof securityItem.column !== "number" &&
        typeof securityItem.column !== "string"
      ) {
        console.log(
          'Invalid field "column" in security report item (must be number or string)'
        );
        continue;
      }
      const parsedColumn = parseInt(securityItem.column, 10);
      if (isNaN(parsedColumn) || parsedColumn <= 0) {
        console.log(`Invalid column number: ${securityItem.column}`);
        continue;
      }
      column = parsedColumn;
    }

    // Parse optional rule ID suffix
    let ruleIdSuffix = null;
    if (securityItem.ruleIdSuffix !== undefined) {
      if (typeof securityItem.ruleIdSuffix !== "string") {
        console.log(
          'Invalid field "ruleIdSuffix" in security report item (must be string)'
        );
        continue;
      }
      // Validate that the suffix doesn't contain invalid characters
      const trimmedSuffix = securityItem.ruleIdSuffix.trim();
      if (trimmedSuffix.length === 0) {
        console.log(
          'Invalid field "ruleIdSuffix" in security report item (cannot be empty)'
        );
        continue;
      }
      // Check for characters that would be problematic in rule IDs
      if (!/^[a-zA-Z0-9_-]+$/.test(trimmedSuffix)) {
        console.log(
          `Invalid ruleIdSuffix "${trimmedSuffix}" (must contain only alphanumeric characters, hyphens, and underscores)`
        );
        continue;
      }
      ruleIdSuffix = trimmedSuffix;
    }

    // Validate severity level and map to SARIF level
    const severityMap = {
      error: "error",
      warning: "warning",
      info: "note",
      note: "note",
    };

    const normalizedSeverity = securityItem.severity.toLowerCase();
    if (!severityMap[normalizedSeverity]) {
      console.log(
        `Invalid severity level: ${securityItem.severity} (must be error, warning, info, or note)`
      );
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
      console.log(`Reached maximum findings limit: ${maxFindings}`);
      break;
    }
  }

  if (validFindings.length === 0) {
    console.log("No valid security findings to report");
    return;
  }

  console.log(`Processing ${validFindings.length} valid security finding(s)`);

  // Generate SARIF file
  const sarifContent = {
    $schema:
      "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
    version: "2.1.0",
    runs: [
      {
        tool: {
          driver: {
            name: driverName,
            version: "1.0.0",
            informationUri: "https://github.com/githubnext/gh-aw-copilots",
          },
        },
        results: validFindings.map((finding, index) => ({
          ruleId: finding.ruleIdSuffix
            ? `${workflowFilename}-${finding.ruleIdSuffix}`
            : `${workflowFilename}-security-finding-${index + 1}`,
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
  const sarifFileName = "security-report.sarif";
  const sarifFilePath = path.join(process.cwd(), sarifFileName);

  try {
    fs.writeFileSync(sarifFilePath, JSON.stringify(sarifContent, null, 2));
    console.log(`✓ Created SARIF file: ${sarifFilePath}`);
    console.log(`SARIF file size: ${fs.statSync(sarifFilePath).size} bytes`);

    // Set outputs for the GitHub Action
    core.setOutput("sarif_file", sarifFilePath);
    core.setOutput("findings_count", validFindings.length);
    core.setOutput("artifact_uploaded", "pending");
    core.setOutput("codeql_uploaded", "pending");

    // Write summary with findings
    let summaryContent = "\n\n## Security Report\n";
    summaryContent += `Found **${validFindings.length}** security finding(s):\n\n`;

    for (const finding of validFindings) {
      const emoji =
        finding.severity === "error"
          ? "🔴"
          : finding.severity === "warning"
            ? "🟡"
            : "🔵";
      summaryContent += `${emoji} **${finding.severity.toUpperCase()}** in \`${finding.file}:${finding.line}\`: ${finding.message}\n`;
    }

    summaryContent += `\n📄 SARIF file created: \`${sarifFileName}\`\n`;
    summaryContent += `🔍 Findings will be uploaded to GitHub Code Scanning\n`;

    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    core.error(
      `✗ Failed to create SARIF file: ${error instanceof Error ? error.message : String(error)}`
    );
    throw error;
  }

  console.log(
    `Successfully created security report with ${validFindings.length} finding(s)`
  );
  return {
    sarifFile: sarifFilePath,
    findingsCount: validFindings.length,
    findings: validFindings,
  };
}
await main();
