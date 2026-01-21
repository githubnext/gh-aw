// @ts-check
/// <reference types="@actions/github-script" />

const core = require("@actions/core");
const exec = require("@actions/exec");
const fs = require("fs");
const path = require("path");

/**
 * Performance Development Environment Setup Action
 * 
 * Sets up the performance development environment for gh-aw and writes
 * a comprehensive step summary with collapsible details following the
 * same markdown best practices as agentic workflows.
 */

// Track step timing and results
const steps = [];
let startTime = Date.now();

/**
 * Execute a step and track its result
 * @param {string} name - Step name
 * @param {() => Promise<string>} fn - Function to execute
 * @returns {Promise<void>}
 */
async function executeStep(name, fn) {
  const stepStart = Date.now();
  console.log(`\n=== ${name} ===`);
  
  try {
    const output = await fn();
    const duration = ((Date.now() - stepStart) / 1000).toFixed(1);
    steps.push({
      name,
      status: "✅",
      duration: `${duration}s`,
      output,
      success: true,
    });
    console.log(`✓ ${name} completed in ${duration}s`);
  } catch (error) {
    const duration = ((Date.now() - stepStart) / 1000).toFixed(1);
    steps.push({
      name,
      status: "❌",
      duration: `${duration}s`,
      output: error.message,
      success: false,
    });
    console.error(`✗ ${name} failed after ${duration}s: ${error.message}`);
    throw error;
  }
}

/**
 * Execute a command and capture output
 * @param {string} command - Command to execute
 * @param {string[]} args - Command arguments
 * @returns {Promise<string>}
 */
async function execCommand(command, args = []) {
  let output = "";
  let errorOutput = "";
  
  await exec.exec(command, args, {
    listeners: {
      stdout: (data) => {
        output += data.toString();
      },
      stderr: (data) => {
        errorOutput += data.toString();
      },
    },
  });
  
  return output + errorOutput;
}

/**
 * Get version information for a command
 * @param {string} command - Command to check
 * @param {string[]} args - Arguments (default: --version)
 * @returns {Promise<string>}
 */
async function getVersion(command, args = ["--version"]) {
  try {
    const output = await execCommand(command, args);
    return output.trim().split("\n")[0];
  } catch (error) {
    return "(not found)";
  }
}

/**
 * Generate the step summary with HTML details/summary tags
 */
async function generateStepSummary() {
  const totalDuration = ((Date.now() - startTime) / 1000).toFixed(1);
  const successCount = steps.filter((s) => s.success).length;
  const failedCount = steps.filter((s) => !s.success).length;
  const status = failedCount === 0 ? "✅" : "❌";
  
  let summary = `## 📦 Performance Development Environment Setup\n\n`;
  summary += `**Status:** ${status} ${failedCount === 0 ? "All steps completed" : `${failedCount} step(s) failed`}\n`;
  summary += `**Total Duration:** ${totalDuration}s\n`;
  summary += `**Steps:** ${successCount}/${steps.length} successful\n\n`;
  
  // Summary table
  summary += `### Summary\n`;
  summary += `| Step | Duration | Status |\n`;
  summary += `|------|----------|--------|\n`;
  for (const step of steps) {
    summary += `| ${step.name} | ${step.duration} | ${step.status} |\n`;
  }
  summary += `\n`;
  
  // Detailed results in collapsible sections
  summary += `### Detailed Results\n`;
  for (const step of steps) {
    const icon = step.success ? "✅" : "❌";
    summary += `<details>\n`;
    summary += `<summary><b>${icon} ${step.name} (${step.duration})</b></summary>\n\n`;
    summary += `\`\`\`\n${step.output.substring(0, 5000)}\`\`\`\n\n`;
    summary += `</details>\n\n`;
  }
  
  await core.summary.addRaw(summary).write();
  console.log("✓ Generated step summary");
}

/**
 * Main action execution
 */
async function run() {
  try {
    console.log("Performance Development Environment Setup");
    console.log(`Started at: ${new Date().toISOString()}`);
    
    // Step 1: Setup Environment
    await executeStep("Setup Environment", async () => {
      const timestamp = new Date().toISOString();
      const cwd = process.cwd();
      return `Timestamp: ${timestamp}\nWorking directory: ${cwd}`;
    });
    
    // Step 2: Verify Go Installation
    await executeStep("Verify Go Installation", async () => {
      const goVersion = await getVersion("go", ["version"]);
      const goPath = process.env.GOPATH || "(not set)";
      const goCacheOutput = await execCommand("go", ["env", "GOCACHE"]);
      const goCache = goCacheOutput.trim();
      return `${goVersion}\nGOPATH: ${goPath}\nGOCACHE: ${goCache}`;
    });
    
    // Step 3: Verify Node Installation
    await executeStep("Verify Node Installation", async () => {
      const nodeVersion = await getVersion("node", ["--version"]);
      const npmVersion = await getVersion("npm", ["--version"]);
      return `Node: ${nodeVersion}\nnpm: ${npmVersion}`;
    });
    
    // Step 4: Install Go Dependencies
    await executeStep("Install Go Dependencies", async () => {
      let output = "";
      output += await execCommand("go", ["mod", "download"]);
      output += await execCommand("go", ["mod", "verify"]);
      output += await execCommand("go", ["mod", "tidy"]);
      return output;
    });
    
    // Step 5: Install Go Development Tools
    await executeStep("Install Go Development Tools", async () => {
      let output = "";
      output += "Installing gopls...\n";
      output += await execCommand("go", ["install", "golang.org/x/tools/gopls@latest"]);
      output += "Installing actionlint...\n";
      output += await execCommand("go", ["install", "github.com/rhysd/actionlint/cmd/actionlint@latest"]);
      output += "Installing golangci-lint...\n";
      output += await execCommand("go", ["install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"]);
      return output;
    });
    
    // Step 6: Install npm Global Dependencies
    await executeStep("Install npm Global Dependencies", async () => {
      const output = await execCommand("npm", ["install", "-g", "prettier"]);
      return output;
    });
    
    // Step 7: Install JavaScript Dependencies
    await executeStep("Install JavaScript Dependencies", async () => {
      process.chdir("pkg/workflow/js");
      const output = await execCommand("npm", ["ci"]);
      process.chdir("../../..");
      return output;
    });
    
    // Step 8: Download GitHub Actions Schema
    await executeStep("Download GitHub Actions Schema", async () => {
      const schemaDir = "pkg/workflow/schemas";
      if (!fs.existsSync(schemaDir)) {
        fs.mkdirSync(schemaDir, { recursive: true });
      }
      
      const schemaUrl = "https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json";
      const schemaPath = path.join(schemaDir, "github-workflow.json");
      
      // Use curl to download (available in GitHub Actions runners)
      await execCommand("curl", ["-s", "-o", schemaPath, schemaUrl]);
      
      const stats = fs.statSync(schemaPath);
      return `Downloaded GitHub Actions schema (${stats.size} bytes)`;
    });
    
    // Step 9: Build gh-aw Binary
    await executeStep("Build gh-aw Binary", async () => {
      const output = await execCommand("make", ["build"]);
      const stats = fs.statSync("./gh-aw");
      const version = await execCommand("./gh-aw", ["--version"]);
      return `${output}\nBuilt gh-aw binary (${stats.size} bytes)\n${version}`;
    });
    
    // Step 10: Verify Build Environment
    await executeStep("Verify Build Environment", async () => {
      const go = await getVersion("go", ["version"]);
      const node = await getVersion("node", ["--version"]);
      const npm = await getVersion("npm", ["--version"]);
      const ghAw = await getVersion("./gh-aw", ["--version"]);
      const gopls = await getVersion("gopls", ["version"]);
      const golangciLint = await getVersion("golangci-lint", ["--version"]);
      const actionlint = await getVersion("actionlint", ["--version"]);
      const prettier = await getVersion("prettier", ["--version"]);
      
      return `✓ Go: ${go}\n✓ Node: ${node}\n✓ npm: ${npm}\n✓ gh-aw: ${ghAw}\n✓ gopls: ${gopls}\n✓ golangci-lint: ${golangciLint}\n✓ actionlint: ${actionlint}\n✓ prettier: ${prettier}`;
    });
    
    // Step 11: Create Performance Testing Directory
    await executeStep("Create Performance Testing Directory", async () => {
      const perfDir = "/tmp/gh-aw/perf";
      const benchDir = "/tmp/gh-aw/benchmarks";
      
      if (!fs.existsSync(perfDir)) {
        fs.mkdirSync(perfDir, { recursive: true });
      }
      if (!fs.existsSync(benchDir)) {
        fs.mkdirSync(benchDir, { recursive: true });
      }
      
      return `Created ${perfDir} for performance test artifacts\nCreated ${benchDir} for benchmark results`;
    });
    
    // Generate step summary
    await generateStepSummary();
    
    console.log("\n✓ Performance development environment ready!");
    console.log(`Completed at: ${new Date().toISOString()}`);
    
  } catch (error) {
    // Still generate summary on failure
    await generateStepSummary();
    core.setFailed(`Action failed: ${error.message}`);
  }
}

run();
