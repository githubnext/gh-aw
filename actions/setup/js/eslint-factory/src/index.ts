import { requireJsonParseTryCatchRule } from "./rules/require-json-parse-try-catch";

const plugin = {
  meta: {
    name: "@github/gh-aw-eslint-factory",
    version: "0.1.0",
  },
  rules: {
    "require-json-parse-try-catch": requireJsonParseTryCatchRule,
  },
};

export = plugin;
