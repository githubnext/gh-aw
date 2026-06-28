import type { PagedResult, WorkflowDefinition, WorkflowRun, WorkflowRunStatus, WorkflowStep, WorkflowStepStatus } from "./models.js";
import { paginate } from "./pagination.js";

type CommandResult = { command: string; output: string };

type FlashKind = "success" | "warn" | "error";
type DashboardTabId = "definitions" | "runs" | "details" | "commands";
type DashboardTab = { id: DashboardTabId; label: string; counter?: "definitions" | "runs" };
declare const Alpine: {
  data: (name: string, callback: () => DashboardState) => void;
};

interface DashboardState {
  tabs: DashboardTab[];
  activeTab: DashboardTabId;
  definitionPage: number;
  runPage: number;
  pageSize: number;
  definitions: WorkflowDefinition[];
  runs: WorkflowRun[];
  definitionsPaged: PagedResult<WorkflowDefinition>;
  runsPaged: PagedResult<WorkflowRun>;
  selectedRun: WorkflowRun | null;
  selectedDefinitionId: string;
  commandInput: string;
  commandOutput: string;
  flashMessage: string;
  flashKind: FlashKind;
  init(): void;
  setActiveTab(tab: DashboardTabId): void;
  isActiveTab(tab: DashboardTabId): boolean;
  tabCount(tab: DashboardTab): number;
  loadDefinitionPage(page: number): void;
  loadRunPage(page: number): void;
  selectRun(id: string): void;
  viewRunDetails(id: string): void;
  dispatchSelectedWorkflow(): void;
  runCommand(): void;
  commandQuickFill(value: string): void;
  auditDiffQuickFill(): string;
  renderMarkdown(markdown: string): string;
  formatDate(iso: string): string;
  runStatusClass(status: WorkflowRunStatus): string;
  stepStatusClass(status: WorkflowStepStatus): string;
}

const definitionCount = 240;
const runCount = 420;
const dashboardTabs: DashboardTab[] = [
  { id: "definitions", label: "Workflows", counter: "definitions" },
  { id: "runs", label: "Runs", counter: "runs" },
  { id: "details", label: "Run details" },
  { id: "commands", label: "Commands" },
];

function isoHoursAgo(hours: number): string {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function buildDefinitions(count: number): WorkflowDefinition[] {
  return Array.from({ length: count }, (_, i) => {
    const index = i + 1;
    return {
      id: `wf-${String(index).padStart(3, "0")}`,
      name: `agentic-workflow-${index}`,
      description: `Automated workflow #${index} for triage, reporting, and repository automation.`,
      inputSchema: {
        type: "object",
        properties: {
          issue: { type: "number" },
          branch: { type: "string" },
        },
        additionalProperties: false,
      },
      enabled: index % 9 !== 0,
    };
  });
}

function buildStep(runNumber: number, idx: number, status: WorkflowStepStatus): WorkflowStep {
  return {
    id: `step-${runNumber}-${idx}`,
    title: ["Resolve context", "Execute engine", "Publish summary", "Finalize run"][idx - 1] ?? `Step ${idx}`,
    status,
    summaryMarkdown: `### ${status === "failed" ? "Action required" : "Step complete"}\n- Run: **${runNumber}**\n- Step: **${idx}**\n- Status: \`${status}\`\n\n[View workflow docs](https://github.com/github/gh-aw/tree/main/docs)`,
  };
}

function buildRuns(count: number, definitions: WorkflowDefinition[]): WorkflowRun[] {
  const statuses: WorkflowRunStatus[] = ["queued", "running", "completed", "failed"];
  const stepStatusMap: Record<WorkflowRunStatus, WorkflowStepStatus[]> = {
    queued: ["pending", "pending", "pending", "pending"],
    running: ["done", "running", "pending", "pending"],
    completed: ["done", "done", "done", "done"],
    failed: ["done", "failed", "pending", "pending"],
  };

  return Array.from({ length: count }, (_, i) => {
    const index = i + 1;
    const status = statuses[index % statuses.length] ?? "queued";
    const definition = definitions[index % definitions.length] ?? definitions[0];
    if (!definition) {
      throw new Error("No workflow definitions available.");
    }
    const stepStatuses = stepStatusMap[status];

    return {
      id: `run-${String(index).padStart(5, "0")}`,
      definitionId: definition.id,
      status,
      createdAt: isoHoursAgo(index * 2),
      updatedAt: isoHoursAgo(index),
      steps: stepStatuses.map((stepStatus, stepIndex) => buildStep(index, stepIndex + 1, stepStatus)),
    };
  });
}

function escapeHtml(value: string): string {
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function renderSafeMarkdown(markdown: string): string {
  const safe = escapeHtml(markdown);
  const lines = safe.split("\n");
  let inList = false;
  const out: string[] = [];

  const closeList = () => {
    if (inList) {
      out.push("</ul>");
      inList = false;
    }
  };

  for (const line of lines) {
    if (line.startsWith("### ")) {
      closeList();
      out.push(`<h4 class=\"h5 mb-2\">${line.slice(4)}</h4>`);
      continue;
    }

    if (line.startsWith("- ")) {
      if (!inList) {
        out.push('<ul class="pl-3 mb-2">');
        inList = true;
      }
      out.push(`<li>${line.slice(2)}</li>`);
      continue;
    }

    if (line.trim().length === 0) {
      closeList();
      out.push('<div class="mb-1"></div>');
      continue;
    }

    closeList();
    out.push(`<p class=\"mb-2\">${line}</p>`);
  }

  closeList();

  return out
    .join("")
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a class="Link--primary" href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString();
}

function runGhCommand(command: string, runs: WorkflowRun[]): CommandResult {
  const normalized = command.trim();

  if (!normalized.startsWith("gh aw ")) {
    return { command: normalized, output: "Only gh aw commands are supported." };
  }

  if (normalized === "gh aw logs") {
    const first = runs[0];
    return {
      command: normalized,
      output: `Showing logs for latest run ${first?.id ?? "n/a"}.\n- status: ${first?.status ?? "unknown"}\n- updated: ${first ? formatDate(first.updatedAt) : "n/a"}`,
    };
  }

  const runFlagMatch = normalized.match(/--run(?:=|\s+)(run-\d{5})/);
  if (normalized.startsWith("gh aw logs --run") && runFlagMatch?.[1]) {
    const run = runs.find(item => item.id === runFlagMatch[1]);
    if (!run) {
      return { command: normalized, output: `Run ${runFlagMatch[1]} not found.` };
    }
    return {
      command: normalized,
      output: `Logs for ${run.id}\n- definition: ${run.definitionId}\n- status: ${run.status}\n- steps: ${run.steps.length}`,
    };
  }

  if (normalized === "gh aw compile") {
    return {
      command: normalized,
      output: `Compile summary\n- definitions loaded: ${definitions.length}\n- runs indexed: ${runs.length}\n- status: success`,
    };
  }

  if (normalized === "gh aw audit") {
    const completed = runs.filter(run => run.status === "completed").length;
    const failed = runs.filter(run => run.status === "failed").length;
    return {
      command: normalized,
      output: `Audit summary\n- total runs: ${runs.length}\n- completed: ${completed}\n- failed: ${failed}`,
    };
  }

  if (normalized.startsWith("gh aw audit-diff")) {
    const referencedRuns = normalized.match(/run-\d{5}/g) ?? [];
    if (referencedRuns.length < 2) {
      return {
        command: normalized,
        output: "Need two valid runs for diff. Example: gh aw audit-diff run-00002 run-00003",
      };
    }

    const baseRun = runs.find(item => item.id === referencedRuns[0]);
    const compareRun = runs.find(item => item.id === referencedRuns[1]);

    if (!baseRun || !compareRun) {
      return {
        command: normalized,
        output: "Need two valid runs for diff. Example: gh aw audit-diff run-00002 run-00003",
      };
    }

    return {
      command: normalized,
      output: `Audit diff\n- base: ${baseRun.id} (${baseRun.status})\n- compare: ${compareRun.id} (${compareRun.status})\n- step delta: ${compareRun.steps.length - baseRun.steps.length}`,
    };
  }

  if (normalized.startsWith("gh aw audit --run") && runFlagMatch?.[1]) {
    const run = runs.find(item => item.id === runFlagMatch[1]);
    if (!run) {
      return { command: normalized, output: `Run ${runFlagMatch[1]} not found.` };
    }
    const failedSteps = run.steps.filter(step => step.status === "failed").length;
    return {
      command: normalized,
      output: `Audit for ${run.id}\n- status: ${run.status}\n- failed steps: ${failedSteps}\n- updated: ${formatDate(run.updatedAt)}`,
    };
  }

  return {
    command: normalized,
    output: "Supported commands: gh aw logs, gh aw logs --run <id>, gh aw compile, gh aw audit, gh aw audit --run <id>, gh aw audit-diff.",
  };
}

function statusClass(status: WorkflowRunStatus): string {
  switch (status) {
    case "completed":
      return "Label Label--success";
    case "failed":
      return "Label Label--danger";
    case "running":
      return "Label Label--attention";
    default:
      return "Label Label--secondary";
  }
}

function stepStatusClass(status: WorkflowStepStatus): string {
  switch (status) {
    case "done":
      return "Label Label--success";
    case "failed":
      return "Label Label--danger";
    case "running":
      return "Label Label--attention";
    default:
      return "Label Label--secondary";
  }
}

const definitions = buildDefinitions(definitionCount);
const runs = buildRuns(runCount, definitions);

document.addEventListener("alpine:init", () => {
  Alpine.data("dashboardApp", () => ({
    tabs: dashboardTabs,
    activeTab: "definitions",
    definitionPage: 1,
    runPage: 1,
    pageSize: 20,
    definitions,
    runs,
    definitionsPaged: paginate(definitions, 1, 20),
    runsPaged: paginate(runs, 1, 20),
    selectedRun: runs[0] ?? null,
    selectedDefinitionId: definitions[0]?.id ?? "",
    commandInput: "gh aw logs",
    commandOutput: "",
    flashMessage: "",
    flashKind: "success",

    init() {
      this.loadDefinitionPage(1);
      this.loadRunPage(1);
      if (this.commandOutput.length === 0) {
        this.runCommand();
      }
    },

    setActiveTab(tab) {
      if (this.tabs.some(item => item.id === tab)) {
        this.activeTab = tab;
      }
    },

    isActiveTab(tab) {
      return this.activeTab === tab;
    },

    tabCount(tab) {
      if (tab.counter === "definitions") {
        return this.definitions.length;
      }
      if (tab.counter === "runs") {
        return this.runs.length;
      }
      return 0;
    },

    loadDefinitionPage(page) {
      this.definitionPage = page;
      this.definitionsPaged = paginate(this.definitions, page, this.pageSize);
    },

    loadRunPage(page) {
      this.runPage = page;
      this.runsPaged = paginate(this.runs, page, this.pageSize);
      if (!this.selectedRun && this.runsPaged.items.length > 0) {
        this.selectedRun = this.runsPaged.items[0] ?? null;
      }
    },

    selectRun(id) {
      this.selectedRun = this.runs.find(run => run.id === id) ?? null;
    },

    viewRunDetails(id) {
      this.selectRun(id);
      this.setActiveTab("details");
    },

    dispatchSelectedWorkflow() {
      const definition = this.definitions.find(item => item.id === this.selectedDefinitionId);
      if (!definition) {
        this.flashKind = "error";
        this.flashMessage = "Select a workflow definition before dispatching.";
        return;
      }

      const sequence = this.runs.length + 1;
      const now = new Date().toISOString();
      const newRun: WorkflowRun = {
        id: `run-${String(sequence).padStart(5, "0")}`,
        definitionId: definition.id,
        status: "queued",
        createdAt: now,
        updatedAt: now,
        steps: [buildStep(sequence, 1, "pending"), buildStep(sequence, 2, "pending"), buildStep(sequence, 3, "pending"), buildStep(sequence, 4, "pending")],
      };

      this.runs = [newRun, ...this.runs];
      this.loadRunPage(1);
      this.viewRunDetails(newRun.id);

      this.flashKind = "success";
      this.flashMessage = `Dispatched ${definition.name} as ${newRun.id}.`;
    },

    runCommand() {
      const result = runGhCommand(this.commandInput, this.runs);
      this.commandOutput = `$ ${result.command}\n${result.output}`;
    },

    commandQuickFill(value) {
      this.commandInput = value;
      this.runCommand();
    },

    auditDiffQuickFill() {
      const selectedId = this.selectedRun?.id;
      if (!selectedId) {
        return "gh aw audit-diff run-00001 run-00002";
      }

      const firstRunId = this.runs[0]?.id ?? "run-00001";
      const secondRunId = this.runs[1]?.id ?? "run-00002";
      const compareId = selectedId === firstRunId ? secondRunId : firstRunId;
      return `gh aw audit-diff ${selectedId} ${compareId}`;
    },

    renderMarkdown(markdown) {
      return renderSafeMarkdown(markdown);
    },

    formatDate,
    runStatusClass: statusClass,
    stepStatusClass,
  }));
});
