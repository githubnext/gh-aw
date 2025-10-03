// Test case sensitivity
global.core = {
  error: (msg) => console.log("ERROR:", msg),
  warning: (msg) => console.log("WARNING:", msg),
};

const { validateErrors } = require("./validate_errors.cjs");

const logContent = `
This line has UNAUTHORIZED in caps
This line has unauthorized in lowercase
This line has UnAuthorized in mixed case
`;

const patterns = [
  {
    pattern: "unauthorized",  // Case-sensitive (no (?i))
    level_group: 0,
    message_group: 0,
    description: "Test pattern",
  },
];

console.log("=== Testing case-sensitive pattern 'unauthorized' ===");
validateErrors(logContent, patterns);
