export type WorkflowJob = {
  id: string;
  needs: string[];
};

export type WorkflowGraph = {
  jobs: WorkflowJob[];
};

export type ParseWorkflowResult =
  | { ok: true; graph: WorkflowGraph }
  | { ok: false; error: string };
