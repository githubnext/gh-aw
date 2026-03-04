// @ts-check

/**
 * Safe Inputs Script Runner
 *
 * Provides the main execution harness for generated safe-input JavaScript tools.
 * Generated .cjs files export an `execute(inputs)` function and delegate to this
 * runner when invoked as a subprocess by the MCP handler.
 *
 * The MCP handler spawns the generated script with `node scriptPath`, passes
 * inputs as JSON on stdin, and reads the JSON result from stdout.
 *
 * Usage (generated scripts call this automatically):
 *   if (require.main === module) {
 *     require('./safe-inputs-runner.cjs')(execute);
 *   }
 */

/**
 * Run a safe-input execute function as a subprocess entry point.
 * Reads JSON inputs from stdin, calls execute(inputs), and writes the JSON
 * result to stdout.  On error, writes to stderr and exits with code 1.
 *
 * @param {function(Object): Promise<any>} execute - The tool execute function
 */
function runSafeInput(execute) {
  let inputJson = "";
  process.stdin.setEncoding("utf8");
  process.stdin.on("data", chunk => {
    inputJson += chunk;
  });
  process.stdin.on("end", async () => {
    let inputs = {};
    try {
      if (inputJson.trim()) {
        inputs = JSON.parse(inputJson.trim());
      }
    } catch (e) {
      process.stderr.write("Warning: Failed to parse inputs: " + (e instanceof Error ? e.message : String(e)) + "\n");
    }
    try {
      const result = await execute(inputs);
      process.stdout.write(JSON.stringify(result));
    } catch (err) {
      process.stderr.write(String(err));
      process.exit(1);
    }
  });
}

module.exports = runSafeInput;
