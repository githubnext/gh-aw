async function main() {
  const fs = require("fs");
  
  /**
   * Sanitizes content for safe output in GitHub Actions
   * @param {string} content - The content to sanitize
   * @returns {string} The sanitized content
   */
  function sanitizeContent(content) {
    if (!content || typeof content !== 'string') {
      return '';
    }

    // Read allowed domains from environment variable
    const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
    const defaultAllowedDomains = [
      'github.com',
      'github.io',
      'githubusercontent.com',
      'githubassets.com',
      'github.dev',
      'codespaces.new'
    ];

    const allowedDomains = allowedDomainsEnv
      ? allowedDomainsEnv.split(',').map(d => d.trim()).filter(d => d)
      : defaultAllowedDomains;

    let sanitized = content;

    // Neutralize @mentions to prevent unintended notifications
    sanitized = neutralizeMentions(sanitized);

    // Remove control characters (except newlines and tabs)
    sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '');

    // XML character escaping
    sanitized = sanitized
      .replace(/&/g, '&amp;')   // Must be first to avoid double-escaping
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&apos;');

    // URI filtering - replace non-https protocols with "(redacted)"
    sanitized = sanitizeUrlProtocols(sanitized);

    // Domain filtering for HTTPS URIs
    sanitized = sanitizeUrlDomains(sanitized);

    // Limit total length to prevent DoS (0.5MB max)
    const maxLength = 524288;
    if (sanitized.length > maxLength) {
      sanitized = sanitized.substring(0, maxLength) + '\n[Content truncated due to length]';
    }

    // Limit number of lines to prevent log flooding (65k max)
    const lines = sanitized.split('\n');
    const maxLines = 65000;
    if (lines.length > maxLines) {
      sanitized = lines.slice(0, maxLines).join('\n') + '\n[Content truncated due to line count]';
    }

    // Remove ANSI escape sequences
    sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, '');

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
      return s.replace(/\bhttps:\/\/([^\/\s\])}'"<>&\x00-\x1f]+)/gi, (match, domain) => {
        // Extract the hostname part (before first slash, colon, or other delimiter)
        const hostname = domain.split(/[\/:\?#]/)[0].toLowerCase();

        // Check if this domain or any parent domain is in the allowlist
        const isAllowed = allowedDomains.some(allowedDomain => {
          const normalizedAllowed = allowedDomain.toLowerCase();
          return hostname === normalizedAllowed || hostname.endsWith('.' + normalizedAllowed);
        });

        return isAllowed ? match : '(redacted)';
      });
    }

    /**
     * Remove unknown protocols except https
     * @param {string} s - The string to process
     * @returns {string} The string with non-https protocols redacted
     */
    function sanitizeUrlProtocols(s) {
      // Match both protocol:// and protocol: patterns
      return s.replace(/\b(\w+):(?:\/\/)?[^\s\])}'"<>&\x00-\x1f]+/gi, (match, protocol) => {
        // Allow https (case insensitive), redact everything else
        return protocol.toLowerCase() === 'https' ? match : '(redacted)';
      });
    }

    /**
     * Neutralizes @mentions by wrapping them in backticks
     * @param {string} s - The string to process
     * @returns {string} The string with neutralized mentions
     */
    function neutralizeMentions(s) {
      // Replace @name or @org/team outside code with `@name`
      return s.replace(/(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
        (_m, p1, p2) => `${p1}\`@${p2}\``);
    }

    /**
     * Neutralizes bot trigger phrases by wrapping them in backticks
     * @param {string} s - The string to process
     * @returns {string} The string with neutralized bot triggers
     */
    function neutralizeBotTriggers(s) {
      // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
      return s.replace(/\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi,
        (match, action, ref) => `\`${action} #${ref}\``);
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
    if (config && config[itemType] && typeof config[itemType] === 'object' && config[itemType].max) {
      return config[itemType].max;
    }
    
    // Use default limits for plural-supported types
    switch (itemType) {
      case 'create-issue':
        return 1; // Only one issue allowed
      case 'add-issue-comment':
        return 1; // Only one comment allowed
      case 'create-pull-request':
        return 1;  // Only one pull request allowed
      case 'add-issue-label':
        return 5;  // Only one labels operation allowed
      default:
        return 1;  // Default to single item for unknown types
    }
  }
  const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
  const safeOutputsConfig = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
  
  if (!outputFile) {
    console.log('GITHUB_AW_SAFE_OUTPUTS not set, no output to collect');
    core.setOutput('output', '');
    return;
  }

  if (!fs.existsSync(outputFile)) {
    console.log('Output file does not exist:', outputFile);
    core.setOutput('output', '');
    return;
  }

  const outputContent = fs.readFileSync(outputFile, 'utf8');
  if (outputContent.trim() === '') {
    console.log('Output file is empty');
    core.setOutput('output', '');
    return;
  }

  console.log('Raw output content length:', outputContent.length);

  // Parse the safe-outputs configuration
  let expectedOutputTypes = {};
  if (safeOutputsConfig) {
    try {
      expectedOutputTypes = JSON.parse(safeOutputsConfig);
      console.log('Expected output types:', Object.keys(expectedOutputTypes));
    } catch (error) {
      console.log('Warning: Could not parse safe-outputs config:', error.message);
    }
  }

  // Parse JSONL content
  const lines = outputContent.trim().split('\n');
  const parsedItems = [];
  const errors = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === '') continue; // Skip empty lines

    try {
      const item = JSON.parse(line);
      
      // Validate that the item has a 'type' field
      if (!item.type) {
        errors.push(`Line ${i + 1}: Missing required 'type' field`);
        continue;
      }

      // Validate against expected output types
      const itemType = item.type;
      if (!expectedOutputTypes[itemType]) {
        errors.push(`Line ${i + 1}: Unexpected output type '${itemType}'. Expected one of: ${Object.keys(expectedOutputTypes).join(', ')}`);
        continue;
      }

      // Check for too many items of the same type
      const typeCount = parsedItems.filter(existing => existing.type === itemType).length;
      const maxAllowed = getMaxAllowedForType(itemType, expectedOutputTypes);
      if (typeCount >= maxAllowed) {
        errors.push(`Line ${i + 1}: Too many items of type '${itemType}'. Maximum allowed: ${maxAllowed}.`);
        continue;
      }

      // Basic validation based on type
      switch (itemType) {
        case 'create-issue':
          if (!item.title || typeof item.title !== 'string') {
            errors.push(`Line ${i + 1}: create-issue requires a 'title' string field`);
            continue;
          }
          if (!item.body || typeof item.body !== 'string') {
            errors.push(`Line ${i + 1}: create-issue requires a 'body' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Sanitize labels if present
          if (item.labels && Array.isArray(item.labels)) {
            item.labels = item.labels.map(label => typeof label === 'string' ? sanitizeContent(label) : label);
          }
          break;

        case 'add-issue-comment':
          if (!item.body || typeof item.body !== 'string') {
            errors.push(`Line ${i + 1}: add-issue-comment requires a 'body' string field`);
            continue;
          }
          // Sanitize text content
          item.body = sanitizeContent(item.body);
          break;

        case 'create-pull-request':
          if (!item.title || typeof item.title !== 'string') {
            errors.push(`Line ${i + 1}: create-pull-request requires a 'title' string field`);
            continue;
          }
          if (!item.body || typeof item.body !== 'string') {
            errors.push(`Line ${i + 1}: create-pull-request requires a 'body' string field`);
            continue;
          }
          // Sanitize text content
          item.title = sanitizeContent(item.title);
          item.body = sanitizeContent(item.body);
          // Sanitize labels if present
          if (item.labels && Array.isArray(item.labels)) {
            item.labels = item.labels.map(label => typeof label === 'string' ? sanitizeContent(label) : label);
          }
          break;

        case 'add-issue-label':
          if (!item.labels || !Array.isArray(item.labels)) {
            errors.push(`Line ${i + 1}: add-issue-label requires a 'labels' array field`);
            continue;
          }
          if (item.labels.some(label => typeof label !== 'string')) {
            errors.push(`Line ${i + 1}: add-issue-label labels array must contain only strings`);
            continue;
          }
          // Sanitize label strings
          item.labels = item.labels.map(label => sanitizeContent(label));
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
    console.log('Validation errors found:');
    errors.forEach(error => console.log(`  - ${error}`));
    
    // For now, we'll continue with valid items but log the errors
    // In the future, we might want to fail the workflow for invalid items
  }

  console.log(`Successfully parsed ${parsedItems.length} valid output items`);

  // Set the parsed and validated items as output
  const validatedOutput = {
    items: parsedItems,
    errors: errors
  };

  core.setOutput('output', JSON.stringify(validatedOutput));
  core.setOutput('raw_output', outputContent);
}

// Call the main function
await main();
