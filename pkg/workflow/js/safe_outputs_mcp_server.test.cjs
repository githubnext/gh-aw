const { execSync } = require("child_process");
const { createRequire } = require("module");
const require = createRequire(import.meta.url);

// Load the function to test
const fs = require("fs");
eval(fs.readFileSync("safe_outputs_mcp_server.js", "utf8"));

// Test that the main function exists
if (typeof safeOutputsMcpServerMain !== "function") {
  throw new Error("safeOutputsMcpServerMain function not found");
}

console.log("✓ safe_outputs_mcp_server.js syntax and function export is valid");
