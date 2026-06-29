import { CACHE_TTL_MS, DEFAULT_LOG_TIMEOUT_MINUTES, MAX_LOG_CONTINUATIONS, type ReportWindow } from "./dashboard-config.js";
import {
  buildLogsArgs,
  continuationToLogsOptions,
  hasFlag,
  logsArgsToOptions,
  logsCommandUsesJSON,
  mergeRuns,
  normalizeLogsCommandArgs,
  normalizeLogsOptions,
  parseGhAwArgs,
  type LogsContinuation,
  type LogsOptions,
  type LogsOptionsInput,
  type RunLike,
} from "./dashboard-logs.js";
import type { AuditReport, ExperimentInfo, WorkflowDefinition } from "./models.js";
import { applyForecastToUsageSummary, buildUsageSummary, forecastDaysForWindow, type ForecastWorkflow, type UsageRun, type UsageSummaryItem } from "./usage-forecast.js";

const LOG = "[dashboard-data]";

interface CacheEntry<T> {
  data: T;
  expiresAt: number;
}

interface LogsBatchResponse {
  runs?: RunLike[];
  continuation?: LogsContinuation;
  summary?: Record<string, unknown>;
}

interface ForecastResponse {
  workflows?: ForecastWorkflow[];
}

interface LogsBatchResult {
  firstBatch: LogsBatchResponse | null;
  runs: RunLike[];
  summary: Record<string, unknown> | null;
  logsFetches: number;
  partial: boolean;
  continuation: LogsContinuation | null;
}

interface LogsDataResult {
  runs: RunLike[];
  summary: Record<string, unknown> | null;
  window: ReportWindow;
  timeout: number;
  logsFetches: number;
  partial: boolean;
  continuation: LogsContinuation | null;
}

interface UsageResult {
  items: UsageSummaryItem[];
  window: ReportWindow;
  timeout: number;
  logsFetches: number;
  partial: boolean;
  continuation: LogsContinuation | null;
  total_runs: number;
  forecast_history_days: number;
}

interface ExecCommandOptions {
  window?: string;
  timeout?: number;
}

interface ExecCommandResult {
  command: string;
  output: string;
  error?: boolean;
}

interface CommandError extends Error {
  stderr?: string;
}

type RunGhAw = (args: string[]) => Promise<string>;

function asError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }
  return new Error(String(value));
}

/**
 * Parse JSON from CLI output robustly.
 *
 * Some gh-aw commands emit a status line (e.g. "✓ Fetched 36 workflows") to
 * stdout before the JSON payload.  This helper tries a direct parse first and,
 * on failure, locates the first `[` or `{` character and retries from there.
 * A descriptive error with a raw-output snippet is thrown when parsing still
 * fails, making silent "Unexpected end of JSON input" errors actionable.
 */
function parseJsonOutput(raw: string, context: string): unknown {
  const trimmed = (raw ?? "").trim();
  if (!trimmed) {
    console.error(`${LOG} parseJsonOutput: no output for context="${context}"`);
    throw new Error(`${context}: command produced no output`);
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    const jsonStart = trimmed.search(/[{[]/);
    if (jsonStart > 0) {
      try {
        return JSON.parse(trimmed.slice(jsonStart));
      } catch {
        // fall through to the descriptive throw below
      }
    }
    const snippet = trimmed.replace(/\s+/g, " ").slice(0, 200);
    console.error(`${LOG} parseJsonOutput: JSON parse failed context="${context}" snippet=${snippet}`);
    throw new Error(`${context}: failed to parse JSON (output: ${snippet})`);
  }
}

export function createDashboardDataAccess({ runGhAw, cacheTTL = CACHE_TTL_MS, logsOutputDir }: { runGhAw: RunGhAw; cacheTTL?: number; logsOutputDir?: string }) {
  const cache = new Map<string, CacheEntry<unknown>>();

  function getCached<T>(key: string): T | null {
    const entry = cache.get(key);
    return entry && Date.now() < entry.expiresAt ? (entry.data as T) : null;
  }

  function setCached<T>(key: string, data: T): void {
    cache.set(key, { data, expiresAt: Date.now() + cacheTTL });
  }

  async function getDefinitions(): Promise<WorkflowDefinition[]> {
    const hit = getCached<WorkflowDefinition[]>("definitions");
    if (hit) {
      console.error(`${LOG} getDefinitions: cache hit count=${hit.length}`);
      return hit;
    }
    console.error(`${LOG} getDefinitions: fetching from CLI`);
    try {
      const raw = await runGhAw(["status", "--json"]);
      const parsed = parseJsonOutput(raw, "gh aw status --json");
      const data = (Array.isArray(parsed) ? parsed : []) as WorkflowDefinition[];
      setCached("definitions", data);
      console.error(`${LOG} getDefinitions: fetched count=${data.length}`);
      return data;
    } catch (err) {
      console.error(`${LOG} getDefinitions error: ${asError(err).message}`);
      throw err;
    }
  }

  async function getExperiments(): Promise<ExperimentInfo[]> {
    const hit = getCached<ExperimentInfo[]>("experiments");
    if (hit) {
      console.error(`${LOG} getExperiments: cache hit count=${hit.length}`);
      return hit;
    }
    console.error(`${LOG} getExperiments: fetching from CLI`);
    try {
      const raw = await runGhAw(["experiments", "list", "--json"]);
      const parsed = parseJsonOutput(raw, "gh aw experiments list --json");
      const experiments = (Array.isArray(parsed) ? parsed : []) as ExperimentInfo[];
      setCached("experiments", experiments);
      console.error(`${LOG} getExperiments: fetched count=${experiments.length}`);
      return experiments;
    } catch (err) {
      console.error(`${LOG} getExperiments error: ${asError(err).message}`);
      throw err;
    }
  }

  async function fetchLogsBatches(initialOptions: LogsOptions, initialArgs: string[] | null = null): Promise<LogsBatchResult> {
    let current: LogsOptions | null = initialOptions;
    let logsFetches = 0;
    let runs: RunLike[] = [];
    let continuation: LogsContinuation | null = null;
    let summary: Record<string, unknown> | null = null;
    let firstBatch: LogsBatchResponse | null = null;

    while (current && logsFetches < MAX_LOG_CONTINUATIONS) {
      let batchArgs = logsFetches === 0 && initialArgs ? initialArgs : buildLogsArgs(current);
      // Inject --output so all sessions for the same repo share one disk cache.
      if (logsOutputDir && !hasFlag(batchArgs, "--output", "-o")) {
        batchArgs = [...batchArgs, "--output", logsOutputDir];
      }
      console.error(`${LOG} fetchLogsBatches: batch=${logsFetches + 1} args=${JSON.stringify(batchArgs)}`);
      const raw = await runGhAw(batchArgs);
      let data: LogsBatchResponse;
      try {
        data = parseJsonOutput(raw, `logs batch ${logsFetches + 1}`) as LogsBatchResponse;
      } catch (error) {
        console.error(`${LOG} fetchLogsBatches: parse error on batch ${logsFetches + 1}: ${asError(error).message}`);
        throw asError(error);
      }

      if (!firstBatch) {
        firstBatch = data;
      }

      const newRuns = Array.isArray(data.runs) ? data.runs : [];
      runs = mergeRuns(runs, newRuns);
      continuation = data.continuation ?? null;
      summary = data.summary ?? summary;
      logsFetches += 1;
      console.error(`${LOG} fetchLogsBatches: batch=${logsFetches} newRuns=${newRuns.length} totalRuns=${runs.length} hasContinuation=${Boolean(continuation)}`);

      if (!continuation) {
        break;
      }

      current = continuationToLogsOptions(continuation, current);
    }

    return {
      firstBatch,
      runs,
      summary,
      logsFetches,
      partial: Boolean(continuation),
      continuation,
    };
  }

  async function getLogsData(options: LogsOptionsInput = {}): Promise<LogsDataResult> {
    const normalized = normalizeLogsOptions(options);
    const key = `logs:${JSON.stringify({
      window: normalized.window.id,
      count: normalized.count,
      timeout: normalized.timeout,
      startDate: normalized.startDate,
      endDate: normalized.endDate,
      beforeRunID: normalized.beforeRunID,
      afterRunID: normalized.afterRunID,
      workflowName: normalized.workflowName,
      engine: normalized.engine,
      branch: normalized.branch,
      artifacts: normalized.artifacts,
    })}`;
    const hit = getCached<LogsDataResult>(key);
    if (hit) {
      console.error(`${LOG} getLogsData: cache hit runs=${hit.runs.length} window=${hit.window.id}`);
      return hit;
    }

    console.error(`${LOG} getLogsData: fetching window=${normalized.window.id} count=${normalized.count} timeout=${normalized.timeout}`);
    const logsResult = await fetchLogsBatches(normalized);
    console.error(`${LOG} getLogsData: fetched runs=${logsResult.runs.length} fetches=${logsResult.logsFetches} partial=${logsResult.partial}`);

    const result: LogsDataResult = {
      runs: logsResult.runs,
      summary: logsResult.summary,
      window: normalized.window,
      timeout: normalized.timeout,
      logsFetches: logsResult.logsFetches,
      partial: logsResult.partial,
      continuation: logsResult.continuation,
    };
    setCached(key, result);
    return result;
  }

  async function getForecastData(workflowIDs: string[], window: ReportWindow, timeout: number): Promise<ForecastWorkflow[]> {
    if (workflowIDs.length === 0) {
      return [];
    }

    const args = ["forecast", "--json", "--period", "month", "--days", String(forecastDaysForWindow(window)), "--timeout", String(timeout), ...workflowIDs];
    console.error(`${LOG} getForecastData: workflowIDs=${workflowIDs.length} window=${window.id} days=${forecastDaysForWindow(window)}`);
    try {
      const raw = await runGhAw(args);
      const data = parseJsonOutput(raw, "gh aw forecast --json") as ForecastResponse;
      const workflows = Array.isArray(data.workflows) ? data.workflows : [];
      console.error(`${LOG} getForecastData: fetched workflows=${workflows.length}`);
      return workflows;
    } catch (err) {
      console.error(`${LOG} getForecastData error: ${asError(err).message}`);
      throw err;
    }
  }

  async function getRuns(options: LogsOptionsInput = {}): Promise<LogsDataResult> {
    return getLogsData(options);
  }

  async function getUsage(options: LogsOptionsInput = {}): Promise<UsageResult> {
    const normalized = normalizeLogsOptions(options);
    const key = `usage:${JSON.stringify({
      window: normalized.window.id,
      count: normalized.count,
      timeout: normalized.timeout,
    })}`;
    const hit = getCached<UsageResult>(key);
    if (hit) return hit;

    const logsData = await getLogsData(normalized);
    const usageItems = buildUsageSummary(logsData.runs as UsageRun[], logsData.window);
    const workflowIDs = usageItems.map(item => item.workflow_id).filter(Boolean);
    const forecastWorkflows = await getForecastData(workflowIDs, logsData.window, logsData.timeout);
    const result: UsageResult = {
      items: applyForecastToUsageSummary(usageItems, forecastWorkflows),
      window: logsData.window,
      timeout: logsData.timeout,
      logsFetches: logsData.logsFetches,
      partial: logsData.partial,
      continuation: logsData.continuation,
      total_runs: logsData.runs.length,
      forecast_history_days: forecastDaysForWindow(logsData.window),
    };
    setCached(key, result);
    return result;
  }

  async function execCommand(rawCmd: string, options: ExecCommandOptions = {}): Promise<ExecCommandResult> {
    console.error(`${LOG} execCommand: cmd="${rawCmd}"`);
    const args = parseGhAwArgs(rawCmd);
    if (!args) {
      console.error(`${LOG} execCommand: rejected unsupported command "${rawCmd}"`);
      return { command: rawCmd, output: "Only 'gh aw <subcommand>' commands are supported.", error: true };
    }

    try {
      if (args[0] === "logs" && logsCommandUsesJSON(args)) {
        const commandArgs = normalizeLogsCommandArgs(args, options.window, options.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES);
        const fallback: LogsOptionsInput = {};
        if (options.window) {
          fallback.window = options.window;
        }
        if (options.timeout != null) {
          fallback.timeout = options.timeout;
        }
        const logsOptions = logsArgsToOptions(commandArgs, fallback);
        const logsResult = await fetchLogsBatches(logsOptions, commandArgs);

        return {
          command: `gh aw ${commandArgs.join(" ")}`,
          output: JSON.stringify(
            {
              ...(logsResult.firstBatch ?? {}),
              runs: logsResult.runs,
              partial: logsResult.partial,
              logs_fetches: logsResult.logsFetches,
              continuation: logsResult.continuation,
            },
            null,
            2
          ),
        };
      }

      const output = await runGhAw(args);
      return { command: rawCmd, output };
    } catch (err) {
      const e = asError(err) as CommandError;
      const msg = e.stderr || e.message || "Unknown error";
      console.error(`${LOG} execCommand error cmd="${rawCmd}": ${msg}`);
      return { command: rawCmd, output: msg, error: true };
    }
  }

  async function getAudit(runId: string | number): Promise<AuditReport | null> {
    if (!runId) return null;
    const key = `audit:${runId}`;
    const hit = getCached<unknown>(key);
    if (hit) {
      console.error(`${LOG} getAudit: cache hit runId=${runId}`);
      return hit;
    }
    console.error(`${LOG} getAudit: fetching runId=${runId}`);
    try {
      const auditArgs: string[] = ["audit", String(runId), "--json"];
      if (logsOutputDir) {
        auditArgs.push("--output", logsOutputDir);
      }
      const raw = await runGhAw(auditArgs);
      const data = parseJsonOutput(raw, `gh aw audit ${runId} --json`) as AuditReport;
      setCached(key, data);
      console.error(`${LOG} getAudit: fetched runId=${runId}`);
      return data;
    } catch (err) {
      console.error(`${LOG} getAudit error runId=${runId}: ${asError(err).message}`);
      throw err;
    }
  }

  return {
    clearCache: () => cache.clear(),
    execCommand,
    getAudit,
    getDefinitions,
    getExperiments,
    getRuns,
    getUsage,
  };
}
