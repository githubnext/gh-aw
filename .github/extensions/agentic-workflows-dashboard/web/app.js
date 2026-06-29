import Alpine from "https://cdn.jsdelivr.net/npm/alpinejs@3.15.0/+esm";
import { paginate } from "./pagination.js";

const dashboardTabs = [
  { id: "definitions", label: "Workflows", counter: "definitions" },
  { id: "runs", label: "Runs", counter: "runs" },
  { id: "details", label: "Run details" },
  { id: "usage", label: "Usage", counter: "usage" },
  { id: "experiments", label: "Experiments", counter: "experiments" },
  { id: "commands", label: "Commands" },
];

const reportWindows = [
  { id: "3d", label: "3 days", startDate: "-3d" },
  { id: "7d", label: "7 days", startDate: "-1w" },
  { id: "1mo", label: "1 month", startDate: "-1mo" },
];
const DEFAULT_LOGS_COMMAND_COUNT = 25;

function runStatusClass(run) {
  const status = run?.status ?? "";
  const conclusion = run?.conclusion ?? "";
  if (status === "completed" || status === "success") {
    return conclusion && conclusion !== "success" ? "Label Label--danger" : "Label Label--success";
  }
  if (status === "failure" || status === "failed") return "Label Label--danger";
  if (status === "in_progress" || status === "running") return "Label Label--attention";
  return "Label Label--secondary";
}

function runStatusLabel(run) {
  if (run?.status === "completed" && run?.conclusion) return run.conclusion;
  return run?.status ?? "unknown";
}

function definitionStatusClass(def) {
  if (def?.status === "disabled") return "Label Label--secondary";
  return def?.compiled === "yes" ? "Label Label--success" : "Label Label--attention";
}

function definitionStatusLabel(def) {
  if (def?.status === "disabled") return "disabled";
  return def?.compiled === "yes" ? "enabled" : "not compiled";
}

function formatDuration(ms) {
  if (ms == null) return "—";
  const secs = Math.round(ms / 1000);
  if (secs < 60) return `${secs}s`;
  return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}

function formatDate(iso) {
  if (!iso) return "—";
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}

function formatNumber(value, options = {}) {
  const numeric = Number(value ?? 0);
  if (!Number.isFinite(numeric)) return "0";
  return new Intl.NumberFormat(undefined, options).format(numeric);
}

function formatAIC(value) {
  const numeric = Number(value ?? 0);
  if (!Number.isFinite(numeric) || numeric <= 0) return "0";
  return formatNumber(Math.ceil(numeric));
}

function reportWindowById(windowId) {
  return reportWindows.find(window => window.id === windowId) ?? reportWindows[1];
}

function buildReportMessage(meta, emptyLabel) {
  if (!meta?.window) {
    return emptyLabel ?? "";
  }

  const fragments = [`Window: ${meta.window.label}`];
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

Alpine.data("dashboardApp", () => ({
  tabs: dashboardTabs,
  reportWindows,
  activeTab: "definitions",
  selectedWindow: "7d",
  logsTimeout: 1,
  pageSize: 20,
  definitions: [],
  runs: [],
  usage: [],
  experiments: [],
  definitionsPaged: paginate([], 1, 20),
  runsPaged: paginate([], 1, 20),
  usagePaged: paginate([], 1, 20),
  experimentsPaged: paginate([], 1, 20),
  selectedRun: null,
  commandInput: "",
  commandOutput: "",
  flashMessage: "",
  flashKind: "success",
  loadingDefinitions: true,
  loadingRuns: true,
  loadingUsage: true,
  loadingExperiments: true,
  errorDefinitions: "",
  errorRuns: "",
  errorUsage: "",
  errorExperiments: "",
  runsMeta: null,
  usageMeta: null,

  async init() {
    this.commandInput = this.buildLogsCommand();
    await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchUsage(), this.fetchExperiments()]);
  },

  currentWindow() {
    return reportWindowById(this.selectedWindow);
  },

  reportWindowClass(windowId) {
    return this.selectedWindow === windowId ? "btn btn-sm btn-primary" : "btn btn-sm";
  },

  async selectReportWindow(windowId) {
    if (this.selectedWindow === windowId) return;
    this.selectedWindow = windowId;
    this.commandInput = this.buildLogsCommand();
    await Promise.all([this.fetchRuns(), this.fetchUsage()]);
  },

  async fetchDefinitions() {
    this.loadingDefinitions = true;
    this.errorDefinitions = "";
    try {
      const resp = await fetch("/api/status");
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error ?? `HTTP ${resp.status}`);
      this.definitions = Array.isArray(data) ? data : [];
      this.loadDefinitionPage(1);
    } catch (error) {
      this.errorDefinitions = `Failed to load workflows: ${error.message}`;
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
      const resp = await fetch(`/api/runs?${params.toString()}`);
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error ?? `HTTP ${resp.status}`);
      this.runsMeta = data;
      this.runs = Array.isArray(data?.runs) ? data.runs : [];
      this.loadRunPage(1);
      this.selectedRun = this.runs.find(run => run.run_id === previousRunId) ?? this.runs[0] ?? null;
    } catch (error) {
      this.errorRuns = `Failed to load runs: ${error.message}`;
      this.runsMeta = null;
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
      const resp = await fetch(`/api/usage?${params.toString()}`);
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error ?? `HTTP ${resp.status}`);
      this.usageMeta = data;
      this.usage = Array.isArray(data?.items) ? data.items : [];
      this.loadUsagePage(1);
    } catch (error) {
      this.errorUsage = `Failed to load usage summary: ${error.message}`;
      this.usageMeta = null;
    } finally {
      this.loadingUsage = false;
    }
  },

  async fetchExperiments() {
    this.loadingExperiments = true;
    this.errorExperiments = "";
    try {
      const resp = await fetch("/api/experiments");
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error ?? `HTTP ${resp.status}`);
      this.experiments = Array.isArray(data) ? data : [];
      this.loadExperimentPage(1);
    } catch (error) {
      this.errorExperiments = `Failed to load experiments: ${error.message}`;
    } finally {
      this.loadingExperiments = false;
    }
  },

  async refresh() {
    await fetch("/api/refresh");
    this.flashMessage = "Refreshing…";
    this.flashKind = "success";
    await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchUsage(), this.fetchExperiments()]);
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
    this.selectedRun = this.runs.find(run => run.run_id === runId) ?? null;
  },

  viewRunDetails(runId) {
    this.selectRun(runId);
    this.setActiveTab("details");
  },

  buildLogsCommand(count = DEFAULT_LOGS_COMMAND_COUNT) {
    const window = this.currentWindow();
    return `gh aw logs --json -c ${count} --start-date ${window.startDate} --timeout ${this.logsTimeout}`;
  },

  buildReportSummaryMessage(meta) {
    return buildReportMessage(meta, "No logs metadata available.");
  },

  runStatusClass,
  runStatusLabel,
  definitionStatusClass,
  definitionStatusLabel,
  formatDuration,
  formatDate,
  formatAIC,
  formatNumber,

  async runCommand() {
    const cmd = this.commandInput.trim();
    this.commandOutput = `$ ${cmd}\n(running…)`;
    try {
      const params = new URLSearchParams({
        cmd,
        window: this.selectedWindow,
        timeout: String(this.logsTimeout),
      });
      const resp = await fetch(`/api/run-command?${params.toString()}`);
      const result = await resp.json();
      this.commandOutput = `$ ${result.command ?? cmd}\n${result.output ?? ""}`;
    } catch (error) {
      this.commandOutput = `$ ${cmd}\nError: ${error.message}`;
    }
  },

  commandQuickFill(value) {
    this.commandInput = value;
    this.runCommand();
  },
}));

Alpine.start();
