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

const DEFAULT_ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC = 60 * 60;
const DEFAULT_LABEL_RETENTION_WINDOW_SEC = 24 * 60 * 60;

/**
 * Read a positive integer from env with fallback.
 * @param {string} key
 * @param {number} fallback
 * @returns {number}
 */
function getEnvPositiveInt(key, fallback) {
  const raw = process.env[key];
  const parsed = Number.parseInt(raw || "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

const ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC = getEnvPositiveInt("OUTCOME_ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC", DEFAULT_ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC);
const LABEL_RETENTION_WINDOW_SEC = getEnvPositiveInt("OUTCOME_LABEL_RETENTION_WINDOW_SEC", DEFAULT_LABEL_RETENTION_WINDOW_SEC);

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
  const positive = (reactions["+1"] || 0) + (reactions.heart || 0) + (reactions.hooray || 0) + (reactions.rocket || 0);
  const negative = (reactions["-1"] || 0) + (reactions.confused || 0);
  const total = reactions.total_count != null ? reactions.total_count : positive + negative + (reactions.laugh || 0) + (reactions.eyes || 0);
  return { total, positive, negative };
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

  if (issue.state === "open" && (hasMergedPRReference || hasCommitReference || hasClosingActionReference)) {
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
    if (typeof out.resolution_sec === "number" && out.resolution_sec <= ISSUE_IMMEDIATE_CLOSE_WINDOW_SEC && closeActor && closeActor !== authorLogin) {
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
    out.result = "rejected";
    out.detail = "rejected:strong deleted";
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

  const labelsBefore = Array.isArray(item.labelsBefore) ? item.labelsBefore.map(l => String(l || "").trim()).filter(Boolean) : [];
  const labelsAdded = Array.isArray(item.labelsAdded) ? item.labelsAdded.map(l => String(l || "").trim()).filter(Boolean) : Array.isArray(item.labels) ? item.labels.map(l => String(l || "").trim()).filter(Boolean) : [];

  if (labelsBefore.length === 0 || labelsAdded.length === 0) {
    out.result = "unknown";
    out.detail = "unknown: missing persisted label before-state";
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
  const trackedAdded = labelsAdded.filter(l => !labelsBefore.includes(l));
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
 * Evaluate a single safe-output item against the GitHub API.
 * @param {object} item
 * @param {string} defaultRepo
 * @param {EvaluateDeps} [deps]
 * @returns {EvalResult}
 */
function evaluateItem(item, defaultRepo, deps = {}) {
  const url = item.url || "";
  const itemRepo = item.repo || defaultRepo;
  const timestamp = item.timestamp || "";
  const apiGet = typeof deps.ghAPI === "function" ? deps.ghAPI : ghAPI;
  const nowMs = typeof deps.nowMs === "number" ? deps.nowMs : Date.now();

  /** @type {EvalResult} */
  const out = {
    result: "pending",
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

  if (!url) {
    out.detail = "no url";
    setPendingAge(out, timestamp, nowMs);
    return out;
  }

  if (item.type === "create_issue") {
    return evaluateCreateIssue(item, itemRepo, timestamp, out, apiGet, nowMs);
  }
  if (item.type === "add_comment") {
    return evaluateAddComment(item, itemRepo, timestamp, out, apiGet, nowMs);
  }
  if (item.type === "add_labels") {
    return evaluateAddLabels(item, itemRepo, timestamp, out, apiGet, nowMs);
  }

  // Issues / issue-comments
  const issueMatch = url.match(/\/(?:issues|pull)\/(\d+)/);
  if (/\/issues\/\d+|\/issuecomment-/.test(url) && issueMatch) {
    const num = issueMatch[1];
    const data = apiGet(`repos/${itemRepo}/issues/${num}`);
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
    const data = apiGet(`repos/${itemRepo}/pulls/${num}`);
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
  out.result = "accepted";
  out.detail = "object exists";
  return out;
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
  const ignored = 0;
  let pending = 0;
  let total = 0;
  let noop = 0;
  let zeroTouchCount = 0;
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
      fs.appendFileSync(
        EVAL_JSONL,
        JSON.stringify({
          type: n.type,
          url: "",
          repo,
          result: "noop",
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

      switch (evalResult.result) {
        case "accepted":
          accepted++;
          if (evalResult.zero_touch === true) {
            zeroTouchCount++;
          }
          break;
        case "rejected":
          rejected++;
          break;
        default:
          pending++;
          break;
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
  evaluateCreateIssue,
  evaluateAddComment,
  evaluateAddLabels,
  readJSONL,
  secondsBetween,
  isoToEpoch,
};
