// Test to demonstrate overly aggressive error patterns
const { validateErrors, extractLevel } = require("./validate_errors.cjs");

// Mock core for testing
global.core = {
  error: (msg) => console.log("ERROR:", msg),
  warning: (msg) => console.log("WARNING:", msg),
  info: (msg) => console.log("INFO:", msg),
  setFailed: (msg) => console.log("FAILED:", msg),
};

// Claude-style log content with informational mentions of "unauthorized"
const logContent = `
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll check if the user is unauthorized to access this resource."}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"The API returned 401 Unauthorized, which means we need to authenticate."}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Permission was denied because the token expired."}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"This endpoint is forbidden without admin privileges."}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Access is restricted to team members only."}]}}
`;

// Current overly aggressive patterns from claude_engine.go
const aggressivePatterns = [
  {
    pattern: "(?i)permission.*denied",
    level_group: 0,
    message_group: 0,
    description: "Generic permission denied error",
  },
  {
    pattern: "(?i)unauthorized",
    level_group: 0,
    message_group: 0,
    description: "Unauthorized access error",
  },
  {
    pattern: "(?i)forbidden",
    level_group: 0,
    message_group: 0,
    description: "Forbidden access error",
  },
  {
    pattern: "(?i)access.*restricted",
    level_group: 0,
    message_group: 0,
    description: "Access restricted error",
  },
];

console.log("=== Testing with AGGRESSIVE patterns ===");
const hasErrors = validateErrors(logContent, aggressivePatterns);
console.log("\nResult:", hasErrors ? "FAILED (has errors)" : "PASSED (no errors)");
console.log("\nNote: These are informational mentions, not actual errors!");
