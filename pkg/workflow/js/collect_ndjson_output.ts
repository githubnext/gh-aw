async function collectNdjsonOutputMain(): Promise<void> {
  const fs = require("fs");

  interface ValidationResult {
    isValid: boolean;
    error?: string;
    normalizedValue?: number;
  }

  interface SafeOutputItem {
    type: string;
    [key: string]: any;
  }

  interface OutputData {
    items: SafeOutputItem[];
    errors: string[];
  }

  /**
   * Sanitizes content for safe output in GitHub Actions
   * @param content - The content to sanitize
   * @returns The sanitized content
   */
  function sanitizeContent(content: string): string {
    if (!content || typeof content !== "string") {
      return "";
    }

    // Read allowed domains from environment variable
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

    // Neutralize @mentions to prevent unintended notifications
    sanitized = neutralizeMentions(sanitized);

    // Remove XML comments to prevent content hiding
    sanitized = removeXmlComments(sanitized);

    // Remove ANSI escape sequences BEFORE removing control characters
    sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");

    // Remove control characters (except newlines and tabs)
    sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

    // URI filtering - replace non-https protocols with "(redacted)"
    sanitized = sanitizeUrlProtocols(sanitized);

    // Domain filtering for HTTPS URIs
    sanitized = sanitizeUrlDomains(sanitized);

    // Limit total length to prevent DoS (0.5MB max)
    const maxLength = 524288;
    if (sanitized.length > maxLength) {
      sanitized = sanitized.substring(0, maxLength) + "\n[Content truncated due to length]";
    }

    // Limit number of lines to prevent log flooding (65k max)
    const lines = sanitized.split("\n");
    const maxLines = 65000;
    if (lines.length > maxLines) {
      sanitized = lines.slice(0, maxLines).join("\n") + "\n[Content truncated due to line count]";
    }

    // ANSI escape sequences already removed earlier in the function

    // Neutralize common bot trigger phrases
    sanitized = neutralizeBotTriggers(sanitized);

    // Trim excessive whitespace
    return sanitized.trim();

    /**
     * Remove unknown domains
     * @param s - The string to process
     * @returns The string with unknown domains redacted
     */
    function sanitizeUrlDomains(s: string): string {
      return s.replace(/\bhttps:\/\/[^\s\])}'"<>&\x00-\x1f,;]+/gi, match => {
        // Extract just the URL part after https://
        const urlAfterProtocol = match.slice(8); // Remove 'https://'

        // Extract the hostname part (before first slash, colon, or other delimiter)
        const hostname = urlAfterProtocol.split(/[\/:\?#]/)[0].toLowerCase();

        // Check if this domain or any parent domain is in the allowlist
        const isAllowed = allowedDomains.some(allowedDomain => {
          const normalizedAllowed = allowedDomain.toLowerCase();
          return hostname === normalizedAllowed || hostname.endsWith("." + normalizedAllowed);
        });

        return isAllowed ? match : "(redacted)";
      });
    }

    /**
     * Remove unknown protocols except https
     * @param s - The string to process
     * @returns The string with non-https protocols redacted
     */
    function sanitizeUrlProtocols(s: string): string {
      // Match protocols that are not https
      return s.replace(/\b(?!https:\/\/)[a-z][a-z0-9+.-]*:\/\/[^\s\])}'"<>&\x00-\x1f,;]+/gi, "(redacted)");
    }

    /**
     * Neutralize @mentions by wrapping them in backticks
     * @param s - The string to process
     * @returns The string with neutralized @mentions
     */
    function neutralizeMentions(s: string): string {
      // Match @username patterns but avoid already neutralized ones (in backticks)
      return s.replace(/(?<!`)@([a-zA-Z0-9_-]+)(?!`)/g, "`@$1`");
    }

    /**
     * Remove XML comments to prevent content hiding
     * @param s - The string to process
     * @returns The string with XML comments removed
     */
    function removeXmlComments(s: string): string {
      // Remove <!-- comment --> patterns
      return s.replace(/<!--[\s\S]*?-->/g, "");
    }

    /**
     * Neutralize common bot trigger phrases by wrapping them in backticks
     * @param s - The string to process
     * @returns The string with neutralized bot triggers
     */
    function neutralizeBotTriggers(s: string): string {
      const botTriggers = [
        /(?<!`)\b(fixes?|closes?|resolves?)\s+#(\d+)(?!`)/gi,
        /(?<!`)\b(re-?open|reopen)\s+#(\d+)(?!`)/gi,
        /(?<!`)\/\w+(?!`)/g, // slash commands like /help, /close, etc.
      ];

      let result = s;
      for (const trigger of botTriggers) {
        result = result.replace(trigger, "`$&`");
      }
      return result;
    }
  }

  /**
   * Simple JSON repair attempt
   * @param jsonStr - The malformed JSON string
   * @returns A potentially repaired JSON string
   */
  function repairJson(jsonStr: string): string {
    let repaired = jsonStr.trim();

    // Convert control characters to proper JSON escape sequences first
    // Preserve common JSON escapes for \b, \f, \n, \r, \t and use \uXXXX for the rest.
    const _ctrl: Record<number, string> = { 8: "\\b", 9: "\\t", 10: "\\n", 12: "\\f", 13: "\\r" };
    repaired = repaired.replace(/[\u0000-\u001F]/g, ch => {
      const c = ch.charCodeAt(0);
      return _ctrl[c] || "\\u" + c.toString(16).padStart(4, "0");
    });

    // Fix single quotes to double quotes (must be done first)
    repaired = repaired.replace(/'/g, '"');

    // Fix missing quotes around object keys
    repaired = repaired.replace(/([{,]\s*)([a-zA-Z_$][a-zA-Z0-9_$]*)\s*:/g, '$1"$2":');

    // Fix newlines and tabs inside strings by escaping them
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

    // Fix unescaped quotes inside string values
    repaired = repaired.replace(
      /"([^"]*)"([^":,}\]]*)"([^"]*)"(\s*[,:}\]])/g,
      (match, p1, p2, p3, p4) => `"${p1}\\"${p2}\\"${p3}"${p4}`
    );

    // Fix wrong bracket/brace types - arrays should end with ] not }
    repaired = repaired.replace(/(\[\s*(?:"[^"]*"(?:\s*,\s*"[^"]*")*\s*),?)\s*}/g, "$1]");

    // Fix missing closing braces/brackets
    const openBraces = (repaired.match(/\{/g) || []).length;
    const closeBraces = (repaired.match(/\}/g) || []).length;

    if (openBraces > closeBraces) {
      repaired += "}".repeat(openBraces - closeBraces);
    } else if (closeBraces > openBraces) {
      repaired = "{".repeat(closeBraces - openBraces) + repaired;
    }

    // Fix missing closing brackets for arrays
    const openBrackets = (repaired.match(/\[/g) || []).length;
    const closeBrackets = (repaired.match(/\]/g) || []).length;

    if (openBrackets > closeBrackets) {
      repaired += "]".repeat(openBrackets - closeBrackets);
    } else if (closeBrackets > openBrackets) {
      repaired = "[".repeat(closeBrackets - openBrackets) + repaired;
    }

    // Fix trailing commas in objects and arrays (AFTER fixing brackets/braces)
    repaired = repaired.replace(/,(\s*[}\]])/g, "$1");

    return repaired;
  }

  /**
   * Validates that a value is a positive integer
   * @param value - The value to validate
   * @param fieldName - The name of the field being validated
   * @param lineNum - The line number for error reporting
   * @returns Validation result
   */
  function validatePositiveInteger(value: any, fieldName: string, lineNum: number): ValidationResult {
    if (value === undefined || value === null) {
      // Match the original error format for create-code-scanning-alert
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
      // Match the original error format for create-code-scanning-alert
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

    // Convert to number if it's a string
    let numValue: number;
    if (typeof value === "string") {
      const parsed = parseInt(value, 10);
      if (isNaN(parsed)) {
        return {
          isValid: false,
          error: `Line ${lineNum}: ${fieldName} must be a valid number`,
        };
      }
      numValue = parsed;
    } else {
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

  /**
   * Validates that a value is a positive integer if provided (optional)
   * @param value - The value to validate
   * @param fieldName - The name of the field being validated
   * @param lineNum - The line number for error reporting
   * @returns Validation result
   */
  function validateOptionalPositiveInteger(value: any, fieldName: string, lineNum: number): ValidationResult {
    if (value === undefined || value === null) {
      return { isValid: true }; // Optional field is valid when not provided
    }

    if (typeof value !== "number" && typeof value !== "string") {
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} must be a number or string`,
      };
    }

    return { isValid: true };
  }

  /**
   * Validates that a value is an issue or PR number (positive integer) if provided (optional)
   * @param value - The value to validate
   * @param fieldName - The name of the field being validated
   * @param lineNum - The line number for error reporting
   * @returns Validation result
   */
  function validateIssueOrPRNumber(value: any, fieldName: string, lineNum: number): ValidationResult {
    if (value === undefined || value === null) {
      return { isValid: true }; // Optional field is valid when not provided
    }

    if (typeof value !== "number" && typeof value !== "string") {
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} must be a number or string`,
      };
    }

    return { isValid: true };
  }

  /**
   * Attempts to parse JSON with repair fallback
   * @param jsonStr - The JSON string to parse
   * @returns The parsed JSON object, or undefined if parsing fails
   */
  function parseJsonWithRepair(jsonStr: string): any {
    try {
      // First, try normal JSON.parse
      return JSON.parse(jsonStr);
    } catch (originalError) {
      try {
        // If that fails, try repairing and parsing again
        const repairedJson = repairJson(jsonStr);
        return JSON.parse(repairedJson);
      } catch (repairError) {
        // If repair also fails, throw the error
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

  // Parse the safe-outputs configuration
  let expectedOutputTypes: any = {};
  if (safeOutputsConfig) {
    try {
      expectedOutputTypes = JSON.parse(safeOutputsConfig);
      core.info(`Expected output types: ${JSON.stringify(Object.keys(expectedOutputTypes))}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      core.info(`Warning: Could not parse safe-outputs config: ${errorMsg}`);
    }
  }

  // Parse JSONL content
  const lines = outputContent.trim().split("\n");
  const parsedItems: SafeOutputItem[] = [];
  const errors: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === "") continue; // Skip empty lines
    try {
      const item: any = parseJsonWithRepair(line);

      // If item is undefined (failed to parse), add error and process next line
      if (item === undefined) {
        errors.push(`Line ${i + 1}: Invalid JSON - JSON parsing failed`);
        continue;
      }

      // Validate that the item has a 'type' field
      if (!item.type) {
        errors.push(`Line ${i + 1}: Missing required 'type' field`);
        continue;
      }

      // Validate against expected output types
      const itemType = item.type;
      if (!expectedOutputTypes[itemType]) {
        errors.push(`Line ${i + 1}: Unexpected output type '${itemType}'. Expected one of: ${Object.keys(expectedOutputTypes).join(", ")}`);
        continue;
      }

      // Validate common fields based on the type
      switch (itemType) {
        case "create-issue":
          // Validate required title field
          if (!item.title || typeof item.title !== "string") {
            errors.push(`Line ${i + 1}: create-issue requires a 'title' string field`);
            continue;
          }
          // Validate required body field
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create-issue requires a 'body' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Validate optional labels field
          if (item.labels !== undefined) {
            if (!Array.isArray(item.labels)) {
              errors.push(`Line ${i + 1}: create-issue 'labels' must be an array`);
              continue;
            }
            // Validate each label is a string
            for (let j = 0; j < item.labels.length; j++) {
              if (typeof item.labels[j] !== "string") {
                errors.push(`Line ${i + 1}: create-issue label at index ${j} must be a string`);
                continue;
              }
              // Sanitize label content
              item.labels[j] = sanitizeContent(item.labels[j]);
            }
          }
          break;

        case "create-discussion":
          // Validate required title field
          if (!item.title || typeof item.title !== "string") {
            errors.push(`Line ${i + 1}: create-discussion requires a 'title' string field`);
            continue;
          }
          // Validate required body field
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create-discussion requires a 'body' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Validate optional category field
          if (item.category !== undefined && typeof item.category !== "string") {
            errors.push(`Line ${i + 1}: create-discussion 'category' must be a string`);
            continue;
          }
          if (item.category) {
            item.category = sanitizeContent(item.category);
          }
          break;

        case "add-comment":
          // Validate required body field
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: add-comment requires a 'body' string field`);
            continue;
          }
          // Sanitize body content
          item.body = sanitizeContent(item.body);
          // Validate optional issue_number field
          const addCommentIssueNumValidation = validateIssueOrPRNumber(
            item.issue_number,
            "add-comment 'issue_number'",
            i + 1
          );
          if (!addCommentIssueNumValidation.isValid) {
            errors.push(addCommentIssueNumValidation.error!);
            continue;
          }
          break;

        case "create-pull-request":
          // Validate required title field
          if (!item.title || typeof item.title !== "string") {
            errors.push(`Line ${i + 1}: create-pull-request requires a 'title' string field`);
            continue;
          }
          // Validate required body field
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create-pull-request requires a 'body' string field`);
            continue;
          }
          // Validate required branch field
          if (!item.branch || typeof item.branch !== "string") {
            errors.push(`Line ${i + 1}: create-pull-request requires a 'branch' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          item.branch = sanitizeContent(item.branch);
          // Validate optional labels field
          if (item.labels !== undefined) {
            if (!Array.isArray(item.labels)) {
              errors.push(`Line ${i + 1}: create-pull-request 'labels' must be an array`);
              continue;
            }
            // Validate each label is a string
            for (let j = 0; j < item.labels.length; j++) {
              if (typeof item.labels[j] !== "string") {
                errors.push(`Line ${i + 1}: create-pull-request label at index ${j} must be a string`);
                continue;
              }
              // Sanitize label content
              item.labels[j] = sanitizeContent(item.labels[j]);
            }
          }
          break;

        case "create-code-scanning-alert":
          // Validate required file field
          if (!item.file || typeof item.file !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'file' string field`);
            continue;
          }
          // Validate required line field
          const codeAlertLineValidation = validatePositiveInteger(item.line, "create-code-scanning-alert 'line'", i + 1);
          if (!codeAlertLineValidation.isValid) {
            errors.push(codeAlertLineValidation.error!);
            continue;
          }
          // Validate required severity field
          if (!item.severity || typeof item.severity !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'severity' string field`);
            continue;
          }
          const validSeverities = ["error", "warning", "info", "note"];
          if (!validSeverities.includes(item.severity)) {
            errors.push(`Line ${i + 1}: create-code-scanning-alert 'severity' must be one of: ${validSeverities.join(", ")}`);
            continue;
          }
          // Validate required message field
          if (!item.message || typeof item.message !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'message' string field`);
            continue;
          }
          // Sanitize text content
          item.file = sanitizeContent(item.file);
          item.message = sanitizeContent(item.message);
          // Validate optional column field
          const columnValidation = validateOptionalPositiveInteger(item.column, "create-code-scanning-alert 'column'", i + 1);
          if (!columnValidation.isValid) {
            errors.push(columnValidation.error!);
            continue;
          }
          // Validate optional ruleIdSuffix field
          if (item.ruleIdSuffix !== undefined && typeof item.ruleIdSuffix !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert 'ruleIdSuffix' must be a string`);
            continue;
          }
          if (item.ruleIdSuffix) {
            item.ruleIdSuffix = sanitizeContent(item.ruleIdSuffix);
          }
          break;

        case "add-labels":
          // Validate required labels field
          if (!item.labels || !Array.isArray(item.labels)) {
            errors.push(`Line ${i + 1}: add-labels requires a 'labels' array field`);
            continue;
          }
          if (item.labels.length === 0) {
            errors.push(`Line ${i + 1}: add-labels 'labels' array cannot be empty`);
            continue;
          }
          // Validate each label is a string
          for (let j = 0; j < item.labels.length; j++) {
            if (typeof item.labels[j] !== "string") {
              errors.push(`Line ${i + 1}: add-labels label at index ${j} must be a string`);
              continue;
            }
            // Sanitize label content
            item.labels[j] = sanitizeContent(item.labels[j]);
          }
          // Validate optional issue_number field
          const addLabelsIssueNumValidation = validateIssueOrPRNumber(
            item.issue_number,
            "add-labels 'issue_number'",
            i + 1
          );
          if (!addLabelsIssueNumValidation.isValid) {
            errors.push(addLabelsIssueNumValidation.error!);
            continue;
          }
          break;

        case "update-issue":
          // Check that at least one updateable field is provided
          const hasValidField = item.status !== undefined || item.title !== undefined || item.body !== undefined;
          if (!hasValidField) {
            errors.push(`Line ${i + 1}: update_issue requires at least one of: 'status', 'title', or 'body' fields`);
            continue;
          }
          // Validate status if provided
          if (item.status !== undefined) {
            if (typeof item.status !== "string" || (item.status !== "open" && item.status !== "closed")) {
              errors.push(`Line ${i + 1}: update_issue 'status' must be 'open' or 'closed'`);
              continue;
            }
          }
          // Validate title if provided
          if (item.title !== undefined) {
            if (typeof item.title !== "string") {
              errors.push(`Line ${i + 1}: update-issue 'title' must be a string`);
              continue;
            }
            item.title = sanitizeContent(item.title);
          }
          // Validate body if provided
          if (item.body !== undefined) {
            if (typeof item.body !== "string") {
              errors.push(`Line ${i + 1}: update-issue 'body' must be a string`);
              continue;
            }
            item.body = sanitizeContent(item.body);
          }
          // Validate issue_number if provided (for target "*")
          const updateIssueNumValidation = validateIssueOrPRNumber(
            item.issue_number,
            "update-issue 'issue_number'",
            i + 1
          );
          if (!updateIssueNumValidation.isValid) {
            errors.push(updateIssueNumValidation.error!);
            continue;
          }
          break;

        case "push-to-pr-branch":
          // Validate required branch field
          if (!item.branch || typeof item.branch !== "string") {
            errors.push(`Line ${i + 1}: push_to_pr_branch requires a 'branch' string field`);
            continue;
          }
          // Validate required message field
          if (!item.message || typeof item.message !== "string") {
            errors.push(`Line ${i + 1}: push_to_pr_branch requires a 'message' string field`);
            continue;
          }
          // Sanitize text content
          item.branch = sanitizeContent(item.branch);
          item.message = sanitizeContent(item.message);
          // Validate pull_request_number if provided (for target "*")
          const pushPRNumValidation = validateIssueOrPRNumber(
            item.pull_request_number,
            "push-to-pr-branch 'pull_request_number'",
            i + 1
          );
          if (!pushPRNumValidation.isValid) {
            errors.push(pushPRNumValidation.error!);
            continue;
          }
          break;
        case "create-pull-request-review-comment":
          // Validate required path field
          if (!item.path || typeof item.path !== "string") {
            errors.push(`Line ${i + 1}: create-pull-request-review-comment requires a 'path' string field`);
            continue;
          }
          // Validate required line field
          const lineValidation = validatePositiveInteger(item.line, "create-pull-request-review-comment 'line'", i + 1);
          if (!lineValidation.isValid) {
            errors.push(lineValidation.error!);
            continue;
          }
          // lineValidation.normalizedValue is guaranteed to be defined when isValid is true
          const lineNumber = lineValidation.normalizedValue!;
          // Validate required body field
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create-pull-request-review-comment requires a 'body' string field`);
            continue;
          }
          // Sanitize required text content
          item.body = sanitizeContent(item.body);
          // Validate optional start_line field
          const startLineValidation = validateOptionalPositiveInteger(
            item.start_line,
            "create-pull-request-review-comment 'start_line'",
            i + 1
          );
          if (!startLineValidation.isValid) {
            errors.push(startLineValidation.error!);
            continue;
          }
          if (
            startLineValidation.normalizedValue !== undefined &&
            lineNumber !== undefined &&
            startLineValidation.normalizedValue > lineNumber
          ) {
            errors.push(`Line ${i + 1}: create-pull-request-review-comment 'start_line' must be less than or equal to 'line'`);
            continue;
          }
          // Validate optional side field
          if (item.side !== undefined) {
            if (typeof item.side !== "string" || (item.side !== "LEFT" && item.side !== "RIGHT")) {
              errors.push(`Line ${i + 1}: create-pull-request-review-comment 'side' must be 'LEFT' or 'RIGHT'`);
              continue;
            }
          }
          break;

        case "missing-tool":
          // Validate required tool field
          if (!item.tool || typeof item.tool !== "string") {
            errors.push(`Line ${i + 1}: missing-tool requires a 'tool' string field`);
            continue;
          }
          // Validate required reason field
          if (!item.reason || typeof item.reason !== "string") {
            errors.push(`Line ${i + 1}: missing-tool requires a 'reason' string field`);
            continue;
          }
          // Sanitize text content
          item.tool = sanitizeContent(item.tool);
          item.reason = sanitizeContent(item.reason);
          // Validate optional alternatives field
          if (item.alternatives !== undefined && typeof item.alternatives !== "string") {
            errors.push(`Line ${i + 1}: missing-tool 'alternatives' must be a string`);
            continue;
          }
          if (item.alternatives) {
            item.alternatives = sanitizeContent(item.alternatives);
          }
          break;

        case "upload-asset":
          // Validate required path field
          if (!item.path || typeof item.path !== "string") {
            errors.push(`Line ${i + 1}: upload-asset requires a 'path' string field`);
            continue;
          }
          // Validate fileName (should be string if present)
          if (item.fileName !== undefined && typeof item.fileName !== "string") {
            errors.push(`Line ${i + 1}: upload-asset 'fileName' must be a string`);
            continue;
          }
          // Validate sha (should be string if present)
          if (item.sha !== undefined && typeof item.sha !== "string") {
            errors.push(`Line ${i + 1}: upload-asset 'sha' must be a string`);
            continue;
          }
          // Validate size (should be number if present)
          if (item.size !== undefined && typeof item.size !== "number") {
            errors.push(`Line ${i + 1}: upload-asset 'size' must be a number`);
            continue;
          }
          // Validate url (should be string if present)
          if (item.url !== undefined && typeof item.url !== "string") {
            errors.push(`Line ${i + 1}: upload-asset 'url' must be a string`);
            continue;
          }
          // Validate targetFileName (should be string if present)
          if (item.targetFileName !== undefined && typeof item.targetFileName !== "string") {
            errors.push(`Line ${i + 1}: upload-asset 'targetFileName' must be a string`);
            continue;
          }
          // Sanitize path content
          item.path = sanitizeContent(item.path);
          if (item.fileName) item.fileName = sanitizeContent(item.fileName);
          if (item.url) item.url = sanitizeContent(item.url);
          if (item.targetFileName) item.targetFileName = sanitizeContent(item.targetFileName);
          break;

        default:
          errors.push(`Line ${i + 1}: Unknown output type '${itemType}'`);
          continue;
      }

      // If we made it here, the item is valid
      parsedItems.push(item);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      errors.push(`Line ${i + 1}: ${errorMsg}`);
    }
  }

  // Check if there are too many errors
  if (errors.length > 0) {
    core.warning(`Found ${errors.length} validation errors:`);
    errors.forEach(error => core.warning(`  ${error}`));

    // For now, fail if there are any errors to maintain strict validation
    if (errors.length > 0) {
      core.setFailed(errors.map(e => `  - ${e}`).join("\n"));
      return;
    }

    // For now, we'll continue with valid items but log the errors
    // In the future, we might want to fail the workflow for invalid items
  }

  core.info(`Successfully parsed ${parsedItems.length} valid output items`);

  // Set the parsed and validated items as output
  const validatedOutput: OutputData = {
    items: parsedItems,
    errors: errors,
  };

  // Store validatedOutput JSON in "agent_output.json" file
  const agentOutputFile = "/tmp/agent_output.json";
  const validatedOutputJson = JSON.stringify(validatedOutput);

  try {
    // Ensure the /tmp directory exists
    fs.mkdirSync("/tmp", { recursive: true });
    fs.writeFileSync(agentOutputFile, validatedOutputJson, "utf8");
    core.info(`Stored validated output to: ${agentOutputFile}`);

    // Set the environment variable GITHUB_AW_AGENT_OUTPUT to the file path
    core.exportVariable("GITHUB_AW_AGENT_OUTPUT", agentOutputFile);
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    core.error(`Failed to write agent output file: ${errorMsg}`);
  }

  core.setOutput("output", JSON.stringify(validatedOutput));
  core.setOutput("raw_output", outputContent);

  // Write processed output to step summary using core.summary
  try {
    await core.summary
      .addRaw("## Processed Output\n\n")
      .addRaw("```json\n")
      .addRaw(JSON.stringify(validatedOutput))
      .addRaw("\n```\n")
      .write();
    core.info("Successfully wrote processed output to step summary");
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    core.warning(`Failed to write to step summary: ${errorMsg}`);
  }
}

(async () => {
  await collectNdjsonOutputMain();
})();