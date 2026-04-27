// @ts-check

/**
 * Shell Script Handler for MCP Scripts
 *
 * This module provides a handler for executing shell scripts in mcp-scripts tools.
 * It follows GitHub Actions conventions for passing inputs and reading outputs.
 */

const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");
const os = require("os");

/** Interval (ms) between progress log messages for long-running commands */
const PROGRESS_INTERVAL_MS = 30000;

/** Maximum accumulated stdout/stderr size in bytes (10 MB) */
const MAX_BUFFER_BYTES = 10 * 1024 * 1024;

/**
 * Create a shell script handler function that executes a .sh file.
 * Uses GitHub Actions convention for passing inputs/outputs:
 * - Inputs are passed as environment variables prefixed with INPUT_ (uppercased, dashes replaced with underscores)
 * - Outputs are read from GITHUB_OUTPUT file (key=value format, one per line)
 * - Returns: { stdout, stderr, outputs }
 *
 * The handler emits a progress message every 30 seconds while the script is
 * running so that long-lived commands (e.g. `make test-unit`) do not cause
 * silent idle periods that can trip MCP connection timeouts.
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

      const startMs = Date.now();
      let stdout = "";
      let stderr = "";
      let stdoutBytes = 0;
      let stderrBytes = 0;
      /** @type {boolean} */
      let settled = false;

      const child = spawn(scriptPath, [], {
        env,
        cwd: process.env.GITHUB_WORKSPACE || process.cwd(),
      });

      // Periodic progress indicator: emit a heartbeat log every 30 seconds so
      // that long-running commands (make build, make test-unit, etc.) are
      // visible in the workflow logs and keep the MCP connection active.
      const progressInterval = setInterval(() => {
        if (settled) return;
        const elapsedSec = Math.round((Date.now() - startMs) / 1000);
        server.debug(`  [${toolName}] Still running... elapsed=${elapsedSec}s`);
      }, PROGRESS_INTERVAL_MS);

      // Hard timeout: kill the process if it exceeds the allowed duration.
      const timeoutHandle = setTimeout(() => {
        if (settled) return;
        settled = true;
        clearInterval(progressInterval);
        child.kill("SIGKILL");
        cleanupOutputFile(outputFile);
        reject(new Error(`Command timed out after ${timeoutSeconds}s: ${scriptPath}`));
      }, timeoutSeconds * 1000);

      // Stream stdout, capped at MAX_BUFFER_BYTES to prevent memory exhaustion.
      child.stdout.on("data", chunk => {
        const bytes = Buffer.isBuffer(chunk) ? chunk.length : Buffer.byteLength(chunk);
        if (stdoutBytes + bytes <= MAX_BUFFER_BYTES) {
          stdout += chunk.toString();
          stdoutBytes += bytes;
        }
      });

      // Stream stderr, capped at MAX_BUFFER_BYTES.
      child.stderr.on("data", chunk => {
        const bytes = Buffer.isBuffer(chunk) ? chunk.length : Buffer.byteLength(chunk);
        if (stderrBytes + bytes <= MAX_BUFFER_BYTES) {
          stderr += chunk.toString();
          stderrBytes += bytes;
        }
      });

      child.on("error", err => {
        if (settled) return;
        settled = true;
        clearTimeout(timeoutHandle);
        clearInterval(progressInterval);
        server.debugError(`  [${toolName}] Shell spawn error: `, err);
        cleanupOutputFile(outputFile);
        reject(err);
      });

      child.on("close", exitCode => {
        if (settled) return;
        settled = true;
        clearTimeout(timeoutHandle);
        clearInterval(progressInterval);

        const elapsedMs = Date.now() - startMs;

        // Log stdout and stderr
        if (stdout) {
          server.debug(`  [${toolName}] stdout: ${stdout.substring(0, 500)}${stdout.length > 500 ? "..." : ""}`);
        }
        if (stderr) {
          server.debug(`  [${toolName}] stderr: ${stderr.substring(0, 500)}${stderr.length > 500 ? "..." : ""}`);
        }

        if (exitCode !== 0) {
          server.debug(`  [${toolName}] Shell script failed: exit code=${exitCode}, elapsed=${elapsedMs}ms`);
          cleanupOutputFile(outputFile);

          // Build an enhanced error message that includes stdout/stderr so the
          // AI agent can see what actually went wrong (not just "Command failed").
          const parts = [`Command failed: ${scriptPath} (exit code: ${exitCode})`];
          if (stderr && stderr.trim()) {
            parts.push(`stderr:\n${stderr.trim()}`);
          }
          if (stdout && stdout.trim()) {
            parts.push(`stdout:\n${stdout.trim()}`);
          }
          reject(new Error(parts.join("\n")));
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

        cleanupOutputFile(outputFile);

        // Build the result
        const result = {
          stdout: stdout || "",
          stderr: stderr || "",
          outputs,
        };

        server.debug(`  [${toolName}] Shell handler completed: elapsed=${elapsedMs}ms, outputs: ${Object.keys(outputs).join(", ") || "(none)"}`);

        // Return MCP format
        resolve({
          content: [
            {
              type: "text",
              text: JSON.stringify(result),
            },
          ],
        });
      });
    });
  };
}

/**
 * Delete the temporary GITHUB_OUTPUT file, ignoring any errors.
 * @param {string} outputFile - Path to the output file
 */
function cleanupOutputFile(outputFile) {
  try {
    if (fs.existsSync(outputFile)) {
      fs.unlinkSync(outputFile);
    }
  } catch {
    // Ignore cleanup errors
  }
}

module.exports = {
  createShellHandler,
};
