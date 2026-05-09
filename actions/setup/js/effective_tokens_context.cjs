// @ts-check

const fs = require("fs");
const path = require("path");

const MAX_EFFECTIVE_TOKENS_FIELDS = new Set(["max_effective_tokens", "maxEffectiveTokens"]);
const EFFECTIVE_TOKENS_FIELDS = new Set(["effective_tokens", "effectiveTokens"]);
const EFFECTIVE_TOKENS_RATE_LIMIT_ERROR_FIELDS = new Set(["effective_tokens_rate_limit_error", "effectiveTokensRateLimitError"]);
const EFFECTIVE_TOKENS_RATE_LIMIT_TEXT_FIELDS = new Set(["error", "message", "reason", "details", "detail"]);
// Effective-token rate-limit indicators seen in runtime/audit payload text, e.g.:
// - "effective_tokens limit exceeded"
// - "rate limit ... effective tokens"
// - "429 too many requests ... ET budget"
// Keep these patterns permissive because providers vary wording across error payloads.
const EFFECTIVE_TOKENS_RATE_LIMIT_PATTERNS = [
  /effective[\s_-]*tokens?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i,
  /(?:rate[\s-]*limit|too many requests).*(?:effective[\s_-]*tokens?|et budget)/i,
  /\b429\b[\s\S]{0,120}(?:rate[\s-]*limit|too many requests|effective[\s_-]*tokens?|et budget)/i,
];
const AWF_REFLECT_RELATIVE_PATH = path.join("sandbox", "firewall", "awf-reflect.json");

/**
 * @param {unknown} value
 * @returns {string}
 */
function parsePositiveIntegerString(value) {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return String(Math.trunc(value));
  }
  if (typeof value === "string" && /^\d+$/.test(value) && Number.parseInt(value, 10) > 0) {
    return value;
  }
  return "";
}

/**
 * Compare two integer strings using BigInt.
 * Returns false when either value is missing or cannot be parsed as an integer.
 *
 * @param {string} left
 * @param {string} right
 * @returns {boolean}
 */
function isIntegerStringGreaterThanOrEqual(left, right) {
  if (!left || !right) {
    return false;
  }

  try {
    return BigInt(left) >= BigInt(right);
  } catch {
    return false;
  }
}

/**
 * Decide whether an ET rate-limit signal should be surfaced as budget exhaustion.
 * A missing signal always means "no". When the signal is present but one of the
 * token counts is unavailable, keep reporting the condition; otherwise require the
 * effective-token count to meet or exceed the configured max.
 *
 * @param {boolean} hasRateLimitSignal
 * @param {string} effectiveTokens
 * @param {string} maxEffectiveTokens
 * @returns {boolean}
 */
function shouldReportEffectiveTokensRateLimitError(hasRateLimitSignal, effectiveTokens, maxEffectiveTokens) {
  if (!hasRateLimitSignal) {
    return false;
  }

  if (!effectiveTokens || !maxEffectiveTokens) {
    // Conservative fallback: when a rate-limit signal exists but the numeric budget
    // values are unavailable, keep surfacing the ET failure instead of suppressing it.
    return true;
  }

  return isIntegerStringGreaterThanOrEqual(effectiveTokens, maxEffectiveTokens);
}

/**
 * @param {unknown} value
 * @returns {boolean}
 */
function isTrueLike(value) {
  return value === true || value === "true" || value === 1 || value === "1";
}

/**
 * Resolve the AWF firewall audit log path.
 * Newer runs write `log.jsonl`; older runs use `audit.jsonl`.
 *
 * @param {string} [auditJsonlPathOverride]
 * @returns {string}
 */
function resolveFirewallAuditLogPath(auditJsonlPathOverride) {
  if (auditJsonlPathOverride) return auditJsonlPathOverride;

  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  const candidateBases = [];
  if (agentOutputFile) {
    candidateBases.push(path.join(path.dirname(agentOutputFile), "sandbox", "firewall", "audit"));
  }
  candidateBases.push("/tmp/gh-aw/sandbox/firewall/audit");

  for (const base of candidateBases) {
    const logPath = path.join(base, "log.jsonl");
    if (fs.existsSync(logPath)) return logPath;
    const auditPath = path.join(base, "audit.jsonl");
    if (fs.existsSync(auditPath)) return auditPath;
  }

  // Default to the latest expected location/name.
  return path.join(candidateBases[0] || "/tmp/gh-aw/sandbox/firewall/audit", "log.jsonl");
}

/**
 * Resolve the AWF firewall reflect file path.
 *
 * @returns {string}
 */
function resolveFirewallReflectPath() {
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (agentOutputFile) {
    return path.join(path.dirname(agentOutputFile), AWF_REFLECT_RELATIVE_PATH);
  }
  return "/tmp/gh-aw/sandbox/firewall/awf-reflect.json";
}

/**
 * Parse effective token totals from AWF firewall reflect JSON.
 *
 * @returns {{effectiveTokens: string, maxEffectiveTokens: string}}
 */
function parseEffectiveTokensFromReflectFile() {
  try {
    const reflectPath = resolveFirewallReflectPath();
    if (!fs.existsSync(reflectPath)) {
      return { effectiveTokens: "", maxEffectiveTokens: "" };
    }

    const content = fs.readFileSync(reflectPath, "utf8");
    if (!content.trim()) {
      return { effectiveTokens: "", maxEffectiveTokens: "" };
    }

    const parsed = JSON.parse(content);
    const effectiveTokens = parsePositiveIntegerString(parsed?.effective_tokens?.total_effective_tokens);
    const maxEffectiveTokens = parsePositiveIntegerString(parsed?.effective_tokens?.max_effective_tokens);
    return { effectiveTokens, maxEffectiveTokens };
  } catch {
    return { effectiveTokens: "", maxEffectiveTokens: "" };
  }
}

/**
 * Parse max effective tokens from a single AWF audit log entry object.
 * Accepts both snake_case and camelCase field names.
 *
 * @param {unknown} entry
 * @returns {string}
 */
function parseMaxEffectiveTokensFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return "";

  /** @type {unknown[]} */
  const stack = [entry];
  while (stack.length > 0) {
    const node = stack.pop();
    if (!node || typeof node !== "object") continue;
    for (const [key, value] of Object.entries(node)) {
      if (MAX_EFFECTIVE_TOKENS_FIELDS.has(key)) {
        const parsed = parsePositiveIntegerString(value);
        if (parsed) return parsed;
      }
      if (value && typeof value === "object") {
        stack.push(value);
      }
    }
  }

  return "";
}

/**
 * Parse effective token error metadata from a single AWF audit log entry object.
 * Accepts both snake_case and camelCase field names.
 *
 * @param {unknown} entry
 * @returns {{effectiveTokens: string, rateLimitError: boolean}}
 */
function parseEffectiveTokensErrorInfoFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return { effectiveTokens: "", rateLimitError: false };

  /** @type {unknown[]} */
  const stack = [entry];
  let effectiveTokens = "";
  let rateLimitError = false;

  while (stack.length > 0) {
    const node = stack.pop();
    if (!node || typeof node !== "object") continue;

    for (const [key, value] of Object.entries(node)) {
      if (EFFECTIVE_TOKENS_FIELDS.has(key)) {
        const parsed = parsePositiveIntegerString(value);
        if (parsed) effectiveTokens = parsed;
      }

      if (EFFECTIVE_TOKENS_RATE_LIMIT_ERROR_FIELDS.has(key)) {
        if (isTrueLike(value)) {
          rateLimitError = true;
        }
      }

      if (EFFECTIVE_TOKENS_RATE_LIMIT_TEXT_FIELDS.has(key) && typeof value === "string") {
        if (EFFECTIVE_TOKENS_RATE_LIMIT_PATTERNS.some(pattern => pattern.test(value))) {
          rateLimitError = true;
        }
      }

      if (value && typeof value === "object") {
        stack.push(value);
      }
    }
  }

  return { effectiveTokens, rateLimitError };
}

/**
 * Parse max effective tokens from AWF firewall audit JSONL.
 *
 * @param {string} [auditJsonlPathOverride]
 * @returns {string}
 */
function parseMaxEffectiveTokensFromAuditLog(auditJsonlPathOverride) {
  try {
    const auditJsonlPath = resolveFirewallAuditLogPath(auditJsonlPathOverride);
    if (!fs.existsSync(auditJsonlPath)) return "";

    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim()) return "";
    if (!/(?:max_effective_tokens|maxEffectiveTokens)/.test(content)) return "";

    let parsedMaxEffectiveTokens = "";
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed[0] !== "{") continue;

      try {
        const entry = JSON.parse(trimmed);
        const value = parseMaxEffectiveTokensFromAuditEntry(entry);
        if (value) parsedMaxEffectiveTokens = value;
      } catch {
        // ignore malformed lines
      }
    }

    return parsedMaxEffectiveTokens;
  } catch {
    return "";
  }
}

/**
 * Parse effective token error metadata from AWF firewall audit JSONL.
 *
 * @param {string} [auditJsonlPathOverride]
 * @returns {{effectiveTokens: string, rateLimitError: boolean}}
 */
function parseEffectiveTokensErrorInfoFromAuditLog(auditJsonlPathOverride) {
  try {
    const auditJsonlPath = resolveFirewallAuditLogPath(auditJsonlPathOverride);
    if (!fs.existsSync(auditJsonlPath)) return { effectiveTokens: "", rateLimitError: false };

    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim()) return { effectiveTokens: "", rateLimitError: false };

    let parsedEffectiveTokens = "";
    let hasRateLimitError = false;

    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed[0] !== "{") continue;

      try {
        const entry = JSON.parse(trimmed);
        const parsed = parseEffectiveTokensErrorInfoFromAuditEntry(entry);
        // AWF audit logs are append-only JSONL; later entries represent newer state.
        if (parsed.effectiveTokens) parsedEffectiveTokens = parsed.effectiveTokens;
        // Sticky OR: any detected ET rate-limit signal is enough to report this failure mode.
        if (parsed.rateLimitError) hasRateLimitError = true;
      } catch {
        // ignore malformed lines
      }
    }

    return { effectiveTokens: parsedEffectiveTokens, rateLimitError: hasRateLimitError };
  } catch {
    return { effectiveTokens: "", rateLimitError: false };
  }
}

/**
 * Compute effective-token failure state with audit JSONL values preferred over env fallbacks.
 * @returns {{effectiveTokens: string, maxEffectiveTokens: string, effectiveTokensRateLimitError: boolean}}
 */
function resolveEffectiveTokensFailureState() {
  const parsedEffectiveTokensErrorInfo = parseEffectiveTokensErrorInfoFromAuditLog();
  const parsedEffectiveTokensFromReflect = parseEffectiveTokensFromReflectFile();
  // Treat invalid env fallbacks as missing so they do not produce misleading ET math.
  const envEffectiveTokens = parsePositiveIntegerString(process.env.GH_AW_EFFECTIVE_TOKENS);
  const envMaxEffectiveTokens = parsePositiveIntegerString(process.env.GH_AW_MAX_EFFECTIVE_TOKENS);
  const effectiveTokens = parsedEffectiveTokensErrorInfo.effectiveTokens || parsedEffectiveTokensFromReflect.effectiveTokens || envEffectiveTokens || "";
  const maxEffectiveTokens = parseMaxEffectiveTokensFromAuditLog() || parsedEffectiveTokensFromReflect.maxEffectiveTokens || envMaxEffectiveTokens || "";
  const rawEffectiveTokensRateLimitError = parsedEffectiveTokensErrorInfo.rateLimitError || process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR === "true";
  const effectiveTokensRateLimitError = shouldReportEffectiveTokensRateLimitError(rawEffectiveTokensRateLimitError, effectiveTokens, maxEffectiveTokens);

  return {
    effectiveTokens,
    maxEffectiveTokens,
    effectiveTokensRateLimitError,
  };
}

module.exports = {
  resolveFirewallAuditLogPath,
  parseMaxEffectiveTokensFromAuditLog,
  parseEffectiveTokensErrorInfoFromAuditLog,
  resolveEffectiveTokensFailureState,
};
