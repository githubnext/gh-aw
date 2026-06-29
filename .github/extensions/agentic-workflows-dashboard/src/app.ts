import Alpine from "alpinejs";

import type { AuditFinding, AuditReport, CLIStatus, ExperimentInfo, PagedResult, UsageSummaryItem, WorkflowDefinition, WorkflowRun } from "./models.js";
import { paginate } from "./pagination.js";
import type { ReportWindow } from "./dashboard-config.js";

type FlashKind = "success" | "warn" | "error";
type DashboardTabId = "definitions" | "runs" | "details" | "usage" | "experiments" | "maintenance" | "commands";
type DashboardTab = { id: DashboardTabId; label: string; counter?: "definitions" | "runs" | "usage" | "experiments" };
type MaintenanceAction = "check-update" | "run-update" | "check-upgrade" | "run-upgrade";
type ReportMeta = {
  window?: { id?: string; label?: string };
  logsFetches?: number;
  partial?: boolean;
  total_runs?: number;
};
type RunsResponse = ReportMeta & { runs?: WorkflowRun[] };
type UsageResponse = ReportMeta & { items?: UsageSummaryItem[] };

interface DashboardState {
  tabs: DashboardTab[];
  reportWindows: ReportWindow[];
  activeTab: DashboardTabId;
  selectedWindow: ReportWindow["id"];
  logsTimeout: number;
  pageSize: number;
  cliStatus: CLIStatus | null;
  definitions: WorkflowDefinition[];
  runs: WorkflowRun[];
  usage: UsageSummaryItem[];
  experiments: ExperimentInfo[];
  definitionsPaged: PagedResult<WorkflowDefinition>;
  runsPaged: PagedResult<WorkflowRun>;
  usagePaged: PagedResult<UsageSummaryItem>;
  experimentsPaged: PagedResult<ExperimentInfo>;
  selectedRun: WorkflowRun | null;
  auditData: AuditReport | null;
  includePreReleases: boolean;
  maintenanceOutput: string;
  maintenanceLastAction: string;
  commandInput: string;
  commandOutput: string;
  flashMessage: string;
  flashKind: FlashKind;
  loadingCliStatus: boolean;
  loadingDefinitions: boolean;
  loadingRuns: boolean;
  loadingUsage: boolean;
  loadingExperiments: boolean;
  loadingAudit: boolean;
  loadingMaintenance: boolean;
  errorCliStatus: string;
  errorDefinitions: string;
  errorRuns: string;
  errorUsage: string;
  errorExperiments: string;
  errorAudit: string;
  runsMeta: ReportMeta | null;
  usageMeta: ReportMeta | null;
  init(): Promise<void>;
  currentWindow(): ReportWindow;
  reportWindowClass(windowId: ReportWindow["id"]): string;
  selectReportWindow(windowId: ReportWindow["id"]): Promise<void>;
  fetchCliStatus(): Promise<void>;
  fetchDefinitions(): Promise<void>;
  fetchRuns(): Promise<void>;
  fetchUsage(): Promise<void>;
  fetchExperiments(): Promise<void>;
  refresh(): Promise<void>;
  setActiveTab(tab: DashboardTabId): void;
  isActiveTab(tab: DashboardTabId): boolean;
  tabCount(tab: DashboardTab): number;
  loadDefinitionPage(page: number): void;
  loadRunPage(page: number): void;
  loadUsagePage(page: number): void;
  loadExperimentPage(page: number): void;
  selectRun(runId: number): void;
  viewRunDetails(runId: number): void;
  loadAudit(): Promise<void>;
  clearAudit(): void;
  buildLogsCommand(count?: number): string;
  buildMaintenanceCommand(action: MaintenanceAction): string;
  maintenanceActionLabel(action: MaintenanceAction): string;
  runMaintenanceAction(action: MaintenanceAction): Promise<void>;
  auditHasFindings(): boolean;
  auditSeverityClass(severity?: string): string;
  auditPriorityClass(priority?: string): string;
  buildReportSummaryMessage(meta: ReportMeta | null): string;
  runCommand(): Promise<void>;
  commandQuickFill(value: string): void;
  runStatusClass(run: WorkflowRun): string;
  runStatusLabel(run: WorkflowRun): string;
  definitionStatusClass(definition: WorkflowDefinition): string;
  definitionStatusLabel(definition: WorkflowDefinition): string;
  formatDuration(ms?: number | null): string;
  formatDate(iso?: string | null): string;
  formatAIC(value?: number | null): string;
  formatNumber(value?: number | null, options?: Intl.NumberFormatOptions): string;
  cliSourceLabel(cliStatus: CLIStatus | null): string;
  cliUnavailableMessage(): string;
}

const dashboardTabs: DashboardTab[] = [
  { id: "definitions", label: "Workflows", counter: "definitions" },
  { id: "runs", label: "Runs", counter: "runs" },
  { id: "details", label: "Run details" },
  { id: "usage", label: "Usage", counter: "usage" },
  { id: "experiments", label: "Experiments", counter: "experiments" },
  { id: "maintenance", label: "Maintenance" },
  { id: "commands", label: "Commands" },
];

const reportWindows: ReportWindow[] = [
  { id: "3d", label: "3 days", startDate: "-3d", days: 3 },
  { id: "7d", label: "7 days", startDate: "-1w", days: 7 },
  { id: "1mo", label: "1 month", startDate: "-1mo", days: 30 },
];

const DEFAULT_LOGS_COMMAND_COUNT = 25;

function cliSourceLabel(cliStatus: CLIStatus | null): string {
  if (!cliStatus?.available) return "not installed";
  if (cliStatus.source === "dev-binary") return "local build";
  if (cliStatus.source === "gh-extension") return "gh extension";
  return "available";
}

function runStatusClass(run: WorkflowRun): string {
  const status = run.status ?? "";
  const conclusion = run.conclusion ?? "";
  if (status === "completed" || status === "success") {
    return conclusion && conclusion !== "success" ? "Label Label--danger" : "Label Label--success";
  }
  if (status === "failure" || status === "failed") return "Label Label--danger";
  if (status === "in_progress" || status === "running") return "Label Label--attention";
  return "Label Label--secondary";
}

function runStatusLabel(run: WorkflowRun): string {
  if (run.status === "completed" && run.conclusion) return run.conclusion;
  return run.status ?? "unknown";
}

function definitionStatusClass(definition: WorkflowDefinition): string {
  if (definition.status === "disabled") return "Label Label--secondary";
  return definition.compiled === "yes" ? "Label Label--success" : "Label Label--attention";
}

function definitionStatusLabel(definition: WorkflowDefinition): string {
  if (definition.status === "disabled") return "disabled";
  return definition.compiled === "yes" ? "enabled" : "not compiled";
}

function formatDuration(ms?: number | null): string {
  if (ms == null) return "—";
  const secs = Math.round(ms / 1000);
  if (secs < 60) return `${secs}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}

function formatDate(iso?: string | null): string {
  if (!iso) return "—";
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}

function formatNumber(value?: number | null, options: Intl.NumberFormatOptions = {}): string {
  const numeric = Number(value ?? 0);
  if (!Number.isFinite(numeric)) return "0";
  return new Intl.NumberFormat(undefined, options).format(numeric);
}

function formatAIC(value?: number | null): string {
  const numeric = Number(value ?? 0);
  if (!Number.isFinite(numeric) || numeric <= 0) return "0";
  return formatNumber(Math.ceil(numeric));
}

function reportWindowById(windowId: ReportWindow["id"]): ReportWindow {
  return reportWindows.find(window => window.id === windowId) ?? reportWindows[1]!;
}

function buildReportMessage(meta: ReportMeta | null, emptyLabel: string): string {
  if (!meta?.window) return emptyLabel ?? "";

  const windowLabel = meta.window.label ?? meta.window.id;
  const fragments = windowLabel ? [`Window: ${windowLabel}`] : [];
  if (meta.logsFetches) {
    fragments.push(`${meta.logsFetches} log request${meta.logsFetches === 1 ? "" : "s"}`);
  }
  if (meta.partial) {
    fragments.push("continuation still available");
  }
  if (meta.total_runs != null) {
    fragments.push(`${meta.total_runs} runs analyzed`);
  }

  return fragments.length > 0 ? fragments.join(" · ") : emptyLabel;
}

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url);
  const data = (await response.json()) as T & { error?: string };
  if (!response.ok) {
    throw new Error(data.error ?? `HTTP ${response.status}`);
  }
  return data;
}

Alpine.data("dashboardApp", (): DashboardState => ({
  tabs: dashboardTabs,
  reportWindows,
  activeTab: "definitions",
  selectedWindow: "7d",
  logsTimeout: 1,
  pageSize: 20,
  cliStatus: null,
  definitions: [],
  runs: [],
  usage: [],
  experiments: [],
  definitionsPaged: paginate([], 1, 20),
  runsPaged: paginate([], 1, 20),
  usagePaged: paginate([], 1, 20),
  experimentsPaged: paginate([], 1, 20),
  selectedRun: null,
  auditData: null,
  includePreReleases: false,
  maintenanceOutput: "Use this view to check dashboard maintenance commands before applying updates or upgrades.",
  maintenanceLastAction: "",
  commandInput: "",
  commandOutput: "",
  flashMessage: "",
  flashKind: "success",
  loadingCliStatus: true,
  loadingDefinitions: false,
  loadingRuns: false,
  loadingUsage: false,
  loadingExperiments: false,
  loadingAudit: false,
  loadingMaintenance: false,
  errorCliStatus: "",
  errorDefinitions: "",
  errorRuns: "",
  errorUsage: "",
  errorExperiments: "",
  errorAudit: "",
  runsMeta: null,
  usageMeta: null,

  async init() {
    this.commandInput = this.buildLogsCommand();
    await this.fetchCliStatus();
    if (this.cliStatus?.available) {
      await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchUsage(), this.fetchExperiments()]);
      this.commandOutput = `$ ${this.cliStatus.command}\ngh aw version ${this.cliStatus.version}`;
      return;
    }

    const unavailableMessage = this.cliUnavailableMessage();
    this.commandInput = this.cliStatus?.command ?? "gh aw version";
    this.commandOutput = unavailableMessage;
  },

  currentWindow() {
    return reportWindowById(this.selectedWindow);
  },

  reportWindowClass(windowId) {
    return this.selectedWindow === windowId ? "BtnGroup-item btn btn-sm btn-primary" : "BtnGroup-item btn btn-sm";
  },

  async selectReportWindow(windowId) {
    if (this.selectedWindow === windowId) return;
    this.selectedWindow = windowId;
    this.commandInput = this.buildLogsCommand();
    if (!this.cliStatus?.available) return;
    await Promise.all([this.fetchRuns(), this.fetchUsage()]);
  },

  async fetchCliStatus() {
    this.loadingCliStatus = true;
    this.errorCliStatus = "";
    try {
      this.cliStatus = await fetchJson<CLIStatus>("/api/cli-status");
    } catch (error) {
      this.cliStatus = null;
      this.errorCliStatus = `Failed to detect gh aw: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingCliStatus = false;
    }
  },

  async fetchDefinitions() {
    this.loadingDefinitions = true;
    this.errorDefinitions = "";
    try {
      this.definitions = await fetchJson<WorkflowDefinition[]>("/api/status");
      this.loadDefinitionPage(1);
    } catch (error) {
      this.errorDefinitions = `Failed to load workflows: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingDefinitions = false;
    }
  },

  async fetchRuns() {
    this.loadingRuns = true;
    this.errorRuns = "";
    try {
      const previousRunId = this.selectedRun?.run_id ?? null;
      const params = new URLSearchParams({
        count: "100",
        window: this.selectedWindow,
        timeout: String(this.logsTimeout),
      });
      const data = await fetchJson<RunsResponse>(`/api/runs?${params.toString()}`);
      this.runsMeta = data;
      this.runs = Array.isArray(data.runs) ? data.runs : [];
      this.loadRunPage(1);
      this.selectedRun = this.runs.find(run => run.run_id === previousRunId) ?? this.runs[0] ?? null;
    } catch (error) {
      this.runsMeta = null;
      this.errorRuns = `Failed to load runs: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingRuns = false;
    }
  },

  async fetchUsage() {
    this.loadingUsage = true;
    this.errorUsage = "";
    try {
      const params = new URLSearchParams({
        count: "100",
        window: this.selectedWindow,
        timeout: String(this.logsTimeout),
      });
      const data = await fetchJson<UsageResponse>(`/api/usage?${params.toString()}`);
      this.usageMeta = data;
      this.usage = Array.isArray(data.items) ? data.items : [];
      this.loadUsagePage(1);
    } catch (error) {
      this.usageMeta = null;
      this.errorUsage = `Failed to load usage summary: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingUsage = false;
    }
  },

  async fetchExperiments() {
    this.loadingExperiments = true;
    this.errorExperiments = "";
    try {
      this.experiments = await fetchJson<ExperimentInfo[]>("/api/experiments");
      this.loadExperimentPage(1);
    } catch (error) {
      this.errorExperiments = `Failed to load experiments: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingExperiments = false;
    }
  },

  async refresh() {
    await fetch("/api/refresh");
    this.flashMessage = "Refreshing…";
    this.flashKind = "success";
    await this.fetchCliStatus();
    if (this.cliStatus?.available) {
      await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchUsage(), this.fetchExperiments()]);
      this.commandOutput = `$ ${this.cliStatus.command}\ngh aw version ${this.cliStatus.version}`;
    } else {
      this.definitions = [];
      this.runs = [];
      this.usage = [];
      this.experiments = [];
      this.selectedRun = null;
      this.runsMeta = null;
      this.usageMeta = null;
      this.loadDefinitionPage(1);
      this.loadRunPage(1);
      this.loadUsagePage(1);
      this.loadExperimentPage(1);
      const unavailableMessage = this.cliUnavailableMessage();
      this.commandInput = this.cliStatus?.command ?? "gh aw version";
      this.commandOutput = unavailableMessage;
    }
    this.flashMessage = "Refreshed.";
    setTimeout(() => {
      this.flashMessage = "";
    }, 3000);
  },

  setActiveTab(tab) {
    if (this.tabs.some(item => item.id === tab)) this.activeTab = tab;
  },

  isActiveTab(tab) {
    return this.activeTab === tab;
  },

  tabCount(tab) {
    if (tab.counter === "definitions") return this.definitions.length;
    if (tab.counter === "runs") return this.runs.length;
    if (tab.counter === "usage") return this.usage.length;
    if (tab.counter === "experiments") return this.experiments.length;
    return 0;
  },

  loadDefinitionPage(page) {
    this.definitionsPaged = paginate(this.definitions, page, this.pageSize);
  },

  loadRunPage(page) {
    this.runsPaged = paginate(this.runs, page, this.pageSize);
  },

  loadUsagePage(page) {
    this.usagePaged = paginate(this.usage, page, this.pageSize);
  },

  loadExperimentPage(page) {
    this.experimentsPaged = paginate(this.experiments, page, this.pageSize);
  },

  selectRun(runId) {
    const previousRunId = this.selectedRun?.run_id;
    this.selectedRun = this.runs.find(run => run.run_id === runId) ?? null;
    if (this.selectedRun?.run_id !== previousRunId) {
      this.clearAudit();
    }
  },

  viewRunDetails(runId) {
    this.selectRun(runId);
    this.setActiveTab("details");
  },

  async loadAudit() {
    if (!this.selectedRun?.run_id) return;
    this.loadingAudit = true;
    this.errorAudit = "";
    try {
      const params = new URLSearchParams({ run_id: String(this.selectedRun.run_id) });
      this.auditData = await fetchJson<AuditReport>(`/api/audit?${params.toString()}`);
    } catch (error) {
      this.auditData = null;
      this.errorAudit = `Failed to load audit: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingAudit = false;
    }
  },

  clearAudit() {
    this.auditData = null;
    this.errorAudit = "";
    this.loadingAudit = false;
  },

  buildLogsCommand(count = DEFAULT_LOGS_COMMAND_COUNT) {
    const window = this.currentWindow();
    return `gh aw logs --json -c ${count} --start-date ${window.startDate} --timeout ${this.logsTimeout}`;
  },

  buildMaintenanceCommand(action) {
    if (action === "check-update") return "gh aw status --json";
    if (action === "run-update") return "gh aw update";
    if (action === "check-upgrade") return `gh aw upgrade --audit${this.includePreReleases ? " --pre-releases" : ""}`;
    return `gh aw upgrade${this.includePreReleases ? " --pre-releases" : ""}`;
  },

  maintenanceActionLabel(action) {
    if (action === "check-update") return "Check updates";
    if (action === "run-update") return "Run update";
    if (action === "check-upgrade") return "Check upgrades";
    return "Run upgrade";
  },

  async runMaintenanceAction(action) {
    const cmd = this.buildMaintenanceCommand(action);
    this.loadingMaintenance = true;
    this.maintenanceLastAction = this.maintenanceActionLabel(action);
    this.maintenanceOutput = `$ ${cmd}\n(running…)`;
    this.commandInput = cmd;
    try {
      const params = new URLSearchParams({
        cmd,
        window: this.selectedWindow,
        timeout: String(this.logsTimeout),
      });
      const result = await fetchJson<{ command?: string; output?: string }>(`/api/run-command?${params.toString()}`);
      this.maintenanceOutput = `$ ${result.command ?? cmd}\n${result.output ?? ""}`;
      if (action === "run-update" || action === "run-upgrade") {
        await fetch("/api/refresh");
        await this.fetchCliStatus();
        if (this.cliStatus?.available) {
          await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchUsage(), this.fetchExperiments()]);
        }
      }
    } catch (error) {
      this.maintenanceOutput = `$ ${cmd}\nError: ${error instanceof Error ? error.message : String(error)}`;
    } finally {
      this.loadingMaintenance = false;
    }
  },

  cliUnavailableMessage() {
    return (this.cliStatus?.message ?? this.errorCliStatus) || "gh aw is not installed.";
  },

  auditHasFindings() {
    return (this.auditData?.key_findings ?? []).length > 0;
  },

  auditSeverityClass(severity) {
    const normalized = (severity ?? "").toLowerCase();
    if (normalized === "critical" || normalized === "high" || normalized === "error") return "Label Label--danger";
    if (normalized === "medium" || normalized === "warning") return "Label Label--attention";
    if (normalized === "low") return "Label Label--accent";
    return "Label Label--secondary";
  },

  auditPriorityClass(priority) {
    const normalized = (priority ?? "").toLowerCase();
    if (normalized === "high" || normalized === "p0" || normalized === "p1") return "Label Label--danger";
    if (normalized === "medium" || normalized === "p2") return "Label Label--attention";
    return "Label Label--secondary";
  },

  buildReportSummaryMessage(meta) {
    return buildReportMessage(meta, "No logs metadata available.");
  },

  async runCommand() {
    const cmd = this.commandInput.trim();
    this.commandOutput = `$ ${cmd}\n(running…)`;
    try {
      const params = new URLSearchParams({
        cmd,
        window: this.selectedWindow,
        timeout: String(this.logsTimeout),
      });
      const result = await fetchJson<{ command?: string; output?: string }>(`/api/run-command?${params.toString()}`);
      this.commandOutput = `$ ${result.command ?? cmd}\n${result.output ?? ""}`;
    } catch (error) {
      this.commandOutput = `$ ${cmd}\nError: ${error instanceof Error ? error.message : String(error)}`;
    }
  },

  commandQuickFill(value) {
    this.commandInput = value;
    this.runCommand().catch(error => {
      this.commandOutput = `$ ${this.commandInput}\nError: ${error instanceof Error ? error.message : String(error)}`;
    });
  },

  runStatusClass,
  runStatusLabel,
  definitionStatusClass,
  definitionStatusLabel,
  formatDuration,
  formatDate,
  formatAIC,
  formatNumber,
  cliSourceLabel,
}));

Alpine.start();
