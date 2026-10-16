// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Only uniquely matched completed jobs can supply a dependency's duration.
 * Matrix and reusable-workflow groups do not have a single job duration.
 * @typedef {{name: string, status?: string, conclusion?: string | null, started_at?: string | null, completed_at?: string | null, run_attempt?: number}} JobTiming
 * @param {{githubClient?: typeof github, runContext?: typeof context, env?: NodeJS.ProcessEnv, jobs?: JobTiming[], warn: (message: string) => void}} options
 * @returns {Promise<Record<string, number>>}
 */
async function collectJobDurations({ githubClient = global.github, runContext = global.context, env = process.env, jobs: providedJobs, warn }) {
  if (!env.GH_AW_JOB_RESULTS) return {};
  if (env.GH_AW_JOB_DURATIONS) {
    try {
      const cached = JSON.parse(env.GH_AW_JOB_DURATIONS);
      if (
        cached &&
        typeof cached === "object" &&
        !Array.isArray(cached) &&
        Object.entries(cached).every(([id, duration]) => /^[A-Za-z_][A-Za-z0-9_-]*$/.test(id) && typeof duration === "number" && Number.isSafeInteger(duration) && duration >= 0)
      )
        return cached;
      warn("Unified session: invalid cached job durations; reading Actions job timing");
    } catch (error) {
      if (!(error instanceof SyntaxError)) throw error;
      warn("Unified session: invalid cached job durations; reading Actions job timing");
    }
  }
  let needs;
  let names;
  let eligible;
  try {
    needs = JSON.parse(env.GH_AW_JOB_RESULTS);
    names = JSON.parse(env.GH_AW_JOB_DISPLAY_NAMES || "{}");
    eligible = JSON.parse(env.GH_AW_JOB_TIMING_ELIGIBLE || "{}");
  } catch (error) {
    if (!(error instanceof SyntaxError)) throw error;
    warn("Unified session: job durations unavailable (invalid job metadata)");
    return {};
  }
  if (!needs || typeof needs !== "object" || Array.isArray(needs) || !names || typeof names !== "object" || Array.isArray(names) || !eligible || typeof eligible !== "object" || Array.isArray(eligible)) {
    warn("Unified session: job durations unavailable (invalid job metadata)");
    return {};
  }
  if (!Object.keys(needs).some(id => eligible[id] === true)) return {};
  const attempt = Number(env.GITHUB_RUN_ATTEMPT);
  if (!githubClient || !runContext?.repo || !Number.isSafeInteger(attempt) || attempt < 1) {
    warn("Unified session: job durations unavailable (missing Actions run context)");
    return {};
  }
  /** @type {JobTiming[]} */
  const jobs = providedJobs ?? [];
  try {
    for (let page = 1; providedJobs === undefined; page++) {
      const response = await githubClient.rest.actions.listJobsForWorkflowRunAttempt({
        ...runContext.repo,
        run_id: runContext.runId,
        attempt_number: attempt,
        per_page: 100,
        page,
      });
      jobs.push(...response.data.jobs);
      if (response.data.jobs.length < 100) break;
    }
  } catch (error) {
    // API errors can include request details; never log the response or metadata.
    warn("Unified session: job durations unavailable (Actions API request failed)");
    return {};
  }
  /** @type {Record<string, number>} */
  const durations = Object.create(null);
  const knownIds = [...new Set([...Object.keys(needs), ...Object.keys(names)])];
  const displayName = id => (typeof names[id] === "string" && names[id] ? names[id] : id);
  const matchesName = (name, display) => name === display || name.startsWith(`${display} (`) || name.startsWith(`${display} / `);
  for (const [jobId, dependency] of Object.entries(needs)) {
    if (eligible[jobId] !== true || !/^[A-Za-z_][A-Za-z0-9_-]*$/.test(jobId) || !["success", "failure", "cancelled"].includes(dependency?.result)) continue;
    const matches = jobs.filter(job => job.name === displayName(jobId) && (job.run_attempt === undefined || job.run_attempt === attempt));
    const ambiguous = knownIds.some(id => id !== jobId && matchesName(displayName(jobId), displayName(id)));
    if (matches.length !== 1 || ambiguous) {
      warn(`Unified session: job duration unavailable for ${jobId} (no unique Actions job match)`);
      continue;
    }
    const job = matches[0];
    const started = typeof job.started_at === "string" ? Date.parse(job.started_at) : NaN;
    const completed = typeof job.completed_at === "string" ? Date.parse(job.completed_at) : NaN;
    const durationMs = completed - started;
    if (job.status !== "completed" || job.conclusion === "skipped" || !Number.isSafeInteger(durationMs) || durationMs < 0) {
      warn(`Unified session: job duration unavailable for ${jobId} (missing or invalid completed-job timing)`);
      continue;
    }
    durations[jobId] = durationMs;
  }
  return durations;
}

module.exports = { collectJobDurations };
