import { noCoreSetOutputNonStringRule } from "./rules/no-core-setoutput-non-string";
import { noJsonStringifyErrorRule } from "./rules/no-json-stringify-error";
import { noUnsafeCatchErrorPropertyRule } from "./rules/no-unsafe-catch-error-property";
import { noUnsafePromiseCatchErrorPropertyRule } from "./rules/no-unsafe-promise-catch-error-property";
import { preferGetErrorMessageRule } from "./rules/prefer-get-error-message";
import { preferNumberIsNanRule } from "./rules/prefer-number-isnan";
import { requireAsyncEntrypointCatchRule } from "./rules/require-async-entrypoint-catch";
import { requireAwaitCoreSummaryWriteRule } from "./rules/require-await-core-summary-write";
import { requireJsonParseTryCatchRule } from "./rules/require-json-parse-try-catch";
import { requireParseIntRadixRule } from "./rules/require-parseInt-radix";

const plugin = {
  meta: {
    name: "@github/gh-aw-eslint-factory",
    version: "0.1.0",
  },
  rules: {
    "no-core-setoutput-non-string": noCoreSetOutputNonStringRule,
    "no-json-stringify-error": noJsonStringifyErrorRule,
    "no-unsafe-catch-error-property": noUnsafeCatchErrorPropertyRule,
    "no-unsafe-promise-catch-error-property": noUnsafePromiseCatchErrorPropertyRule,
    "prefer-get-error-message": preferGetErrorMessageRule,
    "prefer-number-isnan": preferNumberIsNanRule,
    "require-async-entrypoint-catch": requireAsyncEntrypointCatchRule,
    "require-await-core-summary-write": requireAwaitCoreSummaryWriteRule,
    "require-json-parse-try-catch": requireJsonParseTryCatchRule,
    "require-parseInt-radix": requireParseIntRadixRule,
  },
};

export = plugin;
