export type Conclusion =
  | 'success'
  | 'failure'
  | 'cancelled'
  | 'skipped'
  | 'neutral'
  | 'timed_out'
  | 'action_required'
  | 'stale'
  | null;

export interface RunLogGroup {
  title: string;
  lines?: string[];
  omittedLineCount?: number;
  children?: RunLogGroup[];
  truncated?: boolean;
}

export interface RunStep {
  name: string;
  conclusion: Conclusion;

  // Optional richer metadata (available in Actions API mode).
  number?: number;
  status?: string;
  startedAt?: string;
  completedAt?: string;

  // Optional nested log structure parsed from job logs.
  log?: RunLogGroup;
}

export interface RunJob {
  name: string;
  conclusion: Conclusion;
  steps: RunStep[];

  // Optional AI-written summary of what this job did.
  summary?: string;

  // Optional richer metadata (available in Actions API mode).
  id?: number;
  status?: string;
  startedAt?: string;
  completedAt?: string;
  url?: string;

  // Optional full job log tree.
  log?: RunLogGroup;
}

export interface WorkflowRunSnapshot {
  workflowId: string;
  runUrl?: string;
  updatedAt: string;
  conclusion: Conclusion;
  jobs: RunJob[];

  // Optional run-level metadata (available in Actions API mode).
  runId?: number;
  runNumber?: number;
  runAttempt?: number;
  status?: string;
  event?: string;
  headBranch?: string;
  headSha?: string;
  createdAt?: string;
}

export interface HeroWorkflow {
  id: string;
  label: string;
  sourceMarkdown?: string;
  compiledYaml: string;
  snapshot?: WorkflowRunSnapshot;
}
