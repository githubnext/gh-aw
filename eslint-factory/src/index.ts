import { noUnsafeCatchErrorPropertyRule } from "./rules/no-unsafe-catch-error-property";
import { requireJsonParseTryCatchRule } from "./rules/require-json-parse-try-catch";
import { requireParseIntRadixRule } from "./rules/require-parseInt-radix";

const plugin = {
  meta: {
    name: "@github/gh-aw-eslint-factory",
    version: "0.1.0",
  },
  rules: {
    "no-unsafe-catch-error-property": noUnsafeCatchErrorPropertyRule,
    "require-json-parse-try-catch": requireJsonParseTryCatchRule,
    "require-parseInt-radix": requireParseIntRadixRule,
  },
};

export = plugin;
