// @ts-ignore - browser runtime resolves the remote ESM import directly
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
function cliSourceLabel(cliStatus) {
    if (!cliStatus?.available)
        return "not installed";
    if (cliStatus.source === "dev-binary")
        return "local build";
    if (cliStatus.source === "gh-extension")
        return "gh extension";
    return "available";
}
function runStatusClass(run) {
    const status = run.status ?? "";
    const conclusion = run.conclusion ?? "";
    if (status === "completed" || status === "success") {
        return conclusion && conclusion !== "success" ? "Label Label--danger" : "Label Label--success";
    }
    if (status === "failure" || status === "failed")
        return "Label Label--danger";
    if (status === "in_progress" || status === "running")
        return "Label Label--attention";
    return "Label Label--secondary";
}
function runStatusLabel(run) {
    if (run.status === "completed" && run.conclusion)
        return run.conclusion;
    return run.status ?? "unknown";
}
function definitionStatusClass(definition) {
    if (definition.status === "disabled")
        return "Label Label--secondary";
    return definition.compiled === "yes" ? "Label Label--success" : "Label Label--attention";
}
function definitionStatusLabel(definition) {
    if (definition.status === "disabled")
        return "disabled";
    return definition.compiled === "yes" ? "enabled" : "not compiled";
}
function formatDuration(ms) {
    if (ms == null)
        return "—";
    const secs = Math.round(ms / 1000);
    if (secs < 60)
        return `${secs}s`;
    return `${Math.floor(secs / 60)}m ${secs % 60}s`;
}
function formatDate(iso) {
    if (!iso)
        return "—";
    const date = new Date(iso);
    return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}
function formatNumber(value, options = {}) {
    const numeric = Number(value ?? 0);
    if (!Number.isFinite(numeric))
        return "0";
    return new Intl.NumberFormat(undefined, options).format(numeric);
}
function formatAIC(value) {
    const numeric = Number(value ?? 0);
    if (!Number.isFinite(numeric) || numeric <= 0)
        return "0";
    return formatNumber(Math.ceil(numeric));
}
function reportWindowById(windowId) {
    return reportWindows.find(window => window.id === windowId) ?? reportWindows[1];
}
function buildReportMessage(meta, emptyLabel) {
    if (!meta?.window)
        return emptyLabel ?? "";
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
async function fetchJson(url) {
    const response = await fetch(url);
    const data = (await response.json());
    if (!response.ok) {
        throw new Error(data.error ?? `HTTP ${response.status}`);
    }
    return data;
}
Alpine.data("dashboardApp", () => ({
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
    loadingAudit: false,
    errorAudit: "",
    commandInput: "",
    commandOutput: "",
    flashMessage: "",
    flashKind: "success",
    loadingCliStatus: true,
    loadingDefinitions: false,
    loadingRuns: false,
    loadingUsage: false,
    loadingExperiments: false,
    errorCliStatus: "",
    errorDefinitions: "",
    errorRuns: "",
    errorUsage: "",
    errorExperiments: "",
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
        this.commandInput = this.cliStatus?.command ?? "gh aw version";
        this.commandOutput = this.cliStatus?.message ?? "gh aw is not installed.";
    },
    currentWindow() {
        return reportWindowById(this.selectedWindow);
    },
    reportWindowClass(windowId) {
        return this.selectedWindow === windowId ? "btn btn-sm btn-primary" : "btn btn-sm";
    },
    async selectReportWindow(windowId) {
        if (this.selectedWindow === windowId)
            return;
        this.selectedWindow = windowId;
        this.commandInput = this.buildLogsCommand();
        if (!this.cliStatus?.available)
            return;
        await Promise.all([this.fetchRuns(), this.fetchUsage()]);
    },
    async fetchCliStatus() {
        this.loadingCliStatus = true;
        this.errorCliStatus = "";
        try {
            this.cliStatus = await fetchJson("/api/cli-status");
        }
        catch (error) {
            this.cliStatus = null;
            this.errorCliStatus = `Failed to detect gh aw: ${error instanceof Error ? error.message : String(error)}`;
        }
        finally {
            this.loadingCliStatus = false;
        }
    },
    async fetchDefinitions() {
        this.loadingDefinitions = true;
        this.errorDefinitions = "";
        try {
            this.definitions = await fetchJson("/api/status");
            this.loadDefinitionPage(1);
        }
        catch (error) {
            this.errorDefinitions = `Failed to load workflows: ${error instanceof Error ? error.message : String(error)}`;
        }
        finally {
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
            const data = await fetchJson(`/api/runs?${params.toString()}`);
            this.runsMeta = data;
            this.runs = Array.isArray(data.runs) ? data.runs : [];
            this.loadRunPage(1);
            this.selectedRun = this.runs.find(run => run.run_id === previousRunId) ?? this.runs[0] ?? null;
        }
        catch (error) {
            this.runsMeta = null;
            this.errorRuns = `Failed to load runs: ${error instanceof Error ? error.message : String(error)}`;
        }
        finally {
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
            const data = await fetchJson(`/api/usage?${params.toString()}`);
            this.usageMeta = data;
            this.usage = Array.isArray(data.items) ? data.items : [];
            this.loadUsagePage(1);
        }
        catch (error) {
            this.usageMeta = null;
            this.errorUsage = `Failed to load usage summary: ${error instanceof Error ? error.message : String(error)}`;
        }
        finally {
            this.loadingUsage = false;
        }
    },
    async fetchExperiments() {
        this.loadingExperiments = true;
        this.errorExperiments = "";
        try {
            this.experiments = await fetchJson("/api/experiments");
            this.loadExperimentPage(1);
        }
        catch (error) {
            this.errorExperiments = `Failed to load experiments: ${error instanceof Error ? error.message : String(error)}`;
        }
        finally {
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
        }
        else {
            this.definitions = [];
            this.runs = [];
            this.usage = [];
            this.experiments = [];
            this.loadDefinitionPage(1);
            this.loadRunPage(1);
            this.loadUsagePage(1);
            this.loadExperimentPage(1);
            this.commandInput = this.cliStatus?.command ?? "gh aw version";
            this.commandOutput = this.cliStatus?.message ?? "gh aw is not installed.";
        }
        this.flashMessage = "Refreshed.";
        setTimeout(() => {
            this.flashMessage = "";
        }, 3000);
    },
    setActiveTab(tab) {
        if (this.tabs.some(item => item.id === tab))
            this.activeTab = tab;
    },
    isActiveTab(tab) {
        return this.activeTab === tab;
    },
    tabCount(tab) {
        if (tab.counter === "definitions")
            return this.definitions.length;
        if (tab.counter === "runs")
            return this.runs.length;
        if (tab.counter === "usage")
            return this.usage.length;
        if (tab.counter === "experiments")
            return this.experiments.length;
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
        this.auditData = null;
        this.errorAudit = "";
        this.loadingAudit = false;
    },
    viewRunDetails(runId) {
        this.selectRun(runId);
        this.setActiveTab("details");
    },
    async loadAudit() {
        if (!this.selectedRun) return;
        const requestedRunId = this.selectedRun.run_id;
        this.loadingAudit = true;
        this.errorAudit = "";
        this.auditData = null;
        try {
            const params = new URLSearchParams({ run_id: String(requestedRunId) });
            const resp = await fetch(`/api/audit?${params.toString()}`);
            const data = await resp.json();
            if (!resp.ok) throw new Error(data.error ?? `HTTP ${resp.status}`);
            if (this.selectedRun?.run_id !== requestedRunId) return;
            this.auditData = data;
        } catch (error) {
            if (this.selectedRun?.run_id !== requestedRunId) return;
            this.errorAudit = `Audit failed: ${error instanceof Error ? error.message : String(error)}`;
        } finally {
            if (this.selectedRun?.run_id === requestedRunId) {
                this.loadingAudit = false;
            }
        }
    },
    clearAudit() {
        this.auditData = null;
        this.errorAudit = "";
    },
    auditHasFindings() {
        return (this.auditData?.key_findings ?? []).length > 0;
    },
    auditSeverityClass(severity) {
        if (severity === "critical" || severity === "high") return "Label Label--danger";
        if (severity === "medium") return "Label Label--attention";
        if (severity === "low") return "Label Label--secondary";
        return "Label Label--accent";
    },
    auditPriorityClass(priority) {
        if (priority === "high") return "Label Label--danger";
        if (priority === "medium") return "Label Label--attention";
        return "Label Label--secondary";
    },
    buildLogsCommand(count = DEFAULT_LOGS_COMMAND_COUNT) {
        const window = this.currentWindow();
        return `gh aw logs --json -c ${count} --start-date ${window.startDate} --timeout ${this.logsTimeout}`;
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
            const result = await fetchJson(`/api/run-command?${params.toString()}`);
            this.commandOutput = `$ ${result.command ?? cmd}\n${result.output ?? ""}`;
        }
        catch (error) {
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
