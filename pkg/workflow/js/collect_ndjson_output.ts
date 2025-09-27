import type { SafeOutputItem, SafeOutputItems } from "./types/safe-outputs";
import type {
  SafeOutputConfigs,
  SafeOutputConfig,
  SpecificSafeOutputConfig,
  CreateIssueConfig,
  CreateDiscussionConfig,
  AddCommentConfig,
  CreatePullRequestConfig,
  CreatePullRequestReviewCommentConfig,
  CreateCodeScanningAlertConfig,
  AddLabelsConfig,
  UpdateIssueConfig,
  PushToPullRequestBranchConfig,
  UploadAssetConfig,
  EditWikiConfig,
  MissingToolConfig,
  SafeJobInput,
  SafeJobConfig,
} from "./types/safe-outputs-config";

async function main() {
  const fs = require("fs");

  /**
   * Sanitizes content for safe output in GitHub Actions
   * @param {string} content - The content to sanitize
   * @returns {string} The sanitized content
   */
  function sanitizeContent(content: string) {
    if (!content || typeof content !== "string") {
      return "";
    }

    // Read allowed domains from environment variable
    const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
    const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];

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
     * @param {string} s - The string to process
     * @returns {string} The string with unknown domains redacted
     */
    function sanitizeUrlDomains(s: string) {
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
     * @param {string} s - The string to process
     * @returns {string} The string with non-https protocols redacted
     */
    function sanitizeUrlProtocols(s: string) {
      // Match protocol:// patterns (URLs) and standalone protocol: patterns that look like URLs
      // Avoid matching command line flags like -v:10 or z3 -memory:high
      return s.replace(/\b(\w+):\/\/[^\s\])}'"<>&\x00-\x1f]+/gi, (match, protocol) => {
        // Allow https (case insensitive), redact everything else
        return protocol.toLowerCase() === "https" ? match : "(redacted)";
      });
    }

    /**
     * Neutralizes @mentions by wrapping them in backticks
     * @param {string} s - The string to process
     * @returns {string} The string with neutralized mentions
     */
    function neutralizeMentions(s: string) {
      // Replace @name or @org/team outside code with `@name`
      return s.replace(
        /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
        (_m, p1, p2) => `${p1}\`@${p2}\``
      );
    }

    /**
     * Removes XML comments to prevent content hiding
     * @param {string} s - The string to process
     * @returns {string} The string with XML comments removed
     */
    function removeXmlComments(s: string) {
      // Remove XML/HTML comments including malformed ones that might be used to hide content
      // Matches: <!-- ... --> and <!--- ... --> and <!--- ... --!> variations
      return s.replace(/<!--[\s\S]*?-->/g, "").replace(/<!--[\s\S]*?--!>/g, "");
    }

    /**
     * Neutralizes bot trigger phrases by wrapping them in backticks
     * @param {string} s - The string to process
     * @returns {string} The string with neutralized bot triggers
     */
    function neutralizeBotTriggers(s: string) {
      // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
      return s.replace(/\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi, (match, action, ref) => `\`${action} #${ref}\``);
    }
  }

  /**
   * Gets the maximum allowed count for a given output type
   * @param {string} itemType - The output item type
   * @param {SafeOutputConfigs} config - The safe-outputs configuration
   * @returns {number} The maximum allowed count
   */
  function getMaxAllowedForType(itemType: string, config: SafeOutputConfigs): number {
    // Check if max is explicitly specified in config
    const itemConfig = config?.[itemType];
    if (itemConfig && typeof itemConfig === "object" && "max" in itemConfig && itemConfig.max) {
      return itemConfig.max;
    }

    // Use default limits for plural-supported types
    switch (itemType) {
      case "create-issue":
        return 1; // Only one issue allowed
      case "add-comment":
        return 1; // Only one comment allowed
      case "create-pull-request":
        return 1; // Only one pull request allowed
      case "create-pull-request-review-comment":
        return 1; // Default to 1 review comment allowed
      case "add-labels":
        return 5; // Only one labels operation allowed
      case "update-issue":
        return 1; // Only one issue update allowed
      case "push-to-pull-request-branch":
        return 1; // Only one push to branch allowed
      case "create-discussion":
        return 1; // Only one discussion allowed
      case "missing-tool":
        return 1000; // Allow many missing tool reports (default: unlimited)
      case "create-code-scanning-alert":
        return 1000; // Allow many repository security advisories (default: unlimited)
      case "upload-asset":
        return 10; // Default to 10 assets allowed
      case "edit-wiki":
        return 1; // Only one wiki edit operation allowed by default
      default:
        return 1; // Default to single item for unknown types
    }
  }

  /**
   * Attempts to repair common JSON syntax issues in LLM-generated content
   * @param {string} jsonStr - The potentially malformed JSON string
   * @returns {string} The repaired JSON string
   */
  function repairJson(jsonStr: string): string {
    let repaired = jsonStr.trim();

    // remove invalid control characters like
    // U+0014 (DC4) — represented here as "\u0014"
    // Escape control characters not allowed in JSON strings (U+0000 through U+001F)
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
        const escaped = content.replace(/\\/g, "\\\\").replace(/\n/g, "\\n").replace(/\r/g, "\\r").replace(/\t/g, "\\t");
        return `"${escaped}"`;
      }
      return match;
    });

    // Fix unescaped quotes inside string values
    repaired = repaired.replace(/"([^"]*)"([^":,}\]]*)"([^"]*)"(\s*[,:}\]])/g, (match, p1, p2, p3, p4) => `"${p1}\\"${p2}\\"${p3}"${p4}`);

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
   * @param {any} value - The value to validate
   * @param {string} fieldName - The name of the field being validated
   * @param {number} lineNum - The line number for error reporting
   * @returns {{isValid: boolean, error?: string, normalizedValue?: number}} Validation result
   */
  function validatePositiveInteger(value: unknown, fieldName: string, lineNum: number) {
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
          error: `Line ${lineNum}: create-pull-request-review-comment requires a 'line' number or string field`,
        };
      }
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} must be a number or string`,
      };
    }

    const parsed = typeof value === "string" ? parseInt(value, 10) : value;
    if (isNaN(parsed) || parsed <= 0 || !Number.isInteger(parsed)) {
      // Match the original error format for different field types
      if (fieldName.includes("create-code-scanning-alert 'line'")) {
        return {
          isValid: false,
          error: `Line ${lineNum}: create-code-scanning-alert 'line' must be a valid positive integer (got: ${value})`,
        };
      }
      if (fieldName.includes("create-pull-request-review-comment 'line'")) {
        return {
          isValid: false,
          error: `Line ${lineNum}: create-pull-request-review-comment 'line' must be a positive integer`,
        };
      }
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} must be a positive integer (got: ${value})`,
      };
    }

    return { isValid: true, normalizedValue: parsed };
  }

  /**
   * Validates an optional positive integer field
   * @param {any} value - The value to validate
   * @param {string} fieldName - The name of the field being validated
   * @param {number} lineNum - The line number for error reporting
   * @returns {{isValid: boolean, error?: string, normalizedValue?: number}} Validation result
   */
  function validateOptionalPositiveInteger(value: unknown, fieldName: string, lineNum: number) {
    if (value === undefined) {
      return { isValid: true };
    }

    if (typeof value !== "number" && typeof value !== "string") {
      // Match the original error format for specific field types
      if (fieldName.includes("create-pull-request-review-comment 'start_line'")) {
        return {
          isValid: false,
          error: `Line ${lineNum}: create-pull-request-review-comment 'start_line' must be a number or string`,
        };
      }
      if (fieldName.includes("create-code-scanning-alert 'column'")) {
        return {
          isValid: false,
          error: `Line ${lineNum}: create-code-scanning-alert 'column' must be a number or string`,
        };
      }
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} must be a number or string`,
      };
    }

    const parsed = typeof value === "string" ? parseInt(value, 10) : value;
    if (isNaN(parsed) || parsed <= 0 || !Number.isInteger(parsed)) {
      // Match the original error format for different field types
      if (fieldName.includes("create-pull-request-review-comment 'start_line'")) {
        return {
          isValid: false,
          error: `Line ${lineNum}: create-pull-request-review-comment 'start_line' must be a positive integer`,
        };
      }
      if (fieldName.includes("create-code-scanning-alert 'column'")) {
        return {
          isValid: false,
          error: `Line ${lineNum}: create-code-scanning-alert 'column' must be a valid positive integer (got: ${value})`,
        };
      }
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} must be a positive integer (got: ${value})`,
      };
    }

    return { isValid: true, normalizedValue: parsed };
  }

  /**
   * Validates an issue or pull request number (optional field)
   * @param {any} value - The value to validate
   * @param {string} fieldName - The name of the field being validated
   * @param {number} lineNum - The line number for error reporting
   * @returns {{isValid: boolean, error?: string}} Validation result
   */
  function validateIssueOrPRNumber(value: unknown, fieldName: string, lineNum: number) {
    if (value === undefined) {
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

  /**
   * Validates and sanitizes a field value based on SafeJobInput schema
   * @param {any} value - The value to validate
   * @param {string} fieldName - The name of the field
   * @param {SafeJobInput} inputSchema - The input schema to validate against
   * @param {number} lineNum - The line number for error reporting
   * @returns {{isValid: boolean, error?: string, normalizedValue?: any}} Validation result
   */
  function validateFieldWithInputSchema(value: any, fieldName: string, inputSchema: SafeJobInput, lineNum: number) {
    // If field is required and value is missing
    if (inputSchema.required && (value === undefined || value === null)) {
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} is required`,
      };
    }

    // If value is undefined and field is not required, use default or return valid
    if (value === undefined || value === null) {
      return {
        isValid: true,
        normalizedValue: inputSchema.default || undefined,
      };
    }

    // Validate type
    const inputType = inputSchema.type || "string";
    let normalizedValue = value;

    switch (inputType) {
      case "string":
        if (typeof value !== "string") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a string`,
          };
        }
        // Apply sanitization for string types
        normalizedValue = sanitizeContent(value);
        break;

      case "boolean":
        if (typeof value !== "boolean") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a boolean`,
          };
        }
        break;

      case "number":
        if (typeof value !== "number") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a number`,
          };
        }
        break;

      case "choice":
        if (typeof value !== "string") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a string for choice type`,
          };
        }
        if (inputSchema.options && !inputSchema.options.includes(value)) {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be one of: ${inputSchema.options.join(", ")}`,
          };
        }
        // Apply sanitization for string-based choice types
        normalizedValue = sanitizeContent(value);
        break;

      default:
        // For unknown types, treat as string and sanitize
        if (typeof value === "string") {
          normalizedValue = sanitizeContent(value);
        }
        break;
    }

    return {
      isValid: true,
      normalizedValue,
    };
  }

  /**
   * Validates an item using SafeJobConfig inputs schema
   * @param {any} item - The item to validate
   * @param {SafeJobConfig} jobConfig - The safe job configuration
   * @param {number} lineNum - The line number for error reporting
   * @returns {{isValid: boolean, errors: string[], normalizedItem: any}} Validation result
   */
  function validateItemWithSafeJobConfig(item: any, jobConfig: SafeJobConfig, lineNum: number) {
    const errors: string[] = [];
    const normalizedItem = { ...item };

    if (!jobConfig.inputs) {
      // No input schema defined, return item as-is
      return {
        isValid: true,
        errors: [],
        normalizedItem: item,
      };
    }

    // Validate each field defined in the inputs schema
    for (const [fieldName, inputSchema] of Object.entries(jobConfig.inputs)) {
      const fieldValue = item[fieldName];
      const validation = validateFieldWithInputSchema(fieldValue, fieldName, inputSchema, lineNum);

      if (!validation.isValid && validation.error) {
        errors.push(validation.error);
      } else if (validation.normalizedValue !== undefined) {
        normalizedItem[fieldName] = validation.normalizedValue;
      }
    }

    return {
      isValid: errors.length === 0,
      errors,
      normalizedItem,
    };
  }

  /**
   * Attempts to parse JSON with repair fallback
   * @param {string} jsonStr - The JSON string to parse
   * @returns {any|undefined} The parsed JSON object, or undefined if parsing fails
   */
  function parseJsonWithRepair(jsonStr: string): any | undefined {
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
  let expectedOutputTypes: SafeOutputConfigs = {};
  if (safeOutputsConfig) {
    try {
      expectedOutputTypes = JSON.parse(safeOutputsConfig) as SafeOutputConfigs;
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
      const item = parseJsonWithRepair(line) as any;

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

      // Check for too many items of the same type
      const typeCount = parsedItems.filter(existing => existing.type === itemType).length;
      const maxAllowed = getMaxAllowedForType(itemType, expectedOutputTypes);
      if (typeCount >= maxAllowed) {
        errors.push(`Line ${i + 1}: Too many items of type '${itemType}'. Maximum allowed: ${maxAllowed}.`);
        continue;
      }

      core.info(`Line ${i + 1}: type '${itemType}'`);
      // Basic validation based on type
      switch (itemType) {
        case "create-issue":
          if (!item.title || typeof item.title !== "string") {
            errors.push(`Line ${i + 1}: create_issue requires a 'title' string field`);
            continue;
          }
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create_issue requires a 'body' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Sanitize labels if present
          if (item.labels && Array.isArray(item.labels)) {
            item.labels = item.labels.map((label: any) => (typeof label === "string" ? sanitizeContent(label) : label));
          }
          break;

        case "add-comment":
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: add_comment requires a 'body' string field`);
            continue;
          }
          // Validate optional issue_number field
          const issueNumValidation = validateIssueOrPRNumber(item.issue_number, "add_comment 'issue_number'", i + 1);
          if (!issueNumValidation.isValid) {
            if (issueNumValidation.error) errors.push(issueNumValidation.error);
            continue;
          }
          // Sanitize text content
          item.body = sanitizeContent(item.body);
          break;

        case "create-pull-request":
          if (!item.title || typeof item.title !== "string") {
            errors.push(`Line ${i + 1}: create_pull_request requires a 'title' string field`);
            continue;
          }
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create_pull_request requires a 'body' string field`);
            continue;
          }
          if (!item.branch || typeof item.branch !== "string") {
            errors.push(`Line ${i + 1}: create_pull_request requires a 'branch' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          item.branch = sanitizeContent(item.branch);
          // Sanitize labels if present
          if (item.labels && Array.isArray(item.labels)) {
            item.labels = item.labels.map((label: any) => (typeof label === "string" ? sanitizeContent(label) : label));
          }
          break;

        case "add-labels":
          if (!item.labels || !Array.isArray(item.labels)) {
            errors.push(`Line ${i + 1}: add_labels requires a 'labels' array field`);
            continue;
          }
          if (item.labels.some((label: any) => typeof label !== "string")) {
            errors.push(`Line ${i + 1}: add_labels labels array must contain only strings`);
            continue;
          }
          // Validate optional issue_number field
          const labelsIssueNumValidation = validateIssueOrPRNumber(item.issue_number, "add-labels 'issue_number'", i + 1);
          if (!labelsIssueNumValidation.isValid) {
            if (labelsIssueNumValidation.error) errors.push(labelsIssueNumValidation.error);
            continue;
          }
          // Sanitize label strings
          item.labels = item.labels.map((label: any) => sanitizeContent(label));
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
          const updateIssueNumValidation = validateIssueOrPRNumber(item.issue_number, "update-issue 'issue_number'", i + 1);
          if (!updateIssueNumValidation.isValid) {
            if (updateIssueNumValidation.error) errors.push(updateIssueNumValidation.error);
            continue;
          }
          break;

        case "push-to-pull-request-branch":
          // Validate required branch field
          if (!item.branch || typeof item.branch !== "string") {
            errors.push(`Line ${i + 1}: push_to_pull_request_branch requires a 'branch' string field`);
            continue;
          }
          // Validate required message field
          if (!item.message || typeof item.message !== "string") {
            errors.push(`Line ${i + 1}: push_to_pull_request_branch requires a 'message' string field`);
            continue;
          }
          // Sanitize text content
          item.branch = sanitizeContent(item.branch);
          item.message = sanitizeContent(item.message);
          // Validate pull_request_number if provided (for target "*")
          const pushPRNumValidation = validateIssueOrPRNumber(
            item.pull_request_number,
            "push-to-pull-request-branch 'pull_request_number'",
            i + 1
          );
          if (!pushPRNumValidation.isValid) {
            if (pushPRNumValidation.error) errors.push(pushPRNumValidation.error);
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
            if (lineValidation.error) errors.push(lineValidation.error);
            continue;
          }
          // lineValidation.normalizedValue is guaranteed to be defined when isValid is true
          const lineNumber = lineValidation.normalizedValue;
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
            if (startLineValidation.error) errors.push(startLineValidation.error);
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
        case "create-discussion":
          if (!item.title || typeof item.title !== "string") {
            errors.push(`Line ${i + 1}: create_discussion requires a 'title' string field`);
            continue;
          }
          if (!item.body || typeof item.body !== "string") {
            errors.push(`Line ${i + 1}: create_discussion requires a 'body' string field`);
            continue;
          }
          // Validate optional category field
          if (item.category !== undefined) {
            if (typeof item.category !== "string") {
              errors.push(`Line ${i + 1}: create_discussion 'category' must be a string`);
              continue;
            }
            item.category = sanitizeContent(item.category);
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          break;

        case "missing-tool":
          // Validate required tool field
          if (!item.tool || typeof item.tool !== "string") {
            errors.push(`Line ${i + 1}: missing_tool requires a 'tool' string field`);
            continue;
          }
          // Validate required reason field
          if (!item.reason || typeof item.reason !== "string") {
            errors.push(`Line ${i + 1}: missing_tool requires a 'reason' string field`);
            continue;
          }
          // Sanitize text content
          item.tool = sanitizeContent(item.tool);
          item.reason = sanitizeContent(item.reason);
          // Validate optional alternatives field
          if (item.alternatives !== undefined) {
            if (typeof item.alternatives !== "string") {
              errors.push(`Line ${i + 1}: missing-tool 'alternatives' must be a string`);
              continue;
            }
            item.alternatives = sanitizeContent(item.alternatives);
          }
          break;

        case "upload-asset":
          if (!item.path || typeof item.path !== "string") {
            errors.push(`Line ${i + 1}: upload_asset requires a 'path' string field`);
            continue;
          }
          break;

        case "edit-wiki":
          // Validate required path field
          if (!item.path || typeof item.path !== "string") {
            errors.push(`Line ${i + 1}: ${itemType} requires a 'path' string field`);
            continue;
          }
          // Validate required content field
          if (!item.content || typeof item.content !== "string") {
            errors.push(`Line ${i + 1}: ${itemType} requires a 'content' string field`);
            continue;
          }
          // Validate optional message field (defaults to generated message if not provided)
          if (item.message !== undefined && typeof item.message !== "string") {
            errors.push(`Line ${i + 1}: ${itemType} 'message' must be a string if provided`);
            continue;
          }
          // Sanitize text content
          item.path = sanitizeContent(item.path);
          item.content = sanitizeContent(item.content);
          if (item.message) {
            item.message = sanitizeContent(item.message);
          }
          break;

        case "create-code-scanning-alert":
          // Validate required fields
          if (!item.file || typeof item.file !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'file' field (string)`);
            continue;
          }
          const alertLineValidation = validatePositiveInteger(item.line, "create-code-scanning-alert 'line'", i + 1);
          if (!alertLineValidation.isValid) {
            if (alertLineValidation.error) {
              errors.push(alertLineValidation.error);
            }
            continue;
          }
          if (!item.severity || typeof item.severity !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'severity' field (string)`);
            continue;
          }
          if (!item.message || typeof item.message !== "string") {
            errors.push(`Line ${i + 1}: create-code-scanning-alert requires a 'message' field (string)`);
            continue;
          }

          // Validate severity level
          const allowedSeverities = ["error", "warning", "info", "note"];
          if (!allowedSeverities.includes(item.severity.toLowerCase())) {
            errors.push(
              `Line ${i + 1}: create-code-scanning-alert 'severity' must be one of: ${allowedSeverities.join(", ")}, got ${item.severity.toLowerCase()}`
            );
            continue;
          }

          // Validate optional column field
          const columnValidation = validateOptionalPositiveInteger(item.column, "create-code-scanning-alert 'column'", i + 1);
          if (!columnValidation.isValid) {
            if (columnValidation.error) errors.push(columnValidation.error);
            continue;
          }

          // Validate optional ruleIdSuffix field
          if (item.ruleIdSuffix !== undefined) {
            if (typeof item.ruleIdSuffix !== "string") {
              errors.push(`Line ${i + 1}: create-code-scanning-alert 'ruleIdSuffix' must be a string`);
              continue;
            }
            if (!/^[a-zA-Z0-9_-]+$/.test(item.ruleIdSuffix.trim())) {
              errors.push(
                `Line ${i + 1}: create-code-scanning-alert 'ruleIdSuffix' must contain only alphanumeric characters, hyphens, and underscores`
              );
              continue;
            }
          }

          // Normalize severity to lowercase and sanitize string fields
          item.severity = item.severity.toLowerCase();
          item.file = sanitizeContent(item.file);
          item.severity = sanitizeContent(item.severity);
          item.message = sanitizeContent(item.message);
          if (item.ruleIdSuffix) {
            item.ruleIdSuffix = sanitizeContent(item.ruleIdSuffix);
          }
          break;

        default:
          const jobOutputType = expectedOutputTypes[itemType];
          if (!jobOutputType) {
            errors.push(`Line ${i + 1}: Unknown output type '${itemType}'`);
            continue;
          }

          // Check if this is a SafeJobConfig with inputs schema
          const safeJobConfig = jobOutputType as SafeJobConfig;
          if (safeJobConfig && safeJobConfig.inputs) {
            // Use SafeJobConfig inputs schema to validate and sanitize fields
            const validation = validateItemWithSafeJobConfig(item, safeJobConfig, i + 1);

            if (!validation.isValid) {
              errors.push(...validation.errors);
              continue;
            }

            // Update item with normalized/sanitized values
            Object.assign(item, validation.normalizedItem);
          }
          break;
      }

      core.info(`Line ${i + 1}: Valid ${itemType} item`);
      parsedItems.push(item);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      errors.push(`Line ${i + 1}: Invalid JSON - ${errorMsg}`);
    }
  }

  // Report validation results
  if (errors.length > 0) {
    core.warning("Validation errors found:");
    errors.forEach(error => core.warning(`  - ${error}`));

    if (parsedItems.length === 0) {
      core.setFailed(errors.map(e => `  - ${e}`).join("\n"));
      return;
    }

    // For now, we'll continue with valid items but log the errors
    // In the future, we might want to fail the workflow for invalid items
  }

  core.info(`Successfully parsed ${parsedItems.length} valid output items`);

  // Set the parsed and validated items as output
  const validatedOutput = {
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

// Call the main function
await main();
