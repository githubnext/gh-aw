// @ts-check

/**
 * Go Script Handler for Safe-Inputs
 *
 * This module provides a handler for executing Go scripts in safe-inputs tools.
 * It uses `go run` to execute Go source files with inputs via JSON on stdin.
 */

const { execFile } = require("child_process");

/**
 * Create a Go script handler function that executes a .go file using `go run`.
 * Inputs are passed as JSON via stdin:
 * - Inputs are passed as JSON object via stdin (similar to Python tools)
 * - Go script reads and parses JSON from stdin into inputs map
 * - Outputs are read from stdout (JSON format expected)
 *
 * @param {Object} server - The MCP server instance for logging
 * @param {string} toolName - Name of the tool for logging purposes
 * @param {string} scriptPath - Path to the Go script to execute
 * @param {number} [timeoutSeconds=60] - Timeout in seconds for script execution
 * @returns {Function} Async handler function that executes the Go script
 */
function createGoHandler(server, toolName, scriptPath, timeoutSeconds = 60) {
  return async args => {
    server.debug(`  [${toolName}] Invoking Go handler: ${scriptPath}`);
    server.debug(`  [${toolName}] Go handler args: ${JSON.stringify(args)}`);
    server.debug(`  [${toolName}] Timeout: ${timeoutSeconds}s`);

    // Pass inputs as JSON via stdin
    const inputJson = JSON.stringify(args || {});
    server.debug(`  [${toolName}] Input JSON (${inputJson.length} bytes): ${inputJson.substring(0, 200)}${inputJson.length > 200 ? "..." : ""}`);

    return new Promise((resolve, reject) => {
      server.debug(`  [${toolName}] Executing Go script with 'go run'...`);

      const child = execFile(
        "go",
        ["run", scriptPath],
        {
          env: process.env,
          cwd: process.env.GITHUB_WORKSPACE || process.cwd(),
          timeout: timeoutSeconds * 1000, // Convert to milliseconds
          maxBuffer: 10 * 1024 * 1024, // 10MB buffer
        },
        (error, stdout, stderr) => {
          // Log stdout and stderr
          if (stdout) {
            server.debug(`  [${toolName}] stdout: ${stdout.substring(0, 500)}${stdout.length > 500 ? "..." : ""}`);
          }
          if (stderr) {
            server.debug(`  [${toolName}] stderr: ${stderr.substring(0, 500)}${stderr.length > 500 ? "..." : ""}`);
          }

          if (error) {
            server.debugError(`  [${toolName}] Go script error: `, error);
            reject(error);
            return;
          }

          // Parse output from stdout
          let result;
          try {
            // Try to parse stdout as JSON
            if (stdout && stdout.trim()) {
              result = JSON.parse(stdout.trim());
            } else {
              result = { stdout: stdout || "", stderr: stderr || "" };
            }
          } catch (parseError) {
            server.debug(`  [${toolName}] Output is not JSON, returning as text`);
            result = { stdout: stdout || "", stderr: stderr || "" };
          }

          server.debug(`  [${toolName}] Go handler completed successfully`);

          // Return MCP format
          resolve({
            content: [
              {
                type: "text",
                text: JSON.stringify(result),
              },
            ],
          });
        }
      );

      // Write input JSON to stdin
      if (child.stdin) {
        child.stdin.write(inputJson);
        child.stdin.end();
      }
    });
  };
}

module.exports = {
  createGoHandler,
};
