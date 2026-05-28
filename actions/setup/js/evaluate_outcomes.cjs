// @ts-check

/**
 * evaluate_outcomes.cjs
 *
 * Evaluates safe output outcomes for recent successful workflow runs.
 * Replaces the shell-based evaluation logic in the outcome-collector workflow.
 *
 * Responsibilities:
 * - Load previously evaluated run IDs from cache-memory
 * - Fetch recent successful runs via `gh run list`
 * - Download safe-outputs-items artifacts via `gh run download`
 * - Classify each item (accepted/rejected/pending/noop) using the GitHub API
 * - Extract time-to-resolution, PR quality signals, pending age
 * - Write per-item evaluations to outcome-evaluations.jsonl
 * - Compute and write fleet summary to outcome-summary.json
 * - Update the seen-runs cache
 *
 * Outputs:
 *   /tmp/gh-aw/outcome-evaluations.jsonl  — per-item JSONL
 *   /tmp/gh-aw/outcome-summary.json       — fleet summary
 *   /tmp/gh-aw/outcomes/run-*.json        — per-run data
 *
 * Errors in individual run/item evaluation are non-fatal and logged to stderr.
 */

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { execFileSync } = require("child_process");

// ---------------------------------------------------------------------------
// Paths
// ---------------------------------------------------------------------------
const CACHE_DIR = "/tmp/gh-aw/cache-memory/outcome-collector";
const SEEN_FILE = path.join(CACHE_DIR, "seen-runs.json");
const OUTCOMES_DIR = "/tmp/gh-aw/outcomes";
const EVAL_JSONL = "/tmp/gh-aw/outcome-evaluations.jsonl";
const SUMMARY_PATH = "/tmp/gh-aw/outcome-summary.json";

// ---------------------------------------------------------------------------
// Noop types that are tracked but not counted as actionable
// ---------------------------------------------------------------------------
const NOOP_TYPES = new Set(["noop", "missing_tool", "missing_data", "report_incomplete"]);
const CLOSING_LABEL_KEYWORDS = ["not planned", "not_planned", "wontfix", "won't fix", "duplicate", "invalid", "declined", "rejected"];
const CLOSING_COMMENT_KEYWORDS = ["not planned", "won't fix", "wontfix", "duplicate", "invalid", "declined", "rejected", "closing as", "closed as", "closing this"];

const DEFAULT_ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC = 60 * 60;
const DEFAULT_LABEL_RETENTION_WINDOW_SEC = 24 * 60 * 60;

const POSITIVE_REACTIONS = ["+1", "heart", "hooray", "rocket"];
const NEGATIVE_REACTIONS = ["-1", "confused"];

/**
 * Read a positive integer from env with fallback.
 * @param {string} key
 * @param {number} fallback
 * @returns {number}
 */
function getEnvPositiveIntOrDefault(key, fallback) {
  const raw = process.env[key];
  const parsed = Number.parseInt(raw || "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

const ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC = getEnvPositiveIntOrDefault("OUTCOME_ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC", DEFAULT_ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC);
const LABEL_RETENTION_WINDOW_SEC = getEnvPositiveIntOrDefault("OUTCOME_LABEL_RETENTION_WINDOW_SEC", DEFAULT_LABEL_RETENTION_WINDOW_SEC);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Run a `gh` CLI command, returning stdout as a string.
 * Returns null on failure.
 * @param {string[]} args
 * @returns {string | null}
 */
function gh(args) {
  try {
    return execFileSync("gh", args, { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
  } catch {
    return null;
  }
}

/**
 * Run a `gh api` call, returning parsed JSON.
 * Returns null on failure.
 * @param {string} endpoint
 * @returns {object | null}
 */
function ghAPI(endpoint) {
  const raw = gh(["api", endpoint]);
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

/**
 * Read a JSON file, returning a default value on failure.
 * @param {string} filePath
 * @param {any} fallback
 * @returns {any}
 */
function readJSON(filePath, fallback) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return fallback;
  }
}

/**
 * Read a JSONL file, returning an array of parsed objects.
 * @param {string} filePath
 * @returns {object[]}
 */
function readJSONL(filePath) {
  try {
    return fs
      .readFileSync(filePath, "utf8")
      .split("\n")
      .filter(l => l.trim())
      .map(l => {
        try {
          return JSON.parse(l);
        } catch {
          return null;
        }
      })
      .filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * Atomically write JSON to a file using a tmp+rename swap.
 * @param {string} filePath
 * @param {any} data
 */
function writeJSONAtomic(filePath, data) {
  const tmp = filePath + ".tmp";
  fs.writeFileSync(tmp, JSON.stringify(data, null, 2) + "\n");
  fs.renameSync(tmp, filePath);
}

/**
 * Parse an ISO-8601 timestamp to epoch seconds. Returns null on failure.
 * @param {string} ts
 * @returns {number | null}
 */
function isoToEpoch(ts) {
  if (!ts) return null;
  const ms = Date.parse(ts);
  return Number.isFinite(ms) ? Math.floor(ms / 1000) : null;
}

/**
 * Compute seconds between two ISO timestamps. Returns null if either is invalid.
 * @param {string} from
 * @param {string} to
 * @returns {number | null}
 */
function secondsBetween(from, to) {
  const a = isoToEpoch(from);
  const b = isoToEpoch(to);
  if (a === null || b === null) return null;
  return b - a;
}

// ---------------------------------------------------------------------------
// Item evaluation
// ---------------------------------------------------------------------------

/**
 * @typedef {object} EvalResult
 * @property {string} result
 * @property {"accepted"|"rejected"|"pending"|"ignored"|"skipped"|"unknown"} outcome_status
 * @property {"strong"|"medium"|"weak"|"none"} evidence_strength
 * @property {string} signal
 * @property {string} detail
 * @property {number | null} resolution_sec
 * @property {number | null} pending_age_sec
 * @property {number | null} review_comments
 * @property {number | null} changed_files
 * @property {number | null} additions
 * @property {number | null} deletions
 * @property {number | null} reactions_total
 * @property {number | null} reactions_positive
 * @property {number | null} reactions_negative
 * @property {number | null} comments
 * @property {boolean} zero_touch
 */

/**
 * @typedef {object} EvaluateDeps
 * @property {(endpoint: string) => any} [ghAPI]
 * @property {number} [nowMs]
 */

/**
 * Convert issue/PR reaction summary into aggregate counts.
 * @param {any} reactions
 * @returns {{total: number|null, positive: number|null, negative: number|null}}
 */
function summarizeReactions(reactions) {
  if (!reactions || typeof reactions !== "object") {
    return { total: null, positive: null, negative: null };
  }
  const positive = POSITIVE_REACTIONS.reduce((sum, key) => sum + Number(reactions[key] || 0), 0);
  const negative = NEGATIVE_REACTIONS.reduce((sum, key) => sum + Number(reactions[key] || 0), 0);
  const total = reactions.total_count != null ? reactions.total_count : positive + negative + (reactions.laugh || 0) + (reactions.eyes || 0);
  return { total, positive, negative };
}

/**
 * @param {unknown} value
 * @returns {string[]}
 */
function normalizeLabels(value) {
  if (!Array.isArray(value)) return [];
  return value.map(l => String(l || "").trim()).filter(Boolean);
}

/**
 * @param {string} url
 * @returns {number|null}
 */
function parseIssueNumberFromURL(url) {
  const match = String(url || "").match(/\/(?:issues|pull)\/(\d+)/);
  if (!match) return null;
  const num = Number.parseInt(match[1], 10);
  return Number.isInteger(num) && num > 0 ? num : null;
}

/**
 * @param {string} url
 * @returns {string}
 */
function parseCommentIDFromURL(url) {
  const text = String(url || "");
  const issueCommentMatch = text.match(/#issuecomment-(\d+)/);
  if (issueCommentMatch) return issueCommentMatch[1];
  const pathMatch = text.match(/\/comments\/(\d+)/);
  return pathMatch ? pathMatch[1] : "";
}

/**
 * @param {any} issue
 * @returns {boolean}
 */
function hasIssueReactions(issue) {
  const summary = summarizeReactions(issue?.reactions);
  return typeof summary.total === "number" && summary.total > 0;
}

/**
 * Evaluate `create_issue`.
 * @param {object} item
 * @param {string} itemRepo
 * @param {string} timestamp
 * @param {EvalResult} out
 * @param {(endpoint: string) => any} apiGet
 * @param {number} nowMs
 * @returns {EvalResult}
 */
function evaluateCreateIssue(item, itemRepo, timestamp, out, apiGet, nowMs) {
  const num = parseIssueNumberFromURL(item.url || "");
  if (!num) {
    out.result = "unknown";
    out.detail = "unknown: issue number not found";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }
  const issue = apiGet(`repos/${itemRepo}/issues/${num}`);
  if (!issue || !issue.state) {
    out.result = "unknown";
    out.detail = "unknown: issue api error";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  out.comments = typeof issue.comments === "number" ? issue.comments : null;
  const reactionSummary = summarizeReactions(issue.reactions);
  out.reactions_total = reactionSummary.total;
  out.reactions_positive = reactionSummary.positive;
  out.reactions_negative = reactionSummary.negative;

  const authorLogin = issue.user && typeof issue.user.login === "string" ? issue.user.login : "";
  const comments = apiGet(`repos/${itemRepo}/issues/${num}/comments`);
  const hasNonAuthorComment = Array.isArray(comments) && comments.some(c => c && c.user && typeof c.user.login === "string" && c.user.login !== authorLogin && c.user.login !== "");

  const timeline = apiGet(`repos/${itemRepo}/issues/${num}/timeline`);
  const timelineEvents = Array.isArray(timeline) ? timeline : [];
  let hasMergedPRReference = false;
  let hasCommitReference = false;
  let hasClosingActionReference = false;
  let closeActor = "";
  for (const event of timelineEvents) {
    if (!event || typeof event !== "object") continue;
    const eventType = typeof event.event === "string" ? event.event : "";
    if (eventType === "closed") {
      hasClosingActionReference = true;
      const actorLogin = event.actor && typeof event.actor.login === "string" ? event.actor.login : "";
      if (actorLogin) closeActor = actorLogin;
    }
    if (eventType === "referenced" && event.commit_id) {
      hasCommitReference = true;
    }
    if (eventType !== "cross-referenced") continue;
    const sourceIssue = event.source && event.source.issue;
    const prNumber = sourceIssue && typeof sourceIssue.number === "number" ? sourceIssue.number : null;
    if (!prNumber) continue;
    const pr = apiGet(`repos/${itemRepo}/pulls/${prNumber}`);
    if (pr && pr.merged === true) {
      hasMergedPRReference = true;
    }
  }

  if (issue.state === "open" && (hasMergedPRReference || hasCommitReference)) {
    out.result = "accepted";
    out.detail = "accepted:strong";
    return out;
  }

  if (issue.state === "open" && (hasNonAuthorComment || hasIssueReactions(issue))) {
    out.result = "accepted";
    out.detail = "accepted:medium";
    return out;
  }

  if (issue.state === "closed" && issue.created_at && issue.closed_at) {
    out.resolution_sec = secondsBetween(issue.created_at, issue.closed_at);
    const closedByDifferentUser = closeActor !== "" && closeActor !== authorLogin;
    if (typeof out.resolution_sec === "number" && out.resolution_sec <= ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC && closedByDifferentUser) {
      out.result = "rejected";
      out.detail = "rejected:strong";
      return out;
    }
  }

  if (issue.state === "closed") {
    const hasActivity = (typeof issue.comments === "number" && issue.comments > 0) || hasIssueReactions(issue) || hasMergedPRReference || hasCommitReference;
    if (!hasActivity) {
      out.result = "rejected";
      out.detail = "rejected:medium";
      return out;
    }
    out.result = "unknown";
    out.detail = "unknown: closed with activity";
    return out;
  }

  if (issue.state === "open") {
    out.result = "pending";
    out.detail = "pending: open with no engagement";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  out.result = "unknown";
  out.detail = "unknown: unsupported issue state";
  setPendingAge(out, timestamp, nowMs);
  return out;
}

/**
 * Evaluate `add_comment`.
 * @param {object} item
 * @param {string} itemRepo
 * @param {string} timestamp
 * @param {EvalResult} out
 * @param {(endpoint: string) => any} apiGet
 * @param {number} nowMs
 * @returns {EvalResult}
 */
function evaluateAddComment(item, itemRepo, timestamp, out, apiGet, nowMs) {
  const commentID = parseCommentIDFromURL(item.url || "");
  const issueNum = parseIssueNumberFromURL(item.url || "");
  if (!commentID || !issueNum) {
    out.result = "unknown";
    out.detail = "unknown: missing comment or issue id";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  const comment = apiGet(`repos/${itemRepo}/issues/comments/${commentID}`);
  if (!comment || !comment.id) {
    out.result = "unknown";
    out.detail = "unknown: failed to fetch comment from API (may be deleted or inaccessible)";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  const commentAuthor = comment.user && typeof comment.user.login === "string" ? comment.user.login : "";
  const commentCreatedAt = typeof comment.created_at === "string" ? comment.created_at : "";
  const reactionSummary = summarizeReactions(comment.reactions);
  out.reactions_total = reactionSummary.total;
  out.reactions_positive = reactionSummary.positive;
  out.reactions_negative = reactionSummary.negative;

  const issueComments = apiGet(`repos/${itemRepo}/issues/${issueNum}/comments`);
  const commentURL = String(item.url || "");
  let hasReply = false;
  let hasQuote = false;
  let threadActedOn = false;
  if (Array.isArray(issueComments)) {
    for (const c of issueComments) {
      if (!c || typeof c !== "object") continue;
      if (typeof c.created_at === "string" && commentCreatedAt && c.created_at > commentCreatedAt) {
        threadActedOn = true;
      }
      const body = typeof c.body === "string" ? c.body : "";
      const cAuthor = c.user && typeof c.user.login === "string" ? c.user.login : "";
      if (cAuthor && cAuthor !== commentAuthor && typeof c.created_at === "string" && commentCreatedAt && c.created_at > commentCreatedAt) {
        hasReply = true;
      }
      if (body.includes(`#issuecomment-${commentID}`) || (commentURL && body.includes(commentURL))) {
        hasQuote = true;
      }
    }
  }

  if ((typeof reactionSummary.total === "number" && reactionSummary.total > 0) || hasReply || hasQuote) {
    out.result = "accepted";
    out.detail = "accepted:strong";
    return out;
  }

  if (threadActedOn) {
    out.result = "accepted";
    out.detail = "accepted:medium";
    return out;
  }

  out.result = "pending";
  out.detail = "pending: no follow-up";
  setPendingAge(out, timestamp, nowMs);
  return out;
}

/**
 * Evaluate `add_labels`.
 * @param {object} item
 * @param {string} itemRepo
 * @param {string} timestamp
 * @param {EvalResult} out
 * @param {(endpoint: string) => any} apiGet
 * @param {number} nowMs
 * @returns {EvalResult}
 */
function evaluateAddLabels(item, itemRepo, timestamp, out, apiGet, nowMs) {
  const num = parseIssueNumberFromURL(item.url || "");
  if (!num) {
    out.result = "unknown";
    out.detail = "unknown: issue number not found";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  const hasLabelsBefore = Object.hasOwn(item, "labelsBefore") && Array.isArray(item.labelsBefore);
  const labelsBefore = hasLabelsBefore ? normalizeLabels(item.labelsBefore) : [];
  const labelsAdded = normalizeLabels(item.labelsAdded);
  const fallbackLabels = normalizeLabels(item.labels);
  const effectiveLabelsAdded = labelsAdded.length > 0 ? labelsAdded : fallbackLabels;

  if (!hasLabelsBefore) {
    out.result = "unknown";
    out.detail = "unknown: missing persisted label before state";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  if (effectiveLabelsAdded.length === 0) {
    out.result = "unknown";
    out.detail = "unknown: no labels added";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  const labels = apiGet(`repos/${itemRepo}/issues/${num}/labels`);
  if (!Array.isArray(labels)) {
    out.result = "unknown";
    out.detail = "unknown: labels api error";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  const currentLabels = new Set(labels.map(l => (l && typeof l.name === "string" ? l.name : "")).filter(Boolean));
  const trackedAdded = effectiveLabelsAdded.filter(l => !labelsBefore.includes(l));
  const removed = trackedAdded.filter(l => !currentLabels.has(l));

  const nowEpoch = Math.floor(nowMs / 1000);
  const createdEpoch = isoToEpoch(timestamp || "");
  const elapsedSec = createdEpoch === null ? null : nowEpoch - createdEpoch;
  if (elapsedSec === null || elapsedSec < LABEL_RETENTION_WINDOW_SEC) {
    out.result = "pending";
    out.detail = "pending: retention window not elapsed";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  if (removed.length === 0) {
    out.result = "accepted";
    out.detail = "accepted:strong";
    return out;
  }

  const issue = apiGet(`repos/${itemRepo}/issues/${num}`);
  const issueAuthor = issue && issue.user && typeof issue.user.login === "string" ? issue.user.login : "";
  const events = apiGet(`repos/${itemRepo}/issues/${num}/events`);
  const eventList = Array.isArray(events) ? events : [];
  const removedByNonAuthor = eventList.some(event => {
    if (!event || event.event !== "unlabeled") return false;
    const removedLabel = event.label && typeof event.label.name === "string" ? event.label.name : "";
    if (!removed.includes(removedLabel)) return false;
    const actor = event.actor && typeof event.actor.login === "string" ? event.actor.login : "";
    return actor !== "" && actor !== issueAuthor;
  });

  if (removedByNonAuthor) {
    out.result = "rejected";
    out.detail = "rejected:strong";
    return out;
  }

  out.result = "unknown";
  out.detail = "unknown: labels removed with ambiguous actor";
  return out;
}

/**
 * Normalize legacy result/detail pairs into the shared outcome model.
 * @param {string} result
 * @param {string} detail
 * @returns {{ outcome_status: "accepted"|"rejected"|"pending"|"ignored"|"skipped"|"unknown", evidence_strength: "strong"|"medium"|"weak"|"none", signal: string }}
 */
function normalizeOutcome(result, detail) {
  const normalizedDetail = String(detail || "")
    .toLowerCase()
    .trim();

  if (result === "noop") {
    return { outcome_status: "skipped", evidence_strength: "none", signal: "noop" };
  }
  if (normalizedDetail === "object still exists") {
    return { outcome_status: "unknown", evidence_strength: "weak", signal: "target_exists_only" };
  }
  if (normalizedDetail === "review approved") {
    return { outcome_status: "accepted", evidence_strength: "strong", signal: "review_approved" };
  }
  if (normalizedDetail === "review submitted") {
    return { outcome_status: "accepted", evidence_strength: "medium", signal: "review_submitted" };
  }
  if (normalizedDetail === "review request removed") {
    return { outcome_status: "rejected", evidence_strength: "strong", signal: "review_request_removed" };
  }
  if (normalizedDetail === "review dismissed") {
    return { outcome_status: "rejected", evidence_strength: "strong", signal: "review_dismissed" };
  }
  if (normalizedDetail === "changes requested addressed and merged") {
    return { outcome_status: "accepted", evidence_strength: "medium", signal: "changes_requested_addressed" };
  }
  if (normalizedDetail === "closed without merge after review") {
    return { outcome_status: "rejected", evidence_strength: "medium", signal: "closed_without_merge_after_review" };
  }
  if (normalizedDetail === "latest review awaiting outcome") {
    return { outcome_status: "pending", evidence_strength: "medium", signal: "latest_review_pending" };
  }
  if (normalizedDetail === "awaiting review") {
    return { outcome_status: "pending", evidence_strength: "medium", signal: "awaiting_review" };
  }
  if (normalizedDetail === "update retained and merged") {
    return { outcome_status: "accepted", evidence_strength: "strong", signal: "state_retained_and_merged" };
  }
  if (normalizedDetail === "update retained") {
    return { outcome_status: "accepted", evidence_strength: "medium", signal: "state_retained" };
  }
  if (normalizedDetail === "update reverted") {
    return { outcome_status: "rejected", evidence_strength: "strong", signal: "state_reverted" };
  }
  if (normalizedDetail === "update replaced") {
    return { outcome_status: "rejected", evidence_strength: "strong", signal: "state_replaced" };
  }
  if (normalizedDetail === "missing execution state") {
    return { outcome_status: "unknown", evidence_strength: "none", signal: "missing_execution_state" };
  }
  if (normalizedDetail === "no persisted state delta") {
    return { outcome_status: "unknown", evidence_strength: "none", signal: "no_state_delta" };
  }
  if (result === "accepted" && normalizedDetail.startsWith("merged")) {
    return { outcome_status: "accepted", evidence_strength: "strong", signal: "merged" };
  }
  if (result === "rejected" && normalizedDetail === "closed") {
    return { outcome_status: "rejected", evidence_strength: "strong", signal: "closed" };
  }
  if (result === "pending" && normalizedDetail === "open") {
    return { outcome_status: "pending", evidence_strength: "medium", signal: "open" };
  }
  switch (result) {
    case "accepted":
      return { outcome_status: "accepted", evidence_strength: "medium", signal: "acted_on" };
    case "rejected":
      return { outcome_status: "rejected", evidence_strength: "medium", signal: "rejected" };
    case "ignored":
      return { outcome_status: "ignored", evidence_strength: "medium", signal: "ignored" };
    case "pending":
      return { outcome_status: "pending", evidence_strength: "medium", signal: "pending" };
    default:
      return { outcome_status: "unknown", evidence_strength: "weak", signal: "unknown" };
  }
}

/**
 * @param {object} item
 * @returns {number | null}
 */
function getItemNumber(item) {
  if (typeof item.number === "number" && Number.isFinite(item.number)) return item.number;
  const url = item.url || "";
  const issueMatch = url.match(/\/(?:issues|pull)\/(\d+)/);
  if (issueMatch) return Number(issueMatch[1]);
  return null;
}

/**
 * @param {object} item
 * @param {string} defaultRepo
 * @returns {string}
 */
function getItemRepo(item, defaultRepo) {
  if (item.repo) return item.repo;
  const url = item.url || "";
  const match = url.match(/github\.com\/([^/]+\/[^/]+)/);
  return match ? match[1] : defaultRepo;
}

/**
 * @param {object} item
 * @param {string} key
 * @returns {string[]}
 */
function getMetadataStringArray(item, key) {
  const raw = item?.metadata?.[key];
  if (!Array.isArray(raw)) return [];
  return raw.map(value => String(value || "").trim()).filter(Boolean);
}

/**
 * @param {object} item
 * @param {string} key
 * @returns {number | null}
 */
function getMetadataNumber(item, key) {
  const raw = item?.metadata?.[key];
  if (typeof raw === "number" && Number.isFinite(raw)) return raw;
  if (typeof raw === "string" && raw.trim() !== "") {
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

/**
 * @param {string} key
 * @param {any} value
 * @returns {any}
 */
function normalizeStateValue(key, value) {
  if (key === "labels" || key === "assignees") {
    if (!Array.isArray(value)) return [];
    return value
      .map(entry => String(entry || "").trim())
      .filter(Boolean)
      .sort();
  }
  if (key === "draft") {
    return value === true;
  }
  if (typeof value === "string") {
    return value.trim();
  }
  return value ?? "";
}

function hashOutcomeBody(body) {
  const normalized = String(body || "")
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map(line => line.replace(/[ \t]+$/g, ""))
    .join("\n")
    .trim();
  return crypto.createHash("sha256").update(normalized, "utf8").digest("hex");
}

/**
 * @param {string} key
 * @param {any} left
 * @param {any} right
 * @returns {boolean}
 */
function stateValuesEqual(key, left, right) {
  const normalizedLeft = normalizeStateValue(key, left);
  const normalizedRight = normalizeStateValue(key, right);
  return JSON.stringify(normalizedLeft) === JSON.stringify(normalizedRight);
}

/**
 * @param {Record<string, any> | null | undefined} beforeState
 * @param {Record<string, any> | null | undefined} afterState
 * @param {Record<string, any>} currentState
 * @param {string[]} fields
 * @returns {{changed: string[], retained: string[], reverted: string[], replaced: string[]}}
 */
function compareRetainedState(beforeState, afterState, currentState, fields) {
  const changed = [];
  const retained = [];
  const reverted = [];
  const replaced = [];

  for (const field of fields) {
    if (!afterState || !(field in afterState)) continue;
    const beforeValue = beforeState ? beforeState[field] : undefined;
    const afterValue = afterState[field];
    if (stateValuesEqual(field, beforeValue, afterValue)) continue;
    changed.push(field);
    const currentValue = currentState[field];
    if (stateValuesEqual(field, currentValue, afterValue)) {
      retained.push(field);
      continue;
    }
    if (beforeState && field in beforeState && stateValuesEqual(field, currentValue, beforeValue)) {
      reverted.push(field);
      continue;
    }
    replaced.push(field);
  }

  return { changed, retained, reverted, replaced };
}

function extractIssueUpdateState(issue) {
  return {
    title: typeof issue?.title === "string" ? issue.title : "",
    body_hash: hashOutcomeBody(issue?.body),
    state: typeof issue?.state === "string" ? issue.state : "",
    labels: Array.isArray(issue?.labels)
      ? issue.labels
          .map(label => {
            if (typeof label === "string") return label;
            if (label && typeof label.name === "string") return label.name;
            return "";
          })
          .filter(Boolean)
      : [],
    assignees: Array.isArray(issue?.assignees)
      ? issue.assignees
          .map(assignee => {
            if (typeof assignee === "string") return assignee;
            if (assignee && typeof assignee.login === "string") return assignee.login;
            return "";
          })
          .filter(Boolean)
      : [],
  };
}

function extractPullRequestUpdateState(pullRequest) {
  return {
    title: typeof pullRequest?.title === "string" ? pullRequest.title : "",
    body_hash: hashOutcomeBody(pullRequest?.body),
    state: typeof pullRequest?.state === "string" ? pullRequest.state : "",
    base: typeof pullRequest?.base?.ref === "string" ? pullRequest.base.ref : "",
    draft: pullRequest?.draft === true,
    head_sha: typeof pullRequest?.head?.sha === "string" ? pullRequest.head.sha : "",
  };
}

/**
 * @param {object} item
 * @param {string} defaultRepo
 * @param {(endpoint: string) => any} api
 * @param {{fields: string[], loadCurrent: (repo: string, number: number) => { currentState: Record<string, any>, merged?: boolean } | null}} options
 * @returns {EvalResult}
 */
function evaluateRetainedUpdate(item, defaultRepo, api, options) {
  const repo = getItemRepo(item, defaultRepo);
  const number = getItemNumber(item);
  /** @type {EvalResult} */
  const out = {
    result: "unknown",
    outcome_status: "unknown",
    evidence_strength: "none",
    signal: "missing_execution_state",
    detail: "",
    resolution_sec: null,
    pending_age_sec: null,
    review_comments: null,
    changed_files: null,
    additions: null,
    deletions: null,
    reactions_total: null,
    reactions_positive: null,
    reactions_negative: null,
    comments: null,
    zero_touch: false,
  };

  if (!repo || !number) {
    out.detail = "missing execution state";
    return out;
  }

  if (!item.before_state || !item.after_state) {
    out.detail = "missing execution state";
    return out;
  }

  const loaded = options.loadCurrent(repo, number);
  if (!loaded || !loaded.currentState) {
    out.signal = "unknown";
    out.detail = "api error";
    out.evidence_strength = "none";
    return out;
  }

  const comparison = compareRetainedState(item.before_state, item.after_state, loaded.currentState, options.fields);
  if (comparison.changed.length === 0) {
    out.detail = "no persisted state delta";
    out.signal = "no_state_delta";
    return out;
  }

  if (comparison.retained.length === comparison.changed.length) {
    out.result = "accepted";
    if (loaded.merged) {
      out.outcome_status = "accepted";
      out.evidence_strength = "strong";
      out.signal = "state_retained_and_merged";
      out.detail = "update retained and merged";
      return out;
    }
    out.outcome_status = "accepted";
    out.evidence_strength = "medium";
    out.signal = "state_retained";
    out.detail = "update retained";
    return out;
  }

  out.result = "rejected";
  out.outcome_status = "rejected";
  out.evidence_strength = "strong";
  if (comparison.reverted.length === comparison.changed.length) {
    out.signal = "state_reverted";
    out.detail = "update reverted";
    return out;
  }
  out.signal = "state_replaced";
  out.detail = "update replaced";
  return out;
}

/**
 * @param {string | undefined} timestamp
 * @param {string | undefined} threshold
 * @returns {boolean}
 */
function isOnOrAfter(timestamp, threshold) {
  if (!timestamp) return false;
  if (!threshold) return true;
  const a = Date.parse(timestamp);
  const b = Date.parse(threshold);
  if (!Number.isFinite(a) || !Number.isFinite(b)) return false;
  return a >= b;
}

/**
 * @param {any} review
 * @returns {boolean}
 */
function isSubmittedReview(review) {
  const state = String(review?.state || "").toUpperCase();
  return Boolean(review?.submitted_at) && state !== "" && state !== "PENDING";
}

/**
 * @param {object} item
 * @param {string} defaultRepo
 * @param {(endpoint: string) => any} api
 * @returns {EvalResult}
 */
function evaluateAddReviewer(item, defaultRepo, api = ghAPI) {
  const repo = getItemRepo(item, defaultRepo);
  const number = getItemNumber(item);
  const timestamp = item.timestamp || "";
  /** @type {EvalResult} */
  const out = {
    result: "unknown",
    outcome_status: "unknown",
    evidence_strength: "weak",
    signal: "unknown",
    detail: "",
    resolution_sec: null,
    pending_age_sec: null,
    review_comments: null,
    changed_files: null,
    additions: null,
    deletions: null,
    reactions_total: null,
    reactions_positive: null,
    reactions_negative: null,
    comments: null,
    zero_touch: false,
  };

  if (!repo || !number) {
    out.detail = "missing review request reference";
    return out;
  }

  const requestedReviewers = new Set(getMetadataStringArray(item, "requested_reviewers").map(login => login.toLowerCase()));
  const requestedTeams = new Set(getMetadataStringArray(item, "requested_team_reviewers").map(team => team.toLowerCase()));
  const reviews = api(`repos/${repo}/pulls/${number}/reviews`);
  const requested = api(`repos/${repo}/pulls/${number}/requested_reviewers`);

  if (!Array.isArray(reviews) || !requested) {
    out.detail = "api error";
    setPendingAge(out, timestamp);
    return out;
  }

  const latestReviewByRequestedReviewer = new Map();
  for (const review of reviews) {
    const state = String(review?.state || "").toUpperCase();
    const submittedAt = review?.submitted_at;
    if (!state || state === "PENDING" || !submittedAt || !isOnOrAfter(submittedAt, timestamp)) continue;
    const login = String(review?.user?.login || "").toLowerCase();
    if (!requestedReviewers.has(login)) continue;
    const previous = latestReviewByRequestedReviewer.get(login);
    if (!previous || isOnOrAfter(submittedAt, previous?.submitted_at)) {
      latestReviewByRequestedReviewer.set(login, review);
    }
  }
  const relevantReviews = Array.from(latestReviewByRequestedReviewer.values());

  if (relevantReviews.some(review => String(review?.state || "").toUpperCase() === "APPROVED")) {
    out.result = "accepted";
    out.detail = "review approved";
    return out;
  }

  if (relevantReviews.length > 0) {
    out.result = "accepted";
    out.detail = "review submitted";
    return out;
  }

  // We cannot cheaply verify team membership for each reviewer from this endpoint,
  // so any submitted post-request review counts as medium-evidence team activity.
  const anyReviewAfterRequest = reviews.some(review => isSubmittedReview(review) && isOnOrAfter(review?.submitted_at, timestamp));
  if (requestedTeams.size > 0 && anyReviewAfterRequest) {
    out.result = "accepted";
    out.detail = "review submitted";
    return out;
  }

  const pendingUsers = new Set((requested.users || []).map(user => String(user?.login || "").toLowerCase()));
  const pendingTeams = new Set((requested.teams || []).map(team => String(team?.slug || team?.name || "").toLowerCase()));
  const stillPending = Array.from(requestedReviewers).some(login => pendingUsers.has(login)) || Array.from(requestedTeams).some(team => pendingTeams.has(team));

  if (stillPending) {
    out.result = "pending";
    out.detail = "awaiting review";
    setPendingAge(out, timestamp);
    return out;
  }

  if (requestedReviewers.size > 0 || requestedTeams.size > 0) {
    out.result = "rejected";
    out.detail = "review request removed";
    return out;
  }

  out.detail = "unknown review request state";
  return out;
}

/**
 * @param {object} item
 * @param {string} defaultRepo
 * @param {(endpoint: string) => any} api
 * @returns {EvalResult}
 */
function evaluateUpdateIssue(item, defaultRepo, api = ghAPI) {
  return evaluateRetainedUpdate(item, defaultRepo, api, {
    fields: ["title", "body_hash", "state", "labels", "assignees"],
    loadCurrent: (repo, number) => {
      const issue = api(`repos/${repo}/issues/${number}`);
      if (!issue || !issue.state) return null;
      return { currentState: extractIssueUpdateState(issue) };
    },
  });
}

/**
 * @param {object} item
 * @param {string} defaultRepo
 * @param {(endpoint: string) => any} api
 * @returns {EvalResult}
 */
function evaluateUpdatePullRequest(item, defaultRepo, api = ghAPI) {
  return evaluateRetainedUpdate(item, defaultRepo, api, {
    fields: ["title", "body_hash", "state", "base", "draft", "head_sha"],
    loadCurrent: (repo, number) => {
      const pullRequest = api(`repos/${repo}/pulls/${number}`);
      if (!pullRequest || !pullRequest.state) return null;
      return {
        currentState: extractPullRequestUpdateState(pullRequest),
        merged: pullRequest.merged === true,
      };
    },
  });
}

/**
 * @param {object} item
 * @param {string} defaultRepo
 * @param {(endpoint: string) => any} api
 * @returns {EvalResult}
 */
function evaluateSubmitPullRequestReview(item, defaultRepo, api = ghAPI) {
  const repo = getItemRepo(item, defaultRepo);
  const number = getItemNumber(item);
  const timestamp = item.timestamp || "";
  /** @type {EvalResult} */
  const out = {
    result: "unknown",
    outcome_status: "unknown",
    evidence_strength: "weak",
    signal: "unknown",
    detail: "",
    resolution_sec: null,
    pending_age_sec: null,
    review_comments: null,
    changed_files: null,
    additions: null,
    deletions: null,
    reactions_total: null,
    reactions_positive: null,
    reactions_negative: null,
    comments: null,
    zero_touch: false,
  };

  if (!repo || !number) {
    out.detail = "missing review reference";
    return out;
  }

  const reviewId = getMetadataNumber(item, "review_id");
  const pr = api(`repos/${repo}/pulls/${number}`);
  const reviews = api(`repos/${repo}/pulls/${number}/reviews`);

  if (!pr || !Array.isArray(reviews) || !pr.state) {
    out.detail = "api error";
    setPendingAge(out, timestamp);
    return out;
  }

  const submittedReviews = reviews.filter(candidate => isSubmittedReview(candidate));
  const review = submittedReviews.find(candidate => Number(candidate?.id) === reviewId) || submittedReviews.filter(candidate => isOnOrAfter(candidate?.submitted_at, timestamp)).slice(-1)[0];

  if (!review) {
    out.detail = "review not found";
    return out;
  }

  const reviewState = String(review?.state || item?.metadata?.review_state || "").toUpperCase();
  const reviewSubmittedAt = review?.submitted_at || timestamp;
  const latestReview = submittedReviews.sort((a, b) => Date.parse(a.submitted_at) - Date.parse(b.submitted_at)).slice(-1)[0];

  out.review_comments = typeof pr.review_comments === "number" ? pr.review_comments : null;
  out.changed_files = typeof pr.changed_files === "number" ? pr.changed_files : null;
  out.additions = typeof pr.additions === "number" ? pr.additions : null;
  out.deletions = typeof pr.deletions === "number" ? pr.deletions : null;
  out.comments = typeof pr.comments === "number" ? pr.comments : null;

  if (reviewState === "DISMISSED") {
    out.result = "rejected";
    out.detail = "review dismissed";
    return out;
  }

  if (pr.merged === true) {
    if (reviewState === "APPROVED") {
      out.result = "accepted";
      out.detail = "review approved";
      if (pr.created_at && pr.merged_at) {
        out.resolution_sec = secondsBetween(pr.created_at, pr.merged_at);
      }
      return out;
    }

    if (reviewState === "CHANGES_REQUESTED") {
      const commits = api(`repos/${repo}/pulls/${number}/commits`);
      const hasPushAfterReview = Array.isArray(commits) ? commits.some(commit => isOnOrAfter(commit?.commit?.committer?.date || commit?.commit?.author?.date, reviewSubmittedAt)) : false;
      if (hasPushAfterReview) {
        out.result = "accepted";
        out.detail = "changes requested addressed and merged";
        if (pr.created_at && pr.merged_at) {
          out.resolution_sec = secondsBetween(pr.created_at, pr.merged_at);
        }
        return out;
      }
    }
  }

  if (pr.state === "closed" && pr.merged !== true) {
    out.result = "rejected";
    out.detail = "closed without merge after review";
    if (pr.created_at && pr.closed_at) {
      out.resolution_sec = secondsBetween(pr.created_at, pr.closed_at);
    }
    return out;
  }

  if (pr.state === "open" && latestReview && Number(latestReview.id) === Number(review.id)) {
    out.result = "pending";
    out.detail = "latest review awaiting outcome";
    setPendingAge(out, timestamp);
    return out;
  }

  out.detail = "unknown review lifecycle";
  return out;
}

/**
 * Evaluate a single safe-output item against the GitHub API.
 * @param {object} item
 * @param {string} defaultRepo
 * @param {((endpoint: string) => any) | EvaluateDeps} [apiOrOptions]
 * @returns {EvalResult}
 */
function evaluateItem(item, defaultRepo, apiOrOptions) {
  const url = item.url || "";
  const itemRepo = item.repo || defaultRepo;
  const timestamp = item.timestamp || "";
  const type = item.type || "";
  const ghAPIFn = typeof apiOrOptions === "function" ? apiOrOptions : typeof apiOrOptions?.ghAPI === "function" ? apiOrOptions.ghAPI : ghAPI;
  const nowMs = typeof apiOrOptions === "object" && typeof apiOrOptions?.nowMs === "number" ? apiOrOptions.nowMs : Date.now();

  /** @type {EvalResult} */
  const out = {
    result: "pending",
    outcome_status: "pending",
    evidence_strength: "medium",
    signal: "pending",
    detail: "",
    resolution_sec: null,
    pending_age_sec: null,
    review_comments: null,
    changed_files: null,
    additions: null,
    deletions: null,
    reactions_total: null,
    reactions_positive: null,
    reactions_negative: null,
    comments: null,
    zero_touch: false,
  };

  if (type === "create_pull_request") {
    return evaluateCreatePullRequestOutcome(item, itemRepo, out, ghAPIFn);
  }
  if (type === "push_to_pull_request_branch") {
    return evaluatePushToPullRequestBranchOutcome(item, itemRepo, out, ghAPIFn);
  }
  if (type === "update_issue") {
    return evaluateUpdateIssue(item, defaultRepo, ghAPIFn);
  }
  if (type === "update_pull_request") {
    return evaluateUpdatePullRequest(item, defaultRepo, ghAPIFn);
  }

  if (!url) {
    if (item.type === "add_reviewer") {
      return evaluateAddReviewer(item, defaultRepo, ghAPIFn);
    }
    if (item.type === "submit_pull_request_review") {
      return evaluateSubmitPullRequestReview(item, defaultRepo, ghAPIFn);
    }
    out.detail = "no url";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  if (type === "add_reviewer") {
    return evaluateAddReviewer(item, defaultRepo, ghAPIFn);
  }
  if (type === "submit_pull_request_review") {
    return evaluateSubmitPullRequestReview(item, defaultRepo, ghAPIFn);
  }
  if (type === "create_issue") {
    return evaluateCreateIssue(item, itemRepo, timestamp, out, ghAPIFn, nowMs);
  }
  if (type === "add_comment") {
    return evaluateAddComment(item, itemRepo, timestamp, out, ghAPIFn, nowMs);
  }
  if (type === "add_labels") {
    return evaluateAddLabels(item, itemRepo, timestamp, out, ghAPIFn, nowMs);
  }

  // Issues / issue-comments
  const issueMatch = url.match(/\/(?:issues|pull)\/(\d+)/);
  if (/\/issues\/\d+|\/issuecomment-/.test(url) && issueMatch) {
    const num = issueMatch[1];
    const data = ghAPIFn(`repos/${itemRepo}/issues/${num}`);
    if (!data || !data.state) {
      out.detail = "api error";
      setPendingAge(out, timestamp, nowMs);
      return out;
    }
    out.result = "accepted";
    out.detail = data.state;
    out.comments = typeof data.comments === "number" ? data.comments : null;

    // Reactions on issues
    if (data.reactions && typeof data.reactions === "object") {
      const summary = summarizeReactions(data.reactions);
      out.reactions_total = summary.total;
      out.reactions_positive = summary.positive;
      out.reactions_negative = summary.negative;
    }

    if (data.state === "closed" && data.created_at && data.closed_at) {
      out.resolution_sec = secondsBetween(data.created_at, data.closed_at);
    }
    return out;
  }

  // Pull requests
  const prMatch = url.match(/\/pull\/(\d+)/);
  if (prMatch) {
    const num = prMatch[1];
    const data = ghAPIFn(`repos/${itemRepo}/pulls/${num}`);
    if (!data || !data.state) {
      out.detail = "api error";
      setPendingAge(out, timestamp, nowMs);
      return out;
    }

    // PR quality signals
    out.review_comments = typeof data.review_comments === "number" ? data.review_comments : null;
    out.changed_files = typeof data.changed_files === "number" ? data.changed_files : null;
    out.additions = typeof data.additions === "number" ? data.additions : null;
    out.deletions = typeof data.deletions === "number" ? data.deletions : null;
    out.comments = typeof data.comments === "number" ? data.comments : null;

    // Reactions
    if (data.reactions && typeof data.reactions === "object") {
      const summary = summarizeReactions(data.reactions);
      out.reactions_total = summary.total;
      out.reactions_positive = summary.positive;
      out.reactions_negative = summary.negative;
    }

    // Zero-touch: merged with no human review comments and no issue-level comments
    if (data.merged === true && out.review_comments === 0 && out.comments === 0) {
      out.zero_touch = true;
    }

    if (data.merged === true) {
      out.result = "accepted";
      out.detail = "merged";
      if (data.created_at && data.merged_at) {
        out.resolution_sec = secondsBetween(data.created_at, data.merged_at);
      }
    } else if (data.state === "closed") {
      out.result = "rejected";
      out.detail = "closed";
      if (data.created_at && data.closed_at) {
        out.resolution_sec = secondsBetween(data.created_at, data.closed_at);
      }
    } else if (data.state === "open") {
      out.result = "pending";
      out.detail = "open";
      setPendingAge(out, timestamp, nowMs);
    } else {
      out.detail = "api error";
      setPendingAge(out, timestamp, nowMs);
    }
    return out;
  }

  // Comments, labels, etc. — if URL exists, the item was created
  out.result = "unknown";
  out.detail = "object still exists";
  Object.assign(out, normalizeOutcome(out.result, out.detail));
  return out;
}

/**
 * Evaluate outcome for create_pull_request.
 * @param {object} item
 * @param {string} itemRepo
 * @param {EvalResult} out
 * @param {(endpoint: string) => any} [ghAPIFn]
 * @returns {EvalResult}
 */
function evaluateCreatePullRequestOutcome(item, itemRepo, out, ghAPIFn = ghAPI) {
  const num = resolvePRNumber(item);
  const timestamp = item.timestamp || "";

  if (!num || !itemRepo) {
    out.result = "unknown";
    out.detail = "missing pull request reference";
    setPendingAge(out, timestamp);
    return out;
  }

  const data = ghAPIFn(`repos/${itemRepo}/pulls/${num}`);
  if (!data || !data.state) {
    out.result = "unknown";
    out.detail = "api error";
    setPendingAge(out, timestamp);
    return out;
  }

  out.review_comments = typeof data.review_comments === "number" ? data.review_comments : null;
  out.changed_files = typeof data.changed_files === "number" ? data.changed_files : null;
  out.additions = typeof data.additions === "number" ? data.additions : null;
  out.deletions = typeof data.deletions === "number" ? data.deletions : null;
  out.comments = typeof data.comments === "number" ? data.comments : null;

  if (data.merged === true) {
    out.result = "accepted";
    out.detail = "merged (strong)";
    if (data.created_at && data.merged_at) {
      out.resolution_sec = secondsBetween(data.created_at, data.merged_at);
    }
    if (out.review_comments === 0 && out.comments === 0) {
      out.zero_touch = true;
    }
    return out;
  }

  if (data.state === "closed") {
    const closingSignal = hasClosingSignal(itemRepo, num, data, ghAPIFn);
    out.result = "rejected";
    out.detail = closingSignal ? "closed without merge (strong)" : "closed without merge";
    if (data.created_at && data.closed_at) {
      out.resolution_sec = secondsBetween(data.created_at, data.closed_at);
    }
    return out;
  }

  if (data.state === "open") {
    const reviewsRaw = ghAPIFn(`repos/${itemRepo}/pulls/${num}/reviews`);
    if (reviewsRaw === null) {
      out.result = "unknown";
      out.detail = "reviews api error";
      setPendingAge(out, timestamp);
      return out;
    }
    const reviews = Array.isArray(reviewsRaw) ? reviewsRaw : [];
    const hasApproved = reviews.some(r => (r?.state || "").toUpperCase() === "APPROVED");
    const hasChangesRequested = reviews.some(r => (r?.state || "").toUpperCase() === "CHANGES_REQUESTED");

    if (hasApproved && !hasChangesRequested) {
      out.result = "accepted";
      out.detail = "approved without requested changes";
      return out;
    }
    if (hasChangesRequested && !hasApproved) {
      out.result = "pending";
      out.detail = "open with changes requested";
      setPendingAge(out, timestamp);
      return out;
    }
    if (reviews.length === 0) {
      setPendingAge(out, timestamp);
      if (isStalePending(out.pending_age_sec)) {
        out.result = "ignored";
        out.detail = "open and stale";
      } else {
        out.result = "pending";
        out.detail = "open with no reviews";
      }
      return out;
    }
    out.result = "unknown";
    out.detail = "open with mixed review state";
    setPendingAge(out, timestamp);
    return out;
  }

  out.result = "unknown";
  out.detail = "unknown pull request state";
  setPendingAge(out, timestamp);
  return out;
}

/**
 * Evaluate outcome for push_to_pull_request_branch.
 * @param {object} item
 * @param {string} itemRepo
 * @param {EvalResult} out
 * @param {(endpoint: string) => any} [ghAPIFn]
 * @returns {EvalResult}
 */
function evaluatePushToPullRequestBranchOutcome(item, itemRepo, out, ghAPIFn = ghAPI) {
  const num = resolvePRNumber(item);
  const timestamp = item.timestamp || "";
  const pushedShas = extractPushedCommitSHAs(item);
  const beforeHead = extractBeforeHeadSHA(item);

  if (!num || !itemRepo) {
    out.result = "unknown";
    out.detail = "missing pull request reference";
    setPendingAge(out, timestamp);
    return out;
  }

  const data = ghAPIFn(`repos/${itemRepo}/pulls/${num}`);
  if (!data || !data.state) {
    out.result = "unknown";
    out.detail = "api error";
    setPendingAge(out, timestamp);
    return out;
  }

  const currentHead = normalizeCommitSHA(data?.head?.sha);

  const pushedStillHead = currentHead ? pushedShas.some(sha => shaMatches(sha, currentHead)) : false;
  const commitRetentionResults = currentHead && pushedShas.length > 0 ? pushedShas.map(sha => isCommitInBranchHistory(itemRepo, sha, currentHead, ghAPIFn)) : [];
  const pushedIncluded = commitRetentionResults.some(inHistory => inHistory === true);
  const allPushedCommitsMissingFromHistory = commitRetentionResults.length > 0 && commitRetentionResults.every(inHistory => inHistory === false);

  if (data.merged === true) {
    out.result = "accepted";
    out.detail = pushedIncluded ? "merged with pushed commit retained (strong)" : "merged";
    if (data.created_at && data.merged_at) {
      out.resolution_sec = secondsBetween(data.created_at, data.merged_at);
    }
    return out;
  }

  if (data.state === "closed") {
    out.result = "rejected";
    out.detail = "closed without merge";
    if (data.created_at && data.closed_at) {
      out.resolution_sec = secondsBetween(data.created_at, data.closed_at);
    }
    return out;
  }

  if (data.state !== "open") {
    out.result = "unknown";
    out.detail = "unknown pull request state";
    setPendingAge(out, timestamp);
    return out;
  }

  if (pushedStillHead) {
    out.result = "accepted";
    out.detail = "pushed commit is current branch head";
    return out;
  }

  // A strong rejection requires before-head metadata from execution time so we
  // can distinguish "commit not retained" from "insufficient history context".
  if (pushedShas.length > 0 && allPushedCommitsMissingFromHistory && beforeHead) {
    out.result = "rejected";
    out.detail = "pushed commits were force-pushed away or branch reset";
    return out;
  }

  const reviewsRaw = ghAPIFn(`repos/${itemRepo}/pulls/${num}/reviews`);
  if (reviewsRaw === null) {
    out.result = "unknown";
    out.detail = "reviews api error";
    setPendingAge(out, timestamp);
    return out;
  }
  const reviews = Array.isArray(reviewsRaw) ? reviewsRaw : [];
  const hasReviewOnPushedCommit =
    pushedShas.length > 0 &&
    reviews.some(r => {
      const reviewCommit = normalizeCommitSHA(r?.commit_id);
      return reviewCommit ? pushedShas.some(sha => shaMatches(sha, reviewCommit)) : false;
    });

  if (!hasReviewOnPushedCommit) {
    setPendingAge(out, timestamp);
    if (isStalePending(out.pending_age_sec)) {
      out.result = "ignored";
      out.detail = "open and stale with no review on pushed commits";
    } else {
      out.result = "pending";
      out.detail = "open with no review on pushed commits";
    }
    return out;
  }

  out.result = "unknown";
  out.detail = "open with reviewed pushed commits";
  setPendingAge(out, timestamp);
  return out;
}

/**
 * @param {object} item
 * @returns {number}
 */
function resolvePRNumber(item) {
  if (typeof item.number === "number" && item.number > 0) return item.number;
  const candidates = [item.pull_request_number, item.pr_number, item.pr, item.pull_number, item.item_number];
  for (const candidate of candidates) {
    const n = Number.parseInt(String(candidate || ""), 10);
    if (Number.isInteger(n) && n > 0) return n;
  }
  const url = item.url || "";
  const prMatch = url.match(/\/pull\/(\d+)/);
  if (!prMatch) return 0;
  const n = Number.parseInt(prMatch[1], 10);
  return Number.isInteger(n) && n > 0 ? n : 0;
}

/**
 * @param {string | null | undefined} sha
 * @returns {string}
 */
function normalizeCommitSHA(sha) {
  if (!sha || typeof sha !== "string") return "";
  const normalized = sha.trim().toLowerCase();
  return /^[0-9a-f]{7,40}$/.test(normalized) ? normalized : "";
}

/**
 * Match SHAs across short/full representations (7-40 hex chars).
 * Returns true for exact matches and when the longer SHA starts with the
 * shorter SHA prefix (minimum 7 chars).
 *
 * @param {string} a
 * @param {string} b
 * @returns {boolean}
 */
function shaMatches(a, b) {
  const left = normalizeCommitSHA(a);
  const right = normalizeCommitSHA(b);
  if (!left || !right) return false;
  if (left === right) return true;
  const leftIsShorterOrEqual = left.length <= right.length;
  const shorter = leftIsShorterOrEqual ? left : right;
  const longer = leftIsShorterOrEqual ? right : left;
  return shorter.length >= 7 && longer.startsWith(shorter);
}

/**
 * @param {object} item
 * @returns {string[]}
 */
function extractPushedCommitSHAs(item) {
  /** @type {string[]} */
  const shas = [];
  // Intentionally exclude `item.head_sha`: it is ambiguous (tip-at-observation)
  // and not a reliable indicator of what commit(s) were pushed in this action.
  const candidates = [item.commit_sha, item.pushed_commit_sha, item?.metadata?.commit_sha, item?.metadata?.pushed_commit_sha];
  for (const candidate of candidates) {
    const normalized = normalizeCommitSHA(candidate);
    if (normalized) shas.push(normalized);
  }
  const listCandidates = [item.commit_shas, item.pushed_commit_shas, item?.metadata?.commit_shas, item?.metadata?.pushed_commit_shas];
  for (const list of listCandidates) {
    if (!Array.isArray(list)) continue;
    for (const value of list) {
      const normalized = normalizeCommitSHA(value);
      if (normalized) shas.push(normalized);
    }
  }
  return [...new Set(shas)];
}

/**
 * @param {object} item
 * @returns {string}
 */
function extractBeforeHeadSHA(item) {
  const candidates = [item.before_head_sha, item.previous_head_sha, item.head_sha_before, item.branch_head_before, item.pre_push_head_sha, item?.metadata?.before_head_sha, item?.metadata?.previous_head_sha, item?.metadata?.head_sha_before];
  for (const candidate of candidates) {
    const normalized = normalizeCommitSHA(candidate);
    if (normalized) return normalized;
  }
  return "";
}

/**
 * @param {string} repo
 * @param {number} number
 * @param {any} prData
 * @param {(endpoint: string) => object | null} ghAPIFn
 * @returns {boolean}
 */
function hasClosingSignal(repo, number, prData, ghAPIFn) {
  const labels = Array.isArray(prData?.labels) ? prData.labels : [];
  const hasClosingLabel = labels.some(label => {
    const name = String(label?.name || "").toLowerCase();
    return CLOSING_LABEL_KEYWORDS.some(keyword => name.includes(keyword));
  });
  if (hasClosingLabel) return true;

  const commentsRaw = ghAPIFn(`repos/${repo}/issues/${number}/comments`);
  if (!Array.isArray(commentsRaw)) return false;
  return commentsRaw.some(comment => {
    const body = String(comment?.body || "").toLowerCase();
    return CLOSING_COMMENT_KEYWORDS.some(keyword => body.includes(keyword));
  });
}

/**
 * @param {string} repo
 * @param {string} commitSHA
 * @param {string} branchHeadSHA
 * @param {(endpoint: string) => object | null} ghAPIFn
 * @returns {boolean | null}
 */
function isCommitInBranchHistory(repo, commitSHA, branchHeadSHA, ghAPIFn) {
  if (!commitSHA || !branchHeadSHA) return null;
  if (shaMatches(commitSHA, branchHeadSHA)) return true;
  const compareData = ghAPIFn(`repos/${repo}/compare/${commitSHA}...${branchHeadSHA}`);
  if (!compareData || typeof compareData.status !== "string") return null;
  const status = compareData.status.toLowerCase();
  // compare base...head semantics:
  // - ahead/identical => base commit is in head history
  // - behind => evaluated head is behind base, so base is not retained at this tip
  // - diverged => evaluated head diverged from base, so base is not retained
  if (status === "ahead" || status === "identical") return true;
  if (status === "behind" || status === "diverged") return false;
  return null;
}

/**
 * @returns {number}
 */
function staleThresholdSec() {
  const raw = Number.parseInt(String(process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS || ""), 10);
  if (Number.isInteger(raw) && raw > 0) return raw;
  return 7 * 24 * 60 * 60;
}

/**
 * @param {number | null} pendingAgeSec
 * @returns {boolean}
 */
function isStalePending(pendingAgeSec) {
  return typeof pendingAgeSec === "number" && pendingAgeSec >= staleThresholdSec();
}

/**
 * Set pending_age_sec on the result if the item has a timestamp.
 * @param {EvalResult} out
 * @param {string} timestamp
 * @param {number} [nowMs]
 */
function setPendingAge(out, timestamp, nowMs = Date.now()) {
  if (!timestamp) return;
  const itemEpoch = isoToEpoch(timestamp);
  if (itemEpoch === null) return;
  out.pending_age_sec = Math.floor(nowMs / 1000) - itemEpoch;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function main() {
  const repo = process.env.GITHUB_REPOSITORY || "";
  if (!repo) {
    console.error("GITHUB_REPOSITORY is not set");
    process.exit(1);
  }

  // Ensure directories exist
  fs.mkdirSync(CACHE_DIR, { recursive: true });
  fs.mkdirSync(OUTCOMES_DIR, { recursive: true });

  // Load seen-runs cache
  const seenIds = new Set(readJSON(SEEN_FILE, []));

  // Fetch recent successful runs
  const runsRaw = gh(["run", "list", "--repo", repo, "--limit", "200", "--json", "databaseId,conclusion,workflowName,event", "--jq", '[.[] | select(.conclusion == "success")] | .[0:150]']);

  if (!runsRaw || runsRaw === "[]" || runsRaw === "null") {
    console.log("No recent successful runs found");
    writeJSONAtomic(SUMMARY_PATH, { runs_checked: 0, total_outcomes: 0 });
    process.exit(0);
  }

  /** @type {Array<{databaseId: number, workflowName: string, event: string}>} */
  let runs;
  try {
    runs = JSON.parse(runsRaw);
  } catch {
    console.error("Failed to parse run list");
    writeJSONAtomic(SUMMARY_PATH, { runs_checked: 0, total_outcomes: 0 });
    process.exit(0);
  }

  // Counters
  let checked = 0;
  let accepted = 0;
  let rejected = 0;
  let ignored = 0;
  let pending = 0;
  let total = 0;
  let noop = 0;
  let zeroTouchCount = 0;
  let acceptedStrong = 0;
  let acceptedMedium = 0;
  let acceptedWeak = 0;
  let fallbackExistsOnlyCount = 0;
  /** @type {number[]} */
  const resolutionTimes = [];

  // Clear the evaluations file
  fs.writeFileSync(EVAL_JSONL, "");

  /** @type {number[]} */
  const evaluatedIds = [];

  for (const run of runs) {
    const runId = run.databaseId;
    const workflow = run.workflowName || "";
    const event = run.event || "";

    // Skip previously evaluated
    if (seenIds.has(runId)) continue;

    // Download artifact
    const itemDir = path.join(OUTCOMES_DIR, `run-${runId}`);
    const dlResult = gh(["run", "download", String(runId), "--repo", repo, "--name", "safe-outputs-items", "--dir", itemDir]);
    if (dlResult === null) continue;

    const manifestPath = path.join(itemDir, "safe-output-items.jsonl");
    if (!fs.existsSync(manifestPath)) continue;

    const manifest = readJSONL(manifestPath);
    if (manifest.length === 0) continue;

    // Separate actionable items from noops
    const actionable = manifest.filter(m => m.type && !NOOP_TYPES.has(m.type));
    const noops = manifest.filter(m => m.type && NOOP_TYPES.has(m.type));
    const runNoops = noops.length;
    const runItems = actionable.length;

    if (runItems === 0 && runNoops === 0) continue;

    noop += runNoops;

    console.log(`Run ${runId} (${workflow}): ${runItems} item(s), ${runNoops} noop(s) [trigger: ${event}]`);
    checked++;
    total += runItems;

    // Write noop entries
    for (const n of noops) {
      const normalized = normalizeOutcome("noop", n.type || "");
      fs.appendFileSync(
        EVAL_JSONL,
        JSON.stringify({
          type: n.type,
          url: "",
          repo,
          result: "noop",
          outcome_status: normalized.outcome_status,
          evidence_strength: normalized.evidence_strength,
          signal: normalized.signal,
          detail: n.type,
          workflow,
          run_id: runId,
          timestamp: "",
          event,
        }) + "\n"
      );
    }

    if (runItems === 0) {
      // Only noops — still mark as evaluated
      writeJSONAtomic(path.join(OUTCOMES_DIR, `run-${runId}.json`), {
        workflow,
        run_id: runId,
        items: 0,
        noops: runNoops,
        event,
      });
      evaluatedIds.push(runId);
      continue;
    }

    // Evaluate each actionable item
    for (const item of actionable) {
      const evalResult = evaluateItem(item, repo);
      const normalized = normalizeOutcome(evalResult.result, evalResult.detail);

      switch (normalized.outcome_status) {
        case "accepted":
          accepted++;
          switch (normalized.evidence_strength) {
            case "strong":
              acceptedStrong++;
              break;
            case "medium":
              acceptedMedium++;
              break;
            case "weak":
              acceptedWeak++;
              break;
          }
          if (evalResult.zero_touch === true) {
            zeroTouchCount++;
          }
          break;
        case "rejected":
          rejected++;
          break;
        case "ignored":
          ignored++;
          break;
        case "pending":
          pending++;
          break;
      }
      if (normalized.signal === "target_exists_only") {
        fallbackExistsOnlyCount++;
      }
      if (typeof evalResult.resolution_sec === "number" && evalResult.resolution_sec > 0) {
        resolutionTimes.push(evalResult.resolution_sec);
      }

      fs.appendFileSync(
        EVAL_JSONL,
        JSON.stringify({
          type: item.type || "",
          url: item.url || "",
          repo: item.repo || repo,
          result: evalResult.result,
          outcome_status: normalized.outcome_status,
          evidence_strength: normalized.evidence_strength,
          signal: normalized.signal,
          detail: evalResult.detail,
          workflow,
          run_id: runId,
          timestamp: item.timestamp || "",
          event,
          resolution_sec: evalResult.resolution_sec,
          pending_age_sec: evalResult.pending_age_sec,
          review_comments: evalResult.review_comments,
          changed_files: evalResult.changed_files,
          additions: evalResult.additions,
          deletions: evalResult.deletions,
          reactions_total: evalResult.reactions_total,
          reactions_positive: evalResult.reactions_positive,
          reactions_negative: evalResult.reactions_negative,
          comments: evalResult.comments,
          zero_touch: evalResult.zero_touch || false,
        }) + "\n"
      );
    }

    // Save per-run data
    writeJSONAtomic(path.join(OUTCOMES_DIR, `run-${runId}.json`), {
      workflow,
      run_id: runId,
      items: runItems,
      noops: runNoops,
      event,
    });

    evaluatedIds.push(runId);
  }

  // Compute fleet summary
  const resolved = accepted + rejected;
  const acceptanceRate = resolved > 0 ? accepted / resolved : 0;
  const wasteRate = total > 0 ? rejected / total : 0;
  const noopRate = total + noop > 0 ? noop / (total + noop) : 0;

  // Economics: zero-touch rate and median time-to-outcome
  const zeroTouchRate = accepted > 0 ? zeroTouchCount / accepted : 0;
  resolutionTimes.sort((a, b) => a - b);
  let medianResolutionSec = null;
  if (resolutionTimes.length > 0) {
    const mid = Math.floor(resolutionTimes.length / 2);
    medianResolutionSec = resolutionTimes.length % 2 !== 0 ? resolutionTimes[mid] : Math.round((resolutionTimes[mid - 1] + resolutionTimes[mid]) / 2);
  }

  writeJSONAtomic(SUMMARY_PATH, {
    runs_checked: checked,
    total_outcomes: total,
    accepted,
    rejected,
    ignored,
    pending,
    noop,
    accepted_strong: acceptedStrong,
    accepted_medium: acceptedMedium,
    accepted_weak: acceptedWeak,
    fallback_exists_only_count: fallbackExistsOnlyCount,
    acceptance_rate: Math.round(acceptanceRate * 10000) / 10000,
    waste_rate: Math.round(wasteRate * 10000) / 10000,
    noop_rate: Math.round(noopRate * 10000) / 10000,
    zero_touch: zeroTouchCount,
    zero_touch_rate: Math.round(zeroTouchRate * 10000) / 10000,
    median_resolution_sec: medianResolutionSec,
    date: new Date().toISOString().slice(0, 10),
  });

  // Update seen-runs cache: merge old + new, keep last 500
  const merged = [...new Set([...seenIds, ...evaluatedIds])].sort((a, b) => a - b).slice(-500);
  writeJSONAtomic(SEEN_FILE, merged);

  console.log(`✓ Checked ${checked} runs, ${total} outcomes`);
  console.log(`  Accepted: ${accepted}, Rejected: ${rejected}, Ignored: ${ignored}, Pending: ${pending}, Noop: ${noop}`);
  console.log(`  Acceptance rate: ${acceptanceRate.toFixed(4)}`);
  console.log(JSON.stringify(readJSON(SUMMARY_PATH, {}), null, 2));
}

if (require.main === module) {
  main();
}

module.exports = {
  main,
  evaluateItem,
  evaluateAddReviewer,
  evaluateUpdateIssue,
  evaluateUpdatePullRequest,
  evaluateSubmitPullRequestReview,
  evaluateCreateIssue,
  evaluateAddComment,
  evaluateAddLabels,
  evaluateCreatePullRequestOutcome,
  evaluatePushToPullRequestBranchOutcome,
  normalizeOutcome,
  readJSONL,
  secondsBetween,
  isoToEpoch,
};
