import { noCoreExportVariableNonStringRule } from "./rules/no-core-exportvariable-non-string";
import { noCoreSetOutputNonStringRule } from "./rules/no-core-setoutput-non-string";
import { noThrowPlainObjectRule } from "./rules/no-throw-plain-object";
import { noGithubRequestInterpolatedRouteRule } from "./rules/no-github-request-interpolated-route";
import { noJsonStringifyErrorRule } from "./rules/no-json-stringify-error";
import { noUnsafeCatchErrorPropertyRule } from "./rules/no-unsafe-catch-error-property";
import { noUnsafePromiseCatchErrorPropertyRule } from "./rules/no-unsafe-promise-catch-error-property";
import { preferGetErrorMessageRule } from "./rules/prefer-get-error-message";
import { preferNumberIsNanRule } from "./rules/prefer-number-isnan";
import { requireAsyncEntrypointCatchRule } from "./rules/require-async-entrypoint-catch";
import { requireAwaitCoreSummaryWriteRule } from "./rules/require-await-core-summary-write";
import { requireFsSyncTryCatchRule } from "./rules/require-fs-sync-try-catch";
import { requireJsonParseTryCatchRule } from "./rules/require-json-parse-try-catch";
import { requireErrorCauseInRethrowRule } from "./rules/require-error-cause-in-rethrow";
import { requireParseIntRadixRule } from "./rules/require-parseInt-radix";
import { requireMkdirSyncTryCatchRule } from "./rules/require-mkdirsync-try-catch";
import { requireReturnAfterCoreSetFailedRule } from "./rules/require-return-after-core-setfailed";
import { requireSpawnSyncErrorCheckRule } from "./rules/require-spawnsync-error-check";
import { requireNewUrlTryCatchRule } from "./rules/require-new-url-try-catch";
import { preferCoreLoggingRule } from "./rules/prefer-core-logging";
import { noCoreErrorThenProcessExitRule } from "./rules/no-core-error-then-process-exit";
import { noExecInterpolatedCommandRule } from "./rules/no-exec-interpolated-command";

const plugin = {
  meta: {
    name: "@github/gh-aw-eslint-factory",
    version: "0.1.0",
  },
  rules: {
    "no-core-exportvariable-non-string": noCoreExportVariableNonStringRule,
    "no-core-setoutput-non-string": noCoreSetOutputNonStringRule,
    "no-throw-plain-object": noThrowPlainObjectRule,
    "no-github-request-interpolated-route": noGithubRequestInterpolatedRouteRule,
    "no-json-stringify-error": noJsonStringifyErrorRule,
    "no-unsafe-catch-error-property": noUnsafeCatchErrorPropertyRule,
    "no-unsafe-promise-catch-error-property": noUnsafePromiseCatchErrorPropertyRule,
    "prefer-get-error-message": preferGetErrorMessageRule,
    "prefer-number-isnan": preferNumberIsNanRule,
    "require-async-entrypoint-catch": requireAsyncEntrypointCatchRule,
    "require-await-core-summary-write": requireAwaitCoreSummaryWriteRule,
    "require-error-cause-in-rethrow": requireErrorCauseInRethrowRule,
    "require-fs-sync-try-catch": requireFsSyncTryCatchRule,
    "require-json-parse-try-catch": requireJsonParseTryCatchRule,
    "require-mkdirsync-try-catch": requireMkdirSyncTryCatchRule,
    "require-parseInt-radix": requireParseIntRadixRule,
    "require-return-after-core-setfailed": requireReturnAfterCoreSetFailedRule,
    "require-spawnsync-error-check": requireSpawnSyncErrorCheckRule,
    "require-new-url-try-catch": requireNewUrlTryCatchRule,
    "prefer-core-logging": preferCoreLoggingRule,
    "no-core-error-then-process-exit": noCoreErrorThenProcessExitRule,
    "no-exec-interpolated-command": noExecInterpolatedCommandRule,
  },
};

export = plugin;
