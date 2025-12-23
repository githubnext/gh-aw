import mermaid from 'mermaid';

type Conclusion =
  | 'success'
  | 'failure'
  | 'cancelled'
  | 'skipped'
  | 'neutral'
  | 'timed_out'
  | 'action_required'
  | 'stale'
  | null;

type RunLogGroup = {
  title: string;
  lines?: string[];
  omittedLineCount?: number;
  children?: RunLogGroup[];
  truncated?: boolean;
};

type RunStep = {
  name: string;
  conclusion: Conclusion;
  number?: number;
  status?: string;
  startedAt?: string;
  completedAt?: string;
  log?: RunLogGroup;
};

type RunJob = {
  name: string;
  conclusion: Conclusion;
  steps: RunStep[];
  summary?: string;
  id?: number;
  status?: string;
  startedAt?: string;
  completedAt?: string;
  url?: string;
  log?: RunLogGroup;
};

type WorkflowRunSnapshot = {
  workflowId: string;
  runUrl?: string;
  updatedAt: string;
  conclusion: Conclusion;
  jobs: RunJob[];
  runId?: number;
  runNumber?: number;
  runAttempt?: number;
  status?: string;
  event?: string;
  headBranch?: string;
  headSha?: string;
  createdAt?: string;
};

type HeroWorkflowClient = {
  id: string;
  label: string;
  sourceMarkdown?: string;
  compiledYaml: string;
  snapshot?: WorkflowRunSnapshot;
  mermaidSource?: string;
  mermaidError?: string;
};

type LineRange = { start: number; end: number };

type SelectionRanges = {
  toolsRange: LineRange | null;
  safeOutputsRange: LineRange | null;
  jobRanges: Map<string, LineRange | null>;
};

function normalizeName(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '');
}

function formatRunConclusion(snapshot: WorkflowRunSnapshot): string {
  if (snapshot.conclusion) return snapshot.conclusion;

  const jobs = snapshot.jobs || [];
  if (jobs.length === 0) return 'unknown';

  const conclusions = jobs.map((j) => j.conclusion);

  // If any job has a terminal failure-ish state, treat the run as failed.
  if (conclusions.some((c) => c === 'failure' || c === 'timed_out' || c === 'action_required')) return 'failure';
  if (conclusions.some((c) => c === 'cancelled')) return 'cancelled';

  // If some jobs are still running / missing conclusion, call it in progress.
  if (conclusions.some((c) => c === null)) return 'in progress';

  if (conclusions.every((c) => c === 'success')) return 'success';

  return 'unknown';
}

function parseNodeJobIdFromMermaidNode(nodeEl: Element): string {
  const label = nodeEl.querySelector('.label');
  const text = (label?.textContent || '').trim();
  // Labels are like: "✓ activation" or "✗ safe_outputs" or just "activation".
  return text.replace(/^([✓✗]\s*)/, '').trim();
}

function findJobBlockLineRange(sourceMarkdown: string | undefined, jobId: string): LineRange | null {
  const text = sourceMarkdown ?? '';
  const normalizedJobId = normalizeName(jobId);
  if (!text.trim()) return null;

  const lines = text.replaceAll('\r\n', '\n').split('\n');

  // Find YAML frontmatter region (best-effort).
  let fmStart = -1;
  let fmEnd = -1;
  if (lines[0]?.trim() === '---') {
    fmStart = 1;
    for (let i = 1; i < lines.length; i++) {
      if (lines[i]?.trim() === '---') {
        fmEnd = i;
        break;
      }
    }
  }

  const regionStart = fmStart >= 0 && fmEnd > fmStart ? fmStart : 0;
  const regionEnd = fmStart >= 0 && fmEnd > fmStart ? fmEnd : lines.length;

  // Locate "jobs:".
  let jobsLine = -1;
  for (let i = regionStart; i < regionEnd; i++) {
    if (/^\s*jobs\s*:\s*$/.test(lines[i] ?? '')) {
      jobsLine = i;
      break;
    }
  }
  if (jobsLine < 0) return null;

  // Find the job key under jobs.
  let jobStart = -1;
  let jobIndent = 0;
  for (let i = jobsLine + 1; i < regionEnd; i++) {
    const line = lines[i] ?? '';
    if (!line.trim()) continue;

    // Stop if we dedent back to root (end of jobs section).
    if (/^\S/.test(line)) break;

    const m = line.match(/^(\s+)([^\s:#]+)\s*:\s*$/);
    if (!m) continue;

    const key = m[2] ?? '';
    if (normalizeName(key) !== normalizedJobId) continue;

    jobStart = i;
    jobIndent = m[1].length;
    break;
  }
  if (jobStart < 0) return null;

  // Extend until next sibling job key at same indent, or end of frontmatter/jobs.
  let jobEnd = regionEnd - 1;
  for (let i = jobStart + 1; i < regionEnd; i++) {
    const line = lines[i] ?? '';

    // End if we dedent to root.
    if (/^\S/.test(line)) {
      jobEnd = i - 1;
      break;
    }

    const m = line.match(/^(\s+)([^\s:#]+)\s*:\s*$/);
    if (!m) continue;
    const indent = m[1].length;
    if (indent === jobIndent) {
      jobEnd = i - 1;
      break;
    }
  }

  // Convert to 1-based inclusive range.
  return { start: jobStart + 1, end: Math.max(jobStart + 1, jobEnd + 1) };
}

function findFrontmatterKeyBlockLineRange(
  sourceMarkdown: string | undefined,
  keys: string[]
): LineRange | null {
  const text = sourceMarkdown ?? '';
  if (!text.trim()) return null;

  const lines = text.replaceAll('\r\n', '\n').split('\n');

  // Find YAML frontmatter region.
  let fmStart = -1;
  let fmEnd = -1;
  if (lines[0]?.trim() === '---') {
    fmStart = 1;
    for (let i = 1; i < lines.length; i++) {
      if (lines[i]?.trim() === '---') {
        fmEnd = i;
        break;
      }
    }
  }
  if (!(fmStart >= 0 && fmEnd > fmStart)) return null;

  const normalizedKeys = new Set(keys.map((k) => normalizeName(k)));

  // Find key line.
  let keyStart = -1;
  let keyIndent = 0;
  for (let i = fmStart; i < fmEnd; i++) {
    const line = lines[i] ?? '';
    if (!line.trim()) continue;
    const m = line.match(/^(\s*)([^\s:#]+)\s*:\s*$/);
    if (!m) continue;
    const key = m[2] ?? '';
    if (!normalizedKeys.has(normalizeName(key))) continue;
    keyStart = i;
    keyIndent = (m[1] ?? '').length;
    break;
  }
  if (keyStart < 0) return null;

  // Extend to next sibling key at same indentation or end of frontmatter.
  let keyEnd = fmEnd - 1;
  for (let i = keyStart + 1; i < fmEnd; i++) {
    const line = lines[i] ?? '';
    if (!line.trim()) continue;
    const indent = (line.match(/^(\s*)/)?.[1] ?? '').length;
    if (indent <= keyIndent) {
      keyEnd = i - 1;
      break;
    }
  }

  return { start: keyStart + 1, end: Math.max(keyStart + 1, keyEnd + 1) };
}

function clearCodeHighlights(codeContainer: HTMLElement) {
  const lines = codeContainer.querySelectorAll<HTMLElement>('.ec-line.is-active');
  for (const line of lines) line.classList.remove('is-active');
}

function highlightCodeRanges(codeContainer: HTMLElement, ranges: Array<LineRange | null>) {
  clearCodeHighlights(codeContainer);

  const normalizedRanges = ranges.filter((r): r is LineRange => !!r);
  if (normalizedRanges.length === 0) return;

  // Expressive Code renders each line as .ec-line in order.
  const lineEls = codeContainer.querySelectorAll<HTMLElement>('.ec-line');

  for (const range of normalizedRanges) {
    const startIdx = Math.max(0, range.start - 1);
    const endIdx = Math.min(lineEls.length - 1, range.end - 1);
    for (let i = startIdx; i <= endIdx; i++) {
      lineEls[i]?.classList.add('is-active');
    }
  }

  const first = lineEls[Math.max(0, normalizedRanges[0]!.start - 1)];
  first?.scrollIntoView({ block: 'center', inline: 'nearest' });
}

function clearGraphHighlights(graphCanvas: HTMLElement) {
  const active = graphCanvas.querySelectorAll('.node.is-active');
  for (const el of Array.from(active)) el.classList.remove('is-active');
}

function highlightGraphNode(graphCanvas: HTMLElement, jobId: string | null) {
  clearGraphHighlights(graphCanvas);
  if (!jobId) return;
  const target = normalizeName(jobId);
  const nodes = graphCanvas.querySelectorAll<SVGGElement>('.node');
  for (const node of Array.from(nodes)) {
    const labelId = parseNodeJobIdFromMermaidNode(node);
    if (normalizeName(labelId) === target) {
      node.classList.add('is-active');
      break;
    }
  }
}

function getGraphJobIds(graphCanvas: HTMLElement): string[] {
  const ids: string[] = [];
  const nodes = graphCanvas.querySelectorAll<SVGGElement>('.node');
  for (const node of Array.from(nodes)) {
    const labelId = parseNodeJobIdFromMermaidNode(node);
    if (!labelId) continue;
    if (!ids.some((x) => normalizeName(x) === normalizeName(labelId))) {
      ids.push(labelId);
    }
  }
  return ids;
}

function escapeHtml(text: unknown): string {
  return String(text)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

function renderStatusDot(conclusion: Conclusion | 'in progress' | 'unknown'): string {
  const key = conclusion ?? 'unknown';
  return `<span class="status-dot" data-conclusion="${escapeHtml(key)}" aria-hidden="true"></span>`;
}

function formatDuration(startedAt?: string, completedAt?: string): string | null {
  if (!startedAt || !completedAt) return null;
  const start = Date.parse(startedAt);
  const end = Date.parse(completedAt);
  if (!Number.isFinite(start) || !Number.isFinite(end)) return null;
  const ms = end - start;
  if (!Number.isFinite(ms) || ms < 0) return null;

  const totalSeconds = Math.round(ms / 1000);
  if (totalSeconds < 60) return `${totalSeconds}s`;

  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes < 60) return `${minutes}m ${seconds.toString().padStart(2, '0')}s`;

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes.toString().padStart(2, '0')}m`;
}

function renderStepMeta(step: RunStep): string {
  const details: string[] = [];
  if (typeof step.number === 'number') details.push(`#${step.number}`);
  if (step.status) details.push(step.status);
  const duration = formatDuration(step.startedAt, step.completedAt);
  if (duration) details.push(duration);

  const detailsHtml =
    details.length > 0
      ? `<span class="step-details">${details.map((d) => escapeHtml(d)).join(' • ')}</span>`
      : '';

  const conclusionKey = step.conclusion ?? 'unknown';
  const pillHtml = `<span class="pill" data-conclusion="${escapeHtml(conclusionKey)}">${escapeHtml(conclusionKey)}</span>`;
  return `<span class="step-meta">${detailsHtml}${pillHtml}</span>`;
}

function renderLogGroup(group: RunLogGroup, depth: number = 0): string {
  const title = group?.title ? String(group.title) : 'Log group';
  const lines = Array.isArray(group?.lines) ? group.lines : [];
  const omitted = typeof group?.omittedLineCount === 'number' ? group.omittedLineCount : 0;
  const children = Array.isArray(group?.children) ? group.children : [];
  const truncated = group?.truncated === true;

  const suffixParts: string[] = [];
  if (lines.length > 0) suffixParts.push(`${lines.length} line(s)`);
  if (omitted > 0) suffixParts.push(`${omitted} omitted`);
  if (truncated) suffixParts.push('truncated');
  const suffix = suffixParts.length > 0 ? ` <span class="log-group-meta">(${escapeHtml(suffixParts.join(' • '))})</span>` : '';

  const bodyParts: string[] = [];
  if (lines.length > 0) {
    bodyParts.push(`<pre class="log-lines">${escapeHtml(lines.join('\n'))}</pre>`);
  }
  if (omitted > 0) {
    bodyParts.push(`<div class="log-omitted">${escapeHtml(`… ${omitted} line(s) omitted …`)}</div>`);
  }
  if (children.length > 0) {
    for (const child of children) {
      bodyParts.push(renderLogGroup(child, depth + 1));
    }
  }

  // Keep everything collapsed by default; allow deep exploration.
  return [
    `<details class="log-group" data-depth="${escapeHtml(String(depth))}">`,
    `<summary class="log-group-summary">${escapeHtml(title)}${suffix}</summary>`,
    `<div class="log-group-body">${bodyParts.join('')}</div>`,
    `</details>`,
  ].join('');
}

function renderStep(step: RunStep): string {
  const parts: string[] = [];
  const hasLog = !!step.log;

  // If we have logs, render a step details disclosure. Otherwise keep it as a flat row.
  if (hasLog) {
    parts.push('<li class="step-item">');
    parts.push('<details class="step-details">');
    parts.push('<summary class="step">');
    parts.push(renderStatusDot(step.conclusion ?? 'unknown'));
    parts.push(`<span class="step-name">${escapeHtml(step.name)}</span>`);
    parts.push(renderStepMeta(step));
    parts.push('</summary>');
    parts.push('<div class="step-log">');
    parts.push(renderLogGroup(step.log as RunLogGroup, 0));
    parts.push('</div>');
    parts.push('</details>');
    parts.push('</li>');
    return parts.join('');
  }

  parts.push(`<li class="step">`);
  parts.push(renderStatusDot(step.conclusion ?? 'unknown'));
  parts.push(`<span class="step-name">${escapeHtml(step.name)}</span>`);
  parts.push(renderStepMeta(step));
  parts.push(`</li>`);
  return parts.join('');
}

function renderJobs(container: Element, snapshot: WorkflowRunSnapshot | undefined, selectedJobId?: string | null) {
  if (!snapshot) {
    container.textContent = '';
    return;
  }

  const jobs = snapshot.jobs || [];
  const parts: string[] = [];

  for (const job of jobs) {
    const isOpen =
      typeof selectedJobId === 'string' && selectedJobId.length > 0
        ? normalizeName(job.name) === normalizeName(selectedJobId)
        : false;

    parts.push(`<details class="job"${isOpen ? ' open' : ''}>`);
    parts.push(`<summary class="job-summary">`);
    parts.push(renderStatusDot(job.conclusion ?? 'unknown'));
    parts.push(`<span class="job-name">${escapeHtml(job.name)}</span>`);
    parts.push(' ');
    parts.push(
      `<span class="job-meta"><span class="pill" data-conclusion="${escapeHtml(job.conclusion ?? 'unknown')}">${escapeHtml(job.conclusion ?? 'unknown')}</span></span>`
    );
    parts.push(`</summary>`);

    if (typeof job.summary === 'string' && job.summary.trim().length > 0) {
      parts.push(`<div class="job-ai-summary">${escapeHtml(job.summary.trim())}</div>`);
    }

    if (job.steps && job.steps.length > 0) {
      parts.push('<ul class="steps">');
      for (const step of job.steps) {
        parts.push(renderStep(step));
      }
      parts.push('</ul>');
    }

    if (job.log) {
      parts.push('<div class="job-log">');
      parts.push(renderLogGroup(job.log, 0));
      parts.push('</div>');
    }

    parts.push('</details>');
  }

  container.innerHTML = parts.join('');
}

function renderSelectedJob(container: HTMLElement, snapshot: WorkflowRunSnapshot | undefined, jobId: string | null) {
  if (!snapshot || !jobId) {
    container.hidden = true;
    container.innerHTML = '';
    return;
  }

  const match = snapshot.jobs?.find((j) => normalizeName(j.name) === normalizeName(jobId));
  if (!match) {
    container.hidden = true;
    container.innerHTML = '';
    return;
  }

  const parts: string[] = [];
  parts.push(`<details class="job" open>`);
  parts.push(`<summary class="job-summary">`);
  parts.push(renderStatusDot(match.conclusion ?? 'unknown'));
  parts.push(`<span class="job-name">${escapeHtml(match.name)}</span>`);
  parts.push(' ');
  parts.push(
    `<span class="job-meta"><span class="pill" data-conclusion="${escapeHtml(match.conclusion ?? 'unknown')}">${escapeHtml(match.conclusion ?? 'unknown')}</span></span>`
  );
  parts.push(`</summary>`);

  if (typeof match.summary === 'string' && match.summary.trim().length > 0) {
    parts.push(`<div class="job-ai-summary">${escapeHtml(match.summary.trim())}</div>`);
  }

  parts.push('<ul class="steps">');
  for (const step of match.steps || []) {
    parts.push(renderStep(step));
  }
  parts.push('</ul>');

  if (match.log) {
    parts.push('<div class="job-log">');
    parts.push(renderLogGroup(match.log, 0));
    parts.push('</div>');
  }
  parts.push('</details>');

  container.hidden = false;
  container.innerHTML = parts.join('');
}

async function renderGraph(
  canvas: Element,
  errorEl: HTMLElement,
  mermaidSource?: string,
  mermaidError?: string
) {
  errorEl.hidden = true;
  errorEl.textContent = '';
  canvas.innerHTML = '';

  try {
    if (!mermaidSource) {
      throw new Error(mermaidError || 'Unable to build workflow graph');
    }

    const id = `m-${Math.random().toString(16).slice(2)}`;
    const { svg } = await mermaid.render(id, mermaidSource);
    canvas.innerHTML = svg;
  } catch (err: any) {
    errorEl.hidden = false;
    errorEl.textContent = err?.message ? String(err.message) : String(err);
  }
}

function renderRun(
  linkEl: HTMLAnchorElement,
  metaEl: HTMLElement,
  selectedEl: HTMLElement,
  jobsEl: Element,
  snapshot?: WorkflowRunSnapshot,
  selectedJobId?: string | null
) {
  if (!snapshot) {
    linkEl.hidden = true;
    metaEl.textContent = 'No recent run snapshot available.';
    selectedEl.hidden = true;
    selectedEl.innerHTML = '';
    jobsEl.textContent = '';
    return;
  }

  if (snapshot.runUrl) {
    linkEl.hidden = false;
    linkEl.href = snapshot.runUrl;
  } else {
    linkEl.hidden = true;
  }

  const updatedAt = snapshot.updatedAt ? new Date(snapshot.updatedAt).toLocaleString() : 'unknown time';
  metaEl.textContent = `${formatRunConclusion(snapshot)} • updated ${updatedAt}`;
  renderSelectedJob(selectedEl, snapshot, selectedJobId ?? null);
  renderJobs(jobsEl, snapshot, selectedJobId ?? null);
}

type HeroPayload = {
  workflows: HeroWorkflowClient[];
  initialId?: string;
};

function parsePayload(root: HTMLElement): HeroPayload {
  const payloadEl = root.querySelector<HTMLScriptElement>('script[data-hero-payload][type="application/json"]');
  const raw = payloadEl?.textContent ?? '';
  if (!raw.trim()) throw new Error('Hero playground missing JSON payload');

  const parsed = JSON.parse(raw) as unknown;
  if (!parsed || typeof parsed !== 'object') throw new Error('Hero playground payload must be an object');

  const workflows = (parsed as any).workflows as unknown;
  if (!Array.isArray(workflows)) throw new Error('Hero playground payload.workflows must be an array');

  const initialId = (parsed as any).initialId as unknown;
  return {
    workflows: workflows as HeroWorkflowClient[],
    initialId: typeof initialId === 'string' ? initialId : undefined,
  };
}

function getMermaidTheme(): 'default' | 'dark' {
  // Starlight sets this attribute via ThemeToggle.
  const theme = document.documentElement.getAttribute('data-theme');
  return theme === 'dark' ? 'dark' : 'default';
}

function init() {
  try {
    const root = document.querySelector<HTMLElement>('[data-hero-playground]');
    if (!root) return;

    const payload = parsePayload(root);
    const heroWorkflows = payload.workflows;
    const initialId = payload.initialId;
    let activeId = initialId && heroWorkflows.some((w) => w.id === initialId) ? initialId : heroWorkflows[0]?.id;
    if (!activeId) return;

    const select = root.querySelector<HTMLSelectElement>('[data-hero-select]');
    const codeContainer = root.querySelector<HTMLElement>('[data-hero-code]');
    const graphCanvas = root.querySelector<HTMLElement>('[data-hero-graph-canvas]');
    const graphError = root.querySelector<HTMLElement>('[data-hero-graph-error]');
    const graphPane = root.querySelector<HTMLElement>('.hero-pane-graph');
    const runPane = root.querySelector<HTMLElement>('.hero-pane-run');
    const runLink = root.querySelector<HTMLAnchorElement>('[data-hero-run-link]');
    const runMeta = root.querySelector<HTMLElement>('[data-hero-run-meta]');
    const runSelected = root.querySelector<HTMLElement>('[data-hero-run-selected]');
    const runJobs = root.querySelector<HTMLElement>('[data-hero-run-jobs]');

    if (!codeContainer || !graphCanvas || !graphError || !runLink || !runMeta || !runSelected || !runJobs) {
      throw new Error('Hero playground missing required elements');
    }

    // Create non-null aliases for use in closures.
    const codeEl = codeContainer;
    const graphCanvasEl = graphCanvas;
    const graphErrorEl = graphError;
    const runLinkEl = runLink;
    const runMetaEl = runMeta;
    const runSelectedEl = runSelected;
    const runJobsEl = runJobs;

    let mermaidTheme = getMermaidTheme();
    // Keep Mermaid defaults (no ELK / orthogonal routing).
    mermaid.initialize({
      startOnLoad: false,
      theme: mermaidTheme,
      flowchart: {
      },
    });

    const desktopLayoutQuery = window.matchMedia('(min-width: 900px)');

    const selectionRangesByWorkflow = new Map<string, SelectionRanges>();

    function getOrBuildSelectionRanges(wf: HeroWorkflowClient): SelectionRanges {
      const cached = selectionRangesByWorkflow.get(wf.id);
      if (cached) return cached;

      const ranges: SelectionRanges = {
        toolsRange: findFrontmatterKeyBlockLineRange(wf.sourceMarkdown, ['tools']),
        safeOutputsRange: findFrontmatterKeyBlockLineRange(wf.sourceMarkdown, ['safe_outputs', 'safe-outputs']),
        jobRanges: new Map<string, LineRange | null>(),
      };
      selectionRangesByWorkflow.set(wf.id, ranges);
      return ranges;
    }

    function buildJobRangesIfNeeded(wf: HeroWorkflowClient, ranges: SelectionRanges) {
      const graphJobIds = getGraphJobIds(graphCanvasEl);
      for (const id of graphJobIds) {
        const key = normalizeName(id);
        if (ranges.jobRanges.has(key)) continue;
        ranges.jobRanges.set(key, findJobBlockLineRange(wf.sourceMarkdown, id));
      }
    }

    function selectionForLine(wf: HeroWorkflowClient, lineNumber: number): string | null {
      const ranges = getOrBuildSelectionRanges(wf);
      if (ranges.toolsRange && lineNumber >= ranges.toolsRange.start && lineNumber <= ranges.toolsRange.end) {
        return 'agent';
      }
      if (
        ranges.safeOutputsRange &&
        lineNumber >= ranges.safeOutputsRange.start &&
        lineNumber <= ranges.safeOutputsRange.end
      ) {
        return 'safe_outputs';
      }

      buildJobRangesIfNeeded(wf, ranges);
      for (const [jobId, range] of ranges.jobRanges.entries()) {
        if (!range) continue;
        if (lineNumber >= range.start && lineNumber <= range.end) {
          return jobId;
        }
      }

      return null;
    }

    function syncRunPaneHeightToGraph() {
      if (!graphPane || !runPane) return;

      // Only force equal heights when the layout is side-by-side.
      if (!desktopLayoutQuery.matches) {
        runPane.style.height = '';
        return;
      }

      const rect = graphPane.getBoundingClientRect();
      if (!rect.height || rect.height < 1) return;

      // Lock the run pane height to the graph pane height so the run content
      // becomes scrollable instead of stretching the entire row.
      runPane.style.height = `${Math.round(rect.height)}px`;
    }

    const graphResizeObserver =
      graphPane && runPane
        ? new ResizeObserver(() => {
            syncRunPaneHeightToGraph();
          })
        : null;

    graphResizeObserver?.observe(graphPane as Element);
    desktopLayoutQuery.addEventListener('change', syncRunPaneHeightToGraph);

    async function setActive(id: string) {
      const wf = heroWorkflows.find((w) => w.id === id);
      if (!wf) return;
      activeId = id;

      // Clear any existing selection when switching workflows.
      selectedJobId = null;

      const blocks = codeEl.querySelectorAll<HTMLElement>('[data-hero-code-block]');
      for (const block of blocks) {
        const blockId = block.getAttribute('data-hero-id');
        block.hidden = blockId !== id;
      }

      const activeBlock = codeEl.querySelector<HTMLElement>(
        `[data-hero-code-block][data-hero-id="${CSS.escape(id)}"]`
      );
      if (activeBlock) {
        clearCodeHighlights(activeBlock);
      }

      await renderGraph(graphCanvasEl, graphErrorEl, wf.mermaidSource, wf.mermaidError);
      highlightGraphNode(graphCanvasEl, null);
      renderRun(runLinkEl, runMetaEl, runSelectedEl, runJobsEl, wf.snapshot, null);

      // Precompute selection ranges for this workflow.
      getOrBuildSelectionRanges(wf);

      // Ensure the run pane doesn't push the graph row taller.
      syncRunPaneHeightToGraph();
    }

    let selectedJobId: string | null = null;

    function applySelection(wf: HeroWorkflowClient, jobId: string | null) {
      selectedJobId = jobId;

      highlightGraphNode(graphCanvasEl, jobId);

      const activeBlock = codeEl.querySelector<HTMLElement>(
        `[data-hero-code-block][data-hero-id="${CSS.escape(wf.id)}"]`
      );
      if (activeBlock) {
        const ranges = getOrBuildSelectionRanges(wf);
        buildJobRangesIfNeeded(wf, ranges);

        const normalized = jobId ? normalizeName(jobId) : '';
        const selectionRanges: Array<LineRange | null> = [];

        if (jobId) {
          selectionRanges.push(ranges.jobRanges.get(normalized) ?? findJobBlockLineRange(wf.sourceMarkdown, jobId));
        }

        // When selecting the agent node, also highlight the tools section.
        if (normalized === 'agent' && ranges.toolsRange) {
          selectionRanges.push(ranges.toolsRange);
        }

        // When selecting safe_outputs, also highlight the safe_outputs frontmatter block.
        if (normalized === 'safeoutputs' && ranges.safeOutputsRange) {
          selectionRanges.push(ranges.safeOutputsRange);
        }

        highlightCodeRanges(activeBlock, selectionRanges);
      }

      renderRun(runLinkEl, runMetaEl, runSelectedEl, runJobsEl, wf.snapshot, jobId);
      syncRunPaneHeightToGraph();
    }

    // Graph interactions: click a node to highlight it + highlight matching code + show job outputs.
    graphCanvasEl.addEventListener('click', (e) => {
      const wf = heroWorkflows.find((w) => w.id === activeId);
      if (!wf) return;

      const target = e.target as Element | null;
      const node = target?.closest?.('.node');
      if (!node) {
        applySelection(wf, null);
        return;
      }

      const clickedJobId = parseNodeJobIdFromMermaidNode(node);
      if (!clickedJobId) {
        applySelection(wf, null);
        return;
      }

      // Toggle selection.
      const next = selectedJobId && normalizeName(selectedJobId) === normalizeName(clickedJobId) ? null : clickedJobId;
      applySelection(wf, next);
    });

    // Code interactions: click a frontmatter section (tools/safe_outputs) or job block to select the matching node.
    codeEl.addEventListener('click', (e) => {
      const wf = heroWorkflows.find((w) => w.id === activeId);
      if (!wf) return;

      const target = e.target as Element | null;
      const block = target?.closest?.('[data-hero-code-block][data-hero-id]') as HTMLElement | null;
      if (!block) return;

      const blockId = block.getAttribute('data-hero-id');
      if (blockId !== wf.id) return;

      const lineEl = target?.closest?.('.ec-line') as HTMLElement | null;
      if (!lineEl) return;

      const lineEls = block.querySelectorAll<HTMLElement>('.ec-line');
      const idx = Array.from(lineEls).indexOf(lineEl);
      if (idx < 0) return;

      const lineNumber = idx + 1;
      const next = selectionForLine(wf, lineNumber);
      applySelection(wf, next);
    });

    // Keep the graph readable when the site theme changes.
    const themeObserver = new MutationObserver(() => {
      const nextTheme = getMermaidTheme();
      if (nextTheme === mermaidTheme) return;
      mermaidTheme = nextTheme;
      mermaid.initialize({ startOnLoad: false, theme: mermaidTheme });
      void setActive(activeId);
    });

    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme'],
    });

    select?.addEventListener('change', (e) => {
      const next = (e.target as HTMLSelectElement | null)?.value;
      if (typeof next === 'string') setActive(next);
    });

    void setActive(activeId);
  } catch (err: any) {
    // Surface unexpected errors in the UI so failures aren't silent.
    const message = err?.message ? String(err.message) : String(err);
    console.error('WorkflowHeroPlayground init failed:', err);

    const root = document.querySelector<HTMLElement>('[data-hero-playground]');
    const graphError = root?.querySelector<HTMLElement>('[data-hero-graph-error]');
    if (graphError) {
      graphError.hidden = false;
      graphError.textContent = message;
    }
  }
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
