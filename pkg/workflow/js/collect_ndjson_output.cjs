async function main() {
  const fs = require("fs");

  /**
   * Sanitizes content for safe output in GitHub Actions
   * @param {string} content - The content to sanitize
   * @returns {string} The sanitized content
   */
  function sanitizeContent(content) {
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

    // Remove control characters (except newlines and tabs)
    sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

    // XML character escaping
    sanitized = sanitized
      .replace(/&/g, "&amp;") // Must be first to avoid double-escaping
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&apos;");

    // URI filtering - replace non-https protocols with "(redacted)"
    sanitized = sanitizeUrlProtocols(sanitized);

    // Domain filtering for HTTPS URIs
    sanitized = sanitizeUrlDomains(sanitized);

    // Limit total length to prevent DoS (0.5MB max)
    const maxLength = 524288;
    if (sanitized.length > maxLength) {
      sanitized =
        sanitized.substring(0, maxLength) +
        "\n[Content truncated due to length]";
    }

    // Limit number of lines to prevent log flooding (65k max)
    const lines = sanitized.split("\n");
    const maxLines = 65000;
    if (lines.length > maxLines) {
      sanitized =
        lines.slice(0, maxLines).join("\n") +
        "\n[Content truncated due to line count]";
    }

    // Remove ANSI escape sequences
    sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");

    // Neutralize common bot trigger phrases
    sanitized = neutralizeBotTriggers(sanitized);

    // Trim excessive whitespace
    return sanitized.trim();

    /**
     * Remove unknown domains
     * @param {string} s - The string to process
     * @returns {string} The string with unknown domains redacted
     */
    function sanitizeUrlDomains(s) {
      return s.replace(
        /\bhttps:\/\/([^\/\s\])}'"<>&\x00-\x1f]+)/gi,
        (match, domain) => {
          // Extract the hostname part (before first slash, colon, or other delimiter)
          const hostname = domain.split(/[\/:\?#]/)[0].toLowerCase();

          // Check if this domain or any parent domain is in the allowlist
          const isAllowed = allowedDomains.some(allowedDomain => {
            const normalizedAllowed = allowedDomain.toLowerCase();
            return (
              hostname === normalizedAllowed ||
              hostname.endsWith("." + normalizedAllowed)
            );
          });

          return isAllowed ? match : "(redacted)";
        }
      );
    }

    /**
     * Remove unknown protocols except https
     * @param {string} s - The string to process
     * @returns {string} The string with non-https protocols redacted
     */
    function sanitizeUrlProtocols(s) {
      // Match both protocol:// and protocol: patterns
      return s.replace(
        /\b(\w+):(?:\/\/)?[^\s\])}'"<>&\x00-\x1f]+/gi,
        (match, protocol) => {
          // Allow https (case insensitive), redact everything else
          return protocol.toLowerCase() === "https" ? match : "(redacted)";
        }
      );
    }

    /**
     * Neutralizes @mentions by wrapping them in backticks
     * @param {string} s - The string to process
     * @returns {string} The string with neutralized mentions
     */
    function neutralizeMentions(s) {
      // Replace @name or @org/team outside code with `@name`
      return s.replace(
        /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
        (_m, p1, p2) => `${p1}\`@${p2}\``
      );
    }

    /**
     * Neutralizes bot trigger phrases by wrapping them in backticks
     * @param {string} s - The string to process
     * @returns {string} The string with neutralized bot triggers
     */
    function neutralizeBotTriggers(s) {
      // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
      return s.replace(
        /\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi,
        (match, action, ref) => `\`${action} #${ref}\``
      );
    }
  }

  /**
   * Gets the maximum allowed count for a given output type
   * @param {string} itemType - The output item type
   * @param {Object} config - The safe-outputs configuration
   * @returns {number} The maximum allowed count
   */
  function getMaxAllowedForType(itemType, config) {
    // Check if max is explicitly specified in config
    if (
      config &&
      config[itemType] &&
      typeof config[itemType] === "object" &&
      config[itemType].max
    ) {
      return config[itemType].max;
    }

    // Use default limits for plural-supported types
    switch (itemType) {
      case "create-issue":
        return 1; // Only one issue allowed
      case "add-issue-comment":
        return 1; // Only one comment allowed
      case "create-pull-request":
        return 1; // Only one pull request allowed
      case "create-pull-request-review-comment":
        return 10; // Default to 10 review comments allowed
      case "add-issue-label":
        return 5; // Only one labels operation allowed
      case "update-issue":
        return 1; // Only one issue update allowed
      case "push-to-branch":
        return 1; // Only one push to branch allowed
      case "create-discussion":
        return 1; // Only one discussion allowed
      case "missing-tool":
        return 1000; // Allow many missing tool reports (default: unlimited)
      case "create-security-report":
        return 1000; // Allow many security reports (default: unlimited)
      default:
        return 1; // Default to single item for unknown types
    }
  }

  /**
   * Attempts to repair common JSON syntax issues in LLM-generated content
   * @param {string} jsonStr - The potentially malformed JSON string
   * @returns {string} The repaired JSON string
   */
  function repairJson(jsonStr) {
    let repaired = jsonStr.trim();

    // Fix single quotes to double quotes (must be done first)
    repaired = repaired.replace(/'/g, '"');

    // Fix missing quotes around object keys
    repaired = repaired.replace(
      /([{,]\s*)([a-zA-Z_$][a-zA-Z0-9_$]*)\s*:/g,
      '$1"$2":'
    );

    // Fix newlines and tabs inside strings by escaping them
    repaired = repaired.replace(/"([^"\\]*)"/g, (match, content) => {
      if (
        content.includes("\n") ||
        content.includes("\r") ||
        content.includes("\t")
      ) {
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
    repaired = repaired.replace(
      /(\[\s*(?:"[^"]*"(?:\s*,\s*"[^"]*")*\s*),?)\s*}/g,
      "$1]"
    );

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
   * Attempts to parse JSON with repair fallback
   * @param {string} jsonStr - The JSON string to parse
   * @returns {Object|undefined} The parsed JSON object, or undefined if parsing fails
   */
  function parseJsonWithRepair(jsonStr) {
    try {
      // First, try normal JSON.parse
      return JSON.parse(jsonStr);
    } catch (originalError) {
      try {
        // If that fails, try repairing and parsing again
        const repairedJson = repairJson(jsonStr);
        return JSON.parse(repairedJson);
      } catch (repairError) {
        // If repair also fails, print error to console and return undefined
        console.log(
          `JSON parsing failed. Original: ${originalError.message}. After repair: ${repairError.message}`
        );
        return undefined;
      }
    }
  }

  const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
  const safeOutputsConfig = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;

  if (!outputFile) {
    console.log("GITHUB_AW_SAFE_OUTPUTS not set, no output to collect");
    core.setOutput("output", "");
    return;
  }

  if (!fs.existsSync(outputFile)) {
    console.log("Output file does not exist:", outputFile);
    core.setOutput("output", "");
    return;
  }

  const outputContent = fs.readFileSync(outputFile, "utf8");
  if (outputContent.trim() === "") {
    console.log("Output file is empty");
    core.setOutput("output", "");
    return;
  }

  console.log("Raw output content length:", outputContent.length);

  // Parse the safe-outputs configuration
  let expectedOutputTypes = {};
  if (safeOutputsConfig) {
    try {
      expectedOutputTypes = JSON.parse(safeOutputsConfig);
      console.log("Expected output types:", Object.keys(expectedOutputTypes));
    } catch (error) {
      console.log(
        "Warning: Could not parse safe-outputs config:",
        error.message
      );
    }
  }

  // Parse JSONL content
  const lines = outputContent.trim().split("\n");
  const parsedItems = [];
  const errors = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === "") continue; // Skip empty lines
    try {
      const item = parseJsonWithRepair(line);

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
        errors.push(
          `Line ${i + 1}: Unexpected output type '${itemType}'. Expected one of: ${Object.keys(expectedOutputTypes).join(", ")}`
        );
        continue;
      }

      // Check for too many items of the same type
      const typeCount = parsedItems.filter(
        existing => existing.type === itemType
      ).length;
      const maxAllowed = getMaxAllowedForType(itemType, expectedOutputTypes);
      if (typeCount >= maxAllowed) {
        errors.push(
          `Line ${i + 1}: Too many items of type '${itemType}'. Maximum allowed: ${maxAllowed}.`
        );
        continue;
      }

      // Basic validation based on type
      switch (itemType) {
        case "create-issue":
          if (!item.title || typeof item.title !== "string") {
            errors.push(
              `Line ${i + 1}: create-issue requires a 'title' string field`
            );
            continue;
          }
          if (!item.body || typeof item.body !== "string") {
            errors.push(
              `Line ${i + 1}: create-issue requires a 'body' string field`
            );
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Sanitize labels if present
          if (item.labels && Array.isArray(item.labels)) {
            item.labels = item.labels.map(label =>
              typeof label === "string" ? sanitizeContent(label) : label
            );
          }
          break;

        case "add-issue-comment":
          if (!item.body || typeof item.body !== "string") {
            errors.push(
              `Line ${i + 1}: add-issue-comment requires a 'body' string field`
            );
            continue;
          }
          // Sanitize text content
          item.body = sanitizeContent(item.body);
          break;

        case "create-pull-request":
          if (!item.title || typeof item.title !== "string") {
            errors.push(
              `Line ${i + 1}: create-pull-request requires a 'title' string field`
            );
            continue;
          }
          if (!item.body || typeof item.body !== "string") {
            errors.push(
              `Line ${i + 1}: create-pull-request requires a 'body' string field`
            );
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Sanitize branch name if present
          if (item.branch && typeof item.branch === "string") {
            item.branch = sanitizeContent(item.branch);
          }
          // Sanitize labels if present
          if (item.labels && Array.isArray(item.labels)) {
            item.labels = item.labels.map(label =>
              typeof label === "string" ? sanitizeContent(label) : label
            );
          }
          break;

        case "add-issue-label":
          if (!item.labels || !Array.isArray(item.labels)) {
            errors.push(
              `Line ${i + 1}: add-issue-label requires a 'labels' array field`
            );
            continue;
          }
          if (item.labels.some(label => typeof label !== "string")) {
            errors.push(
              `Line ${i + 1}: add-issue-label labels array must contain only strings`
            );
            continue;
          }
          // Sanitize label strings
          item.labels = item.labels.map(label => sanitizeContent(label));
          break;

        case "update-issue":
          // Check that at least one updateable field is provided
          const hasValidField =
            item.status !== undefined ||
            item.title !== undefined ||
            item.body !== undefined;
          if (!hasValidField) {
            errors.push(
              `Line ${i + 1}: update-issue requires at least one of: 'status', 'title', or 'body' fields`
            );
            continue;
          }
          // Validate status if provided
          if (item.status !== undefined) {
            if (
              typeof item.status !== "string" ||
              (item.status !== "open" && item.status !== "closed")
            ) {
              errors.push(
                `Line ${i + 1}: update-issue 'status' must be 'open' or 'closed'`
              );
              continue;
            }
          }
          // Validate title if provided
          if (item.title !== undefined) {
            if (typeof item.title !== "string") {
              errors.push(
                `Line ${i + 1}: update-issue 'title' must be a string`
              );
              continue;
            }
            item.title = sanitizeContent(item.title);
          }
          // Validate body if provided
          if (item.body !== undefined) {
            if (typeof item.body !== "string") {
              errors.push(
                `Line ${i + 1}: update-issue 'body' must be a string`
              );
              continue;
            }
            item.body = sanitizeContent(item.body);
          }
          // Validate issue_number if provided (for target "*")
          if (item.issue_number !== undefined) {
            if (
              typeof item.issue_number !== "number" &&
              typeof item.issue_number !== "string"
            ) {
              errors.push(
                `Line ${i + 1}: update-issue 'issue_number' must be a number or string`
              );
              continue;
            }
          }
          break;

        case "push-to-branch":
          // Validate message if provided (optional)
          if (item.message !== undefined) {
            if (typeof item.message !== "string") {
              errors.push(
                `Line ${i + 1}: push-to-branch 'message' must be a string`
              );
              continue;
            }
            item.message = sanitizeContent(item.message);
          }
          // Validate pull_request_number if provided (for target "*")
          if (item.pull_request_number !== undefined) {
            if (
              typeof item.pull_request_number !== "number" &&
              typeof item.pull_request_number !== "string"
            ) {
              errors.push(
                `Line ${i + 1}: push-to-branch 'pull_request_number' must be a number or string`
              );
              continue;
            }
          }
          break;
        case "create-pull-request-review-comment":
          // Validate required path field
          if (!item.path || typeof item.path !== "string") {
            errors.push(
              `Line ${i + 1}: create-pull-request-review-comment requires a 'path' string field`
            );
            continue;
          }
          // Validate required line field
          if (
            item.line === undefined ||
            (typeof item.line !== "number" && typeof item.line !== "string")
          ) {
            errors.push(
              `Line ${i + 1}: create-pull-request-review-comment requires a 'line' number or string field`
            );
            continue;
          }
          // Validate line is a positive integer
          const lineNumber =
            typeof item.line === "string" ? parseInt(item.line, 10) : item.line;
          if (
            isNaN(lineNumber) ||
            lineNumber <= 0 ||
            !Number.isInteger(lineNumber)
          ) {
            errors.push(
              `Line ${i + 1}: create-pull-request-review-comment 'line' must be a positive integer`
            );
            continue;
          }
          // Validate required body field
          if (!item.body || typeof item.body !== "string") {
            errors.push(
              `Line ${i + 1}: create-pull-request-review-comment requires a 'body' string field`
            );
            continue;
          }
          // Sanitize required text content
          item.body = sanitizeContent(item.body);
          // Validate optional start_line field
          if (item.start_line !== undefined) {
            if (
              typeof item.start_line !== "number" &&
              typeof item.start_line !== "string"
            ) {
              errors.push(
                `Line ${i + 1}: create-pull-request-review-comment 'start_line' must be a number or string`
              );
              continue;
            }
            const startLineNumber =
              typeof item.start_line === "string"
                ? parseInt(item.start_line, 10)
                : item.start_line;
            if (
              isNaN(startLineNumber) ||
              startLineNumber <= 0 ||
              !Number.isInteger(startLineNumber)
            ) {
              errors.push(
                `Line ${i + 1}: create-pull-request-review-comment 'start_line' must be a positive integer`
              );
              continue;
            }
            if (startLineNumber > lineNumber) {
              errors.push(
                `Line ${i + 1}: create-pull-request-review-comment 'start_line' must be less than or equal to 'line'`
              );
              continue;
            }
          }
          // Validate optional side field
          if (item.side !== undefined) {
            if (
              typeof item.side !== "string" ||
              (item.side !== "LEFT" && item.side !== "RIGHT")
            ) {
              errors.push(
                `Line ${i + 1}: create-pull-request-review-comment 'side' must be 'LEFT' or 'RIGHT'`
              );
              continue;
            }
          }
          break;
        case "create-discussion":
          if (!item.title || typeof item.title !== "string") {
            errors.push(
              `Line ${i + 1}: create-discussion requires a 'title' string field`
            );
            continue;
          }
          if (!item.body || typeof item.body !== "string") {
            errors.push(
              `Line ${i + 1}: create-discussion requires a 'body' string field`
            );
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          break;

        case "missing-tool":
          // Validate required tool field
          if (!item.tool || typeof item.tool !== "string") {
            errors.push(
              `Line ${i + 1}: missing-tool requires a 'tool' string field`
            );
            continue;
          }
          // Validate required reason field
          if (!item.reason || typeof item.reason !== "string") {
            errors.push(
              `Line ${i + 1}: missing-tool requires a 'reason' string field`
            );
            continue;
          }
          // Sanitize text content
          item.tool = sanitizeContent(item.tool);
          item.reason = sanitizeContent(item.reason);
          // Validate optional alternatives field
          if (item.alternatives !== undefined) {
            if (typeof item.alternatives !== "string") {
              errors.push(
                `Line ${i + 1}: missing-tool 'alternatives' must be a string`
              );
              continue;
            }
            item.alternatives = sanitizeContent(item.alternatives);
          }
          break;

        case "create-security-report":
          // Validate required sarif field
          if (!item.sarif) {
            errors.push(
              `Line ${i + 1}: create-security-report requires a 'sarif' field`
            );
            continue;
          }
          // SARIF content can be object or string
          if (
            typeof item.sarif !== "object" &&
            typeof item.sarif !== "string"
          ) {
            errors.push(
              `Line ${i + 1}: create-security-report 'sarif' must be an object or string`
            );
            continue;
          }
          // If SARIF is a string, sanitize it
          if (typeof item.sarif === "string") {
            item.sarif = sanitizeContent(item.sarif);
          }
          // Validate optional category field
          if (item.category !== undefined) {
            if (typeof item.category !== "string") {
              errors.push(
                `Line ${i + 1}: create-security-report 'category' must be a string`
              );
              continue;
            }
            item.category = sanitizeContent(item.category);
          }
          break;

        default:
          errors.push(`Line ${i + 1}: Unknown output type '${itemType}'`);
          continue;
      }

      console.log(`Line ${i + 1}: Valid ${itemType} item`);
      parsedItems.push(item);
    } catch (error) {
      errors.push(`Line ${i + 1}: Invalid JSON - ${error.message}`);
    }
  }

  // Report validation results
  if (errors.length > 0) {
    console.log("Validation errors found:");
    errors.forEach(error => console.log(`  - ${error}`));

    // For now, we'll continue with valid items but log the errors
    // In the future, we might want to fail the workflow for invalid items
  }

  console.log(`Successfully parsed ${parsedItems.length} valid output items`);

  // Set the parsed and validated items as output
  const validatedOutput = {
    items: parsedItems,
    errors: errors,
  };

  core.setOutput("output", JSON.stringify(validatedOutput));
  core.setOutput("raw_output", outputContent);
}

// Call the main function
await main();
