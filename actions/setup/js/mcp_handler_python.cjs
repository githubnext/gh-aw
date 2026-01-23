// @ts-check

/**
 * Python Script Handler for Safe-Inputs
 *
 * This module provides a handler for executing Python scripts in safe-inputs tools.
 * It uses a Pythonic approach for passing inputs via JSON on stdin.
 */

const { execFile } = require("child_process");

/**
 * Create a Python script handler function that executes a .py file.
 * Inputs are passed as JSON via stdin for a more Pythonic approach:
 * - Inputs are passed as JSON object via stdin (similar to JavaScript tools)
 * - Python script reads and parses JSON from stdin into 'inputs' dictionary
 * - Outputs are read from stdout (JSON format expected)
 *
 * @param {Object} server - The MCP server instance for logging
 * @param {string} toolName - Name of the tool for logging purposes
 * @param {string} scriptPath - Path to the Python script to execute
 * @param {number} [timeoutSeconds=60] - Timeout in seconds for script execution
 * @returns {Function} Async handler function that executes the Python script
 */
function createPythonHandler(server, toolName, scriptPath, timeoutSeconds = 60) {
  return async args => {
    server.debug(`  [${toolName}] Invoking Python handler: ${scriptPath}`);
    server.debug(`  [${toolName}] Python handler args: ${JSON.stringify(args)}`);
    server.debug(`  [${toolName}] Timeout: ${timeoutSeconds}s`);

    // Pass inputs as JSON via stdin (more Pythonic approach)
    const inputJson = JSON.stringify(args || {});
    server.debug(`  [${toolName}] Input JSON (${inputJson.length} bytes): ${inputJson.substring(0, 200)}${inputJson.length > 200 ? "..." : ""}`);

    return new Promise((resolve, reject) => {
      server.debug(`  [${toolName}] Executing Python script...`);

      const child = execFile(
        "python3",
        [scriptPath],
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
            server.debugError(`  [${toolName}] Python script error: `, error);
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

          server.debug(`  [${toolName}] Python handler completed successfully`);

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
  createPythonHandler,
};
