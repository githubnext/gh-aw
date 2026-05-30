import YAML from 'yaml';

export interface WorkflowGraph {
  jobs: Record<string, { needs: string[] }>;
}

function toStringArray(value: unknown): string[] {
  if (typeof value === 'string') return [value];
  if (Array.isArray(value)) {
    return value.filter((v): v is string => typeof v === 'string');
  }
  return [];
}

export function parseActionsWorkflowGraph(yamlText: string): WorkflowGraph {
  const doc = YAML.parse(yamlText) as unknown;
  if (!doc || typeof doc !== 'object') {
    throw new Error('Invalid YAML');
  }

  const jobs = (doc as any).jobs as unknown;
  if (!jobs || typeof jobs !== 'object' || Array.isArray(jobs)) {
    throw new Error('No jobs found');
  }

  const graph: WorkflowGraph = { jobs: {} };
  for (const [jobId, jobConfig] of Object.entries(jobs as Record<string, unknown>)) {
    if (!jobConfig || typeof jobConfig !== 'object') continue;
    const needs = toStringArray((jobConfig as any).needs);
    graph.jobs[jobId] = { needs };
  }

  if (Object.keys(graph.jobs).length === 0) {
    throw new Error('No jobs found');
  }

  return graph;
}
