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
      "gh-aw-custom/no-core-exportvariable-non-string": "warn",
      "gh-aw-custom/no-core-setoutput-non-string": "warn",
      "gh-aw-custom/no-throw-plain-object": "warn",
      "gh-aw-custom/no-github-request-interpolated-route": "warn",
      "gh-aw-custom/no-json-stringify-error": "warn",
      "gh-aw-custom/no-unsafe-catch-error-property": "warn",
      "gh-aw-custom/no-unsafe-promise-catch-error-property": "warn",
      "gh-aw-custom/prefer-get-error-message": "warn",
      "gh-aw-custom/prefer-number-isnan": "warn",
      "gh-aw-custom/require-async-entrypoint-catch": "warn",
      "gh-aw-custom/require-await-core-summary-write": "warn",
      "gh-aw-custom/require-error-cause-in-rethrow": "warn",
      "gh-aw-custom/require-fs-sync-try-catch": "warn",
      "gh-aw-custom/require-json-parse-try-catch": "warn",
      "gh-aw-custom/require-parseInt-radix": "warn",
      "gh-aw-custom/require-return-after-core-setfailed": "warn",
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
