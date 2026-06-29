import Alpine from "https://cdn.jsdelivr.net/npm/alpinejs@3.15.0/+esm";
import { paginate } from "./pagination.js";

const dashboardTabs = [
    { id: "definitions", label: "Workflows", counter: "definitions" },
    { id: "runs", label: "Runs", counter: "runs" },
    { id: "details", label: "Run details" },
    { id: "experiments", label: "Experiments", counter: "experiments" },
    { id: "commands", label: "Commands" },
];

function runStatusClass(run) {
    const s = run?.status ?? "";
    const c = run?.conclusion ?? "";
    if (s === "completed" || s === "success") {
        return c && c !== "success" ? "Label Label--danger" : "Label Label--success";
    }
    if (s === "failure" || s === "failed") return "Label Label--danger";
    if (s === "in_progress" || s === "running") return "Label Label--attention";
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
    const d = new Date(iso);
    return isNaN(d.getTime()) ? "—" : d.toLocaleString();
}

Alpine.data("dashboardApp", () => ({
    tabs: dashboardTabs,
    activeTab: "definitions",
    pageSize: 20,
    definitions: [],
    runs: [],
    experiments: [],
    definitionsPaged: paginate([], 1, 20),
    runsPaged: paginate([], 1, 20),
    experimentsPaged: paginate([], 1, 20),
    selectedRun: null,
    commandInput: "gh aw status",
    commandOutput: "",
    flashMessage: "",
    flashKind: "success",
    loadingDefinitions: true,
    loadingRuns: true,
    loadingExperiments: true,
    errorDefinitions: "",
    errorRuns: "",
    errorExperiments: "",

    async init() {
        await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchExperiments()]);
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
        } catch (e) {
            this.errorDefinitions = `Failed to load workflows: ${e.message}`;
        } finally {
            this.loadingDefinitions = false;
        }
    },

    async fetchRuns() {
        this.loadingRuns = true;
        this.errorRuns = "";
        try {
            const resp = await fetch("/api/runs?count=50");
            const data = await resp.json();
            if (!resp.ok) throw new Error(data.error ?? `HTTP ${resp.status}`);
            this.runs = Array.isArray(data) ? data : [];
            this.loadRunPage(1);
            if (!this.selectedRun && this.runs.length > 0) this.selectedRun = this.runs[0];
        } catch (e) {
            this.errorRuns = `Failed to load runs: ${e.message}`;
        } finally {
            this.loadingRuns = false;
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
        } catch (e) {
            this.errorExperiments = `Failed to load experiments: ${e.message}`;
        } finally {
            this.loadingExperiments = false;
        }
    },

    async refresh() {
        await fetch("/api/refresh");
        this.flashMessage = "Refreshing…";
        this.flashKind = "success";
        await Promise.all([this.fetchDefinitions(), this.fetchRuns(), this.fetchExperiments()]);
        this.flashMessage = "Refreshed.";
        setTimeout(() => { this.flashMessage = ""; }, 3000);
    },

    setActiveTab(tab) {
        if (this.tabs.some(t => t.id === tab)) this.activeTab = tab;
    },
    isActiveTab(tab) { return this.activeTab === tab; },
    tabCount(tab) {
        if (tab.counter === "definitions") return this.definitions.length;
        if (tab.counter === "runs") return this.runs.length;
        if (tab.counter === "experiments") return this.experiments.length;
        return 0;
    },

    loadDefinitionPage(page) {
        this.definitionsPaged = paginate(this.definitions, page, this.pageSize);
    },
    loadRunPage(page) {
        this.runsPaged = paginate(this.runs, page, this.pageSize);
    },
    loadExperimentPage(page) {
        this.experimentsPaged = paginate(this.experiments, page, this.pageSize);
    },

    selectRun(runId) {
        this.selectedRun = this.runs.find(r => r.run_id === runId) ?? null;
    },
    viewRunDetails(runId) {
        this.selectRun(runId);
        this.setActiveTab("details");
    },

    runStatusClass,
    runStatusLabel,
    definitionStatusClass,
    definitionStatusLabel,
    formatDuration,
    formatDate,

    async runCommand() {
        const cmd = this.commandInput.trim();
        this.commandOutput = `$ ${cmd}\n(running…)`;
        try {
            const resp = await fetch(`/api/run-command?cmd=${encodeURIComponent(cmd)}`);
            const result = await resp.json();
            this.commandOutput = `$ ${result.command ?? cmd}\n${result.output ?? ""}`;
        } catch (e) {
            this.commandOutput = `$ ${cmd}\nError: ${e.message}`;
        }
    },
    commandQuickFill(value) {
        this.commandInput = value;
        this.runCommand();
    },
}));

Alpine.start();
