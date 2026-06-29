import { DEFAULT_LOG_TIMEOUT_MINUTES, DEFAULT_RUN_COUNT, getReportWindow, type ReportWindow, type ReportWindowID } from "./dashboard-config.js";

export interface RunLike {
  run_id?: number | string | null;
  [key: string]: unknown;
}

export interface LogsOptionsInput {
  window?: ReportWindow | ReportWindowID | string | undefined;
  count?: number | string | undefined;
  timeout?: number | string | undefined;
  startDate?: string | undefined;
  endDate?: string | undefined;
  beforeRunID?: number | string | undefined;
  afterRunID?: number | string | undefined;
  workflowName?: string | undefined;
  engine?: string | undefined;
  branch?: string | undefined;
  artifacts?: string[] | undefined;
}

export interface LogsOptions {
  window: ReportWindow;
  count: number;
  timeout: number;
  startDate: string;
  endDate: string;
  beforeRunID: number;
  afterRunID: number;
  workflowName: string;
  engine: string;
  branch: string;
  artifacts: string[];
}

export interface LogsContinuation {
  workflow_name?: string;
  count?: number | string;
  start_date?: string;
  end_date?: string;
  engine?: string;
  branch?: string;
  after_run_id?: number | string;
  before_run_id?: number | string;
  timeout?: number | string;
}

function parsePositiveInt(value: unknown, fallback: number): number {
  const numeric = Number.parseInt(String(value ?? fallback), 10);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : fallback;
}

function readFlagValue(args: string[], index: number, arg: string): { value: string; nextIndex: number } {
  const equalsIndex = arg.indexOf("=");
  if (equalsIndex >= 0) {
    return { value: arg.slice(equalsIndex + 1), nextIndex: index };
  }
  return { value: args[index + 1] ?? "", nextIndex: index + 1 };
}

export function normalizeLogsOptions(options: LogsOptionsInput = {}): LogsOptions {
  const windowId = typeof options.window === "string" ? options.window : options.window?.id;
  const window = getReportWindow(windowId);
  const artifacts = Array.isArray(options.artifacts) && options.artifacts.length > 0 ? options.artifacts : ["usage"];

  return {
    window,
    count: parsePositiveInt(options.count, DEFAULT_RUN_COUNT),
    timeout: parsePositiveInt(options.timeout, DEFAULT_LOG_TIMEOUT_MINUTES),
    startDate: typeof options.startDate === "string" && options.startDate.trim() ? options.startDate.trim() : window.startDate,
    endDate: typeof options.endDate === "string" && options.endDate.trim() ? options.endDate.trim() : "",
    beforeRunID: Number.isFinite(Number(options.beforeRunID)) && Number(options.beforeRunID) > 0 ? Number(options.beforeRunID) : 0,
    afterRunID: Number.isFinite(Number(options.afterRunID)) && Number(options.afterRunID) > 0 ? Number(options.afterRunID) : 0,
    workflowName: typeof options.workflowName === "string" ? options.workflowName.trim() : "",
    engine: typeof options.engine === "string" ? options.engine.trim() : "",
    branch: typeof options.branch === "string" ? options.branch.trim() : "",
    artifacts,
  };
}

export function buildLogsArgs(options: LogsOptions): string[] {
  const args = ["logs", "--json", "-c", String(options.count), "--timeout", String(options.timeout)];

  if (options.workflowName) args.push(options.workflowName);
  if (options.startDate) args.push("--start-date", options.startDate);
  if (options.endDate) args.push("--end-date", options.endDate);
  if (options.engine) args.push("--engine", options.engine);
  if (options.branch) args.push("--ref", options.branch);
  if (options.beforeRunID > 0) args.push("--before-run-id", String(options.beforeRunID));
  if (options.afterRunID > 0) args.push("--after-run-id", String(options.afterRunID));
  if (options.artifacts.length > 0) args.push("--artifacts", options.artifacts.join(","));

  return args;
}

export function continuationToLogsOptions(continuation: LogsContinuation | null | undefined, fallback: LogsOptions): LogsOptions | null {
  if (!continuation) return null;

  return normalizeLogsOptions({
    window: fallback.window.id,
    workflowName: continuation.workflow_name || fallback.workflowName,
    count: continuation.count || fallback.count,
    startDate: continuation.start_date || fallback.startDate,
    endDate: continuation.end_date || fallback.endDate,
    engine: continuation.engine || fallback.engine,
    branch: continuation.branch || fallback.branch,
    afterRunID: continuation.after_run_id || fallback.afterRunID,
    beforeRunID: continuation.before_run_id || fallback.beforeRunID,
    timeout: continuation.timeout || fallback.timeout,
    artifacts: fallback.artifacts,
  });
}

export function mergeRuns(existingRuns: RunLike[], nextRuns: RunLike[]): RunLike[] {
  const merged = new Map(existingRuns.map(run => [run.run_id, run]));
  for (const run of nextRuns) {
    if (run?.run_id != null) {
      merged.set(run.run_id, run);
    }
  }

  return Array.from(merged.values()).sort((a, b) => Number(b.run_id ?? 0) - Number(a.run_id ?? 0));
}

export function parseGhAwArgs(raw: string): string[] | null {
  const match = raw.trim().match(/^(?:gh\s+aw\s+)(.+)$/);
  return match?.[1] ? match[1].trim().split(/\s+/) : null;
}

export function hasFlag(args: string[], longFlag: string, shortFlag = ""): boolean {
  return args.some(arg => {
    if (arg.startsWith(`${longFlag}=`)) return true;
    if (shortFlag && arg.startsWith(`${shortFlag}=`)) return true;
    return arg === longFlag || (shortFlag !== "" && arg === shortFlag);
  });
}

export function logsCommandUsesJSON(args: string[]): boolean {
  return hasFlag(args, "--json", "-j");
}

export function normalizeLogsCommandArgs(args: string[], windowId: string | undefined, timeoutMinutes: number): string[] {
  const nextArgs = [...args];
  if (!hasFlag(nextArgs, "--start-date") && !hasFlag(nextArgs, "--end-date") && !hasFlag(nextArgs, "--after-run-id") && !hasFlag(nextArgs, "--before-run-id")) {
    nextArgs.push("--start-date", getReportWindow(windowId).startDate);
  }
  if (!hasFlag(nextArgs, "--timeout")) {
    nextArgs.push("--timeout", String(timeoutMinutes));
  }
  if (!hasFlag(nextArgs, "--artifacts")) {
    nextArgs.push("--artifacts", "usage");
  }
  return nextArgs;
}

export function logsArgsToOptions(args: string[], fallback: LogsOptionsInput = {}): LogsOptions {
  const options: LogsOptionsInput = {
    window: typeof fallback.window === "string" ? fallback.window : fallback.window?.id,
    count: fallback.count,
    timeout: fallback.timeout,
    startDate: fallback.startDate,
    endDate: fallback.endDate,
    beforeRunID: fallback.beforeRunID,
    afterRunID: fallback.afterRunID,
    workflowName: fallback.workflowName,
    engine: fallback.engine,
    branch: fallback.branch,
    artifacts: fallback.artifacts,
  };

  for (let index = 1; index < args.length; index += 1) {
    const arg = args[index] ?? "";

    if (!arg.startsWith("-")) {
      if (!options.workflowName) {
        options.workflowName = arg;
      }
      continue;
    }

    if (arg === "--json" || arg === "-j") {
      continue;
    }

    if (arg === "-c" || arg.startsWith("-c=") || arg === "--count" || arg.startsWith("--count=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.count = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--timeout" || arg.startsWith("--timeout=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.timeout = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--start-date" || arg.startsWith("--start-date=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.startDate = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--end-date" || arg.startsWith("--end-date=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.endDate = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--before-run-id" || arg.startsWith("--before-run-id=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.beforeRunID = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--after-run-id" || arg.startsWith("--after-run-id=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.afterRunID = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--engine" || arg.startsWith("--engine=") || arg === "-e" || arg.startsWith("-e=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.engine = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--ref" || arg.startsWith("--ref=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.branch = value;
      index = nextIndex;
      continue;
    }

    if (arg === "--artifacts" || arg.startsWith("--artifacts=")) {
      const { value, nextIndex } = readFlagValue(args, index, arg);
      options.artifacts = value
        .split(",")
        .map(item => item.trim())
        .filter(Boolean);
      index = nextIndex;
    }
  }

  return normalizeLogsOptions(options);
}
