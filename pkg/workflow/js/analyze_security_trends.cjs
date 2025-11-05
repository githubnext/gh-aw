// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Analyzes security scan trends over time
 * Compares current scan results with historical data to track improvements and regressions
 */

const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * Parse scan results from agent output
 * @param {Array} items - Agent output items
 * @returns {Object} Parsed scan data with findings by tool and severity
 */
function parseScanResults(items) {
  const scanData = {
    total: 0,
    byTool: {
      zizmor: { total: 0, bySeverity: { critical: 0, high: 0, medium: 0, low: 0 } },
      poutine: { total: 0, bySeverity: { critical: 0, high: 0, medium: 0, low: 0 } },
      actionlint: { total: 0 },
    },
    findings: [],
    workflowsAffected: new Set(),
  };

  // Parse create_discussion items for scan results
  for (const item of items) {
    if (item.type !== "create_discussion" || !item.body) {
      continue;
    }

    // Extract findings from the discussion body
    const body = item.body;

    // Parse zizmor findings
    const zizmor = parseFindingsByTool(body, "zizmor");
    scanData.byTool.zizmor = zizmor;
    scanData.total += zizmor.total;

    // Parse poutine findings
    const poutine = parseFindingsByTool(body, "poutine");
    scanData.byTool.poutine = poutine;
    scanData.total += poutine.total;

    // Parse actionlint findings
    const actionlint = parseActionlintFindings(body);
    scanData.byTool.actionlint = actionlint;
    scanData.total += actionlint.total;
  }

  return scanData;
}

/**
 * Parse findings for a specific tool from discussion body
 * @param {string} body - Discussion body text
 * @param {string} tool - Tool name (zizmor or poutine)
 * @returns {Object} Findings data for the tool
 */
function parseFindingsByTool(body, tool) {
  const result = {
    total: 0,
    bySeverity: { critical: 0, high: 0, medium: 0, low: 0 },
    byType: {},
  };

  // Look for tool findings table row in the body
  // Format: | zizmor (security) | 50 | 0 | 0 | 5 | 45 |
  const lines = body.split("\n");
  for (const line of lines) {
    if (line.toLowerCase().includes(tool)) {
      // Parse table cells: | tool | total | critical | high | medium | low |
      const cells = line
        .split("|")
        .map(cell => cell.trim())
        .filter(cell => cell);
      if (cells.length >= 6) {
        result.total = parseInt(cells[1]) || 0;
        result.bySeverity.critical = parseInt(cells[2]) || 0;
        result.bySeverity.high = parseInt(cells[3]) || 0;
        result.bySeverity.medium = parseInt(cells[4]) || 0;
        result.bySeverity.low = parseInt(cells[5]) || 0;
        break;
      }
    }
  }

  return result;
}

/**
 * Parse actionlint findings from discussion body
 * @param {string} body - Discussion body text
 * @returns {Object} Actionlint findings data
 */
function parseActionlintFindings(body) {
  const result = { total: 0, byType: {} };

  // Look for actionlint total in the findings table
  // Format: | actionlint (linting) | 20 | - | - | - | - |
  const lines = body.split("\n");
  for (const line of lines) {
    if (line.toLowerCase().includes("actionlint")) {
      const cells = line
        .split("|")
        .map(cell => cell.trim())
        .filter(cell => cell);
      if (cells.length >= 2) {
        result.total = parseInt(cells[1]) || 0;
        break;
      }
    }
  }

  return result;
}

/**
 * Load historical scan data from cache
 * @param {string} cacheDir - Cache directory path
 * @returns {Object} Historical scan data and index
 */
function loadHistoricalData(cacheDir) {
  const fs = require("fs");
  const path = require("path");

  const indexPath = path.join(cacheDir, "security-scans", "index.json");
  const trendsDir = path.join(cacheDir, "security-scans", "trends");

  let index = { scans: [] };
  let weeklyData = {};
  let monthlyData = {};

  // Load scan index
  if (fs.existsSync(indexPath)) {
    try {
      const indexContent = fs.readFileSync(indexPath, "utf8");
      index = JSON.parse(indexContent);
    } catch (error) {
      core.warning(`Failed to load scan index: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  // Load weekly trends
  const weeklyPath = path.join(trendsDir, "weekly.json");
  if (fs.existsSync(weeklyPath)) {
    try {
      const weeklyContent = fs.readFileSync(weeklyPath, "utf8");
      weeklyData = JSON.parse(weeklyContent);
    } catch (error) {
      core.warning(`Failed to load weekly trends: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  // Load monthly trends
  const monthlyPath = path.join(trendsDir, "monthly.json");
  if (fs.existsSync(monthlyPath)) {
    try {
      const monthlyContent = fs.readFileSync(monthlyPath, "utf8");
      monthlyData = JSON.parse(monthlyContent);
    } catch (error) {
      core.warning(`Failed to load monthly trends: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  return { index, weeklyData, monthlyData };
}

/**
 * Compare current scan with previous scan to identify changes
 * @param {Object} currentScan - Current scan data
 * @param {Object} previousScan - Previous scan data
 * @returns {Object} Comparison results with new and resolved issues
 */
function compareScanResults(currentScan, previousScan) {
  const comparison = {
    totalChange: currentScan.total - previousScan.total,
    totalChangePercent: previousScan.total > 0 ? ((currentScan.total - previousScan.total) / previousScan.total) * 100 : 0,
    byTool: {},
    newIssues: 0,
    resolvedIssues: 0,
  };

  // Calculate if this is improvement, regression, or stable
  if (comparison.totalChange < 0) {
    comparison.trend = "improvement";
    comparison.resolvedIssues = Math.abs(comparison.totalChange);
  } else if (comparison.totalChange > 0) {
    comparison.trend = "regression";
    comparison.newIssues = comparison.totalChange;
  } else {
    comparison.trend = "stable";
  }

  // Compare by tool
  for (const tool of ["zizmor", "poutine", "actionlint"]) {
    const current = currentScan.byTool[tool] || { total: 0 };
    const previous = previousScan.byTool[tool] || { total: 0 };

    comparison.byTool[tool] = {
      change: current.total - previous.total,
      changePercent: previous.total > 0 ? ((current.total - previous.total) / previous.total) * 100 : 0,
      current: current.total,
      previous: previous.total,
    };
  }

  return comparison;
}

/**
 * Save current scan data to cache
 * @param {string} cacheDir - Cache directory path
 * @param {Object} scanData - Current scan data
 * @param {Object} comparison - Comparison results
 */
function saveScanData(cacheDir, scanData, comparison) {
  const fs = require("fs");
  const path = require("path");

  // Ensure directories exist
  const scansDir = path.join(cacheDir, "security-scans");
  const trendsDir = path.join(scansDir, "trends");

  if (!fs.existsSync(scansDir)) {
    fs.mkdirSync(scansDir, { recursive: true });
  }
  if (!fs.existsSync(trendsDir)) {
    fs.mkdirSync(trendsDir, { recursive: true });
  }

  // Get current date
  const now = new Date();
  const dateStr = now.toISOString().split("T")[0]; // YYYY-MM-DD

  // Save daily scan
  const dailyScanPath = path.join(scansDir, `${dateStr}.json`);
  const dailyScan = {
    date: dateStr,
    timestamp: now.toISOString(),
    ...scanData,
    comparison: comparison || null,
  };
  fs.writeFileSync(dailyScanPath, JSON.stringify(dailyScan, null, 2));
  core.info(`Saved scan data to ${dailyScanPath}`);

  // Update scan index
  const indexPath = path.join(scansDir, "index.json");
  let index = { scans: [] };
  if (fs.existsSync(indexPath)) {
    try {
      const indexContent = fs.readFileSync(indexPath, "utf8");
      index = JSON.parse(indexContent);
    } catch (error) {
      core.warning(`Failed to load existing index: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  // Add current scan to index (avoid duplicates)
  const existingIndex = index.scans.findIndex(s => s.date === dateStr);
  if (existingIndex >= 0) {
    index.scans[existingIndex] = { date: dateStr, total: scanData.total };
  } else {
    index.scans.push({ date: dateStr, total: scanData.total });
  }

  // Keep only last 90 days
  index.scans = index.scans.sort((a, b) => b.date.localeCompare(a.date)).slice(0, 90);

  fs.writeFileSync(indexPath, JSON.stringify(index, null, 2));

  // Update weekly and monthly aggregates
  updateAggregates(trendsDir, index.scans);
}

/**
 * Update weekly and monthly aggregate data
 * @param {string} trendsDir - Trends directory path
 * @param {Array} scans - Array of scan summaries
 */
function updateAggregates(trendsDir, scans) {
  const fs = require("fs");
  const path = require("path");

  // Calculate weekly aggregates (last 12 weeks)
  const weeklyAggregates = calculateWeeklyAggregates(scans);
  const weeklyPath = path.join(trendsDir, "weekly.json");
  fs.writeFileSync(weeklyPath, JSON.stringify(weeklyAggregates, null, 2));

  // Calculate monthly aggregates (last 12 months)
  const monthlyAggregates = calculateMonthlyAggregates(scans);
  const monthlyPath = path.join(trendsDir, "monthly.json");
  fs.writeFileSync(monthlyPath, JSON.stringify(monthlyAggregates, null, 2));
}

/**
 * Calculate weekly aggregates from scan data
 * @param {Array} scans - Array of scan summaries
 * @returns {Object} Weekly aggregates
 */
function calculateWeeklyAggregates(scans) {
  const weeks = {};

  for (const scan of scans) {
    const date = new Date(scan.date);
    const weekStart = new Date(date);
    weekStart.setDate(date.getDate() - date.getDay()); // Start of week (Sunday)
    const weekKey = weekStart.toISOString().split("T")[0];

    if (!weeks[weekKey]) {
      weeks[weekKey] = { total: 0, count: 0, dates: [] };
    }

    weeks[weekKey].total += scan.total;
    weeks[weekKey].count += 1;
    weeks[weekKey].dates.push(scan.date);
  }

  // Calculate averages
  const weeklyData = {};
  for (const [week, data] of Object.entries(weeks)) {
    weeklyData[week] = {
      average: Math.round(data.total / data.count),
      total: data.total,
      scans: data.count,
    };
  }

  return weeklyData;
}

/**
 * Calculate monthly aggregates from scan data
 * @param {Array} scans - Array of scan summaries
 * @returns {Object} Monthly aggregates
 */
function calculateMonthlyAggregates(scans) {
  const months = {};

  for (const scan of scans) {
    const date = new Date(scan.date);
    const monthKey = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;

    if (!months[monthKey]) {
      months[monthKey] = { total: 0, count: 0, dates: [] };
    }

    months[monthKey].total += scan.total;
    months[monthKey].count += 1;
    months[monthKey].dates.push(scan.date);
  }

  // Calculate averages
  const monthlyData = {};
  for (const [month, data] of Object.entries(months)) {
    monthlyData[month] = {
      average: Math.round(data.total / data.count),
      total: data.total,
      scans: data.count,
    };
  }

  return monthlyData;
}

/**
 * Format trend output for inclusion in discussion
 * @param {Object} scanData - Current scan data
 * @param {Object} comparison - Comparison with previous scan
 * @returns {string} Formatted trend analysis markdown
 */
function formatTrendAnalysis(scanData, comparison) {
  if (!comparison) {
    return "## 📈 Trend Analysis\n\nNo historical data available for comparison. This is the first scan.";
  }

  const trendIcon = comparison.trend === "improvement" ? "↓" : comparison.trend === "regression" ? "↑" : "→";
  const trendText =
    comparison.trend === "improvement"
      ? `${trendIcon} ${Math.abs(comparison.totalChangePercent).toFixed(1)}% improvement`
      : comparison.trend === "regression"
        ? `${trendIcon} ${Math.abs(comparison.totalChangePercent).toFixed(1)}% regression`
        : `${trendIcon} No change`;

  let output = `## 📈 Trend Analysis\n\n`;
  output += `**Week-over-Week Change**: ${trendText}\n\n`;
  output += `| Metric | Current | Previous | Change |\n`;
  output += `|--------|---------|----------|--------|\n`;
  output += `| Total Findings | ${scanData.total} | ${scanData.total - comparison.totalChange} | ${comparison.totalChange >= 0 ? "+" : ""}${comparison.totalChange} (${comparison.totalChangePercent >= 0 ? "+" : ""}${comparison.totalChangePercent.toFixed(1)}%) |\n`;

  // Add tool-specific changes
  for (const tool of ["zizmor", "poutine", "actionlint"]) {
    if (comparison.byTool[tool]) {
      const toolData = comparison.byTool[tool];
      const toolIcon = toolData.change < 0 ? "↓" : toolData.change > 0 ? "↑" : "→";
      output += `| ${tool.charAt(0).toUpperCase() + tool.slice(1)} | ${toolData.current} | ${toolData.previous} | ${toolIcon} ${toolData.change >= 0 ? "+" : ""}${toolData.change} (${toolData.changePercent >= 0 ? "+" : ""}${toolData.changePercent.toFixed(1)}%) |\n`;
    }
  }

  output += `\n`;

  // Add improvements or regressions
  if (comparison.trend === "improvement") {
    output += `**Improvements**:\n`;
    output += `- Resolved ${comparison.resolvedIssues} issue${comparison.resolvedIssues !== 1 ? "s" : ""}\n\n`;
  } else if (comparison.trend === "regression") {
    output += `**New Issues**:\n`;
    output += `- ${comparison.newIssues} new issue${comparison.newIssues !== 1 ? "s" : ""} detected\n\n`;
  }

  return output;
}

async function main() {
  const fs = require("fs");

  // Get cache directory from environment
  const cacheDir = process.env.GH_AW_CACHE_DIR || "/tmp/gh-aw/cache-memory";

  // Load agent output
  const result = loadAgentOutput();
  if (!result.success || result.items.length === 0) {
    core.warning("No agent output found, cannot perform trend analysis");
    core.setOutput("trend_analysis", "");
    return;
  }

  core.info(`Analyzing trends from ${result.items.length} agent output item(s)`);

  // Parse current scan results
  const currentScan = parseScanResults(result.items);
  core.info(`Current scan: ${currentScan.total} total findings`);

  // Load historical data
  const { index } = loadHistoricalData(cacheDir);
  core.info(`Loaded ${index.scans.length} historical scan(s)`);

  // Get previous scan for comparison
  let previousScan = null;
  let comparison = null;

  if (index.scans.length > 0) {
    // Find most recent previous scan (not today)
    const today = new Date().toISOString().split("T")[0];
    const previousScans = index.scans.filter(s => s.date !== today).sort((a, b) => b.date.localeCompare(a.date));

    if (previousScans.length > 0) {
      const previousDate = previousScans[0].date;
      const previousScanPath = `${cacheDir}/security-scans/${previousDate}.json`;

      if (fs.existsSync(previousScanPath)) {
        try {
          const previousContent = fs.readFileSync(previousScanPath, "utf8");
          previousScan = JSON.parse(previousContent);
          core.info(`Comparing with previous scan from ${previousDate}`);

          // Compare scans
          comparison = compareScanResults(currentScan, previousScan);
          core.info(
            `Trend: ${comparison.trend} (${comparison.totalChange >= 0 ? "+" : ""}${comparison.totalChange} findings, ${comparison.totalChangePercent >= 0 ? "+" : ""}${comparison.totalChangePercent.toFixed(1)}%)`
          );
        } catch (error) {
          core.warning(`Failed to load previous scan: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
    }
  }

  // Save current scan data
  saveScanData(cacheDir, currentScan, comparison);

  // Format trend analysis
  const trendAnalysis = formatTrendAnalysis(currentScan, comparison);

  // Output trend analysis for use in discussion
  core.setOutput("trend_analysis", trendAnalysis);
  core.info("Trend analysis complete");

  // Add to step summary
  await core.summary.addRaw("## Trend Analysis Complete\n\n" + trendAnalysis).write();
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
