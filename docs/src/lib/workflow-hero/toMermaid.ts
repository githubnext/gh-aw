import type { WorkflowGraph } from './parseActionsYaml';
import type { Conclusion } from './types';

function sanitizeId(id: string): string {
  const cleaned = id.replace(/[^a-zA-Z0-9_]/g, '_');
  return cleaned.length === 0 ? 'job' : cleaned;
}

function getStatusPrefix(status: Conclusion | undefined): string {
  if (status === 'success') return '✓ ';
  if (status === 'failure') return '✗ ';
  return '';
}

export function workflowGraphToMermaid(
  graph: WorkflowGraph,
  jobConclusions?: Record<string, Conclusion | undefined>
): string {
  const idMap = new Map<string, string>();
  const used = new Set<string>();

  const getNodeId = (jobId: string): string => {
    const existing = idMap.get(jobId);
    if (existing) return existing;

    const base = sanitizeId(jobId);
    let candidate = base;
    let i = 2;
    while (used.has(candidate)) {
      candidate = `${base}_${i++}`;
    }
    used.add(candidate);
    idMap.set(jobId, candidate);
    return candidate;
  };

  const lines: string[] = [
    // Keep Mermaid's default flowchart renderer (non-ELK).
    "%%{init: { 'flowchart': { 'nodeSpacing': 40, 'rankSpacing': 60 } }}%%",
    'flowchart TD',
  ];

  // Declare nodes (with original job ids as labels)
  for (const jobId of Object.keys(graph.jobs)) {
    const nodeId = getNodeId(jobId);
    const statusPrefix = getStatusPrefix(jobConclusions?.[jobId]);
    const label = `${statusPrefix}${jobId}`.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
    lines.push(`  ${nodeId}["${label}"]`);
  }

  // Edges based on needs
  for (const [jobId, job] of Object.entries(graph.jobs)) {
    const to = getNodeId(jobId);
    for (const dep of job.needs) {
      if (!(dep in graph.jobs)) continue;
      const from = getNodeId(dep);
      lines.push(`  ${from} --> ${to}`);
    }
  }

  return lines.join('\n');
}
