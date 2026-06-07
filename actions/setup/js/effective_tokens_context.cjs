// @ts-check

const fs = require("fs");
const path = require("path");
const { parsePositiveEffectiveTokenLimitString } = require("./effective_token_limits.cjs");
const { isMaxEffectiveTokensExceededError } = require("./effective_tokens_hard_rail.cjs");

const MAX_EFFECTIVE_TOKENS_FIELDS = new Set(["max_effective_tokens", "maxEffectiveTokens"]);
const EFFECTIVE_TOKENS_FIELDS = new Set(["effective_tokens", "effectiveTokens"]);
const EFFECTIVE_TOKENS_RATE_LIMIT_ERROR_FIELDS = new Set(["effective_tokens_rate_limit_error", "effectiveTokensRateLimitError"]);
const EFFECTIVE_TOKENS_RATE_LIMIT_TEXT_FIELDS = new Set(["error", "message", "reason", "details", "detail"]);
const EFFECTIVE_TOKENS_RATE_LIMIT_PATTERNS = [
  /effective[\s_-]*tokens?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i,
  /(?:rate[\s-]*limit|too many requests).*(?:effective[\s_-]*tokens?|et budget)/i,
  /\b429\b[\s\S]{0,120}(?:rate[\s-]*limit|too many requests|effective[\s_-]*tokens?|et budget)/i,
];

const MAX_AI_CREDITS_FIELDS = new Set(["max_ai_credits", "maxAiCredits"]);
const AI_CREDITS_FIELDS = new Set(["ai_credits", "aiCredits"]);
const AI_CREDITS_RATE_LIMIT_ERROR_FIELDS = new Set(["ai_credits_rate_limit_error", "aiCreditsRateLimitError"]);
const AI_CREDITS_RATE_LIMIT_TEXT_FIELDS = new Set(["error", "message", "reason", "details", "detail", "type", "code"]);
const AI_CREDITS_RATE_LIMIT_PATTERNS = [/ai[\s_-]*credits?.*(?:rate[\s-]*limit|limit exceeded|budget exceeded|exceeded)/i, /(?:rate[\s-]*limit|too many requests).*(?:ai[\s_-]*credits?)/i, /\bai_credits_limit_exceeded\b/i];
const EFFECTIVE_TOKENS_PER_AI_CREDIT = 10000;

const AWF_REFLECT_RELATIVE_PATH = path.join("sandbox", "firewall", "awf-reflect.json");

/** @param {unknown} value */
function parsePositiveIntegerString(value) {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return String(Math.trunc(value));
  }
  if (typeof value === "string" && /^\d+$/.test(value) && Number.parseInt(value, 10) > 0) {
    return value;
  }
  return "";
}

/** @param {unknown} value */
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

function isIntegerStringGreaterThanOrEqual(left, right) {
  if (!left || !right) return false;
  try {
    return BigInt(left) >= BigInt(right);
  } catch {
    return false;
  }
}

function isNumberStringGreaterThanOrEqual(left, right) {
  if (!left || !right) return false;
  const leftNumber = Number.parseFloat(left);
  const rightNumber = Number.parseFloat(right);
  return Number.isFinite(leftNumber) && Number.isFinite(rightNumber) && leftNumber >= rightNumber;
}

function shouldReportEffectiveTokensRateLimitError(hasRateLimitSignal, effectiveTokens, maxEffectiveTokens) {
  if (!hasRateLimitSignal) return false;
  if (!effectiveTokens || !maxEffectiveTokens) return true;
  return isIntegerStringGreaterThanOrEqual(effectiveTokens, maxEffectiveTokens);
}

function shouldReportAICreditsRateLimitError(hasRateLimitSignal, aiCredits, maxAICredits) {
  if (!hasRateLimitSignal) return false;
  if (!aiCredits || !maxAICredits) return true;
  return isNumberStringGreaterThanOrEqual(aiCredits, maxAICredits);
}

/**
 * Derive an ET-equivalent budget from AI credits when only max-ai-credits is available.
 * Conversion uses 1 AIC = 10,000 ET and rounds down to avoid over-reporting ET capacity.
 * @param {string} maxAICredits
 * @returns {string}
 */
function deriveMaxEffectiveTokensFromAICredits(maxAICredits) {
  if (!maxAICredits) return "";
  const normalized = String(maxAICredits).trim();
  const decimalMatch = normalized.match(/^(\d+)(?:\.(\d+))?$/);
  if (decimalMatch) {
    const wholePart = decimalMatch[1];
    const fractionalPart = decimalMatch[2] || "";
    // Scale to ET units (1 AIC = 10,000 ET) by right-padding fractional digits.
    const scaledFraction = (fractionalPart + "0000").slice(0, 4);
    try {
      const converted = BigInt(wholePart) * BigInt(EFFECTIVE_TOKENS_PER_AI_CREDIT) + BigInt(scaledFraction || "0");
      if (converted > 0n) {
        return converted.toString();
      }
    } catch {
      // Fall through to numeric parsing.
    }
  }

  // Fallback for numeric strings not matched by decimal regex (for example scientific notation).
  const parsed = Number.parseFloat(normalized);
  if (!Number.isFinite(parsed) || parsed <= 0) return "";
  const converted = Math.floor(parsed * EFFECTIVE_TOKENS_PER_AI_CREDIT);
  if (!Number.isSafeInteger(converted) || converted <= 0) return "";
  return String(converted);
}

/** @param {unknown} value */
function isTrueLike(value) {
  return value === true || value === "true" || value === 1 || value === "1";
}

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

function resolveAgentStdioLogPath(stdioLogPathOverride) {
  if (stdioLogPathOverride) return stdioLogPathOverride;
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (agentOutputFile) return path.join(path.dirname(agentOutputFile), "agent-stdio.log");
  return "/tmp/gh-aw/agent-stdio.log";
}

function hasMaxEffectiveTokensExceededSignal(stdioLogPathOverride) {
  try {
    const stdioLogPath = resolveAgentStdioLogPath(stdioLogPathOverride);
    if (!fs.existsSync(stdioLogPath)) return false;
    const content = fs.readFileSync(stdioLogPath, "utf8");
    return isMaxEffectiveTokensExceededError(content);
  } catch {
    return false;
  }
}

function resolveFirewallReflectPath() {
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (agentOutputFile) return path.join(path.dirname(agentOutputFile), AWF_REFLECT_RELATIVE_PATH);
  return "/tmp/gh-aw/sandbox/firewall/awf-reflect.json";
}

function parseEffectiveTokensFromReflectFile() {
  try {
    const reflectPath = resolveFirewallReflectPath();
    if (!fs.existsSync(reflectPath)) return { effectiveTokens: "", maxEffectiveTokens: "" };
    const content = fs.readFileSync(reflectPath, "utf8");
    if (!content.trim()) return { effectiveTokens: "", maxEffectiveTokens: "" };
    const parsed = JSON.parse(content);
    const effectiveTokens = parsePositiveIntegerString(parsed?.effective_tokens?.total_effective_tokens);
    const maxEffectiveTokens = parsePositiveEffectiveTokenLimitString(parsed?.effective_tokens?.max_effective_tokens);
    return { effectiveTokens, maxEffectiveTokens };
  } catch {
    return { effectiveTokens: "", maxEffectiveTokens: "" };
  }
}

function parseMaxEffectiveTokensFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return "";
  const stack = [entry];
  while (stack.length > 0) {
    const node = stack.pop();
    if (!node || typeof node !== "object") continue;
    for (const [key, value] of Object.entries(node)) {
      if (MAX_EFFECTIVE_TOKENS_FIELDS.has(key)) {
        const parsed = parsePositiveEffectiveTokenLimitString(value);
        if (parsed) return parsed;
      }
      if (value && typeof value === "object") stack.push(value);
    }
  }
  return "";
}

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

function parseEffectiveTokensErrorInfoFromAuditEntry(entry) {
  if (!entry || typeof entry !== "object") return { effectiveTokens: "", rateLimitError: false };
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
      if (EFFECTIVE_TOKENS_RATE_LIMIT_ERROR_FIELDS.has(key) && isTrueLike(value)) rateLimitError = true;
      if (EFFECTIVE_TOKENS_RATE_LIMIT_TEXT_FIELDS.has(key) && typeof value === "string") {
        if (EFFECTIVE_TOKENS_RATE_LIMIT_PATTERNS.some(pattern => pattern.test(value))) rateLimitError = true;
      }
      if (value && typeof value === "object") stack.push(value);
    }
  }
  return { effectiveTokens, rateLimitError };
}

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

function parseMaxEffectiveTokensFromAuditLog(auditJsonlPathOverride) {
  try {
    const auditJsonlPath = resolveFirewallAuditLogPath(auditJsonlPathOverride);
    if (!fs.existsSync(auditJsonlPath)) return "";
    const content = fs.readFileSync(auditJsonlPath, "utf8");
    if (!content.trim() || !/(?:max_effective_tokens|maxEffectiveTokens)/.test(content)) return "";
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
        if (parsed.effectiveTokens) parsedEffectiveTokens = parsed.effectiveTokens;
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

function resolveEffectiveTokensFailureState() {
  const parsedEffectiveTokensErrorInfo = parseEffectiveTokensErrorInfoFromAuditLog();
  const parsedEffectiveTokensFromReflect = parseEffectiveTokensFromReflectFile();
  const envEffectiveTokens = parsePositiveIntegerString(process.env.GH_AW_EFFECTIVE_TOKENS);
  const envMaxAICredits = parsePositiveNumberString(process.env.GH_AW_MAX_AI_CREDITS);
  const effectiveTokens = parsedEffectiveTokensErrorInfo.effectiveTokens || parsedEffectiveTokensFromReflect.effectiveTokens || envEffectiveTokens || "";
  const maxAICredits = parseMaxAICreditsFromAuditLog() || envMaxAICredits || "";
  // max_effective_tokens is deprecated for ET failure-state reconciliation.
  // We only derive the ET ceiling from max_ai_credits.
  const maxEffectiveTokens = deriveMaxEffectiveTokensFromAICredits(maxAICredits);
  const rawEffectiveTokensRateLimitError = parsedEffectiveTokensErrorInfo.rateLimitError || hasMaxEffectiveTokensExceededSignal() || process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR === "true";
  const effectiveTokensRateLimitError = shouldReportEffectiveTokensRateLimitError(rawEffectiveTokensRateLimitError, effectiveTokens, maxEffectiveTokens);
  return { effectiveTokens, maxEffectiveTokens, effectiveTokensRateLimitError };
}

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
  parseMaxEffectiveTokensFromAuditLog,
  parseEffectiveTokensErrorInfoFromAuditLog,
  parseMaxAICreditsFromAuditLog,
  parseAICreditsErrorInfoFromAuditLog,
  hasMaxEffectiveTokensExceededSignal,
  resolveEffectiveTokensFailureState,
  resolveAICreditsFailureState,
};
