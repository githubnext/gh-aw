// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Test the health of known secrets used in agentic workflows.
 * Performs benign, cheap queries to verify token life without mutations.
 * Generates a report in the step summary with test results.
 *
 * @returns {Promise<void>}
 */
async function main() {
  core.info("Testing health of known secrets used in agentic workflows");

  const results = [];

  // Test COPILOT_GITHUB_TOKEN
  await testCopilotToken(results);

  // Test GH_AW_GITHUB_TOKEN
  await testGitHubToken(results, "GH_AW_GITHUB_TOKEN");

  // Test GH_AW_GITHUB_MCP_SERVER_TOKEN
  await testGitHubToken(results, "GH_AW_GITHUB_MCP_SERVER_TOKEN");

  // Test GH_AW_PROJECT_GITHUB_TOKEN
  await testGitHubToken(results, "GH_AW_PROJECT_GITHUB_TOKEN");

  // Test ANTHROPIC_API_KEY
  await testAnthropicKey(results);

  // Test OPENAI_API_KEY
  await testOpenAIKey(results);

  // Test BRAVE_API_KEY
  await testBraveKey(results);

  // Test NOTION_API_TOKEN
  await testNotionToken(results);

  // Test TAVILY_API_KEY
  await testTavilyKey(results);

  // Generate summary report
  await generateSummaryReport(results);
}

/**
 * Test COPILOT_GITHUB_TOKEN presence and validity
 * @param {Array} results - Array to store test results
 */
async function testCopilotToken(results) {
  const secretName = "COPILOT_GITHUB_TOKEN";
  const token = process.env[secretName];

  if (!token) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage: "GitHub Copilot CLI and agent sessions",
    });
    return;
  }

  try {
    // Make a simple read-only request to verify the token
    // We'll use a lightweight request to check token validity
    const response = await fetch("https://api.github.com/user", {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/vnd.github+json",
      },
    });

    if (response.ok) {
      const data = await response.json();
      const login = data && typeof data === "object" && "login" in data ? data.login : "unknown";
      results.push({
        name: secretName,
        status: "valid",
        message: `Token is valid (authenticated as ${login})`,
        usage: "GitHub Copilot CLI and agent sessions",
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `Token validation failed: ${response.status} ${errorText}`,
        usage: "GitHub Copilot CLI and agent sessions",
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `Token validation failed: ${getErrorMessage(error)}`,
      usage: "GitHub Copilot CLI and agent sessions",
    });
  }
}

/**
 * Test GitHub token (generic function for various GitHub tokens)
 * @param {Array} results - Array to store test results
 * @param {string} secretName - Name of the secret to test
 */
async function testGitHubToken(results, secretName) {
  const token = process.env[secretName];

  const usageMap = {
    GH_AW_GITHUB_TOKEN: "Cross-repo operations and GitHub MCP server",
    GH_AW_GITHUB_MCP_SERVER_TOKEN: "GitHub MCP server isolation (optional)",
    GH_AW_PROJECT_GITHUB_TOKEN: "GitHub Projects operations",
  };

  const usage = usageMap[secretName] || "GitHub API operations";

  if (!token) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage,
    });
    return;
  }

  try {
    // Make a simple read-only request to verify the token
    const response = await fetch("https://api.github.com/user", {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/vnd.github+json",
      },
    });

    if (response.ok) {
      const data = await response.json();
      const login = data && typeof data === "object" && "login" in data ? data.login : "unknown";
      results.push({
        name: secretName,
        status: "valid",
        message: `Token is valid (authenticated as ${login})`,
        usage,
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `Token validation failed: ${response.status} ${errorText}`,
        usage,
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `Token validation failed: ${getErrorMessage(error)}`,
      usage,
    });
  }
}

/**
 * Test Anthropic API key (Claude)
 * @param {Array} results - Array to store test results
 */
async function testAnthropicKey(results) {
  const secretName = "ANTHROPIC_API_KEY";
  const apiKey = process.env[secretName];

  if (!apiKey) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage: "Claude AI engine",
    });
    return;
  }

  try {
    // Make a minimal request to test the API key
    // We'll use a simple messages endpoint with a very short prompt
    const response = await fetch("https://api.anthropic.com/v1/messages", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "x-api-key": apiKey,
        "anthropic-version": "2023-06-01",
      },
      body: JSON.stringify({
        model: "claude-3-haiku-20240307",
        max_tokens: 1,
        messages: [{ role: "user", content: "Hi" }],
      }),
    });

    if (response.ok) {
      results.push({
        name: secretName,
        status: "valid",
        message: "API key is valid",
        usage: "Claude AI engine",
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `API key validation failed: ${response.status} ${errorText}`,
        usage: "Claude AI engine",
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `API key validation failed: ${getErrorMessage(error)}`,
      usage: "Claude AI engine",
    });
  }
}

/**
 * Test OpenAI API key
 * @param {Array} results - Array to store test results
 */
async function testOpenAIKey(results) {
  const secretName = "OPENAI_API_KEY";
  const apiKey = process.env[secretName];

  if (!apiKey) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage: "OpenAI integrations",
    });
    return;
  }

  try {
    // Test by listing models (read-only operation)
    const response = await fetch("https://api.openai.com/v1/models", {
      method: "GET",
      headers: {
        Authorization: `Bearer ${apiKey}`,
      },
    });

    if (response.ok) {
      results.push({
        name: secretName,
        status: "valid",
        message: "API key is valid",
        usage: "OpenAI integrations",
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `API key validation failed: ${response.status} ${errorText}`,
        usage: "OpenAI integrations",
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `API key validation failed: ${getErrorMessage(error)}`,
      usage: "OpenAI integrations",
    });
  }
}

/**
 * Test Brave API key
 * @param {Array} results - Array to store test results
 */
async function testBraveKey(results) {
  const secretName = "BRAVE_API_KEY";
  const apiKey = process.env[secretName];

  if (!apiKey) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage: "Brave Search API",
    });
    return;
  }

  try {
    // Make a minimal search request
    const response = await fetch("https://api.search.brave.com/res/v1/web/search?q=test&count=1", {
      method: "GET",
      headers: {
        Accept: "application/json",
        "X-Subscription-Token": apiKey,
      },
    });

    if (response.ok) {
      results.push({
        name: secretName,
        status: "valid",
        message: "API key is valid",
        usage: "Brave Search API",
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `API key validation failed: ${response.status} ${errorText}`,
        usage: "Brave Search API",
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `API key validation failed: ${getErrorMessage(error)}`,
      usage: "Brave Search API",
    });
  }
}

/**
 * Test Notion API token
 * @param {Array} results - Array to store test results
 */
async function testNotionToken(results) {
  const secretName = "NOTION_API_TOKEN";
  const token = process.env[secretName];

  if (!token) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage: "Notion integration",
    });
    return;
  }

  try {
    // Test by getting user info (read-only operation)
    const response = await fetch("https://api.notion.com/v1/users/me", {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        "Notion-Version": "2022-06-28",
      },
    });

    if (response.ok) {
      results.push({
        name: secretName,
        status: "valid",
        message: "Token is valid",
        usage: "Notion integration",
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `Token validation failed: ${response.status} ${errorText}`,
        usage: "Notion integration",
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `Token validation failed: ${getErrorMessage(error)}`,
      usage: "Notion integration",
    });
  }
}

/**
 * Test Tavily API key
 * @param {Array} results - Array to store test results
 */
async function testTavilyKey(results) {
  const secretName = "TAVILY_API_KEY";
  const apiKey = process.env[secretName];

  if (!apiKey) {
    results.push({
      name: secretName,
      status: "not_configured",
      message: "Secret not configured",
      usage: "Tavily Search API",
    });
    return;
  }

  try {
    // Make a minimal search request
    const response = await fetch("https://api.tavily.com/search", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        api_key: apiKey,
        query: "test",
        max_results: 1,
      }),
    });

    if (response.ok) {
      results.push({
        name: secretName,
        status: "valid",
        message: "API key is valid",
        usage: "Tavily Search API",
      });
    } else {
      const errorText = await response.text();
      results.push({
        name: secretName,
        status: "invalid",
        message: `API key validation failed: ${response.status} ${errorText}`,
        usage: "Tavily Search API",
      });
    }
  } catch (error) {
    results.push({
      name: secretName,
      status: "invalid",
      message: `API key validation failed: ${getErrorMessage(error)}`,
      usage: "Tavily Search API",
    });
  }
}

/**
 * Generate summary report in GitHub step summary
 * @param {Array} results - Array of test results
 */
async function generateSummaryReport(results) {
  let summaryContent = "## Secret Health Report\n\n";

  // Count statuses
  const validCount = results.filter(r => r.status === "valid").length;
  const invalidCount = results.filter(r => r.status === "invalid").length;
  const notConfiguredCount = results.filter(r => r.status === "not_configured").length;

  summaryContent += `**Summary**: ${validCount} valid, ${invalidCount} invalid, ${notConfiguredCount} not configured\n\n`;

  // Group results by status
  const validResults = results.filter(r => r.status === "valid");
  const invalidResults = results.filter(r => r.status === "invalid");
  const notConfiguredResults = results.filter(r => r.status === "not_configured");

  // Valid secrets
  if (validResults.length > 0) {
    summaryContent += "### ✅ Valid Secrets\n\n";
    summaryContent += "| Secret | Usage | Status |\n";
    summaryContent += "|--------|-------|--------|\n";
    for (const result of validResults) {
      summaryContent += `| \`${result.name}\` | ${result.usage} | ${result.message} |\n`;
    }
    summaryContent += "\n";
  }

  // Invalid secrets
  if (invalidResults.length > 0) {
    summaryContent += "### ❌ Invalid Secrets\n\n";
    summaryContent += "| Secret | Usage | Error |\n";
    summaryContent += "|--------|-------|-------|\n";
    for (const result of invalidResults) {
      summaryContent += `| \`${result.name}\` | ${result.usage} | ${result.message} |\n`;
    }
    summaryContent += "\n";
  }

  // Not configured secrets
  if (notConfiguredResults.length > 0) {
    summaryContent += "### ⚠️ Not Configured Secrets\n\n";
    summaryContent += "| Secret | Usage |\n";
    summaryContent += "|--------|-------|\n";
    for (const result of notConfiguredResults) {
      summaryContent += `| \`${result.name}\` | ${result.usage} |\n`;
    }
    summaryContent += "\n";
  }

  // Add documentation links
  summaryContent += "---\n\n";
  summaryContent += "📚 **Documentation**: [Token Configuration Guide](https://githubnext.github.io/gh-aw/reference/tokens/)\n\n";
  summaryContent += "🔧 **Setup**: Run `gh aw secrets bootstrap` to check and configure secrets\n";

  await core.summary.addRaw(summaryContent).write();

  // Log summary to console
  core.info(`Secret health check complete: ${validCount} valid, ${invalidCount} invalid, ${notConfiguredCount} not configured`);

  // Fail the step if any secrets are invalid
  if (invalidCount > 0) {
    core.warning(`${invalidCount} secret(s) failed validation. Check the step summary for details.`);
  }
}

module.exports = { main };
