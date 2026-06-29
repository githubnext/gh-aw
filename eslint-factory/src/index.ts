import { noUnsafeCatchErrorPropertyRule } from "./rules/no-unsafe-catch-error-property";
import { noUnsafePromiseCatchErrorPropertyRule } from "./rules/no-unsafe-promise-catch-error-property";
import { requireJsonParseTryCatchRule } from "./rules/require-json-parse-try-catch";
import { requireParseIntRadixRule } from "./rules/require-parseInt-radix";

const plugin = {
  meta: {
    name: "@github/gh-aw-eslint-factory",
    version: "0.1.0",
  },
  rules: {
    "no-unsafe-catch-error-property": noUnsafeCatchErrorPropertyRule,
    "no-unsafe-promise-catch-error-property": noUnsafePromiseCatchErrorPropertyRule,
    "require-json-parse-try-catch": requireJsonParseTryCatchRule,
    "require-parseInt-radix": requireParseIntRadixRule,
  },
};

export = plugin;
