async function collectNdjsonOutputMain() {
    const fs = require("fs");
    function sanitizeContent(content) {
        if (!content || typeof content !== "string") {
            return "";
        }
        const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
        const defaultAllowedDomains = [
            "github.com",
            "github.io",
            "githubusercontent.com",
            "githubassets.com",
            "github.dev",
            "codespaces.new",
        ];
        const allowedDomains = allowedDomainsEnv
            ? allowedDomainsEnv
                .split(",")
                .map(d => d.trim())
                .filter(d => d)
            : defaultAllowedDomains;
        let sanitized = content;
        sanitized = neutralizeMentions(sanitized);
        sanitized = removeXmlComments(sanitized);
        sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
        sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
        sanitized = sanitizeUrlProtocols(sanitized);
        sanitized = sanitizeUrlDomains(sanitized);
        const maxLength = 524288;
        if (sanitized.length > maxLength) {
            sanitized = sanitized.substring(0, maxLength) + "\n[Content truncated due to length]";
        }
        const lines = sanitized.split("\n");
        const maxLines = 65000;
        if (lines.length > maxLines) {
            sanitized = lines.slice(0, maxLines).join("\n") + "\n[Content truncated due to line count]";
        }
        sanitized = neutralizeBotTriggers(sanitized);
        return sanitized.trim();
        function sanitizeUrlDomains(s) {
            return s.replace(/\bhttps:\/\/[^\s\])}'"<>&\x00-\x1f,;]+/gi, match => {
                const urlAfterProtocol = match.slice(8);
                const hostname = urlAfterProtocol.split(/[\/:\?#]/)[0].toLowerCase();
                const isAllowed = allowedDomains.some(allowedDomain => {
                    const normalizedAllowed = allowedDomain.toLowerCase();
                    return hostname === normalizedAllowed || hostname.endsWith("." + normalizedAllowed);
                });
                return isAllowed ? match : "(redacted)";
            });
        }
        function sanitizeUrlProtocols(s) {
            return s.replace(/\b(?!https:\/\/)[a-z][a-z0-9+.-]*:\/\/[^\s\])}'"<>&\x00-\x1f,;]+/gi, "(redacted)");
        }
        function neutralizeMentions(s) {
            return s.replace(/(?<!`)@([a-zA-Z0-9_-]+)(?!`)/g, "`@$1`");
        }
        function removeXmlComments(s) {
            return s.replace(/<!--[\s\S]*?-->/g, "");
        }
        function neutralizeBotTriggers(s) {
            const botTriggers = [
                /(?<!`)\b(fixes?|closes?|resolves?)\s+#(\d+)(?!`)/gi,
                /(?<!`)\b(re-?open|reopen)\s+#(\d+)(?!`)/gi,
                /(?<!`)\/\w+(?!`)/g,
            ];
            let result = s;
            for (const trigger of botTriggers) {
                result = result.replace(trigger, "`$&`");
            }
            return result;
        }
    }
    function repairJson(jsonStr) {
        let repaired = jsonStr.trim();
        const _ctrl = { 8: "\\b", 9: "\\t", 10: "\\n", 12: "\\f", 13: "\\r" };
        repaired = repaired.replace(/[\u0000-\u001F]/g, ch => {
            const c = ch.charCodeAt(0);
            return _ctrl[c] || "\\u" + c.toString(16).padStart(4, "0");
        });
        repaired = repaired.replace(/'/g, '"');
        repaired = repaired.replace(/([{,]\s*)([a-zA-Z_$][a-zA-Z0-9_$]*)\s*:/g, '$1"$2":');
        repaired = repaired.replace(/"([^"\\]*)"/g, (match, content) => {
            if (content.includes("\n") || content.includes("\r") || content.includes("\t")) {
                const escaped = content
                    .replace(/\\/g, "\\\\")
                    .replace(/\n/g, "\\n")
                    .replace(/\r/g, "\\r")
                    .replace(/\t/g, "\\t");
                return `"${escaped}"`;
            }
            return match;
        });
        repaired = repaired.replace(/"([^"]*)"([^":,}\]]*)"([^"]*)"(\s*[,:}\]])/g, (match, p1, p2, p3, p4) => `"${p1}\\"${p2}\\"${p3}"${p4}`);
        repaired = repaired.replace(/(\[\s*(?:"[^"]*"(?:\s*,\s*"[^"]*")*\s*),?)\s*}/g, "$1]");
        const openBraces = (repaired.match(/\{/g) || []).length;
        const closeBraces = (repaired.match(/\}/g) || []).length;
        if (openBraces > closeBraces) {
            repaired += "}".repeat(openBraces - closeBraces);
        }
        else if (closeBraces > openBraces) {
            repaired = "{".repeat(closeBraces - openBraces) + repaired;
        }
        const openBrackets = (repaired.match(/\[/g) || []).length;
        const closeBrackets = (repaired.match(/\]/g) || []).length;
        if (openBrackets > closeBrackets) {
            repaired += "]".repeat(openBrackets - closeBrackets);
        }
        else if (closeBrackets > openBrackets) {
            repaired = "[".repeat(closeBrackets - openBrackets) + repaired;
        }
        repaired = repaired.replace(/,(\s*[}\]])/g, "$1");
        return repaired;
    }
    function validatePositiveInteger(value, fieldName, lineNum) {
        if (value === undefined || value === null) {
            if (fieldName.includes("create-code-scanning-alert 'line'")) {
                return {
                    isValid: false,
                    error: `Line ${lineNum}: create-code-scanning-alert requires a 'line' field (number or string)`,
                };
            }
            if (fieldName.includes("create-pull-request-review-comment 'line'")) {
                return {
                    isValid: false,
                    error: `Line ${lineNum}: create-pull-request-review-comment requires a 'line' number`,
                };
            }
            return {
                isValid: false,
                error: `Line ${lineNum}: ${fieldName} is required`,
            };
        }
        if (typeof value !== "number" && typeof value !== "string") {
            if (fieldName.includes("create-code-scanning-alert 'line'")) {
                return {
                    isValid: false,
                    error: `Line ${lineNum}: create-code-scanning-alert requires a 'line' field (number or string)`,
                };
            }
            if (fieldName.includes("create-pull-request-review-comment 'line'")) {
                return {
                    isValid: false,
                    error: `Line ${lineNum}: create-pull-request-review-comment requires a 'line' number`,
                };
            }
            return {
                isValid: false,
                error: `Line ${lineNum}: ${fieldName} must be a number or string`,
            };
        }
        let numValue;
        if (typeof value === "string") {
            const parsed = parseInt(value, 10);
            if (isNaN(parsed)) {
                return {
                    isValid: false,
                    error: `Line ${lineNum}: ${fieldName} must be a valid number`,
                };
            }
            numValue = parsed;
        }
        else {
            numValue = value;
        }
        if (!Number.isInteger(numValue) || numValue <= 0) {
            return {
                isValid: false,
                error: `Line ${lineNum}: ${fieldName} must be a positive integer`,
            };
        }
        return { isValid: true, normalizedValue: numValue };
    }
    function validateOptionalPositiveInteger(value, fieldName, lineNum) {
        if (value === undefined || value === null) {
            return { isValid: true };
        }
        if (typeof value !== "number" && typeof value !== "string") {
            return {
                isValid: false,
                error: `Line ${lineNum}: ${fieldName} must be a number or string`,
            };
        }
        return { isValid: true };
    }
    function validateIssueOrPRNumber(value, fieldName, lineNum) {
        if (value === undefined || value === null) {
            return { isValid: true };
        }
        if (typeof value !== "number" && typeof value !== "string") {
            return {
                isValid: false,
                error: `Line ${lineNum}: ${fieldName} must be a number or string`,
            };
        }
        return { isValid: true };
    }
    function parseJsonWithRepair(jsonStr) {
        try {
            return JSON.parse(jsonStr);
        }
        catch (originalError) {
            try {
                const repairedJson = repairJson(jsonStr);
                return JSON.parse(repairedJson);
            }
            catch (repairError) {
                core.info(`invalid input json: ${jsonStr}`);
                const originalMsg = originalError instanceof Error ? originalError.message : String(originalError);
                const repairMsg = repairError instanceof Error ? repairError.message : String(repairError);
                throw new Error(`JSON parsing failed. Original: ${originalMsg}. After attempted repair: ${repairMsg}`);
            }
        }
    }
    const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
    const safeOutputsConfig = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
    if (!outputFile) {
        core.info("GITHUB_AW_SAFE_OUTPUTS not set, no output to collect");
        core.setOutput("output", "");
        return;
    }
    if (!fs.existsSync(outputFile)) {
        core.info(`Output file does not exist: ${outputFile}`);
        core.setOutput("output", "");
        return;
    }
    const outputContent = fs.readFileSync(outputFile, "utf8");
    if (outputContent.trim() === "") {
        core.info("Output file is empty");
        core.setOutput("output", "");
        return;
    }
    core.info(`Raw output content length: ${outputContent.length}`);
    let expectedOutputTypes = {};
    if (safeOutputsConfig) {
        try {
            expectedOutputTypes = JSON.parse(safeOutputsConfig);
            core.info(`Expected output types: ${JSON.stringify(Object.keys(expectedOutputTypes))}`);
        }
        catch (error) {
            const errorMsg = error instanceof Error ? error.message : String(error);
            core.info(`Warning: Could not parse safe-outputs config: ${errorMsg}`);
        }
    }
    const lines = outputContent.trim().split("\n");
    const parsedItems = [];
    const errors = [];
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        if (line === "")
            continue;
        try {
            const item = parseJsonWithRepair(line);
            if (item === undefined) {
                errors.push(`Line ${i + 1}: Invalid JSON - JSON parsing failed`);
                continue;
            }
            if (!item.type) {
                errors.push(`Line ${i + 1}: Missing required 'type' field`);
                continue;
            }
            const itemType = item.type;
            if (!expectedOutputTypes[itemType]) {
                errors.push(`Line ${i + 1}: Unexpected output type '${itemType}'. Expected one of: ${Object.keys(expectedOutputTypes).join(", ")}`);
                continue;
            }
            switch (itemType) {
                case "create-issue":
                    if (!item.title || typeof item.title !== "string") {
                        errors.push(`Line ${i + 1}: create-issue requires a 'title' string field`);
                        continue;
                    }
                    if (!item.body || typeof item.body !== "string") {
                        errors.push(`Line ${i + 1}: create-issue requires a 'body' string field`);
                        continue;
                    }
                    item.title = sanitizeContent(item.title);
                    item.body = sanitizeContent(item.body);
                    if (item.labels !== undefined) {
                        if (!Array.isArray(item.labels)) {
                            errors.push(`Line ${i + 1}: create-issue 'labels' must be an array`);
                            continue;
                        }
                        for (let j = 0; j < item.labels.length; j++) {
                            if (typeof item.labels[j] !== "string") {
                                errors.push(`Line ${i + 1}: create-issue label at index ${j} must be a string`);
                                continue;
                            }
                            item.labels[j] = sanitizeContent(item.labels[j]);
                        }
                    }
                    break;
                case "create-discussion":
                    if (!item.title || typeof item.title !== "string") {
                        errors.push(`Line ${i + 1}: create-discussion requires a 'title' string field`);
                        continue;
                    }
                    if (!item.body || typeof item.body !== "string") {
                        errors.push(`Line ${i + 1}: create-discussion requires a 'body' string field`);
                        continue;
                    }
                    item.title = sanitizeContent(item.title);
                    item.body = sanitizeContent(item.body);
                    if (item.category !== undefined && typeof item.category !== "string") {
                        errors.push(`Line ${i + 1}: create-discussion 'category' must be a string`);
                        continue;
                    }
                    if (item.category) {
                        item.category = sanitizeContent(item.category);
                    }
                    break;
                case "add-comment":
                    if (!item.body || typeof item.body !== "string") {
                        errors.push(`Line ${i + 1}: add-comment requires a 'body' string field`);
                        continue;
                    }
                    item.body = sanitizeContent(item.body);
                    const addCommentIssueNumValidation = validateIssueOrPRNumber(item.issue_number, "add-comment 'issue_number'", i + 1);
                    if (!addCommentIssueNumValidation.isValid) {
                        errors.push(addCommentIssueNumValidation.error);
                        continue;
                    }
                    break;
                case "create-pull-request":
                    if (!item.title || typeof item.title !== "string") {
                        errors.push(`Line ${i + 1}: create-pull-request requires a 'title' string field`);
                        continue;
                    }
                    if (!item.body || typeof item.body !== "string") {
                        errors.push(`Line ${i + 1}: create-pull-request requires a 'body' string field`);
                        continue;
                    }
                    if (!item.branch || typeof item.branch !== "string") {
                        errors.push(`Line ${i + 1}: create-pull-request requires a 'branch' string field`);
                        continue;
                    }
                    item.title = sanitizeContent(item.title);
                    item.body = sanitizeContent(item.body);
                    item.branch = sanitizeContent(item.branch);
                    if (item.labels !== undefined) {
                        if (!Array.isArray(item.labels)) {
                            errors.push(`Line ${i + 1}: create-pull-request 'labels' must be an array`);
                            continue;
                        }
                        for (let j = 0; j < item.labels.length; j++) {
                            if (typeof item.labels[j] !== "string") {
                                errors.push(`Line ${i + 1}: create-pull-request label at index ${j} must be a string`);
                                continue;
                            }
                            item.labels[j] = sanitizeContent(item.labels[j]);
                        }
                    }
                    break;
                case "create-code-scanning-alert":
                    if (!item.file || typeof item.file !== "string") {
                        errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'file' string field`);
                        continue;
                    }
                    const codeAlertLineValidation = validatePositiveInteger(item.line, "create-code-scanning-alert 'line'", i + 1);
                    if (!codeAlertLineValidation.isValid) {
                        errors.push(codeAlertLineValidation.error);
                        continue;
                    }
                    if (!item.severity || typeof item.severity !== "string") {
                        errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'severity' string field`);
                        continue;
                    }
                    const validSeverities = ["error", "warning", "info", "note"];
                    if (!validSeverities.includes(item.severity)) {
                        errors.push(`Line ${i + 1}: create-code-scanning-alert 'severity' must be one of: ${validSeverities.join(", ")}`);
                        continue;
                    }
                    if (!item.message || typeof item.message !== "string") {
                        errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'message' string field`);
                        continue;
                    }
                    item.file = sanitizeContent(item.file);
                    item.message = sanitizeContent(item.message);
                    const columnValidation = validateOptionalPositiveInteger(item.column, "create-code-scanning-alert 'column'", i + 1);
                    if (!columnValidation.isValid) {
                        errors.push(columnValidation.error);
                        continue;
                    }
                    if (item.ruleIdSuffix !== undefined && typeof item.ruleIdSuffix !== "string") {
                        errors.push(`Line ${i + 1}: create-code-scanning-alert 'ruleIdSuffix' must be a string`);
                        continue;
                    }
                    if (item.ruleIdSuffix) {
                        item.ruleIdSuffix = sanitizeContent(item.ruleIdSuffix);
                    }
                    break;
                case "add-labels":
                    if (!item.labels || !Array.isArray(item.labels)) {
                        errors.push(`Line ${i + 1}: add-labels requires a 'labels' array field`);
                        continue;
                    }
                    if (item.labels.length === 0) {
                        errors.push(`Line ${i + 1}: add-labels 'labels' array cannot be empty`);
                        continue;
                    }
                    for (let j = 0; j < item.labels.length; j++) {
                        if (typeof item.labels[j] !== "string") {
                            errors.push(`Line ${i + 1}: add-labels label at index ${j} must be a string`);
                            continue;
                        }
                        item.labels[j] = sanitizeContent(item.labels[j]);
                    }
                    const addLabelsIssueNumValidation = validateIssueOrPRNumber(item.issue_number, "add-labels 'issue_number'", i + 1);
                    if (!addLabelsIssueNumValidation.isValid) {
                        errors.push(addLabelsIssueNumValidation.error);
                        continue;
                    }
                    break;
                case "update-issue":
                    const hasValidField = item.status !== undefined || item.title !== undefined || item.body !== undefined;
                    if (!hasValidField) {
                        errors.push(`Line ${i + 1}: update_issue requires at least one of: 'status', 'title', or 'body' fields`);
                        continue;
                    }
                    if (item.status !== undefined) {
                        if (typeof item.status !== "string" || (item.status !== "open" && item.status !== "closed")) {
                            errors.push(`Line ${i + 1}: update_issue 'status' must be 'open' or 'closed'`);
                            continue;
                        }
                    }
                    if (item.title !== undefined) {
                        if (typeof item.title !== "string") {
                            errors.push(`Line ${i + 1}: update-issue 'title' must be a string`);
                            continue;
                        }
                        item.title = sanitizeContent(item.title);
                    }
                    if (item.body !== undefined) {
                        if (typeof item.body !== "string") {
                            errors.push(`Line ${i + 1}: update-issue 'body' must be a string`);
                            continue;
                        }
                        item.body = sanitizeContent(item.body);
                    }
                    const updateIssueNumValidation = validateIssueOrPRNumber(item.issue_number, "update-issue 'issue_number'", i + 1);
                    if (!updateIssueNumValidation.isValid) {
                        errors.push(updateIssueNumValidation.error);
                        continue;
                    }
                    break;
                case "push-to-pr-branch":
                    if (!item.branch || typeof item.branch !== "string") {
                        errors.push(`Line ${i + 1}: push_to_pr_branch requires a 'branch' string field`);
                        continue;
                    }
                    if (!item.message || typeof item.message !== "string") {
                        errors.push(`Line ${i + 1}: push_to_pr_branch requires a 'message' string field`);
                        continue;
                    }
                    item.branch = sanitizeContent(item.branch);
                    item.message = sanitizeContent(item.message);
                    const pushPRNumValidation = validateIssueOrPRNumber(item.pull_request_number, "push-to-pr-branch 'pull_request_number'", i + 1);
                    if (!pushPRNumValidation.isValid) {
                        errors.push(pushPRNumValidation.error);
                        continue;
                    }
                    break;
                case "create-pull-request-review-comment":
                    if (!item.path || typeof item.path !== "string") {
                        errors.push(`Line ${i + 1}: create-pull-request-review-comment requires a 'path' string field`);
                        continue;
                    }
                    const lineValidation = validatePositiveInteger(item.line, "create-pull-request-review-comment 'line'", i + 1);
                    if (!lineValidation.isValid) {
                        errors.push(lineValidation.error);
                        continue;
                    }
                    const lineNumber = lineValidation.normalizedValue;
                    if (!item.body || typeof item.body !== "string") {
                        errors.push(`Line ${i + 1}: create-pull-request-review-comment requires a 'body' string field`);
                        continue;
                    }
                    item.body = sanitizeContent(item.body);
                    const startLineValidation = validateOptionalPositiveInteger(item.start_line, "create-pull-request-review-comment 'start_line'", i + 1);
                    if (!startLineValidation.isValid) {
                        errors.push(startLineValidation.error);
                        continue;
                    }
                    if (startLineValidation.normalizedValue !== undefined &&
                        lineNumber !== undefined &&
                        startLineValidation.normalizedValue > lineNumber) {
                        errors.push(`Line ${i + 1}: create-pull-request-review-comment 'start_line' must be less than or equal to 'line'`);
                        continue;
                    }
                    if (item.side !== undefined) {
                        if (typeof item.side !== "string" || (item.side !== "LEFT" && item.side !== "RIGHT")) {
                            errors.push(`Line ${i + 1}: create-pull-request-review-comment 'side' must be 'LEFT' or 'RIGHT'`);
                            continue;
                        }
                    }
                    break;
                case "missing-tool":
                    if (!item.tool || typeof item.tool !== "string") {
                        errors.push(`Line ${i + 1}: missing-tool requires a 'tool' string field`);
                        continue;
                    }
                    if (!item.reason || typeof item.reason !== "string") {
                        errors.push(`Line ${i + 1}: missing-tool requires a 'reason' string field`);
                        continue;
                    }
                    item.tool = sanitizeContent(item.tool);
                    item.reason = sanitizeContent(item.reason);
                    if (item.alternatives !== undefined && typeof item.alternatives !== "string") {
                        errors.push(`Line ${i + 1}: missing-tool 'alternatives' must be a string`);
                        continue;
                    }
                    if (item.alternatives) {
                        item.alternatives = sanitizeContent(item.alternatives);
                    }
                    break;
                case "upload-asset":
                    if (!item.path || typeof item.path !== "string") {
                        errors.push(`Line ${i + 1}: upload-asset requires a 'path' string field`);
                        continue;
                    }
                    if (item.fileName !== undefined && typeof item.fileName !== "string") {
                        errors.push(`Line ${i + 1}: upload-asset 'fileName' must be a string`);
                        continue;
                    }
                    if (item.sha !== undefined && typeof item.sha !== "string") {
                        errors.push(`Line ${i + 1}: upload-asset 'sha' must be a string`);
                        continue;
                    }
                    if (item.size !== undefined && typeof item.size !== "number") {
                        errors.push(`Line ${i + 1}: upload-asset 'size' must be a number`);
                        continue;
                    }
                    if (item.url !== undefined && typeof item.url !== "string") {
                        errors.push(`Line ${i + 1}: upload-asset 'url' must be a string`);
                        continue;
                    }
                    if (item.targetFileName !== undefined && typeof item.targetFileName !== "string") {
                        errors.push(`Line ${i + 1}: upload-asset 'targetFileName' must be a string`);
                        continue;
                    }
                    item.path = sanitizeContent(item.path);
                    if (item.fileName)
                        item.fileName = sanitizeContent(item.fileName);
                    if (item.url)
                        item.url = sanitizeContent(item.url);
                    if (item.targetFileName)
                        item.targetFileName = sanitizeContent(item.targetFileName);
                    break;
                default:
                    errors.push(`Line ${i + 1}: Unknown output type '${itemType}'`);
                    continue;
            }
            parsedItems.push(item);
        }
        catch (error) {
            const errorMsg = error instanceof Error ? error.message : String(error);
            errors.push(`Line ${i + 1}: ${errorMsg}`);
        }
    }
    if (errors.length > 0) {
        core.warning(`Found ${errors.length} validation errors:`);
        errors.forEach(error => core.warning(`  ${error}`));
        if (errors.length > 0) {
            core.setFailed(errors.map(e => `  - ${e}`).join("\n"));
            return;
        }
    }
    core.info(`Successfully parsed ${parsedItems.length} valid output items`);
    const validatedOutput = {
        items: parsedItems,
        errors: errors,
    };
    const agentOutputFile = "/tmp/agent_output.json";
    const validatedOutputJson = JSON.stringify(validatedOutput);
    try {
        fs.mkdirSync("/tmp", { recursive: true });
        fs.writeFileSync(agentOutputFile, validatedOutputJson, "utf8");
        core.info(`Stored validated output to: ${agentOutputFile}`);
        core.exportVariable("GITHUB_AW_AGENT_OUTPUT", agentOutputFile);
    }
    catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        core.error(`Failed to write agent output file: ${errorMsg}`);
    }
    core.setOutput("output", JSON.stringify(validatedOutput));
    core.setOutput("raw_output", outputContent);
    try {
        await core.summary
            .addRaw("## Processed Output\n\n")
            .addRaw("```json\n")
            .addRaw(JSON.stringify(validatedOutput))
            .addRaw("\n```\n")
            .write();
        core.info("Successfully wrote processed output to step summary");
    }
    catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        core.warning(`Failed to write to step summary: ${errorMsg}`);
    }
}
(async () => {
    await collectNdjsonOutputMain();
})();
