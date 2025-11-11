// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Bootstrap helper for log parser entry points.
 * Handles common logic for environment variable lookup, file existence checks,
 * content reading (file or directory), and summary emission.
 *
 * @param {Object} options - Configuration options
 * @param {function(string): string|{markdown: string, mcpFailures?: string[], maxTurnsHit?: boolean}} options.parseLog - Parser function that takes log content and returns markdown or result object
 * @param {string} options.parserName - Name of the parser (e.g., "Codex", "Claude", "Copilot")
 * @param {boolean} [options.supportsDirectories=false] - Whether the parser supports reading from directories
 * @returns {void}
 */
function runLogParser(options) {
  const fs = require("fs");
  const path = require("path");
  const { parseLog, parserName, supportsDirectories = false } = options;

  try {
    const logPath = process.env.GH_AW_AGENT_OUTPUT;
    if (!logPath) {
      core.info("No agent log file specified");
      return;
    }

    if (!fs.existsSync(logPath)) {
      core.info(`Log path not found: ${logPath}`);
      return;
    }

    let content = "";

    // Check if logPath is a directory or a file
    const stat = fs.statSync(logPath);
    if (stat.isDirectory()) {
      if (!supportsDirectories) {
        core.info(`Log path is a directory but ${parserName} parser does not support directories: ${logPath}`);
        return;
      }

      // Read all log files from the directory and concatenate them
      const files = fs.readdirSync(logPath);
      const logFiles = files.filter(file => file.endsWith(".log") || file.endsWith(".txt"));

      if (logFiles.length === 0) {
        core.info(`No log files found in directory: ${logPath}`);
        return;
      }

      // Sort log files by name to ensure consistent ordering
      logFiles.sort();

      // Concatenate all log files
      for (const file of logFiles) {
        const filePath = path.join(logPath, file);
        const fileContent = fs.readFileSync(filePath, "utf8");

        // Add a newline before this file if the previous content doesn't end with one
        if (content.length > 0 && !content.endsWith("\n")) {
          content += "\n";
        }

        content += fileContent;
      }
    } else {
      // Read the single log file
      content = fs.readFileSync(logPath, "utf8");
    }

    const result = parseLog(content);

    // Handle result that may be a simple string or an object with metadata
    let markdown = "";
    let mcpFailures = [];
    let maxTurnsHit = false;

    if (typeof result === "string") {
      markdown = result;
    } else if (result && typeof result === "object") {
      markdown = result.markdown || "";
      mcpFailures = result.mcpFailures || [];
      maxTurnsHit = result.maxTurnsHit || false;
    }

    if (markdown) {
      core.info(markdown);
      core.summary.addRaw(markdown);

      // Add workflow file links if available
      const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE;
      const workflowCompiled = process.env.GH_AW_WORKFLOW_COMPILED;

      if (workflowSource || workflowCompiled) {
        core.summary.addRaw("\n\n---\n\n");
        core.summary.addRaw("## 📄 Workflow Files\n\n");

        // Check if we have the necessary context for building URLs
        const repository = context.payload && context.payload.repository;
        const sha = context.sha;

        if (workflowSource) {
          // Extract relative path from workflow source (remove leading .github/workflows/ if present)
          const sourcePath = workflowSource.replace(/^\.github\/workflows\//, "");

          if (repository && repository.html_url && sha) {
            core.summary.addRaw(`- **Source**: [\`${sourcePath}\`](${repository.html_url}/blob/${sha}/${workflowSource})\n`);
          } else {
            core.summary.addRaw(`- **Source**: \`${sourcePath}\`\n`);
          }
        }

        if (workflowCompiled) {
          // Extract relative path from compiled workflow
          const compiledPath = workflowCompiled.replace(/^\.github\/workflows\//, "");

          if (repository && repository.html_url && sha) {
            core.summary.addRaw(`- **Compiled**: [\`${compiledPath}\`](${repository.html_url}/blob/${sha}/${workflowCompiled})\n`);
          } else {
            core.summary.addRaw(`- **Compiled**: \`${compiledPath}\`\n`);
          }
        }
      }

      core.summary.write();
      core.info(`${parserName} log parsed successfully`);
    } else {
      core.error(`Failed to parse ${parserName} log`);
    }

    // Handle MCP server failures if present
    if (mcpFailures && mcpFailures.length > 0) {
      const failedServers = mcpFailures.join(", ");
      core.setFailed(`MCP server(s) failed to launch: ${failedServers}`);
    }

    // Handle max-turns limit if hit
    if (maxTurnsHit) {
      core.setFailed(`Agent execution stopped: max-turns limit reached. The agent did not complete its task successfully.`);
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

// Export for testing and usage
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    runLogParser,
  };
}
