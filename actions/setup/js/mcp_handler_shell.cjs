// @ts-check

/**
 * Shell Script Handler for Safe-Inputs
 *
 * This module provides a handler for executing shell scripts in safe-inputs tools.
 * It follows GitHub Actions conventions for passing inputs and reading outputs.
 */

const fs = require("fs");
const path = require("path");
const { execFile } = require("child_process");
const os = require("os");

/**
 * Create a shell script handler function that executes a .sh file.
 * Uses GitHub Actions convention for passing inputs/outputs:
 * - Inputs are passed as environment variables prefixed with INPUT_ (uppercased, dashes replaced with underscores)
 * - Outputs are read from GITHUB_OUTPUT file (key=value format, one per line)
 * - Returns: { stdout, stderr, outputs }
 *
 * @param {Object} server - The MCP server instance for logging
 * @param {string} toolName - Name of the tool for logging purposes
 * @param {string} scriptPath - Path to the shell script to execute
 * @param {number} [timeoutSeconds=60] - Timeout in seconds for script execution
 * @returns {Function} Async handler function that executes the shell script
 */
function createShellHandler(server, toolName, scriptPath, timeoutSeconds = 60) {
  return async args => {
    server.debug(`  [${toolName}] Invoking shell handler: ${scriptPath}`);
    server.debug(`  [${toolName}] Shell handler args: ${JSON.stringify(args)}`);
    server.debug(`  [${toolName}] Timeout: ${timeoutSeconds}s`);

    // Create environment variables from args (GitHub Actions convention: INPUT_NAME)
    const env = { ...process.env };
    for (const [key, value] of Object.entries(args || {})) {
      const envKey = `INPUT_${key.toUpperCase().replace(/-/g, "_")}`;
      env[envKey] = String(value);
      server.debug(`  [${toolName}] Set env: ${envKey}=${String(value).substring(0, 100)}${String(value).length > 100 ? "..." : ""}`);
    }

    // Create a temporary file for outputs (GitHub Actions convention: GITHUB_OUTPUT)
    const outputFile = path.join(os.tmpdir(), `mcp-shell-output-${Date.now()}-${Math.random().toString(36).substring(2)}.txt`);
    env.GITHUB_OUTPUT = outputFile;
    server.debug(`  [${toolName}] Output file: ${outputFile}`);

    // Create the output file (empty)
    fs.writeFileSync(outputFile, "");

    return new Promise((resolve, reject) => {
      server.debug(`  [${toolName}] Executing shell script...`);

      execFile(
        scriptPath,
        [],
        {
          env,
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
            server.debugError(`  [${toolName}] Shell script error: `, error);

            // Clean up output file
            try {
              if (fs.existsSync(outputFile)) {
                fs.unlinkSync(outputFile);
              }
            } catch {
              // Ignore cleanup errors
            }

            reject(error);
            return;
          }

          // Read outputs from the GITHUB_OUTPUT file
          /** @type {Record<string, string>} */
          const outputs = {};
          try {
            if (fs.existsSync(outputFile)) {
              const outputContent = fs.readFileSync(outputFile, "utf-8");
              server.debug(`  [${toolName}] Output file content: ${outputContent.substring(0, 500)}${outputContent.length > 500 ? "..." : ""}`);

              // Parse outputs (key=value format, one per line)
              const lines = outputContent.split("\n");
              for (const line of lines) {
                const trimmed = line.trim();
                if (trimmed && trimmed.includes("=")) {
                  const eqIndex = trimmed.indexOf("=");
                  const key = trimmed.substring(0, eqIndex);
                  const value = trimmed.substring(eqIndex + 1);
                  outputs[key] = value;
                  server.debug(`  [${toolName}] Parsed output: ${key}=${value.substring(0, 100)}${value.length > 100 ? "..." : ""}`);
                }
              }
            }
          } catch (readError) {
            server.debugError(`  [${toolName}] Error reading output file: `, readError);
          }

          // Clean up output file
          try {
            if (fs.existsSync(outputFile)) {
              fs.unlinkSync(outputFile);
            }
          } catch {
            // Ignore cleanup errors
          }

          // Build the result
          const result = {
            stdout: stdout || "",
            stderr: stderr || "",
            outputs,
          };

          server.debug(`  [${toolName}] Shell handler completed, outputs: ${Object.keys(outputs).join(", ") || "(none)"}`);

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
    });
  };
}

module.exports = {
  createShellHandler,
};
