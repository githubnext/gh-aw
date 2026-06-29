export interface WorkflowDefinition {
  workflow: string;
  engine_id?: string;
  compiled?: string;
  labels?: string[];
  status?: string;
  time_remaining?: string;
}

export interface WorkflowRun {
  run_id: number;
  workflow_name: string;
  status?: string;
  conclusion?: string;
  duration?: number;
  aic?: number;
  token_usage?: number;
  turns?: number;
  error_count?: number;
  warning_count?: number;
  missing_tool_count?: number;
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

export interface ExperimentInfo {
  workflow_id: string;
  branch: string;
  experiments: number;
  total_runs: number;
  last_run: string;
}

export interface UsageSummaryItem {
  workflow_id: string;
  workflow_name: string;
  run_count: number;
  total_aic: number;
  cost_per_run: number;
  daily_aic: number;
  monthly_forecast_aic: number;
  last_run_at?: string;
}

export interface CLIStatus {
  available: boolean;
  source: string;
  version: string;
  command: string;
  installCommand: string;
  installUrl?: string;
  message?: string;
}
