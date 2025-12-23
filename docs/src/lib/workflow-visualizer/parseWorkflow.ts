import YAML from 'yaml';

import type { ParseWorkflowResult, WorkflowGraph } from './types.js';

type YamlValue = unknown;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function asStringArray(value: unknown): string[] {
  if (typeof value === 'string') return [value];
  if (Array.isArray(value)) {
    return value.filter((v): v is string => typeof v === 'string');
  }
  return [];
}

function extractFrontmatterOrWholeDoc(input: string): string {
  // Supports gh-aw markdown workflows:
  // ---
  // <yaml>
  // ---
  // <markdown>
  const normalized = input.replace(/\r\n/g, '\n');
  if (!normalized.startsWith('---\n')) return input;

  const endIndex = normalized.indexOf('\n---\n', 4);
  if (endIndex === -1) return input;

  return normalized.slice(4, endIndex + 1);
}

function parseWorkflowObject(yamlText: string): YamlValue {
  // We intentionally parse YAML into JS values and then defensively read it.
  return YAML.parse(yamlText);
}

export function parseWorkflowGraph(input: string): ParseWorkflowResult {
  const yamlText = extractFrontmatterOrWholeDoc(input);

  let parsed: YamlValue;
  try {
    parsed = parseWorkflowObject(yamlText);
  } catch (error) {
    return { ok: false, error: `Failed to parse YAML: ${String(error)}` };
  }

  if (!isRecord(parsed)) {
    return { ok: false, error: 'Workflow must be a YAML mapping/object.' };
  }

  const jobsValue = parsed['jobs'];
  if (!isRecord(jobsValue)) {
    return { ok: false, error: 'No jobs found. Expected a top-level "jobs" mapping.' };
  }

  const jobs: WorkflowGraph['jobs'] = [];
  for (const [jobId, jobValue] of Object.entries(jobsValue)) {
    if (!isRecord(jobValue)) continue;
    const needs = asStringArray(jobValue['needs']);
    jobs.push({ id: jobId, needs });
  }

  if (jobs.length === 0) {
    return { ok: false, error: 'No jobs found. The "jobs" mapping was empty or invalid.' };
  }

  return { ok: true, graph: { jobs } };
}
