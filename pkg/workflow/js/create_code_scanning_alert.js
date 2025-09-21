async function createCodeScanningAlertMain() {
    const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!outputContent) {
        core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
        return [];
    }
    if (outputContent.trim() === "") {
        core.info("Agent output content is empty");
        return [];
    }
    core.info(`Agent output content length: ${outputContent.length}`);
    let validatedOutput;
    try {
        validatedOutput = JSON.parse(outputContent);
    }
    catch (error) {
        core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
        return [];
    }
    if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
        core.info("No valid items found in agent output");
        return [];
    }
    const securityItems = validatedOutput.items.filter(item => item.type === "create-code-scanning-alert");
    if (securityItems.length === 0) {
        core.info("No create-code-scanning-alert items found in agent output");
        return [];
    }
    core.info(`Found ${securityItems.length} create-code-scanning-alert item(s)`);
    if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
        let summaryContent = "## 🎭 Staged Mode: Create Code Scanning Alerts Preview\n\n";
        summaryContent += "The following code scanning alerts would be created if staged mode was disabled:\n\n";
        for (let i = 0; i < securityItems.length; i++) {
            const item = securityItems[i];
            summaryContent += `### Security Finding ${i + 1}\n`;
            summaryContent += `**File:** ${item.file || "No file provided"}\n\n`;
            summaryContent += `**Line:** ${item.line || "No line provided"}\n\n`;
            summaryContent += `**Rule:** ${item.rule_id || "No rule ID provided"}\n\n`;
            summaryContent += `**Severity:** ${item.severity || "No severity provided"}\n\n`;
            summaryContent += `**Message:** ${item.message || "No message provided"}\n\n`;
            summaryContent += "---\n\n";
        }
        await core.summary.addRaw(summaryContent).write();
        core.info("📝 Code scanning alert creation preview written to step summary");
        return [];
    }
    const createdAlerts = [];
    for (let i = 0; i < securityItems.length; i++) {
        const securityItem = securityItems[i];
        core.info(`Processing create-code-scanning-alert item ${i + 1}/${securityItems.length}: rule=${securityItem.rule_id}`);
        const file = securityItem.file;
        const line = securityItem.line;
        const ruleId = securityItem.rule_id;
        const severity = securityItem.severity || "warning";
        const message = securityItem.message;
        const endLine = securityItem.end_line || line;
        const startColumn = securityItem.start_column;
        const endColumn = securityItem.end_column;
        if (!file || !ruleId || !message) {
            core.warning(`Skipping security alert ${i + 1}: missing required fields (file, rule_id, or message)`);
            continue;
        }
        const alertPayload = {
            tool_name: "gh-aw-security-scanner",
            results: [
                {
                    rule_id: ruleId,
                    level: severity,
                    message: {
                        text: message,
                    },
                    locations: [
                        {
                            physical_location: {
                                artifact_location: {
                                    uri: file,
                                },
                                region: {
                                    start_line: parseInt(line?.toString() || "1", 10),
                                    end_line: parseInt(endLine?.toString() || line?.toString() || "1", 10),
                                    ...(startColumn && { start_column: parseInt(startColumn.toString(), 10) }),
                                    ...(endColumn && { end_column: parseInt(endColumn.toString(), 10) }),
                                },
                            },
                        },
                    ],
                },
            ],
        };
        core.info(`Creating code scanning alert for ${file}:${line}`);
        core.info(`Rule: ${ruleId}, Severity: ${severity}`);
        core.info(`Message: ${message}`);
        try {
            const { data: alert } = await github.rest.codeScanning.uploadSarif({
                owner: context.repo.owner,
                repo: context.repo.repo,
                commit_sha: context.sha,
                ref: context.ref,
                sarif: JSON.stringify({
                    version: "2.1.0",
                    $schema: "https://json.schemastore.org/sarif-2.1.0.json",
                    runs: [
                        {
                            tool: {
                                driver: {
                                    name: "gh-aw-security-scanner",
                                    version: "1.0.0",
                                    rules: [
                                        {
                                            id: ruleId,
                                            name: ruleId,
                                            short_description: {
                                                text: `Security issue: ${ruleId}`,
                                            },
                                            help: {
                                                text: message,
                                            },
                                            default_configuration: {
                                                level: severity,
                                            },
                                        },
                                    ],
                                },
                            },
                            results: alertPayload.results,
                        },
                    ],
                }),
            });
            core.info("Created code scanning alert: " + alert.id);
            createdAlerts.push({
                number: alert.id,
                url: alert.url || "",
                html_url: `${context.payload.repository?.html_url}/security/code-scanning`,
            });
            if (i === securityItems.length - 1) {
                core.setOutput("alert_id", alert.id);
                core.setOutput("alert_url", alert.url);
            }
        }
        catch (error) {
            core.error(`✗ Failed to create code scanning alert: ${error instanceof Error ? error.message : String(error)}`);
            throw error;
        }
    }
    if (createdAlerts.length > 0) {
        let summaryContent = "\n\n## GitHub Code Scanning Alerts\n";
        for (const alert of createdAlerts) {
            summaryContent += `- Alert #${alert.number}: [View Alert](${alert.html_url})\n`;
        }
        await core.summary.addRaw(summaryContent).write();
    }
    core.info(`Successfully created ${createdAlerts.length} code scanning alert(s)`);
    return createdAlerts;
}
(async () => {
    await createCodeScanningAlertMain();
})();

