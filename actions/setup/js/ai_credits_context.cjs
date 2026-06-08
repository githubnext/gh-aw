// @ts-check

const fs = require("fs");
const path = require("path");

const MAX_AI_CREDITS_FIELDS = new Set(["max_ai_credits", "maxAiCredits"]);
const AI_CREDITS_FIELDS = new Set(["ai_credits", "aiCredits"]);
const AI_CREDITS_RATE_LIMIT_ERROR_FIELDS = new Set(["ai_credits_rate_limit_error", "aiCreditsRateLimitError"]);
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
    const logPath = path.join(base, "log.jsonl");
    if (fs.existsSync(logPath)) return logPath;
    const auditPath = path.join(base, "audit.jsonl");
    if (fs.existsSync(auditPath)) return auditPath;
  }
  return path.join(candidateBases[0] || "/tmp/gh-aw/sandbox/firewall/audit", "log.jsonl");
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
 * @param {string} [auditJsonlPathOverride]
 * @returns {string}
 */
function parseMaxAICreditsFromAuditLog(auditJsonlPathOverride) {
  try {
    const auditJsonlPath = resolveFirewallAuditLogPath(auditJsonlPathOverride);
    if (!fs.existsSync(auditJsonlPath)) return "";
    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim() || !/(?:max_ai_credits|maxAiCredits)/.test(content)) return "";
    let parsedMaxAICredits = "";
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed[0] !== "{") continue;
      try {
        const entry = JSON.parse(trimmed);
        const value = parseMaxAICreditsFromAuditEntry(entry);
        if (value) parsedMaxAICredits = value;
      } catch {
        // ignore malformed lines
      }
    }
    return parsedMaxAICredits;
  } catch {
    return "";
  }
}

/**
 * @param {string} [auditJsonlPathOverride]
 * @returns {{ aiCredits: string, rateLimitError: boolean }}
 */
function parseAICreditsErrorInfoFromAuditLog(auditJsonlPathOverride) {
  try {
    const auditJsonlPath = resolveFirewallAuditLogPath(auditJsonlPathOverride);
    if (!fs.existsSync(auditJsonlPath)) return { aiCredits: "", rateLimitError: false };
    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim()) return { aiCredits: "", rateLimitError: false };
    let parsedAICredits = "";
    let hasRateLimitError = false;
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed[0] !== "{") continue;
      try {
        const entry = JSON.parse(trimmed);
        const parsed = parseAICreditsErrorInfoFromAuditEntry(entry);
        if (parsed.aiCredits) parsedAICredits = parsed.aiCredits;
        if (parsed.rateLimitError) hasRateLimitError = true;
      } catch {
        // ignore malformed lines
      }
    }
    return { aiCredits: parsedAICredits, rateLimitError: hasRateLimitError };
  } catch {
    return { aiCredits: "", rateLimitError: false };
  }
}

/**
 * @returns {{ aiCredits: string, maxAICredits: string, aiCreditsRateLimitError: boolean }}
 */
function resolveAICreditsFailureState() {
  const parsedAICreditsErrorInfo = parseAICreditsErrorInfoFromAuditLog();
  const envAICredits = parsePositiveNumberString(process.env.GH_AW_AIC);
  const envMaxAICredits = parsePositiveNumberString(process.env.GH_AW_MAX_AI_CREDITS);
  const aiCredits = parsedAICreditsErrorInfo.aiCredits || envAICredits || "";
  const maxAICredits = parseMaxAICreditsFromAuditLog() || envMaxAICredits || "";
  const rawAICreditsRateLimitError = parsedAICreditsErrorInfo.rateLimitError || process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR === "true";
  const aiCreditsRateLimitError = shouldReportAICreditsRateLimitError(rawAICreditsRateLimitError, aiCredits, maxAICredits);
  return { aiCredits, maxAICredits, aiCreditsRateLimitError };
}

module.exports = {
  resolveFirewallAuditLogPath,
  parseMaxAICreditsFromAuditLog,
  parseAICreditsErrorInfoFromAuditLog,
  resolveAICreditsFailureState,
};
