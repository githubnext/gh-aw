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
const CLOSING_LABEL_KEYWORDS = ["not planned", "not_planned", "wontfix", "won't fix", "duplicate", "invalid", "declined", "rejected"];
const CLOSING_COMMENT_KEYWORDS = ["not planned", "won't fix", "wontfix", "duplicate", "invalid", "declined", "rejected", "closing as", "closed as", "closing this"];

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
 * @property {"strong"|"medium"|"weak"} evidence_strength
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
 * Normalize legacy result/detail pairs into the shared outcome model.
 * @param {string} result
 * @param {string} detail
 * @returns {{ outcome_status: "accepted"|"rejected"|"pending"|"ignored"|"skipped"|"unknown", evidence_strength: "strong"|"medium"|"weak", signal: string }}
 */
function normalizeOutcome(result, detail) {
  const normalizedDetail = String(detail || "")
    .toLowerCase()
    .trim();

  if (result === "noop") {
    return { outcome_status: "skipped", evidence_strength: "weak", signal: "noop" };
  }
  if (normalizedDetail === "object still exists") {
    return { outcome_status: "unknown", evidence_strength: "weak", signal: "target_exists_only" };
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
 * Evaluate a single safe-output item against the GitHub API.
 * @param {object} item
 * @param {string} defaultRepo
 * @param {{ ghAPI?: (endpoint: string) => object | null }} [options]
 * @returns {EvalResult}
 */
function evaluateItem(item, defaultRepo, options = {}) {
  const url = item.url || "";
  const itemRepo = item.repo || defaultRepo;
  const timestamp = item.timestamp || "";
  const type = item.type || "";
  const ghAPIFn = typeof options.ghAPI === "function" ? options.ghAPI : ghAPI;

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

  if (!url) {
    out.detail = "no url";
    setPendingAge(out, timestamp);
    return out;
  }

  // Issues / issue-comments
  const issueMatch = url.match(/\/(?:issues|pull)\/(\d+)/);
  if (/\/issues\/\d+|\/issuecomment-/.test(url) && issueMatch) {
    const num = issueMatch[1];
    const data = ghAPIFn(`repos/${itemRepo}/issues/${num}`);
    if (!data || !data.state) {
      out.detail = "api error";
      setPendingAge(out, timestamp);
      return out;
    }
    out.result = "accepted";
    out.detail = data.state;
    out.comments = typeof data.comments === "number" ? data.comments : null;

    // Reactions on issues
    if (data.reactions && typeof data.reactions === "object") {
      const r = data.reactions;
      const positive = (r["+1"] || 0) + (r.heart || 0) + (r.hooray || 0) + (r.rocket || 0);
      const negative = (r["-1"] || 0) + (r.confused || 0);
      out.reactions_total = r.total_count != null ? r.total_count : positive + negative + (r.laugh || 0) + (r.eyes || 0);
      out.reactions_positive = positive;
      out.reactions_negative = negative;
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
      setPendingAge(out, timestamp);
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
      const r = data.reactions;
      const positive = (r["+1"] || 0) + (r.heart || 0) + (r.hooray || 0) + (r.rocket || 0);
      const negative = (r["-1"] || 0) + (r.confused || 0);
      out.reactions_total = r.total_count != null ? r.total_count : positive + negative + (r.laugh || 0) + (r.eyes || 0);
      out.reactions_positive = positive;
      out.reactions_negative = negative;
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
      setPendingAge(out, timestamp);
    } else {
      out.detail = "api error";
      setPendingAge(out, timestamp);
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
 */
function setPendingAge(out, timestamp) {
  if (!timestamp) return;
  const itemEpoch = isoToEpoch(timestamp);
  if (itemEpoch === null) return;
  out.pending_age_sec = Math.floor(Date.now() / 1000) - itemEpoch;
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
  evaluateCreatePullRequestOutcome,
  evaluatePushToPullRequestBranchOutcome,
  normalizeOutcome,
  readJSONL,
  secondsBetween,
  isoToEpoch,
};
