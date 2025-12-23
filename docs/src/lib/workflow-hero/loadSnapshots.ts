import type { WorkflowRunSnapshot } from './types';

export function loadPlaygroundSnapshots(): Record<string, WorkflowRunSnapshot> {
  const modules = import.meta.glob('../../assets/playground-snapshots/*.json', {
    eager: true,
  });

  const snapshots: Record<string, WorkflowRunSnapshot> = {};

  for (const [path, mod] of Object.entries(modules)) {
    const anyMod = mod as any;
    const snapshot = (anyMod?.default ?? anyMod) as WorkflowRunSnapshot;

    const filename = path.split('/').pop() ?? '';
    const idFromFilename = filename.endsWith('.json') ? filename.slice(0, -'.json'.length) : filename;
    const id = snapshot?.workflowId || idFromFilename;

    if (id) {
      snapshots[id] = snapshot;
    }
  }

  return snapshots;
}
