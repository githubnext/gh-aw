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
} from "./dashboard-logs.js";
import { applyForecastToUsageSummary, buildUsageSummary, forecastDaysForWindow, type ForecastWorkflow } from "./usage-forecast.js";
import type { CLIStatus, ExperimentInfo, UsageSummaryItem, WorkflowDefinition, WorkflowRun } from "./src/models.js";

export interface CommandExecutionResult {
  command: string;
  error?: boolean;
  output: string;
}

export interface LogsDataResult {
  continuation: LogsContinuation | null;
  logsFetches: number;
  partial: boolean;
  runs: WorkflowRun[];
  summary: Record<string, unknown> | null;
  timeout: number;
  window: ReportWindow;
}

export interface UsageDataResult {
  continuation: LogsContinuation | null;
  forecast_history_days: number;
  items: UsageSummaryItem[];
  logsFetches: number;
  partial: boolean;
  timeout: number;
  total_runs: number;
  window: ReportWindow;
}

export interface DashboardDataAccess {
  clearCache: () => void;
  execCommand: (rawCmd: string, options?: { timeout?: number; window?: string }) => Promise<CommandExecutionResult>;
  getAudit: (runId: string) => Promise<Record<string, unknown> | null>;
  getDefinitions: () => Promise<WorkflowDefinition[]>;
  getExperiments: () => Promise<ExperimentInfo[]>;
  getRuns: (options?: LogsOptionsInput) => Promise<LogsDataResult>;
  getUsage: (options?: LogsOptionsInput) => Promise<UsageDataResult>;
}

type RunGhAw = ((args: string[]) => Promise<string>) & Partial<{ getStatus: () => Promise<CLIStatus> }>;

interface LogsBatchResponse {
  continuation?: LogsContinuation | null;
  runs?: WorkflowRun[];
  summary?: Record<string, unknown> | null;
  [key: string]: unknown;
}

interface LogsBatchResult {
  continuation: LogsContinuation | null;
  firstBatch: LogsBatchResponse | null;
  logsFetches: number;
  partial: boolean;
  runs: WorkflowRun[];
  summary: Record<string, unknown> | null;
}

interface CacheEntry<T> {
  data: T;
  expiresAt: number;
}

interface DashboardDataAccessOptions {
  cacheTTL?: number;
  runGhAw: RunGhAw;
}

interface CommandErrorLike {
  message?: string;
  stderr?: string;
}

function parseJSON<T>(raw: string, errorPrefix: string, snippetLength = 200): T {
  try {
    return JSON.parse(raw) as T;
  } catch (error) {
    const snippet = String(raw ?? "")
      .replace(/\s+/g, " ")
      .slice(0, snippetLength);
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`${errorPrefix}: ${message}${snippet ? ` (output: ${snippet})` : ""}`);
  }
}

function getErrorMessage(error: unknown): string {
  if (typeof error === "object" && error !== null) {
    const commandError = error as CommandErrorLike;
    return commandError.stderr || commandError.message || "Unknown error";
  }
  return String(error ?? "Unknown error");
}

export function createDashboardDataAccess({ runGhAw, cacheTTL = CACHE_TTL_MS }: DashboardDataAccessOptions): DashboardDataAccess {
  const cache = new Map<string, CacheEntry<unknown>>();

  function getCached<T>(key: string): T | null {
    const entry = cache.get(key) as CacheEntry<T> | undefined;
    return entry && Date.now() < entry.expiresAt ? entry.data : null;
  }

  function setCached<T>(key: string, data: T): void {
    cache.set(key, { data, expiresAt: Date.now() + cacheTTL });
  }

  async function getDefinitions(): Promise<WorkflowDefinition[]> {
    const hit = getCached<WorkflowDefinition[]>("definitions");
    if (hit) return hit;
    const raw = await runGhAw(["status", "--json"]);
    const data = parseJSON<WorkflowDefinition[]>(raw, "Failed to parse workflow definitions");
    setCached("definitions", data);
    return data;
  }

  async function getExperiments(): Promise<ExperimentInfo[]> {
    const hit = getCached<ExperimentInfo[]>("experiments");
    if (hit) return hit;
    const raw = await runGhAw(["experiments", "list", "--json"]);
    const data = parseJSON<unknown>(raw, "Failed to parse experiments list");
    const experiments = Array.isArray(data) ? (data as ExperimentInfo[]) : [];
    setCached("experiments", experiments);
    return experiments;
  }

  async function fetchLogsBatches(initialOptions: LogsOptions, initialArgs: string[] | null = null): Promise<LogsBatchResult> {
    let current: LogsOptions | null = initialOptions;
    let logsFetches = 0;
    let runs: WorkflowRun[] = [];
    let continuation: LogsContinuation | null = null;
    let summary: Record<string, unknown> | null = null;
    let firstBatch: LogsBatchResponse | null = null;

    while (current && logsFetches < MAX_LOG_CONTINUATIONS) {
      const raw = await runGhAw(logsFetches === 0 && initialArgs ? initialArgs : buildLogsArgs(current));
      const data = parseJSON<LogsBatchResponse>(raw, `Failed to parse logs batch ${logsFetches + 1}`);
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
    const data = parseJSON<{ workflows?: ForecastWorkflow[] }>(raw, "Failed to parse forecast output");
    return Array.isArray(data.workflows) ? data.workflows : [];
  }

  async function getRuns(options: LogsOptionsInput = {}): Promise<LogsDataResult> {
    return getLogsData(options);
  }

  async function getUsage(options: LogsOptionsInput = {}): Promise<UsageDataResult> {
    const normalized = normalizeLogsOptions(options);
    const key = `usage:${JSON.stringify({
      window: normalized.window.id,
      count: normalized.count,
      timeout: normalized.timeout,
    })}`;
    const hit = getCached<UsageDataResult>(key);
    if (hit) return hit;

    const logsData = await getLogsData(normalized);
    const usageItems = buildUsageSummary(logsData.runs, logsData.window);
    const workflowIDs = usageItems.map(item => item.workflow_id).filter(Boolean);
    const forecastWorkflows = await getForecastData(workflowIDs, logsData.window, logsData.timeout);
    const result: UsageDataResult = {
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

  async function execCommand(rawCmd: string, options: { timeout?: number; window?: string } = {}): Promise<CommandExecutionResult> {
    const args = parseGhAwArgs(rawCmd);
    if (!args) {
      return { command: rawCmd, output: "Only 'gh aw <subcommand>' commands are supported.", error: true };
    }

    try {
      if (args[0] === "logs" && logsCommandUsesJSON(args)) {
        const commandArgs = normalizeLogsCommandArgs(args, options.window, options.timeout ?? DEFAULT_LOG_TIMEOUT_MINUTES);
        const logsFallback: LogsOptionsInput = {};
        if (options.window) logsFallback.window = options.window;
        if (options.timeout != null) logsFallback.timeout = options.timeout;
        const logsOptions = logsArgsToOptions(commandArgs, logsFallback);
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
    } catch (error) {
      return { command: rawCmd, output: getErrorMessage(error), error: true };
    }
  }

  async function getAudit(runId: string): Promise<Record<string, unknown> | null> {
    if (!runId) return null;
    const key = `audit:${runId}`;
    const hit = getCached<Record<string, unknown>>(key);
    if (hit) return hit;

    const raw = await runGhAw(["audit", String(runId), "--json"]);
    const data = parseJSON<Record<string, unknown>>(raw, `Failed to parse audit output for run ${runId}`, 100);
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
