// @ts-check

const fs = require("fs");
const path = require("path");

const MAX_AI_CREDITS_FIELDS = new Set(["max_ai_credits", "maxAiCredits"]);
const AI_CREDITS_FIELDS = new Set(["ai_credits", "aiCredits"]);
const AI_CREDITS_RATE_LIMIT_ERROR_FIELDS = new Set(["ai_credits_rate_limit_error", "aiCreditsRateLimitError"]);
// Note: these text fields are intentionally broad (common field names like "error", "message") because
// rate-limit signals can appear in any of them. This asymmetry vs parseMaxAICreditsFromAuditLog is deliberate.
const AI_CREDITS_RATE_LIMIT_TEXT_FIELDS = new Set(["error", "message", "reason", "details", "detail", "type", "code"]);
const AI_CREDITS_RATE_LIMIT_PATTERNS = [/ai[\s_-]*credits?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i, /(?:rate[\s-]*limit|too many requests).*(?:ai[\s_-]*credits?)/i, /\bai_credits_limit_exceeded\b/i];

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
 * @param {unknown} entry
 * @returns {string}
 */
function parseMaxAICreditsFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return "";
  const stack = [entry];
  while (stack.length > 0) {
    const node = stack.pop();
    if (!node || typeof node !== "object") continue;
    for (const [key, value] of Object.entries(node)) {
      if (MAX_AI_CREDITS_FIELDS.has(key)) {
        const parsed = parsePositiveNumberString(value);
        if (parsed) return parsed;
      }
      if (value && typeof value === "object") stack.push(value);
    }
  }
  return "";
}

/**
 * @param {unknown} entry
 * @returns {{ aiCredits: string, rateLimitError: boolean }}
 */
function parseAICreditsErrorInfoFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return { aiCredits: "", rateLimitError: false };
  const stack = [entry];
  let aiCredits = "";
  let rateLimitError = false;
  while (stack.length > 0) {
    const node = stack.pop();
    if (!node || typeof node !== "object") continue;
    for (const [key, value] of Object.entries(node)) {
      if (AI_CREDITS_FIELDS.has(key)) {
        const parsed = parsePositiveNumberString(value);
        if (parsed) aiCredits = parsed;
      }
      if (AI_CREDITS_RATE_LIMIT_ERROR_FIELDS.has(key) && isTrueLike(value)) rateLimitError = true;
      if (AI_CREDITS_RATE_LIMIT_TEXT_FIELDS.has(key) && typeof value === "string") {
        if (AI_CREDITS_RATE_LIMIT_PATTERNS.some(pattern => pattern.test(value))) rateLimitError = true;
      }
      if (value && typeof value === "object") stack.push(value);
    }
  }
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
 * Single-pass combined read of the audit log, returning all AI credits fields at once.
 * Used by resolveAICreditsFailureState to avoid reading the same file twice.
 * No contentGuard is applied: rate-limit signal detection must scan all entries anyway,
 * so a single full pass is cheaper than two guarded passes.
 *
 * @param {string} [auditJsonlPathOverride]
 * @returns {{ aiCredits: string, maxAICredits: string, rateLimitError: boolean }}
 */
function parseAuditLogCombined(auditJsonlPathOverride) {
  /** @type {{ aiCredits: string, maxAICredits: string, rateLimitError: boolean }} */
  const initial = { aiCredits: "", maxAICredits: "", rateLimitError: false };
  return iterateAuditEntries(auditJsonlPathOverride, initial, null, (acc, entry) => {
    const errorInfo = parseAICreditsErrorInfoFromAuditEntry(entry);
    const max = parseMaxAICreditsFromAuditEntry(entry);
    return {
      aiCredits: errorInfo.aiCredits || acc.aiCredits,
      maxAICredits: max || acc.maxAICredits,
      rateLimitError: acc.rateLimitError || errorInfo.rateLimitError,
    };
  });
}

/**
 * @returns {{ aiCredits: string, maxAICredits: string, aiCreditsRateLimitError: boolean }}
 */
function resolveAICreditsFailureState() {
  const { aiCredits: auditAICredits, maxAICredits: auditMaxAICredits, rateLimitError: auditRateLimitError } = parseAuditLogCombined();
  const envAICredits = parsePositiveNumberString(process.env.GH_AW_AIC);
  const envMaxAICredits = parsePositiveNumberString(process.env.GH_AW_MAX_AI_CREDITS);
  const aiCredits = auditAICredits || envAICredits || "";
  const maxAICredits = auditMaxAICredits || envMaxAICredits || "";
  const rawAICreditsRateLimitError = auditRateLimitError || process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR === "true";
  const aiCreditsRateLimitError = shouldReportAICreditsRateLimitError(rawAICreditsRateLimitError, aiCredits, maxAICredits);
  return { aiCredits, maxAICredits, aiCreditsRateLimitError };
}

module.exports = {
  resolveFirewallAuditLogPath,
  parseMaxAICreditsFromAuditLog,
  parseAICreditsErrorInfoFromAuditLog,
  resolveAICreditsFailureState,
};
