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
 * Evaluate a single safe-output item against the GitHub API.
 * @param {object} item
 * @param {string} defaultRepo
 * @returns {EvalResult}
 */
function evaluateItem(item, defaultRepo) {
  const url = item.url || "";
  const itemRepo = item.repo || defaultRepo;
  const timestamp = item.timestamp || "";

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
    setPendingAge(out, timestamp);
    return out;
  }

  // Issues / issue-comments
  const issueMatch = url.match(/\/(?:issues|pull)\/(\d+)/);
  if (/\/issues\/\d+|\/issuecomment-/.test(url) && issueMatch) {
    const num = issueMatch[1];
    const data = ghAPI(`repos/${itemRepo}/issues/${num}`);
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
    const data = ghAPI(`repos/${itemRepo}/pulls/${num}`);
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
  out.result = "accepted";
  out.detail = "object exists";
  return out;
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

module.exports = { main, evaluateItem, readJSONL, secondsBetween, isoToEpoch };
