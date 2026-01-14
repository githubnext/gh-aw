#!/usr/bin/env node

import fs from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { spawnSync } from "node:child_process";

const repo = process.env.PLAYGROUND_SNAPSHOTS_REPO; // "owner/repo"
const ref = process.env.PLAYGROUND_SNAPSHOTS_REF || "main";
const snapshotsPath = process.env.PLAYGROUND_SNAPSHOTS_PATH || "snapshots";
const token = process.env.PLAYGROUND_SNAPSHOTS_TOKEN || process.env.GITHUB_TOKEN;

// Default keeps backward-compatible behavior (download JSON snapshots from a repo path).
// Set PLAYGROUND_SNAPSHOTS_MODE=actions to generate snapshots from GitHub Actions runs.
const mode = process.env.PLAYGROUND_SNAPSHOTS_MODE || "contents";
const workflowsDir = process.env.PLAYGROUND_SNAPSHOTS_WORKFLOWS_DIR || ".github/workflows";
const workflowIdsCsv = process.env.PLAYGROUND_SNAPSHOTS_WORKFLOW_IDS || "";
const branch = process.env.PLAYGROUND_SNAPSHOTS_BRANCH || ref || "main";

const outDir = path.resolve("src/assets/playground-snapshots");

const MAX_FILES = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_FILES || 50);
const MAX_FILE_BYTES = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_FILE_BYTES || 256 * 1024);
const MAX_TOTAL_BYTES = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_TOTAL_BYTES || 2 * 1024 * 1024);

const INCLUDE_LOGS = String(process.env.PLAYGROUND_SNAPSHOTS_INCLUDE_LOGS || "1") !== "0";
const MAX_LOG_BYTES = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_LOG_BYTES || 512 * 1024);
const MAX_LOG_LINES_TOTAL = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_LOG_LINES_TOTAL || 1200);
const MAX_LOG_LINES_PER_GROUP = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_LOG_LINES_PER_GROUP || 120);
const MAX_LOG_LINE_CHARS = Number(process.env.PLAYGROUND_SNAPSHOTS_MAX_LOG_LINE_CHARS || 300);

const SAFE_FILENAME = /^[a-z0-9][a-z0-9._-]{0,120}\.json$/;

function headerAuth() {
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

async function ghJson(url) {
  const res = await fetch(url, {
    headers: {
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
      ...headerAuth(),
    },
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`GitHub API ${res.status} ${res.statusText}: ${text}`);
  }

  return res.json();
}

async function download(url) {
  const res = await fetch(url, { headers: { ...headerAuth() } });
  if (!res.ok) throw new Error(`Download failed ${res.status} ${res.statusText}: ${url}`);
  return Buffer.from(await res.arrayBuffer());
}

async function downloadJobLogsZip(jobId) {
  if (!repo) throw new Error("[playground-snapshots] Missing PLAYGROUND_SNAPSHOTS_REPO");

  const url = `https://api.github.com/repos/${repo}/actions/jobs/${encodeURIComponent(String(jobId))}/logs`;
  const res = await fetch(url, {
    headers: {
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
      ...headerAuth(),
    },
    redirect: "manual",
  });

  if (res.status >= 300 && res.status < 400 && res.headers.get("location")) {
    const loc = res.headers.get("location");
    // Signed URLs typically don't need (or accept) Authorization.
    const res2 = await fetch(loc, { redirect: "follow" });
    if (!res2.ok) throw new Error(`Download job logs redirect failed ${res2.status} ${res2.statusText}`);
    return Buffer.from(await res2.arrayBuffer());
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`Download job logs failed ${res.status} ${res.statusText}: ${text}`);
  }

  return Buffer.from(await res.arrayBuffer());
}

function looksLikeZip(bytes) {
  // ZIP files start with: 0x50 0x4B ("PK")
  return Buffer.isBuffer(bytes) && bytes.length >= 2 && bytes[0] === 0x50 && bytes[1] === 0x4b;
}

async function extractJobLogsText(bytes) {
  // GitHub Actions job logs are served via redirect and may be either:
  // - a ZIP archive (legacy / some hosts)
  // - a plain text file (e.g. job-logs.txt on blob storage)
  if (!looksLikeZip(bytes)) {
    return Buffer.from(bytes).toString("utf8");
  }

  // Prefer system unzip to avoid extra npm dependencies.
  const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), "gh-aw-playground-logs-"));
  const zipPath = path.join(tmpDir, "logs.zip");
  try {
    await fs.writeFile(zipPath, bytes);

    const res = spawnSync("unzip", ["-p", zipPath], {
      encoding: null,
      maxBuffer: Math.max(MAX_LOG_BYTES, 512 * 1024),
    });

    if (res.error) throw res.error;
    if (res.status !== 0) {
      const stderr = res.stderr ? res.stderr.toString("utf8") : "";
      throw new Error(`unzip failed (exit ${res.status}): ${stderr}`);
    }

    const out = Buffer.isBuffer(res.stdout) ? res.stdout : Buffer.from(String(res.stdout || ""), "utf8");
    return out.toString("utf8");
  } finally {
    await fs.rm(tmpDir, { recursive: true, force: true }).catch(() => undefined);
  }
}

function normalizeLogLine(line) {
  let text = String(line ?? "");
  if (text.length > MAX_LOG_LINE_CHARS) text = text.slice(0, MAX_LOG_LINE_CHARS) + "…";
  return text;
}

function normalizeKey(value) {
  return String(value || "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "");
}

function parseGroupedLogs(text) {
  const root = { title: "Job logs", lines: [], children: [] };
  const stack = [root];

  let totalKept = 0;

  const rawLines = String(text || "")
    .replace(/\r\n/g, "\n")
    .replace(/\r/g, "\n")
    .split("\n");

  for (const raw of rawLines) {
    const line = normalizeLogLine(raw);

    // GitHub-hosted logs often prefix timestamps before group markers.
    // Also support `::group::` markers.
    const ghaGroupIdx = line.indexOf("##[group]");
    const ghaEndGroupIdx = line.indexOf("##[endgroup]");
    const ghaAltGroupIdx = line.indexOf("::group::");
    const ghaAltEndGroupIdx = line.indexOf("::endgroup::");

    if (ghaGroupIdx !== -1) {
      const title = line.slice(ghaGroupIdx + "##[group]".length).trim() || "Group";
      const group = { title, lines: [], children: [] };
      stack[stack.length - 1].children.push(group);
      stack.push(group);
      continue;
    }

    if (ghaEndGroupIdx !== -1) {
      if (stack.length > 1) stack.pop();
      continue;
    }

    if (ghaAltGroupIdx !== -1) {
      const title = line.slice(ghaAltGroupIdx + "::group::".length).trim() || "Group";
      const group = { title, lines: [], children: [] };
      stack[stack.length - 1].children.push(group);
      stack.push(group);
      continue;
    }

    if (ghaAltEndGroupIdx !== -1) {
      if (stack.length > 1) stack.pop();
      continue;
    }

    const current = stack[stack.length - 1];
    if (totalKept >= MAX_LOG_LINES_TOTAL) {
      current.truncated = true;
      continue;
    }

    if ((current.lines?.length ?? 0) >= MAX_LOG_LINES_PER_GROUP) {
      current.omittedLineCount = (current.omittedLineCount ?? 0) + 1;
      continue;
    }

    current.lines.push(line);
    totalKept += 1;
  }

  // Prune empty groups (keep ones that have children).
  const prune = g => {
    const kids = Array.isArray(g.children) ? g.children.map(prune).filter(Boolean) : [];
    const lines = Array.isArray(g.lines) ? g.lines : [];
    const hasContent = lines.length > 0 || kids.length > 0 || (g.omittedLineCount ?? 0) > 0;
    if (!hasContent) return null;
    return {
      title: g.title,
      ...(lines.length > 0 ? { lines } : {}),
      ...((g.omittedLineCount ?? 0) > 0 ? { omittedLineCount: g.omittedLineCount } : {}),
      ...(kids.length > 0 ? { children: kids } : {}),
      ...(g.truncated ? { truncated: true } : {}),
    };
  };

  return prune(root);
}

function findBestGroupForStep(jobLogGroup, stepName) {
  if (!jobLogGroup || !stepName) return undefined;
  const target = normalizeKey(stepName);
  if (!target) return undefined;

  // Walk depth-first, look for the closest title match.
  const candidates = [];
  const visit = g => {
    if (!g || typeof g !== "object") return;
    const title = String(g.title || "");
    const titleKey = normalizeKey(title);
    if (titleKey === target) candidates.push({ score: 100, g });
    else if (titleKey.endsWith(target) || titleKey.includes(target)) candidates.push({ score: 50, g });
    else if (titleKey.startsWith("run" + target) || titleKey.startsWith("post" + target)) candidates.push({ score: 40, g });
    const kids = Array.isArray(g.children) ? g.children : [];
    for (const k of kids) visit(k);
  };
  visit(jobLogGroup);

  candidates.sort((a, b) => b.score - a.score);
  return candidates[0]?.g;
}

function asString(value, label) {
  if (typeof value !== "string") throw new Error(`Invalid snapshot field '${label}': expected string`);
  return value;
}

function asOptionalString(value, label) {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== "string") throw new Error(`Invalid snapshot field '${label}': expected string or undefined`);
  return value;
}

function asArray(value, label) {
  if (!Array.isArray(value)) throw new Error(`Invalid snapshot field '${label}': expected array`);
  return value;
}

function validateConclusion(value, label) {
  const allowed = new Set(["success", "failure", "cancelled", "skipped", "neutral", "timed_out", "action_required", "stale", null]);

  if (!allowed.has(value)) {
    throw new Error(`Invalid snapshot field '${label}': unexpected conclusion '${String(value)}'`);
  }
  return value;
}

function validateSnapshotJson(raw, fallbackWorkflowId) {
  if (raw === null || typeof raw !== "object") throw new Error("Snapshot JSON must be an object");

  const workflowId = asString(raw.workflowId ?? fallbackWorkflowId, "workflowId");
  const updatedAt = asString(raw.updatedAt, "updatedAt");
  const runUrl = asOptionalString(raw.runUrl ?? raw?.run?.html_url, "runUrl");
  const conclusion = validateConclusion(raw.conclusion ?? null, "conclusion");
  const jobsRaw = asArray(raw.jobs ?? [], "jobs");

  /** @type {Array<any>} */
  const jobs = [];
  for (const job of jobsRaw) {
    if (job === null || typeof job !== "object") throw new Error("Invalid job entry: expected object");
    const jobName = asString(job.name, "jobs[].name");
    const jobConclusion = validateConclusion(job.conclusion ?? null, "jobs[].conclusion");
    const stepsRaw = asArray(job.steps ?? [], "jobs[].steps");

    const jobSummary = asOptionalString(job.summary, "jobs[].summary");

    const jobId = asOptionalNumber(job.id, "jobs[].id");
    const jobStatus = asOptionalString(job.status, "jobs[].status");
    const jobStartedAt = asOptionalIsoDateString(job.startedAt ?? job.started_at, "jobs[].startedAt");
    const jobCompletedAt = asOptionalIsoDateString(job.completedAt ?? job.completed_at, "jobs[].completedAt");
    const jobUrl = asOptionalString(job.url, "jobs[].url");

    const jobLog = validateOptionalLogGroup(job.log, "jobs[].log");

    /** @type {Array<any>} */
    const steps = [];
    for (const step of stepsRaw) {
      if (step === null || typeof step !== "object") throw new Error("Invalid step entry: expected object");

      const stepNumber = asOptionalNumber(step.number, "jobs[].steps[].number");
      const stepStatus = asOptionalString(step.status, "jobs[].steps[].status");
      const stepStartedAt = asOptionalIsoDateString(step.startedAt ?? step.started_at, "jobs[].steps[].startedAt");
      const stepCompletedAt = asOptionalIsoDateString(step.completedAt ?? step.completed_at, "jobs[].steps[].completedAt");

      const stepLog = validateOptionalLogGroup(step.log, "jobs[].steps[].log");

      steps.push({
        name: asString(step.name, "jobs[].steps[].name"),
        conclusion: validateConclusion(step.conclusion ?? null, "jobs[].steps[].conclusion"),
        ...(typeof stepNumber === "number" ? { number: stepNumber } : {}),
        ...(typeof stepStatus === "string" ? { status: stepStatus } : {}),
        ...(typeof stepStartedAt === "string" ? { startedAt: stepStartedAt } : {}),
        ...(typeof stepCompletedAt === "string" ? { completedAt: stepCompletedAt } : {}),
        ...(stepLog ? { log: stepLog } : {}),
      });
    }

    jobs.push({
      name: jobName,
      conclusion: jobConclusion,
      steps,
      ...(typeof jobSummary === "string" && jobSummary.trim().length > 0 ? { summary: jobSummary } : {}),
      ...(typeof jobId === "number" ? { id: jobId } : {}),
      ...(typeof jobStatus === "string" ? { status: jobStatus } : {}),
      ...(typeof jobStartedAt === "string" ? { startedAt: jobStartedAt } : {}),
      ...(typeof jobCompletedAt === "string" ? { completedAt: jobCompletedAt } : {}),
      ...(typeof jobUrl === "string" ? { url: jobUrl } : {}),
      ...(jobLog ? { log: jobLog } : {}),
    });
  }

  // Normalize output to the minimal schema used by the docs UI.
  return {
    workflowId,
    ...(runUrl ? { runUrl } : {}),
    updatedAt,
    conclusion,
    jobs,
  };
}

function validateOptionalLogGroup(value, label) {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== "object") throw new Error(`Invalid snapshot field '${label}': expected object or undefined`);

  const title = asString(value.title, `${label}.title`);
  const linesRaw = value.lines;
  const lines = Array.isArray(linesRaw) ? linesRaw.filter(x => typeof x === "string") : undefined;
  const omittedLineCount = asOptionalNumber(value.omittedLineCount, `${label}.omittedLineCount`);
  const truncated = typeof value.truncated === "boolean" ? value.truncated : undefined;

  const childrenRaw = value.children;
  const children = Array.isArray(childrenRaw) ? childrenRaw.map((c, idx) => validateOptionalLogGroup(c, `${label}.children[${idx}]`)).filter(Boolean) : undefined;

  return {
    title,
    ...(lines && lines.length > 0 ? { lines } : {}),
    ...(typeof omittedLineCount === "number" && omittedLineCount > 0 ? { omittedLineCount } : {}),
    ...(children && children.length > 0 ? { children } : {}),
    ...(typeof truncated === "boolean" ? { truncated } : {}),
  };
}

function asOptionalNumber(value, label) {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== "number") throw new Error(`Invalid snapshot field '${label}': expected number or undefined`);
  return value;
}

function asOptionalConclusion(value, label) {
  if (value === undefined) return undefined;
  return validateConclusion(value, label);
}

function asOptionalIsoDateString(value, label) {
  const v = asOptionalString(value, label);
  if (!v) return v;
  // Minimal sanity check; we don't want to reject slightly different formats.
  if (!/\d{4}-\d{2}-\d{2}T/.test(v)) return v;
  return v;
}

async function listWorkflowIdsFromLocalAssets() {
  const workflowsAssetsDir = path.resolve("src/assets/playground-workflows/user-owned");
  const entries = await fs.readdir(workflowsAssetsDir).catch(() => []);
  return entries
    .filter(f => f.endsWith(".lock.yml"))
    .map(f => f.slice(0, -".lock.yml".length))
    .filter(Boolean)
    .sort();
}

async function fetchLatestRunSnapshotFromActionsApi(workflowId) {
  if (!repo) throw new Error("[playground-snapshots] Missing PLAYGROUND_SNAPSHOTS_REPO");

  // GitHub API allows workflow identifier to be either numeric ID or file name.
  // These playground workflows are typically stored as {id}.lock.yml under .github/workflows.
  const workflowFileName = `${workflowId}.lock.yml`;

  const runsUrl = `https://api.github.com/repos/${repo}/actions/workflows/${encodeURIComponent(workflowFileName)}/runs?per_page=1&branch=${encodeURIComponent(branch)}`;
  const runsJson = await ghJson(runsUrl);
  const runs = Array.isArray(runsJson?.workflow_runs) ? runsJson.workflow_runs : [];
  const run = runs[0];

  if (!run || typeof run !== "object") {
    return {
      workflowId,
      updatedAt: new Date().toISOString(),
      conclusion: null,
      jobs: [],
    };
  }

  const runId = asOptionalNumber(run.id, "run.id");
  const runUrl = asOptionalString(run.html_url, "run.html_url");
  const updatedAt = asOptionalIsoDateString(run.updated_at, "run.updated_at") || new Date().toISOString();
  const conclusion = asOptionalConclusion(run.conclusion ?? null, "run.conclusion") ?? null;

  /** @type {Array<any>} */
  let jobs = [];
  if (typeof runId === "number") {
    const jobsUrl = `https://api.github.com/repos/${repo}/actions/runs/${encodeURIComponent(String(runId))}/jobs?per_page=100`;
    const jobsJson = await ghJson(jobsUrl);
    const jobsRaw = Array.isArray(jobsJson?.jobs) ? jobsJson.jobs : [];

    jobs = jobsRaw.map(j => {
      const jobName = asString(j?.name ?? "Unnamed job", "jobs[].name");
      const jobConclusion = asOptionalConclusion(j?.conclusion ?? null, "jobs[].conclusion") ?? null;
      const stepsRaw = Array.isArray(j?.steps) ? j.steps : [];
      const steps = stepsRaw.slice(0, 200).map(s => ({
        name: asString(s?.name ?? "Unnamed step", "jobs[].steps[].name"),
        conclusion: asOptionalConclusion(s?.conclusion ?? null, "jobs[].steps[].conclusion") ?? null,
        // Extra fields for richer UI (ignored by current renderer but useful for future improvements)
        ...(typeof s?.number === "number" ? { number: s.number } : {}),
        ...(typeof s?.status === "string" ? { status: s.status } : {}),
        ...(typeof s?.started_at === "string" ? { startedAt: s.started_at } : {}),
        ...(typeof s?.completed_at === "string" ? { completedAt: s.completed_at } : {}),
      }));

      return {
        name: jobName,
        conclusion: jobConclusion,
        steps,
        ...(typeof j?.id === "number" ? { id: j.id } : {}),
        ...(typeof j?.status === "string" ? { status: j.status } : {}),
        ...(typeof j?.started_at === "string" ? { startedAt: j.started_at } : {}),
        ...(typeof j?.completed_at === "string" ? { completedAt: j.completed_at } : {}),
        ...(typeof j?.html_url === "string" ? { url: j.html_url } : {}),
      };
    });

    if (INCLUDE_LOGS) {
      for (const job of jobs) {
        if (typeof job?.id !== "number") continue;
        try {
          const zipBytes = await downloadJobLogsZip(job.id);
          if (zipBytes.length > MAX_LOG_BYTES) {
            job.log = {
              title: "Job logs",
              omittedLineCount: 0,
              truncated: true,
              lines: [`(logs payload is ${zipBytes.length} bytes; max ${MAX_LOG_BYTES} bytes)`],
            };
            continue;
          }

          const text = await extractJobLogsText(zipBytes);
          const grouped = parseGroupedLogs(text);
          if (grouped) {
            job.log = grouped;
            // Attach per-step logs.
            // Best effort: try to find the step's group. Fallback to a tiny placeholder
            // so every step remains expandable (and users can jump to job-level logs).
            for (const step of job.steps || []) {
              const candidates = [step.name, `Run ${step.name}`, `Post ${step.name}`].filter(Boolean);

              let match;
              for (const candidate of candidates) {
                match = findBestGroupForStep(grouped, candidate);
                if (match) break;
              }

              step.log = match || {
                title: `Step logs: ${step.name}`,
                lines: ["(No separate log group found for this step. See job logs above.)"],
              };
            }
          }
        } catch (err) {
          job.log = {
            title: "Job logs (unavailable)",
            lines: [String(err?.message || err)],
            truncated: true,
          };
        }
      }
    }
  }

  return {
    workflowId,
    ...(runUrl ? { runUrl } : {}),
    updatedAt,
    conclusion,
    jobs,
    // Extra run-level metadata (ignored by current renderer).
    ...(typeof runId === "number" ? { runId } : {}),
    ...(typeof run?.run_number === "number" ? { runNumber: run.run_number } : {}),
    ...(typeof run?.run_attempt === "number" ? { runAttempt: run.run_attempt } : {}),
    ...(typeof run?.status === "string" ? { status: run.status } : {}),
    ...(typeof run?.event === "string" ? { event: run.event } : {}),
    ...(typeof run?.head_branch === "string" ? { headBranch: run.head_branch } : {}),
    ...(typeof run?.head_sha === "string" ? { headSha: run.head_sha } : {}),
    ...(typeof run?.created_at === "string" ? { createdAt: run.created_at } : {}),
  };
}

async function fetchFromContentsApi() {
  await fs.mkdir(outDir, { recursive: true });

  const url = `https://api.github.com/repos/${repo}/contents/${encodeURIComponent(snapshotsPath)}?ref=${encodeURIComponent(ref)}`;
  console.log(`[playground-snapshots] Fetching ${repo}@${ref}:${snapshotsPath}`);

  const listing = await ghJson(url);
  if (!Array.isArray(listing)) {
    throw new Error("[playground-snapshots] Expected directory listing (array) from GitHub contents API.");
  }

  const jsonFiles = listing.filter(i => i && i.type === "file" && typeof i.name === "string" && i.name.endsWith(".json")).filter(i => SAFE_FILENAME.test(i.name));

  if (jsonFiles.length > MAX_FILES) {
    throw new Error(`[playground-snapshots] Refusing to fetch ${jsonFiles.length} files (max ${MAX_FILES}).`);
  }

  if (jsonFiles.length === 0) {
    console.warn("[playground-snapshots] No .json files found; leaving existing snapshots as-is.");
    return;
  }

  // Clean output directory first so removals in the snapshots repo are reflected.
  const existing = await fs.readdir(outDir).catch(() => []);
  await Promise.all(existing.filter(f => f.endsWith(".json")).map(f => fs.rm(path.join(outDir, f), { force: true })));

  let totalBytes = 0;
  for (const file of jsonFiles) {
    const bytes = await download(file.download_url);

    if (bytes.length > MAX_FILE_BYTES) {
      throw new Error(`[playground-snapshots] Refusing oversized snapshot ${file.name} (${bytes.length} bytes; max ${MAX_FILE_BYTES}).`);
    }

    totalBytes += bytes.length;
    if (totalBytes > MAX_TOTAL_BYTES) {
      throw new Error(`[playground-snapshots] Refusing snapshots total ${totalBytes} bytes (max ${MAX_TOTAL_BYTES}).`);
    }

    const fallbackWorkflowId = file.name.slice(0, -".json".length);
    const raw = JSON.parse(bytes.toString("utf8"));
    const normalized = validateSnapshotJson(raw, fallbackWorkflowId);

    // Validate filename at point of use to prevent path traversal attacks
    const safeFilename = path.basename(file.name);
    if (!SAFE_FILENAME.test(safeFilename)) {
      throw new Error(`[playground-snapshots] Refusing unsafe filename: ${safeFilename}`);
    }
    const outputPath = path.join(outDir, safeFilename);
    if (!outputPath.startsWith(outDir + path.sep) && outputPath !== outDir) {
      throw new Error(`[playground-snapshots] Refusing path outside output directory: ${outputPath}`);
    }

    await fs.writeFile(outputPath, JSON.stringify(normalized, null, 2) + "\n", "utf8");
    console.log(`[playground-snapshots] Wrote ${safeFilename}`);
  }
}

async function fetchFromActionsApi() {
  if (!repo) {
    console.warn("[playground-snapshots] PLAYGROUND_SNAPSHOTS_REPO not set; skipping fetch.");
    return;
  }
  if (!token) {
    throw new Error("[playground-snapshots] Missing token for Actions API mode. Set PLAYGROUND_SNAPSHOTS_TOKEN or GITHUB_TOKEN.");
  }

  await fs.mkdir(outDir, { recursive: true });

  if (workflowsDir && workflowsDir !== ".github/workflows") {
    console.warn(
      `[playground-snapshots] Note: Actions API mode can only fetch runs for workflows located in '.github/workflows'. ` +
        `You have PLAYGROUND_SNAPSHOTS_WORKFLOWS_DIR='${workflowsDir}'. ` +
        `If the workflows aren’t in '.github/workflows' in that repo, use PLAYGROUND_SNAPSHOTS_MODE=contents instead.`
    );
  }

  let ids = workflowIdsCsv
    .split(",")
    .map(s => s.trim())
    .filter(Boolean);

  if (ids.length === 0) {
    ids = await listWorkflowIdsFromLocalAssets();
  }

  if (ids.length === 0) {
    console.warn("[playground-snapshots] No workflow IDs found for Actions API mode; leaving existing snapshots as-is.");
    return;
  }

  if (ids.length > MAX_FILES) {
    throw new Error(`[playground-snapshots] Refusing to fetch ${ids.length} workflows (max ${MAX_FILES}).`);
  }

  // Clean output directory first so removals are reflected.
  const existing = await fs.readdir(outDir).catch(() => []);
  await Promise.all(existing.filter(f => f.endsWith(".json")).map(f => fs.rm(path.join(outDir, f), { force: true })));

  console.log(`[playground-snapshots] Fetching latest Actions runs from ${repo} (branch: ${branch})`);
  console.log(`[playground-snapshots] Workflows dir in repo: ${workflowsDir}`);

  let totalBytes = 0;
  for (const id of ids) {
    const safeName = `${id}.json`;
    if (!SAFE_FILENAME.test(safeName)) {
      throw new Error(`[playground-snapshots] Refusing unsafe filename: ${safeName}`);
    }

    const snapshot = await fetchLatestRunSnapshotFromActionsApi(id);
    const json = JSON.stringify(snapshot, null, 2) + "\n";
    const bytes = Buffer.from(json, "utf8");

    if (bytes.length > MAX_FILE_BYTES) {
      throw new Error(`[playground-snapshots] Refusing oversized snapshot ${safeName} (${bytes.length} bytes; max ${MAX_FILE_BYTES}).`);
    }

    totalBytes += bytes.length;
    if (totalBytes > MAX_TOTAL_BYTES) {
      throw new Error(`[playground-snapshots] Refusing snapshots total ${totalBytes} bytes (max ${MAX_TOTAL_BYTES}).`);
    }

    // Additional validation at point of use to prevent path traversal
    const outputPath = path.join(outDir, path.basename(safeName));
    if (!outputPath.startsWith(outDir + path.sep) && outputPath !== outDir) {
      throw new Error(`[playground-snapshots] Refusing path outside output directory: ${outputPath}`);
    }

    await fs.writeFile(outputPath, json, "utf8");
    console.log(`[playground-snapshots] Wrote ${safeName}`);
  }
}

async function main() {
  if (!repo) {
    console.warn("[playground-snapshots] PLAYGROUND_SNAPSHOTS_REPO not set; skipping fetch.");
    return;
  }

  if (mode === "actions") {
    await fetchFromActionsApi();
    return;
  }

  // Default: download pre-baked snapshots from a repo path.
  await fetchFromContentsApi();
}

main().catch(err => {
  console.error(String(err?.stack || err));
  process.exitCode = 1;
});
