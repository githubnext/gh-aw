const plugin = require("./dist/index.js");

module.exports = [
  {
    files: ["*.cjs", "**/*.cjs"],
    ignores: ["**/*.test.cjs", "**/*.test.js"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "commonjs",
    },
    plugins: {
      "gh-aw-custom": plugin,
    },
    rules: {
      "gh-aw-custom/require-json-parse-try-catch": "warn",
      "gh-aw-custom/require-parseInt-radix": "warn",
    },
  },
  {
    files: ["**/*.test.cjs", "**/*.test.js"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
    },
  },
];
