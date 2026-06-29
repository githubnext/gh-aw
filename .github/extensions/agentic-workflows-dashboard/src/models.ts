export type WorkflowRunStatus = "queued" | "running" | "completed" | "failed";
export type WorkflowStepStatus = "pending" | "running" | "done" | "failed";

export interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
  enabled: boolean;
}

export interface WorkflowStep {
  id: string;
  title: string;
  status: WorkflowStepStatus;
  summaryMarkdown: string;
}

export interface WorkflowRun {
  id: string;
  definitionId: string;
  status: WorkflowRunStatus;
  createdAt: string;
  updatedAt: string;
  steps: WorkflowStep[];
}

export interface PagedResult<T> {
  items: T[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
}
