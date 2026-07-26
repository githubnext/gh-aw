// @ts-check
require("./shim.cjs");

const fs = require("fs");
const path = require("path");

const MAX_AI_CREDITS_FIELDS = new Set(["max_ai_credits", "maxAiCredits"]);
const AI_CREDITS_FIELDS = new Set(["ai_credits", "aiCredits"]);
const AI_CREDITS_RATE_LIMIT_ERROR_FIELDS = new Set(["ai_credits_rate_limit_error", "aiCreditsRateLimitError"]);
// Note: these text fields are intentionally broad (common field names like "error", "message") because
// rate-limit signals can appear in any of them. This asymmetry vs parseMaxAICreditsFromAuditLog is deliberate.
const AI_CREDITS_RATE_LIMIT_TEXT_FIELDS = new Set(["error", "message", "reason", "details", "detail", "type", "code"]);
const AI_CREDITS_RATE_LIMIT_PATTERNS = [/ai[\s_-]*credits?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i, /(?:rate[\s-]*limit|too many requests).*(?:ai[\s_-]*credits?)/i, /\bai_credits_limit_exceeded\b/i];
const MAX_AI_CREDITS_EXCEEDED_FIELDS = new Set(["max_ai_credits_exceeded", "maxAiCreditsExceeded"]);
/** @type {{ aiCredits: string, maxAICredits: string, rateLimitError: boolean, maxAICreditsExceeded: boolean }} */
const EMPTY_AI_CREDITS_STATE = { aiCredits: "", maxAICredits: "", rateLimitError: false, maxAICreditsExceeded: false };
const BUDGET_EXCEEDED_EVENT = "budget_exceeded";
// The literal error type emitted by the AWF API proxy (HTTP 400) when maxAiCredits is active
// and the requested model is not in the built-in pricing table.
const UNKNOWN_MODEL_AI_CREDITS_TYPE = "unknown_model_ai_credits";
const MAX_AI_CREDITS_EXCEEDED_STDIO_RE = /maximum ai credits exceeded(?:\s*\((\d+(?:\.\d+)?)\s*\/\s*(\d+(?:\.\d+)?)\))?/i;
const DEFAULT_AGENT_STDIO_LOG = "/tmp/gh-aw/agent-stdio.log";
const AGENT_STDIO_LOG_MAX_TAIL = 64 * 1024; // 64 KB — sufficient for any realistic error block

/**
 * @param {unknown} value
 * @returns {string}
 */
function parsePositiveNumberString(value) {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return String(value);
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed === "") return "";
    const parsed = Number.parseFloat(trimmed);
    if (Number.isFinite(parsed) && parsed > 0) return trimmed;
  }
  return "";
}

/**
 * @param {string} left
 * @param {string} right
 * @returns {boolean}
 */
function isNumberStringGreaterThanOrEqual(left, right) {
  if (!left || !right) return false;
  const leftNumber = Number.parseFloat(left);
  const rightNumber = Number.parseFloat(right);
  return Number.isFinite(leftNumber) && Number.isFinite(rightNumber) && leftNumber >= rightNumber;
}

/**
 * @param {boolean} hasRateLimitSignal
 * @param {string} aiCredits
 * @param {string} maxAICredits
 * @returns {boolean}
 */
function shouldReportAICreditsRateLimitError(hasRateLimitSignal, aiCredits, maxAICredits) {
  if (!hasRateLimitSignal) return false;
  if (!aiCredits || !maxAICredits) return true;
  return isNumberStringGreaterThanOrEqual(aiCredits, maxAICredits);
}

/**
 * @param {unknown} value
 * @returns {boolean}
 */
function isTrueLike(value) {
  return value === true || value === "true" || value === 1 || value === "1";
}

/**
 * @param {string} [auditJsonlPathOverride]
 * @returns {string}
 */
function resolveFirewallAuditLogPath(auditJsonlPathOverride) {
  if (auditJsonlPathOverride) return auditJsonlPathOverride;
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  const candidateBases = [];
  if (agentOutputFile) {
    candidateBases.push(path.join(path.dirname(agentOutputFile), "sandbox", "firewall", "audit"));
    candidateBases.push(path.join(path.dirname(agentOutputFile), "sandbox", "firewall", "logs"));
  }
  candidateBases.push("/tmp/gh-aw/sandbox/firewall/audit");
  candidateBases.push("/tmp/gh-aw/sandbox/firewall/logs");

  for (const base of candidateBases) {
    for (const filename of ["log.jsonl", "audit.jsonl"]) {
      const candidate = path.join(base, filename);
      if (fs.existsSync(candidate)) return candidate;
    }
  }
  return path.join(candidateBases[0], "log.jsonl");
}

/**
 * Depth-first traversal of a nested object, calling visitor for each [key, value] pair.
 * Traversal stops early if visitor returns true.
 *
 * @param {unknown} entry
 * @param {(key: string, value: unknown) => boolean | void} visitor - return true to stop early
 * @returns {boolean} true if visitor stopped traversal early
 */
function traverseObjectTree(entry, visitor) {
  if (!entry || typeof entry !== "object") return false;
  const stack = [entry];
  while (stack.length > 0) {
    const node = stack.pop();
    if (!node || typeof node !== "object") continue;
    for (const [key, value] of Object.entries(node)) {
      if (visitor(key, value) === true) return true;
      if (value && typeof value === "object") stack.push(value);
    }
  }
  return false;
}

/**
 * @param {unknown} entry
 * @returns {string}
 */
function parseMaxAICreditsFromAuditEntry(entry) {
  let result = "";
  traverseObjectTree(entry, (key, value) => {
    if (MAX_AI_CREDITS_FIELDS.has(key)) {
      const parsed = parsePositiveNumberString(value);
      if (parsed) {
        result = parsed;
        return true;
      }
    }
    return false;
  });
  return result;
}

/**
 * @param {unknown} entry
 * @returns {{ aiCredits: string, rateLimitError: boolean }}
 */
function parseAICreditsErrorInfoFromAuditEntry(entry) {
  let aiCredits = "";
  let rateLimitError = false;
  traverseObjectTree(entry, (key, value) => {
    if (AI_CREDITS_FIELDS.has(key)) {
      const parsed = parsePositiveNumberString(value);
      if (parsed) aiCredits = parsed;
    }
    if (AI_CREDITS_RATE_LIMIT_ERROR_FIELDS.has(key) && isTrueLike(value)) rateLimitError = true;
    if (AI_CREDITS_RATE_LIMIT_TEXT_FIELDS.has(key) && typeof value === "string") {
      if (AI_CREDITS_RATE_LIMIT_PATTERNS.some(pattern => pattern.test(value))) rateLimitError = true;
    }
  });
  return { aiCredits, rateLimitError };
}

/**
 * Reads a firewall audit JSONL file and calls accumulate for each parsed entry.
 * Returns the accumulated result, or defaultValue on missing file or any error.
 *
 * @template T
 * @param {string | undefined} auditJsonlPathOverride
 * @param {T} defaultValue
 * @param {((content: string) => boolean) | null} contentGuard - When non-null, called with raw file
 *   content before iteration; return false to skip parsing entirely (fast-path optimization).
 * @param {(acc: T, entry: unknown) => T | undefined} accumulate - Callers should return a defined
 *   value; undefined is ignored defensively to preserve the previous accumulator.
 * @returns {T}
 */
function iterateAuditEntries(auditJsonlPathOverride, defaultValue, contentGuard, accumulate) {
  try {
    const auditJsonlPath = resolveFirewallAuditLogPath(auditJsonlPathOverride);
    if (!fs.existsSync(auditJsonlPath)) return defaultValue;
    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim()) return defaultValue;
    if (contentGuard && !contentGuard(content)) return defaultValue;
    let result = defaultValue;
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed[0] !== "{") continue;
      try {
        const nextResult = accumulate(result, JSON.parse(trimmed));
        if (nextResult !== undefined) result = nextResult;
      } catch {
        // ignore malformed lines
      }
    }
    return result;
  } catch {
    return defaultValue;
  }
}

/**
 * @param {string} [auditJsonlPathOverride]
 * @returns {string}
 */
function parseMaxAICreditsFromAuditLog(auditJsonlPathOverride) {
  return iterateAuditEntries(
    auditJsonlPathOverride,
    "",
    content => /(?:max_ai_credits|maxAiCredits)/.test(content),
    (acc, entry) => parseMaxAICreditsFromAuditEntry(entry) || acc
  );
}

/**
 * @param {string} [auditJsonlPathOverride]
 * @returns {{ aiCredits: string, rateLimitError: boolean }}
 */
function parseAICreditsErrorInfoFromAuditLog(auditJsonlPathOverride) {
  // No content-guard fast-path: the rate-limit signal appears in common field names
  // (error, message, reason…) that are present in almost every entry, making a
  // field-name pre-scan near-useless. The asymmetry vs parseMaxAICreditsFromAuditLog
  // is intentional — see AI_CREDITS_RATE_LIMIT_TEXT_FIELDS comment above.
  /** @type {{ aiCredits: string, rateLimitError: boolean }} */
  const initial = { aiCredits: "", rateLimitError: false };
  return iterateAuditEntries(auditJsonlPathOverride, initial, null, (acc, entry) => {
    const parsed = parseAICreditsErrorInfoFromAuditEntry(entry);
    return {
      aiCredits: parsed.aiCredits || acc.aiCredits,
      rateLimitError: acc.rateLimitError || parsed.rateLimitError,
    };
  });
}

/**
 * Detects a `max_ai_credits_exceeded` signal from a single firewall audit log entry.
 * Checks for the explicit `max_ai_credits_exceeded` boolean field, its camelCase variant,
 * or a `budget_exceeded` event with `reason: "hard_limit"` and `forced_termination: true`
 * as written by the aw-harness upon hard-limit abort (§11.2.2).
 * Only inspects top-level fields to avoid false positives from nested provider responses.
 *
 * @param {unknown} entry
 * @returns {boolean}
 */
function parseMaxAICreditsExceededFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return false;
  /** @type {unknown} */
  let event;
  /** @type {unknown} */
  let reason;
  /** @type {unknown} */
  let forcedTermination;

  for (const [key, value] of Object.entries(entry)) {
    if (MAX_AI_CREDITS_EXCEEDED_FIELDS.has(key) && isTrueLike(value)) return true;
    if (key === "event") event = value;
    if (key === "reason") reason = value;
    if (key === "forced_termination") forcedTermination = value;
  }
  return typeof event === "string" && event === BUDGET_EXCEEDED_EVENT && typeof reason === "string" && reason === "hard_limit" && isTrueLike(forcedTermination);
}

/**
 * @param {string} [auditJsonlPathOverride]
 * @returns {boolean}
 */
function parseMaxAICreditsExceededFromAuditLog(auditJsonlPathOverride) {
  return iterateAuditEntries(
    auditJsonlPathOverride,
    false,
    content => /(?:max_ai_credits_exceeded|maxAiCreditsExceeded|budget_exceeded)/.test(content),
    (acc, entry) => acc || parseMaxAICreditsExceededFromAuditEntry(entry)
  );
}

/**
 * @param {unknown} entry
 * @returns {boolean}
 */
function parseUnknownModelAICreditsFromAuditEntry(entry) {
  return traverseObjectTree(entry, (_key, value) => {
    if (value === UNKNOWN_MODEL_AI_CREDITS_TYPE) return true;
  });
}

/**
 * Detects an `unknown_model_ai_credits` error from the firewall audit log.
 * This HTTP 400 error is emitted by the AWF API proxy when `maxAiCredits` is active and
 * the requested model is not in the built-in pricing table and no `defaultAiCreditsPricing`
 * fallback is configured.
 *
 * @param {string} [auditJsonlPathOverride]
 * @returns {boolean}
 */
function parseUnknownModelAICreditsFromAuditLog(auditJsonlPathOverride) {
  return iterateAuditEntries(
    auditJsonlPathOverride,
    false,
    content => content.includes(UNKNOWN_MODEL_AI_CREDITS_TYPE),
    (acc, entry) => acc || parseUnknownModelAICreditsFromAuditEntry(entry)
  );
}

/**
 * Single-pass combined read of the audit log, returning all AI credits fields at once.
 * Used by resolveAICreditsFailureState to avoid reading the same file twice.
 * No contentGuard is applied: rate-limit signal detection must scan all entries anyway,
 * so a single full pass is cheaper than two guarded passes.
 *
 * @param {string} [auditJsonlPathOverride]
 * @returns {{ aiCredits: string, maxAICredits: string, rateLimitError: boolean, maxAICreditsExceeded: boolean }}
 */
function parseAuditLogCombined(auditJsonlPathOverride) {
  return iterateAuditEntries(auditJsonlPathOverride, EMPTY_AI_CREDITS_STATE, null, (acc, entry) => {
    const errorInfo = parseAICreditsErrorInfoFromAuditEntry(entry);
    const max = parseMaxAICreditsFromAuditEntry(entry);
    const maxAICreditsExceeded = parseMaxAICreditsExceededFromAuditEntry(entry);
    return {
      aiCredits: errorInfo.aiCredits || acc.aiCredits,
      maxAICredits: max || acc.maxAICredits,
      rateLimitError: acc.rateLimitError || errorInfo.rateLimitError,
      maxAICreditsExceeded: acc.maxAICreditsExceeded || maxAICreditsExceeded,
    };
  });
}

/**
 * Logs the provenance source and value for a single AI credits field.
 * Outputs exactly one line regardless of which source resolved the value.
 *
 * @param {string} label
 * @param {string} auditValue
 * @param {string} stdioValue
 * @param {string} envValue
 * @param {string} envVarName
 */
function logAICreditSource(label, auditValue, stdioValue, envValue, envVarName) {
  if (auditValue) core.info(`[ai-credits] ${label} source=audit_log value=${auditValue}`);
  else if (stdioValue) core.info(`[ai-credits] ${label} source=agent_stdio value=${stdioValue}`);
  else if (envValue) core.info(`[ai-credits] ${label} source=env(${envVarName}) value=${envValue}`);
  else core.info(`[ai-credits] ${label} source=none ${envVarName}=${process.env[envVarName] || "(unset)"}`);
}

/**
 * @param {{ logProvenance?: boolean }} [options]
 * @returns {{ aiCredits: string, maxAICredits: string, aiCreditsRateLimitError: boolean, maxAICreditsExceeded: boolean }}
 */
function resolveAICreditsFailureState({ logProvenance = true } = {}) {
  const stdioSignals = parseAICreditsExceededFromAgentStdio();
  const { aiCredits: auditAICredits, maxAICredits: auditMaxAICredits, rateLimitError: auditRateLimitError, maxAICreditsExceeded: auditMaxAICreditsExceeded } = parseAuditLogCombined();
  const envAICredits = parsePositiveNumberString(process.env.GH_AW_AIC);
  const envMaxAICredits = parsePositiveNumberString(process.env.GH_AW_MAX_AI_CREDITS);
  const envRateLimitSignal = process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR === "true";
  const envRateLimitSignalHasEvidence = envRateLimitSignal && !!(auditAICredits || stdioSignals.aiCredits || envAICredits);

  // Log provenance so failing issues can be diagnosed when credit data is missing.
  if (logProvenance) {
    logAICreditSource("aiCredits", auditAICredits, stdioSignals.aiCredits, envAICredits, "GH_AW_AIC");
    logAICreditSource("maxAICredits", auditMaxAICredits, stdioSignals.maxAICredits, envMaxAICredits, "GH_AW_MAX_AI_CREDITS");

    const rawRateLimitSignalSource = auditRateLimitError
      ? "audit_log"
      : stdioSignals.rateLimitError
        ? "agent_stdio"
        : envRateLimitSignalHasEvidence
          ? "env(GH_AW_AI_CREDITS_RATE_LIMIT_ERROR)"
          : envRateLimitSignal
            ? "env_ignored_no_ai_credits"
            : "none";
    core.info(`[ai-credits] rateLimitSignal source=${rawRateLimitSignalSource}`);
  }

  const aiCredits = auditAICredits || stdioSignals.aiCredits || envAICredits || "";
  const maxAICredits = auditMaxAICredits || stdioSignals.maxAICredits || envMaxAICredits || "";
  const rawAICreditsRateLimitError = auditRateLimitError || stdioSignals.rateLimitError || envRateLimitSignalHasEvidence;
  const aiCreditsRateLimitError = shouldReportAICreditsRateLimitError(rawAICreditsRateLimitError, aiCredits, maxAICredits);
  return { aiCredits, maxAICredits, aiCreditsRateLimitError, maxAICreditsExceeded: auditMaxAICreditsExceeded || stdioSignals.maxAICreditsExceeded };
}

/**
 * @returns {{ aiCredits: string, maxAICredits: string, rateLimitError: boolean, maxAICreditsExceeded: boolean }}
 */
function parseAICreditsExceededFromAgentStdio() {
  try {
    const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
    // Derive the stdio log path from GH_AW_AGENT_OUTPUT when set, but always
    // fall back to the well-known default so directory-valued env vars don't
    // silently break detection.
    const derivedPath = agentOutputFile ? path.join(path.dirname(agentOutputFile), "agent-stdio.log") : null;
    const stdioLogPath = derivedPath && fs.existsSync(derivedPath) ? derivedPath : DEFAULT_AGENT_STDIO_LOG;
    if (!fs.existsSync(stdioLogPath)) return EMPTY_AI_CREDITS_STATE;
    // Read only the tail to avoid OOM on large logs; the error token always
    // appears near the end of the file.
    const stat = fs.statSync(stdioLogPath);
    if (stat.size === 0) return EMPTY_AI_CREDITS_STATE;
    const readSize = Math.min(stat.size, AGENT_STDIO_LOG_MAX_TAIL);
    const buf = Buffer.alloc(readSize);
    const fd = fs.openSync(stdioLogPath, "r");
    try {
      fs.readSync(fd, buf, 0, readSize, stat.size - readSize);
    } finally {
      fs.closeSync(fd);
    }
    const content = buf.toString("utf8");
    // Use matchAll and take the last occurrence — in retried runs the final
    // entry carries the authoritative (highest) credit values.
    const RE_G = new RegExp(MAX_AI_CREDITS_EXCEEDED_STDIO_RE.source, "gi");
    const allMatches = [...content.matchAll(RE_G)];
    const match = allMatches.at(-1);
    if (!match) return EMPTY_AI_CREDITS_STATE;
    const aiCredits = parsePositiveNumberString(match[1] || "");
    const maxAICredits = parsePositiveNumberString(match[2] || "");
    return {
      aiCredits,
      maxAICredits,
      rateLimitError: true,
      maxAICreditsExceeded: true,
    };
  } catch {
    return EMPTY_AI_CREDITS_STATE;
  }
}

module.exports = {
  resolveFirewallAuditLogPath,
  parseMaxAICreditsFromAuditLog,
  parseAICreditsErrorInfoFromAuditLog,
  parseMaxAICreditsExceededFromAuditLog,
  parseUnknownModelAICreditsFromAuditLog,
  resolveAICreditsFailureState,
};
