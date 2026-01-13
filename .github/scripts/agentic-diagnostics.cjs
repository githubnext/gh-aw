#!/usr/bin/env node
// @ts-check

/**
 * Agentic Diagnostics Script
 * 
 * Tests each known secret in the repository and generates a diagnostic report
 * about their configuration status and availability.
 */

const https = require('https');
const { exec } = require('child_process');
const { promisify } = require('util');
const execAsync = promisify(exec);

// ANSI color codes for terminal output
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  gray: '\x1b[90m'
};

/**
 * Test result status
 */
const Status = {
  SUCCESS: 'success',
  FAILURE: 'failure',
  NOT_SET: 'not_set',
  SKIPPED: 'skipped'
};

/**
 * Make an HTTPS request
 * @param {string} hostname 
 * @param {string} path 
 * @param {Object} headers 
 * @param {string} method 
 * @returns {Promise<{statusCode: number, data: string}>}
 */
function makeRequest(hostname, path, headers, method = 'GET') {
  return new Promise((resolve, reject) => {
    const options = {
      hostname,
      path,
      method,
      headers
    };

    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        resolve({ statusCode: res.statusCode || 0, data });
      });
    });

    req.on('error', (err) => {
      reject(err);
    });

    req.setTimeout(10000, () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    req.end();
  });
}

/**
 * Test GitHub token with REST API
 * @param {string} token 
 * @param {string} owner 
 * @param {string} repo 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testGitHubRESTAPI(token, owner, repo) {
  if (!token) {
    return { status: Status.NOT_SET, message: 'Token not set' };
  }

  try {
    const result = await makeRequest(
      'api.github.com',
      `/repos/${owner}/${repo}`,
      {
        'User-Agent': 'gh-aw-agentic-diagnostics',
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
      }
    );

    if (result.statusCode === 200) {
      const data = JSON.parse(result.data);
      return {
        status: Status.SUCCESS,
        message: 'REST API access successful',
        details: {
          statusCode: result.statusCode,
          repoName: data.full_name,
          repoPrivate: data.private
        }
      };
    } else {
      return {
        status: Status.FAILURE,
        message: `REST API returned status ${result.statusCode}`,
        details: { statusCode: result.statusCode }
      };
    }
  } catch (error) {
    return {
      status: Status.FAILURE,
      message: `REST API error: ${error.message}`,
      details: { error: error.message }
    };
  }
}

/**
 * Test GitHub token with GraphQL API
 * @param {string} token 
 * @param {string} owner 
 * @param {string} repo 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testGitHubGraphQLAPI(token, owner, repo) {
  if (!token) {
    return { status: Status.NOT_SET, message: 'Token not set' };
  }

  const query = `
    query {
      repository(owner: "${owner}", name: "${repo}") {
        name
        owner {
          login
        }
        isPrivate
      }
    }
  `;

  try {
    const result = await new Promise((resolve, reject) => {
      const postData = JSON.stringify({ query });
      const options = {
        hostname: 'api.github.com',
        path: '/graphql',
        method: 'POST',
        headers: {
          'User-Agent': 'gh-aw-agentic-diagnostics',
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData)
        }
      };

      const req = https.request(options, (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          resolve({ statusCode: res.statusCode || 0, data });
        });
      });

      req.on('error', reject);
      req.setTimeout(10000, () => {
        req.destroy();
        reject(new Error('Request timeout'));
      });
      req.write(postData);
      req.end();
    });

    if (result.statusCode === 200) {
      const data = JSON.parse(result.data);
      if (data.errors) {
        return {
          status: Status.FAILURE,
          message: 'GraphQL query returned errors',
          details: { errors: data.errors }
        };
      }
      return {
        status: Status.SUCCESS,
        message: 'GraphQL API access successful',
        details: {
          statusCode: result.statusCode,
          repoName: data.data?.repository?.name,
          repoPrivate: data.data?.repository?.isPrivate
        }
      };
    } else {
      return {
        status: Status.FAILURE,
        message: `GraphQL API returned status ${result.statusCode}`,
        details: { statusCode: result.statusCode }
      };
    }
  } catch (error) {
    return {
      status: Status.FAILURE,
      message: `GraphQL API error: ${error.message}`,
      details: { error: error.message }
    };
  }
}

/**
 * Test Copilot CLI availability
 * @param {string} token 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testCopilotCLI(token) {
  if (!token) {
    return { status: Status.NOT_SET, message: 'Token not set' };
  }

  try {
    // Check if copilot CLI is installed
    const { stdout, stderr } = await execAsync('which copilot 2>/dev/null || echo ""');
    if (!stdout.trim()) {
      return {
        status: Status.SKIPPED,
        message: 'Copilot CLI not installed (skipped)',
        details: { note: 'Install @github/copilot to test' }
      };
    }

    return {
      status: Status.SUCCESS,
      message: 'Copilot CLI is available',
      details: { cliPath: stdout.trim() }
    };
  } catch (error) {
    return {
      status: Status.SKIPPED,
      message: 'Copilot CLI check skipped',
      details: { note: 'Install @github/copilot to test' }
    };
  }
}

/**
 * Test Anthropic API
 * @param {string} apiKey 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testAnthropicAPI(apiKey) {
  if (!apiKey) {
    return { status: Status.NOT_SET, message: 'API key not set' };
  }

  try {
    // Test with a minimal API call to check authentication
    const result = await new Promise((resolve, reject) => {
      const postData = JSON.stringify({
        model: 'claude-3-haiku-20240307',
        max_tokens: 1,
        messages: [{ role: 'user', content: 'test' }]
      });

      const options = {
        hostname: 'api.anthropic.com',
        path: '/v1/messages',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': apiKey,
          'anthropic-version': '2023-06-01',
          'Content-Length': Buffer.byteLength(postData)
        }
      };

      const req = https.request(options, (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          resolve({ statusCode: res.statusCode || 0, data });
        });
      });

      req.on('error', reject);
      req.setTimeout(10000, () => {
        req.destroy();
        reject(new Error('Request timeout'));
      });
      req.write(postData);
      req.end();
    });

    if (result.statusCode === 200) {
      return {
        status: Status.SUCCESS,
        message: 'Anthropic API access successful',
        details: { statusCode: result.statusCode }
      };
    } else if (result.statusCode === 401) {
      return {
        status: Status.FAILURE,
        message: 'Invalid Anthropic API key',
        details: { statusCode: result.statusCode }
      };
    } else {
      return {
        status: Status.FAILURE,
        message: `Anthropic API returned status ${result.statusCode}`,
        details: { statusCode: result.statusCode }
      };
    }
  } catch (error) {
    return {
      status: Status.FAILURE,
      message: `Anthropic API error: ${error.message}`,
      details: { error: error.message }
    };
  }
}

/**
 * Test OpenAI API
 * @param {string} apiKey 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testOpenAIAPI(apiKey) {
  if (!apiKey) {
    return { status: Status.NOT_SET, message: 'API key not set' };
  }

  try {
    // Test with models endpoint which is lightweight
    const result = await makeRequest(
      'api.openai.com',
      '/v1/models',
      {
        'Authorization': `Bearer ${apiKey}`
      }
    );

    if (result.statusCode === 200) {
      return {
        status: Status.SUCCESS,
        message: 'OpenAI API access successful',
        details: { statusCode: result.statusCode }
      };
    } else if (result.statusCode === 401) {
      return {
        status: Status.FAILURE,
        message: 'Invalid OpenAI API key',
        details: { statusCode: result.statusCode }
      };
    } else {
      return {
        status: Status.FAILURE,
        message: `OpenAI API returned status ${result.statusCode}`,
        details: { statusCode: result.statusCode }
      };
    }
  } catch (error) {
    return {
      status: Status.FAILURE,
      message: `OpenAI API error: ${error.message}`,
      details: { error: error.message }
    };
  }
}

/**
 * Test Brave Search API
 * @param {string} apiKey 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testBraveSearchAPI(apiKey) {
  if (!apiKey) {
    return { status: Status.NOT_SET, message: 'API key not set' };
  }

  try {
    const result = await makeRequest(
      'api.search.brave.com',
      '/res/v1/web/search?q=test&count=1',
      {
        'Accept': 'application/json',
        'X-Subscription-Token': apiKey
      }
    );

    if (result.statusCode === 200) {
      return {
        status: Status.SUCCESS,
        message: 'Brave Search API access successful',
        details: { statusCode: result.statusCode }
      };
    } else if (result.statusCode === 401 || result.statusCode === 403) {
      return {
        status: Status.FAILURE,
        message: 'Invalid Brave Search API key',
        details: { statusCode: result.statusCode }
      };
    } else {
      return {
        status: Status.FAILURE,
        message: `Brave Search API returned status ${result.statusCode}`,
        details: { statusCode: result.statusCode }
      };
    }
  } catch (error) {
    return {
      status: Status.FAILURE,
      message: `Brave Search API error: ${error.message}`,
      details: { error: error.message }
    };
  }
}

/**
 * Test Notion API
 * @param {string} token 
 * @returns {Promise<{status: string, message: string, details?: any}>}
 */
async function testNotionAPI(token) {
  if (!token) {
    return { status: Status.NOT_SET, message: 'Token not set' };
  }

  try {
    // Test with users/me endpoint
    const result = await makeRequest(
      'api.notion.com',
      '/v1/users/me',
      {
        'Authorization': `Bearer ${token}`,
        'Notion-Version': '2022-06-28'
      }
    );

    if (result.statusCode === 200) {
      return {
        status: Status.SUCCESS,
        message: 'Notion API access successful',
        details: { statusCode: result.statusCode }
      };
    } else if (result.statusCode === 401) {
      return {
        status: Status.FAILURE,
        message: 'Invalid Notion API token',
        details: { statusCode: result.statusCode }
      };
    } else {
      return {
        status: Status.FAILURE,
        message: `Notion API returned status ${result.statusCode}`,
        details: { statusCode: result.statusCode }
      };
    }
  } catch (error) {
    return {
      status: Status.FAILURE,
      message: `Notion API error: ${error.message}`,
      details: { error: error.message }
    };
  }
}

/**
 * Format status emoji
 * @param {string} status 
 * @returns {string}
 */
function statusEmoji(status) {
  switch (status) {
    case Status.SUCCESS: return '✅';
    case Status.FAILURE: return '❌';
    case Status.NOT_SET: return '⚪';
    case Status.SKIPPED: return '⏭️';
    default: return '❓';
  }
}

/**
 * Format status for terminal
 * @param {string} status 
 * @returns {string}
 */
function statusColor(status) {
  switch (status) {
    case Status.SUCCESS: return colors.green;
    case Status.FAILURE: return colors.red;
    case Status.NOT_SET: return colors.gray;
    case Status.SKIPPED: return colors.yellow;
    default: return colors.reset;
  }
}

/**
 * Main diagnostic function
 */
async function runDiagnostics() {
  console.log(`${colors.blue}Starting agentic diagnostics...${colors.reset}\n`);

  const owner = process.env.GITHUB_REPOSITORY?.split('/')[0] || 'unknown';
  const repo = process.env.GITHUB_REPOSITORY?.split('/')[1] || 'unknown';

  const results = [];

  // Test GH_AW_GITHUB_TOKEN
  console.log('Testing GH_AW_GITHUB_TOKEN...');
  const ghAwToken = process.env.GH_AW_GITHUB_TOKEN;
  const restResult = await testGitHubRESTAPI(ghAwToken, owner, repo);
  results.push({
    secret: 'GH_AW_GITHUB_TOKEN',
    test: 'GitHub REST API',
    ...restResult
  });
  console.log(`  ${statusColor(restResult.status)}${statusEmoji(restResult.status)} ${restResult.message}${colors.reset}`);

  const graphqlResult = await testGitHubGraphQLAPI(ghAwToken, owner, repo);
  results.push({
    secret: 'GH_AW_GITHUB_TOKEN',
    test: 'GitHub GraphQL API',
    ...graphqlResult
  });
  console.log(`  ${statusColor(graphqlResult.status)}${statusEmoji(graphqlResult.status)} ${graphqlResult.message}${colors.reset}`);

  // Test GH_AW_GITHUB_MCP_SERVER_TOKEN
  console.log('\nTesting GH_AW_GITHUB_MCP_SERVER_TOKEN...');
  const mcpToken = process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN;
  const mcpRestResult = await testGitHubRESTAPI(mcpToken, owner, repo);
  results.push({
    secret: 'GH_AW_GITHUB_MCP_SERVER_TOKEN',
    test: 'GitHub REST API',
    ...mcpRestResult
  });
  console.log(`  ${statusColor(mcpRestResult.status)}${statusEmoji(mcpRestResult.status)} ${mcpRestResult.message}${colors.reset}`);

  // Test GH_AW_PROJECT_GITHUB_TOKEN
  console.log('\nTesting GH_AW_PROJECT_GITHUB_TOKEN...');
  const projectToken = process.env.GH_AW_PROJECT_GITHUB_TOKEN;
  const projectRestResult = await testGitHubRESTAPI(projectToken, owner, repo);
  results.push({
    secret: 'GH_AW_PROJECT_GITHUB_TOKEN',
    test: 'GitHub REST API',
    ...projectRestResult
  });
  console.log(`  ${statusColor(projectRestResult.status)}${statusEmoji(projectRestResult.status)} ${projectRestResult.message}${colors.reset}`);

  // Test GH_AW_COPILOT_TOKEN
  console.log('\nTesting GH_AW_COPILOT_TOKEN...');
  const copilotToken = process.env.GH_AW_COPILOT_TOKEN;
  const copilotResult = await testCopilotCLI(copilotToken);
  results.push({
    secret: 'GH_AW_COPILOT_TOKEN',
    test: 'Copilot CLI Availability',
    ...copilotResult
  });
  console.log(`  ${statusColor(copilotResult.status)}${statusEmoji(copilotResult.status)} ${copilotResult.message}${colors.reset}`);

  // Test ANTHROPIC_API_KEY
  console.log('\nTesting ANTHROPIC_API_KEY...');
  const anthropicKey = process.env.ANTHROPIC_API_KEY;
  const anthropicResult = await testAnthropicAPI(anthropicKey);
  results.push({
    secret: 'ANTHROPIC_API_KEY',
    test: 'Anthropic Claude API',
    ...anthropicResult
  });
  console.log(`  ${statusColor(anthropicResult.status)}${statusEmoji(anthropicResult.status)} ${anthropicResult.message}${colors.reset}`);

  // Test OPENAI_API_KEY
  console.log('\nTesting OPENAI_API_KEY...');
  const openaiKey = process.env.OPENAI_API_KEY;
  const openaiResult = await testOpenAIAPI(openaiKey);
  results.push({
    secret: 'OPENAI_API_KEY',
    test: 'OpenAI API',
    ...openaiResult
  });
  console.log(`  ${statusColor(openaiResult.status)}${statusEmoji(openaiResult.status)} ${openaiResult.message}${colors.reset}`);

  // Test BRAVE_API_KEY
  console.log('\nTesting BRAVE_API_KEY...');
  const braveKey = process.env.BRAVE_API_KEY;
  const braveResult = await testBraveSearchAPI(braveKey);
  results.push({
    secret: 'BRAVE_API_KEY',
    test: 'Brave Search API',
    ...braveResult
  });
  console.log(`  ${statusColor(braveResult.status)}${statusEmoji(braveResult.status)} ${braveResult.message}${colors.reset}`);

  // Test NOTION_API_TOKEN
  console.log('\nTesting NOTION_API_TOKEN...');
  const notionToken = process.env.NOTION_API_TOKEN;
  const notionResult = await testNotionAPI(notionToken);
  results.push({
    secret: 'NOTION_API_TOKEN',
    test: 'Notion API',
    ...notionResult
  });
  console.log(`  ${statusColor(notionResult.status)}${statusEmoji(notionResult.status)} ${notionResult.message}${colors.reset}`);

  // Generate markdown report
  console.log(`\n${colors.blue}Generating report...${colors.reset}`);
  const report = generateMarkdownReport(results);
  
  // Write to file
  const fs = require('fs');
  fs.writeFileSync('diagnostics.md', report);
  console.log(`${colors.green}✅ Report written to diagnostics.md${colors.reset}`);

  // Return results
  return results;
}

/**
 * Generate markdown diagnostic report
 * @param {Array} results 
 * @returns {string}
 */
function generateMarkdownReport(results) {
  const timestamp = new Date().toISOString();
  
  let report = `## 📊 Summary\n\n`;
  report += `**Generated:** ${timestamp} | **Repository:** ${process.env.GITHUB_REPOSITORY || 'unknown'}\n\n`;
  
  const summary = {
    total: results.length,
    success: results.filter(r => r.status === Status.SUCCESS).length,
    failure: results.filter(r => r.status === Status.FAILURE).length,
    notSet: results.filter(r => r.status === Status.NOT_SET).length,
    skipped: results.filter(r => r.status === Status.SKIPPED).length
  };
  
  // Create a summary table
  report += `| Status | Count | Percentage |\n`;
  report += `|--------|-------|------------|\n`;
  report += `| ✅ Successful | ${summary.success} | ${Math.round((summary.success / summary.total) * 100)}% |\n`;
  report += `| ❌ Failed | ${summary.failure} | ${Math.round((summary.failure / summary.total) * 100)}% |\n`;
  report += `| ⚪ Not Set | ${summary.notSet} | ${Math.round((summary.notSet / summary.total) * 100)}% |\n`;
  if (summary.skipped > 0) {
    report += `| ⏭️ Skipped | ${summary.skipped} | ${Math.round((summary.skipped / summary.total) * 100)}% |\n`;
  }
  report += `| **Total** | **${summary.total}** | **100%** |\n\n`;
  
  // Add recommendations section early with callouts
  const notSetSecrets = [...new Set(results.filter(r => r.status === Status.NOT_SET).map(r => r.secret))];
  const failedSecrets = [...new Set(results.filter(r => r.status === Status.FAILURE).map(r => r.secret))];
  
  if (notSetSecrets.length === 0 && failedSecrets.length === 0) {
    report += `> [!TIP]\n`;
    report += `> ✅ All configured secrets are working correctly!\n\n`;
  } else {
    if (failedSecrets.length > 0) {
      report += `> [!WARNING]\n`;
      report += `> **Failed Tests:** ${failedSecrets.length} secret(s) failed validation\n`;
      report += `>\n`;
      failedSecrets.forEach(secret => {
        report += `> - \`${secret}\`\n`;
      });
      report += `>\n`;
      report += `> Review the secret values and ensure they have proper permissions.\n\n`;
    }
    
    if (notSetSecrets.length > 0) {
      report += `> [!NOTE]\n`;
      report += `> **Not Configured:** ${notSetSecrets.length} secret(s) not set\n`;
      report += `>\n`;
      notSetSecrets.forEach(secret => {
        report += `> - \`${secret}\`\n`;
      });
      report += `>\n`;
      report += `> Configure these secrets in repository settings if needed.\n\n`;
    }
  }
  
  report += `---\n\n`;
  report += `## 🔍 Detailed Results\n\n`;
  
  // Group by secret
  const bySecret = {};
  results.forEach(result => {
    if (!bySecret[result.secret]) {
      bySecret[result.secret] = [];
    }
    bySecret[result.secret].push(result);
  });
  
  Object.entries(bySecret).forEach(([secret, tests]) => {
    // Determine overall status for the secret
    const hasFailure = tests.some(t => t.status === Status.FAILURE);
    const hasNotSet = tests.some(t => t.status === Status.NOT_SET);
    const allSuccess = tests.every(t => t.status === Status.SUCCESS);
    
    let statusIcon = '✅';
    if (hasFailure) statusIcon = '❌';
    else if (hasNotSet) statusIcon = '⚪';
    else if (tests.some(t => t.status === Status.SKIPPED)) statusIcon = '⏭️';
    
    // Use collapsible sections for each secret
    report += `<details>\n`;
    report += `<summary><strong>${statusIcon} ${secret}</strong> (${tests.length} test${tests.length > 1 ? 's' : ''})</summary>\n\n`;
    
    tests.forEach(test => {
      report += `### ${statusEmoji(test.status)} ${test.test}\n\n`;
      report += `**Status:** ${test.status} | **Message:** ${test.message}\n\n`;
      
      if (test.details) {
        report += `<details>\n`;
        report += `<summary>View details</summary>\n\n`;
        report += `\`\`\`json\n`;
        report += `${JSON.stringify(test.details, null, 2)}\n`;
        report += `\`\`\`\n\n`;
        report += `</details>\n\n`;
      }
    });
    
    report += `</details>\n\n`;
  });
  
  return report;
}

// Run diagnostics
if (require.main === module) {
  runDiagnostics()
    .then(results => {
      const failures = results.filter(r => r.status === Status.FAILURE).length;
      if (failures > 0) {
        console.log(`\n${colors.yellow}⚠️  ${failures} test(s) failed${colors.reset}`);
        process.exit(0); // Don't fail the workflow, just report
      } else {
        console.log(`\n${colors.green}✅ All tests completed${colors.reset}`);
        process.exit(0);
      }
    })
    .catch(error => {
      console.error(`${colors.red}Error running diagnostics: ${error.message}${colors.reset}`);
      console.error(error.stack);
      process.exit(1);
    });
}

module.exports = {
  runDiagnostics,
  testGitHubRESTAPI,
  testGitHubGraphQLAPI,
  testCopilotCLI,
  testAnthropicAPI,
  testOpenAIAPI,
  testBraveSearchAPI,
  testNotionAPI,
  generateMarkdownReport
};
