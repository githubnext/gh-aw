import type { WorkflowGraph } from './types.js';

function sanitizeId(raw: string): string {
  // Mermaid IDs should be simple. Keep alphanumerics and underscores.
  const cleaned = raw.replace(/[^a-zA-Z0-9_]/g, '_');
  return cleaned.length > 0 ? cleaned : 'job';
}

function escapeLabel(raw: string): string {
  // Mermaid labels in brackets are fairly permissive, but newlines can break diagrams.
  return raw.replace(/\r\n/g, '\n').replace(/\n/g, ' ');
}

export function workflowGraphToMermaid(graph: WorkflowGraph): string {
  const lines: string[] = ['flowchart TD'];

  // Stable, safe IDs even if job names contain dashes.
  const idMap = new Map<string, string>();
  for (const job of graph.jobs) {
    const safe = sanitizeId(job.id);
    // Handle collisions by appending an index.
    let candidate = safe;
    let suffix = 1;
    while ([...idMap.values()].includes(candidate)) {
      suffix += 1;
      candidate = `${safe}_${suffix}`;
    }
    idMap.set(job.id, candidate);
  }

  // Nodes
  for (const job of graph.jobs) {
    const nodeId = idMap.get(job.id)!;
    lines.push(`  ${nodeId}[${escapeLabel(job.id)}]`);
  }

  // Edges (needs -> job)
  for (const job of graph.jobs) {
    const targetId = idMap.get(job.id)!;
    for (const dep of job.needs) {
      const sourceId = idMap.get(dep);
      if (!sourceId) continue;
      lines.push(`  ${sourceId} --> ${targetId}`);
    }
  }

  return lines.join('\n');
}
