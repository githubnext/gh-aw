// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Parse Threat Detection Results
 *
 * This module parses the threat detection results from the detection log file
 * (written by the detection copilot via tee) and determines whether any
 * security threats were detected (prompt injection, secret leak, malicious
 * patch). It sets the appropriate output and fails the workflow if threats
 * are detected.
 *
 * The detection copilot writes its verdict to stdout which is piped through
 * `tee -a` to detection.log. This parser reads that file — NOT agent_output.json
 * (which is the main agent's structured output used as *input* to the detection
 * copilot).
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { listFilesRecursively } = require("./file_helpers.cjs");
const { DETECTION_LOG_FILENAME } = require("./constants.cjs");
const { ERR_SYSTEM, ERR_PARSE, ERR_VALIDATION } = require("./error_codes.cjs");

const RESULT_PREFIX = "THREAT_DETECTION_RESULT:";

/**
 * Extract a complete JSON object from a string that starts with RESULT_PREFIX,
 * using character-by-character brace counting to find the matching closing brace.
 * Tracks string context so that braces inside JSON string values are not counted.
 * This avoids regex and correctly handles escape sequences (e.g. \", \\).
 *
 * The input may contain actual newline characters inside JSON string values when
 * the outer stream-json decoder has unescaped \n sequences. The extraction still
 * produces the complete JSON object; callers must normalize those newlines before
 * passing the JSON to JSON.parse.
 *
 * @param {string} text - String beginning with RESULT_PREFIX followed by a JSON object
 * @returns {string|null} RESULT_PREFIX + complete JSON object, or null if not found
 */
function extractResultFromText(text) {
  const jsonStartPos = text.indexOf("{", RESULT_PREFIX.length);
  if (jsonStartPos === -1) return null;

  let depth = 0;
  let inString = false;
  let escaped = false;
  let jsonEndPos = -1;
  for (let i = jsonStartPos; i < text.length; i++) {
    const ch = text[i];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (ch === "\\" && inString) {
      escaped = true;
      continue;
    }
    if (ch === '"') {
      inString = !inString;
      continue;
    }
    if (inString) {
      continue;
    }
    if (ch === "{") {
      depth++;
    } else if (ch === "}") {
      depth--;
      if (depth === 0) {
        jsonEndPos = i;
        break;
      }
    }
  }
  if (jsonEndPos === -1) return null;

  return RESULT_PREFIX + text.substring(jsonStartPos, jsonEndPos + 1);
}

/**
 * Try to extract a THREAT_DETECTION_RESULT value from a stream-json line.
 * Stream-json output from Claude wraps the result in JSON envelopes like:
 *   {"type":"result","result":"THREAT_DETECTION_RESULT:{\"prompt_injection\":...}"}
 *
 * The same result also appears in {"type":"assistant"} messages, but we only
 * extract from "type":"result" which is the authoritative final summary.
 *
 * @param {string} line - A single line from the detection log
 * @returns {string|null} The raw THREAT_DETECTION_RESULT:... string if found, null otherwise
 */
function extractFromStreamJson(line) {
  const trimmed = line.trim();
  if (!trimmed.startsWith("{")) return null;

  try {
    const obj = JSON.parse(trimmed);
    // Only extract from the authoritative "result" summary, not "assistant" messages.
    // In stream-json mode, the same content appears in both; using only "result"
    // avoids double-counting.
    if (obj.type === "result" && typeof obj.result === "string") {
      // The result field contains the model's full response text, which may
      // include analysis before the THREAT_DETECTION_RESULT line.
      // Split by newlines to find the line that starts with the prefix.
      //
      // IMPORTANT: The outer JSON.parse unescapes \n sequences into actual newline
      // characters. If the model placed a literal newline inside a reasons string
      // value, the JSON object for the verdict gets split across multiple lines here.
      // To handle this robustly, we find the prefix line by index, rejoin all
      // subsequent lines, then use brace-counting to locate the complete JSON object.
      const resultLines = obj.result.split("\n");
      let prefixLineIdx = -1;
      for (let i = 0; i < resultLines.length; i++) {
        if (resultLines[i].trim().startsWith(RESULT_PREFIX)) {
          prefixLineIdx = i;
          break;
        }
      }
      if (prefixLineIdx === -1) return null;

      // Rejoin all lines from the prefix line onward so that any JSON string
      // values split by actual newlines are reassembled.
      const joined = resultLines.slice(prefixLineIdx).join("\n").trim();

      // Extract the complete JSON object using brace-counting.
      return extractResultFromText(joined);
    }
  } catch {
    // Not valid JSON — not a stream-json line
  }
  return null;
}

/**
 * Parse the detection log and extract the THREAT_DETECTION_RESULT.
 *
 * Supports two output formats:
 * 1. **Stream-json** (--output-format stream-json): The result is embedded inside
 *    a JSON envelope: {"type":"result","result":"THREAT_DETECTION_RESULT:{...}"}
 * 2. **Raw/text** (--output-format text or --print): The result appears as a
 *    bare line: THREAT_DETECTION_RESULT:{...}
 *
 * Strategy: extract from stream-json "type":"result" lines first (authoritative).
 * If none found, fall back to raw line matching. This avoids double-counting
 * since stream-json mode produces both "assistant" and "result" envelopes
 * containing the same string.
 *
 * @param {string} content - The raw detection log content
 * @returns {{ verdict?: { prompt_injection: boolean, secret_leak: boolean, malicious_patch: boolean, reasons: string[] }, error?: string }}
 */
function parseDetectionLog(content) {
  const lines = content.split("\n");

  // Phase 1: Try stream-json extraction (from "type":"result" lines only)
  const streamMatches = [];
  for (const line of lines) {
    const extracted = extractFromStreamJson(line);
    if (extracted) {
      streamMatches.push(extracted);
    }
  }

  // Phase 2: If no stream-json results, try raw line matching.
  // Apply the same join-and-brace-count approach to handle cases where the
  // reasons values contain actual newlines that split the JSON across lines.
  const rawMatches = [];
  if (streamMatches.length === 0) {
    let i = 0;
    while (i < lines.length) {
      if (lines[i].trim().startsWith(RESULT_PREFIX)) {
        const joined = lines.slice(i).join("\n").trim();
        const extracted = extractResultFromText(joined);
        if (extracted !== null) {
          // Successfully extracted a complete JSON object; advance past consumed lines.
          rawMatches.push(extracted);
          // Count how many lines were consumed by this match so the loop
          // skips past them and does not re-match continuation lines.
          const jsonPart = extracted.substring(RESULT_PREFIX.length);
          const extraLines = jsonPart.split("\n").length - 1;
          i += extraLines + 1;
        } else {
          // No complete {…} object found (e.g. null, [], string, truncated JSON);
          // fall back to the trimmed line so the parsing step reports a useful error.
          rawMatches.push(lines[i].trim());
          i++;
        }
      } else {
        i++;
      }
    }
  }

  const matches = streamMatches.length > 0 ? streamMatches : rawMatches;

  if (matches.length === 0) {
    return { error: "No THREAT_DETECTION_RESULT found in detection log. The detection model may have failed to follow the output format." };
  }

  // Deduplicate identical results. The detection command writes to the same file
  // via both --debug-file and tee, so the same line often appears 2-3 times.
  // Only error if the entries actually disagree (different verdicts).
  const uniqueMatches = [...new Set(matches)];

  if (uniqueMatches.length > 1) {
    return {
      error: `Multiple conflicting THREAT_DETECTION_RESULT entries found (${uniqueMatches.length} unique out of ${matches.length} total) in detection log. Expected one consistent verdict. Entries: ${uniqueMatches.map((m, i) => `\n  [${i + 1}] ${m}`).join("")}`,
    };
  }

  const jsonPart = uniqueMatches[0].substring(RESULT_PREFIX.length);
  try {
    // Normalize literal newline characters to JSON escape sequences before parsing.
    // When the outer stream-json decoder unescapes \n sequences, actual newline
    // characters may end up inside JSON string values (e.g. in reasons entries).
    // Replacing them with the two-character sequence \n restores valid JSON so
    // that JSON.parse can handle them correctly.
    const normalizedJson = jsonPart.split("\n").join("\\n");
    const parsed = JSON.parse(normalizedJson);

    // The result must be a plain object, not null, an array, or a primitive.
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
      return { error: `THREAT_DETECTION_RESULT JSON must be an object, got ${parsed === null ? "null" : Array.isArray(parsed) ? "array" : typeof parsed}. Raw value: ${matches[0]}` };
    }

    // Validate that threat flags are actual booleans.
    // Boolean("false") === true, so accepting non-boolean types would cause
    // false positives (string "false" treated as a detection).
    for (const field of ["prompt_injection", "secret_leak", "malicious_patch"]) {
      if (typeof parsed[field] !== "boolean") {
        return { error: `Invalid type for "${field}": expected boolean, got ${typeof parsed[field]} (${JSON.stringify(parsed[field])}). Raw value: ${matches[0]}` };
      }
    }

    const verdict = {
      prompt_injection: parsed.prompt_injection,
      secret_leak: parsed.secret_leak,
      malicious_patch: parsed.malicious_patch,
      reasons: Array.isArray(parsed.reasons) ? parsed.reasons : [],
    };
    return { verdict };
  } catch (/** @type {any} */ parseError) {
    return { error: `Failed to parse JSON from THREAT_DETECTION_RESULT: ${getErrorMessage(parseError)}\nRaw value: ${matches[0]}` };
  }
}

/**
 * Main entry point for parsing threat detection results and concluding the detection job.
 *
 * This function consolidates three responsibilities previously split across two steps:
 *  1. Parse the detection log and extract the threat verdict
 *  2. Log all results with extensive diagnostic detail
 *  3. Set job outputs (success, conclusion) and fail the job if threats were detected
 *
 * The RUN_DETECTION environment variable controls whether detection was needed at all:
 *  - "true"  → parse the log and evaluate the verdict
 *  - anything else → mark as skipped (no agent output to analyze)
 *
 * @returns {Promise<void>}
 */
async function main() {
  const threatDetectionDir = "/tmp/gh-aw/threat-detection";
  const logPath = path.join(threatDetectionDir, DETECTION_LOG_FILENAME);
  const runDetection = process.env.RUN_DETECTION;
  const continueOnError = process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR !== "false";
  const isWarnMode = continueOnError;

  core.info("════════════════════════════════════════════════════════");
  core.info("🛡️  Threat Detection: Parse Results & Conclude");
  core.info("════════════════════════════════════════════════════════");
  core.info(`📋 RUN_DETECTION env: ${JSON.stringify(runDetection)}`);
  core.info(`📋 continue-on-error: ${continueOnError}`);
  core.info(`📁 Threat detection directory: ${threatDetectionDir}`);
  core.info(`📄 Detection log path: ${logPath}`);

  /**
   * Helper to set detection failure/warning outputs based on continue-on-error mode.
   * In warn mode: sets conclusion=warning, success=false, does NOT fail the job.
   * In error mode: sets conclusion=failure, success=false, fails the job.
   * @param {string} reason - Categorized reason (e.g. "threat_detected", "agent_failure", "parse_error")
   * @param {string} message - Human-readable error message
   */
  function setDetectionFailure(reason, message) {
    core.setOutput("reason", reason);
    if (isWarnMode) {
      core.warning(`⚠️ ${message}`);
      core.setOutput("conclusion", "warning");
      core.setOutput("success", "false");
    } else {
      core.setOutput("conclusion", "failure");
      core.setOutput("success", "false");
      core.setFailed(message);
    }
  }

  // ── Step 1: Check whether detection was needed ──────────────────────────
  if (runDetection !== "true") {
    core.info("⏭️  Detection guard output indicates detection was not needed.");
    core.info("   Reason: no agent output types or patch files were produced.");
    core.info("   Setting conclusion=skipped, success=true.");
    core.setOutput("conclusion", "skipped");
    core.setOutput("success", "true");
    core.setOutput("reason", "");
    core.info("✅ Detection skipped — no threats to evaluate.");
    return;
  }

  core.info("🔍 Detection is required. Proceeding to parse detection log...");

  // ── Step 2: Verify the detection log file exists ─────────────────────────
  if (!fs.existsSync(logPath)) {
    core.error("❌ Detection log file not found at: " + logPath);
    core.info("📁 Listing all files in artifact directory for diagnosis: " + threatDetectionDir);
    try {
      const files = listFilesRecursively(threatDetectionDir, threatDetectionDir);
      if (files.length === 0) {
        core.warning("   ⚠️  No files found in " + threatDetectionDir);
      } else {
        core.info("   Found " + files.length + " file(s):");
        files.forEach(file => core.info("     - " + file));
      }
    } catch {
      core.warning("   ⚠️  Could not list files in " + threatDetectionDir);
    }
    setDetectionFailure("agent_failure", `${ERR_SYSTEM}: ❌ Detection log file not found at: ${logPath}`);
    return;
  }

  core.info("✔️  Detection log file exists: " + logPath);

  // ── Step 3: Read the detection log ───────────────────────────────────────
  let logContent;
  try {
    logContent = fs.readFileSync(logPath, "utf8");
  } catch (/** @type {any} */ readError) {
    core.error("❌ Failed to read detection log: " + getErrorMessage(readError));
    setDetectionFailure("agent_failure", `${ERR_SYSTEM}: ❌ Failed to read detection log: ${getErrorMessage(readError)}`);
    return;
  }

  const logLines = logContent.split("\n");
  core.info(`📊 Detection log stats: ${logLines.length} lines, ${logContent.length} bytes`);

  // Log lines containing THREAT_DETECTION_RESULT for focused diagnosis
  const resultLineMatches = logLines.map((line, i) => ({ line, idx: i + 1 })).filter(({ line }) => line.includes(RESULT_PREFIX));
  if (resultLineMatches.length > 0) {
    core.info(`📄 Lines containing THREAT_DETECTION_RESULT (${resultLineMatches.length} of ${logLines.length}):`);
    resultLineMatches.forEach(({ line, idx }) => core.info(`   [${idx}] ${line}`));
  } else {
    core.info(`📄 No lines containing THREAT_DETECTION_RESULT found in ${logLines.length} lines`);
  }

  // ── Step 4: Parse the detection result ───────────────────────────────────
  core.info("🔎 Parsing THREAT_DETECTION_RESULT from detection log...");
  const { verdict, error } = parseDetectionLog(logContent);

  if (error || !verdict) {
    const errorMsg = error || "No verdict returned from detection log parser";
    core.error("❌ Failed to parse detection result: " + errorMsg);
    core.info("💡 This usually means the AI engine did not produce the expected output format.");
    core.info('   Expected format: THREAT_DETECTION_RESULT:{"prompt_injection":bool,"secret_leak":bool,"malicious_patch":bool,"reasons":[...]}');
    setDetectionFailure("parse_error", `${ERR_PARSE}: ❌ ${errorMsg}`);
    return;
  }

  // ── Step 5: Log the full verdict ─────────────────────────────────────────
  core.info("📋 Threat detection verdict:");
  core.info(`   prompt_injection : ${verdict.prompt_injection}`);
  core.info(`   secret_leak      : ${verdict.secret_leak}`);
  core.info(`   malicious_patch  : ${verdict.malicious_patch}`);
  if (verdict.reasons && verdict.reasons.length > 0) {
    core.info(`   reasons (${verdict.reasons.length}):`);
    verdict.reasons.forEach((reason, i) => core.info(`     [${i + 1}] ${reason}`));
  } else {
    core.info("   reasons          : (none)");
  }

  // ── Step 6: Evaluate verdict and set conclusion ───────────────────────────
  const threatsDetected = verdict.prompt_injection || verdict.secret_leak || verdict.malicious_patch;

  if (threatsDetected) {
    const threats = [];
    if (verdict.prompt_injection) threats.push("prompt injection");
    if (verdict.secret_leak) threats.push("secret leak");
    if (verdict.malicious_patch) threats.push("malicious patch");

    const reasonsText = verdict.reasons && verdict.reasons.length > 0 ? "\nReasons: " + verdict.reasons.join("; ") : "";

    core.error("🚨 Security threats detected: " + threats.join(", "));
    if (verdict.reasons && verdict.reasons.length > 0) {
      core.error("   Reasons: " + verdict.reasons.join("; "));
    }

    setDetectionFailure("threat_detected", `${ERR_VALIDATION}: ❌ Security threats detected: ${threats.join(", ")}${reasonsText}`);
  } else {
    core.info("✅ No security threats detected. Safe outputs may proceed.");
    core.setOutput("conclusion", "success");
    core.setOutput("success", "true");
    core.setOutput("reason", "");
  }

  core.info("════════════════════════════════════════════════════════");
  core.info("🛡️  Threat detection conclusion complete.");
  core.info("════════════════════════════════════════════════════════");
}

module.exports = { main, parseDetectionLog, extractFromStreamJson, extractResultFromText };
