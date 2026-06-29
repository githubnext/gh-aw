import { CACHE_TTL_MS, DEFAULT_LOG_TIMEOUT_MINUTES, MAX_LOG_CONTINUATIONS, type ReportWindow } from "./dashboard-config.js";
import {
  buildLogsArgs,
  continuationToLogsOptions,
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
import { applyForecastToUsageSummary, buildUsageSummary, forecastDaysForWindow, type ForecastWorkflow, type UsageRun, type UsageSummaryItem } from "./usage-forecast.js";

interface CacheEntry<T> {
  data: T;
  expiresAt: number;
}

interface LogsBatchResponse {
  runs?: RunLike[];
  continuation?: LogsContinuation;
  summary?: unknown;
  [key: string]: unknown;
}

interface ForecastResponse {
  workflows?: ForecastWorkflow[];
}

interface LogsBatchResult {
  firstBatch: LogsBatchResponse | null;
  runs: RunLike[];
  summary: unknown;
  logsFetches: number;
  partial: boolean;
  continuation: LogsContinuation | null;
}

interface LogsDataResult {
  runs: RunLike[];
  summary: unknown;
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

type RunGhAw = (args: string[]) => Promise<string>;

function asError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }
  return new Error(String(value));
}

export function createDashboardDataAccess({ runGhAw, cacheTTL = CACHE_TTL_MS }: { runGhAw: RunGhAw; cacheTTL?: number }) {
  const cache = new Map<string, CacheEntry<unknown>>();

  function getCached<T>(key: string): T | null {
    const entry = cache.get(key);
    return entry && Date.now() < entry.expiresAt ? (entry.data as T) : null;
  }

  function setCached<T>(key: string, data: T): void {
    cache.set(key, { data, expiresAt: Date.now() + cacheTTL });
  }

  async function getDefinitions(): Promise<unknown[]> {
    const hit = getCached<unknown[]>("definitions");
    if (hit) return hit;
    const raw = await runGhAw(["status", "--json"]);
    const parsed = JSON.parse(raw);
    const data = Array.isArray(parsed) ? parsed : [];
    setCached("definitions", data);
    return data;
  }

  async function getExperiments(): Promise<unknown[]> {
    const hit = getCached<unknown[]>("experiments");
    if (hit) return hit;
    const raw = await runGhAw(["experiments", "list", "--json"]);
    const parsed = JSON.parse(raw);
    const experiments = Array.isArray(parsed) ? parsed : [];
    setCached("experiments", experiments);
    return experiments;
  }

  async function fetchLogsBatches(initialOptions: LogsOptions, initialArgs: string[] | null = null): Promise<LogsBatchResult> {
    let current: LogsOptions | null = initialOptions;
    let logsFetches = 0;
    let runs: RunLike[] = [];
    let continuation: LogsContinuation | null = null;
    let summary: unknown = null;
    let firstBatch: LogsBatchResponse | null = null;

    while (current && logsFetches < MAX_LOG_CONTINUATIONS) {
      const raw = await runGhAw(logsFetches === 0 && initialArgs ? initialArgs : buildLogsArgs(current));
      let data: LogsBatchResponse;
      try {
        data = JSON.parse(raw) as LogsBatchResponse;
      } catch (error) {
        const parsedError = asError(error);
        throw new Error(`Failed to parse logs batch ${logsFetches + 1}: ${parsedError.message}`);
      }

      if (!firstBatch) {
        firstBatch = data;
      }

      runs = mergeRuns(runs, Array.isArray(data.runs) ? data.runs : []);
      continuation = data.continuation ?? null;
      summary = data.summary ?? summary;
      logsFetches += 1;

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
    if (hit) return hit;

    const logsResult = await fetchLogsBatches(normalized);

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
    const raw = await runGhAw(args);
    let data: ForecastResponse;
    try {
      data = JSON.parse(raw) as ForecastResponse;
    } catch (error) {
      const parsedError = asError(error);
      const snippet = String(raw ?? "")
        .replace(/\s+/g, " ")
        .slice(0, 200);
      throw new Error(`Failed to parse forecast output: ${parsedError.message}${snippet ? ` (output: ${snippet})` : ""}`);
    }
    return Array.isArray(data.workflows) ? data.workflows : [];
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
    const args = parseGhAwArgs(rawCmd);
    if (!args) {
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
      const error = err as { stderr?: string; message?: string };
      return { command: rawCmd, output: error.stderr || error.message || "Unknown error", error: true };
    }
  }

  async function getAudit(runId: string | number): Promise<unknown | null> {
    if (!runId) return null;
    const key = `audit:${runId}`;
    const hit = getCached<unknown>(key);
    if (hit) return hit;

    const raw = await runGhAw(["audit", String(runId), "--json"]);
    let data: unknown;
    try {
      data = JSON.parse(raw);
    } catch (error) {
      const parsedError = asError(error);
      const snippet = String(raw ?? "")
        .replace(/\s+/g, " ")
        .slice(0, 100);
      throw new Error(`Failed to parse audit output for run ${runId}: ${parsedError.message}${snippet ? ` (output: ${snippet})` : ""}`);
    }
    setCached(key, data);
    return data;
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
