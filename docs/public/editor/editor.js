// ================================================================
// gh-aw Playground - Application Logic
// ================================================================

import { EditorView, basicSetup } from 'https://esm.sh/codemirror@6.0.2';
import { EditorState, Compartment } from 'https://esm.sh/@codemirror/state@6.5.4';
import { keymap } from 'https://esm.sh/@codemirror/view@6.39.14';
import { yaml } from 'https://esm.sh/@codemirror/lang-yaml@6.1.2';
import { markdown } from 'https://esm.sh/@codemirror/lang-markdown@6.5.0';
import { indentUnit } from 'https://esm.sh/@codemirror/language@6.12.1';
import { oneDark } from 'https://esm.sh/@codemirror/theme-one-dark@6.1.3';
import { createWorkerCompiler } from '/gh-aw/wasm/compiler-loader.js';

// ---------------------------------------------------------------
// Sample workflow registry (fetched from GitHub on demand)
// ---------------------------------------------------------------
const AGENTICS_RAW = 'https://raw.githubusercontent.com/githubnext/agentics/main/workflows';

const SAMPLES = {
  'hello-world': {
    label: 'Hello World',
    content: `---
name: hello-world
description: A simple hello world workflow
on:
  workflow_dispatch:
engine: copilot
---

# Mission

Say hello to the world! Check the current date and time, and greet the user warmly.
`,
  },
  'issue-triage': {
    label: 'Issue Triage',
    url: `${AGENTICS_RAW}/issue-triage.md`,
  },
  'ci-doctor': {
    label: 'CI Doctor',
    url: `${AGENTICS_RAW}/ci-doctor.md`,
  },
  'contribution-check': {
    label: 'Contribution Guidelines Checker',
    url: `${AGENTICS_RAW}/contribution-guidelines-checker.md`,
  },
  'daily-repo-status': {
    label: 'Daily Repo Status',
    url: `${AGENTICS_RAW}/daily-repo-status.md`,
  },
};

// Cache for fetched content (keyed by URL)
const contentCache = new Map();

const DEFAULT_CONTENT = SAMPLES['hello-world'].content;

// ---------------------------------------------------------------
// GitHub URL helpers
// ---------------------------------------------------------------

/** Convert github.com blob/tree URLs to raw.githubusercontent.com */
function toRawGitHubUrl(url) {
  // https://github.com/{owner}/{repo}/blob/{ref}/{path}
  const blobMatch = url.match(
    /^https?:\/\/github\.com\/([^/]+)\/([^/]+)\/blob\/([^/]+)\/(.+)$/
  );
  if (blobMatch) {
    const [, owner, repo, ref, path] = blobMatch;
    return `https://raw.githubusercontent.com/${owner}/${repo}/${ref}/${path}`;
  }
  return url;
}

/** Fetch markdown content from a URL (with cache) */
async function fetchContent(url) {
  const rawUrl = toRawGitHubUrl(url);
  if (contentCache.has(rawUrl)) return contentCache.get(rawUrl);
  const resp = await fetch(rawUrl);
  if (!resp.ok) throw new Error(`Failed to fetch ${rawUrl}: ${resp.status}`);
  const text = await resp.text();
  contentCache.set(rawUrl, text);
  return text;
}

// ---------------------------------------------------------------
// Hash-based deep linking
//
// Supported formats:
//   #hello-world              — built-in sample key
//   #issue-triage             — built-in sample key
//   #https://raw.github...    — arbitrary raw URL
//   #https://github.com/o/r/blob/main/file.md — auto-converted
// ---------------------------------------------------------------

function getHashValue() {
  const h = location.hash.slice(1); // strip leading #
  return decodeURIComponent(h).trim();
}

function setHashQuietly(value) {
  // Replace state so we don't spam the history
  history.replaceState(null, '', '#' + encodeURIComponent(value));
}

// ---------------------------------------------------------------
// DOM Elements
// ---------------------------------------------------------------
const $ = (id) => document.getElementById(id);

const sampleSelect = $('sampleSelect');
const editorMount = $('editorMount');
const outputPlaceholder = $('outputPlaceholder');
const outputMount = $('outputMount');
const outputContainer = $('outputContainer');
const statusBadge = $('statusBadge');
const statusText = $('statusText');
const statusDot = $('statusDot');
const loadingOverlay = $('loadingOverlay');
const errorBanner = $('errorBanner');
const errorText = $('errorText');
const warningBanner = $('warningBanner');
const warningText = $('warningText');
const divider = $('divider');
const panelEditor = $('panelEditor');
const panelOutput = $('panelOutput');
const panels = $('panels');
const tabBar = $('tabBar');
const tabStatusDot = $('tabStatusDot');
const fabCompile = $('fabCompile');

// ---------------------------------------------------------------
// State
// ---------------------------------------------------------------
const STORAGE_KEY = 'gh-aw-playground-content';
let compiler = null;
let isReady = false;
let isCompiling = false;
let compileTimer = null;
let currentYaml = '';
let pendingCompile = false;
let activeTab = 'editor';       // 'editor' | 'output'
let outputIsStale = false;      // true when editor changed since last compile
let lastCompileStatus = 'ok';   // 'ok' | 'error'
let isDragging = false;         // divider drag state (used by both divider + swipe logic)

// ---------------------------------------------------------------
// Theme — follows browser's prefers-color-scheme automatically.
// Primer CSS handles the page via data-color-mode="auto".
// We only need to toggle the CodeMirror theme (oneDark vs default).
// ---------------------------------------------------------------
const editorThemeConfig = new Compartment();
const outputThemeConfig = new Compartment();
const darkMq = window.matchMedia('(prefers-color-scheme: dark)');

function isDark() {
  return darkMq.matches;
}

function cmThemeFor(dark) {
  return dark ? oneDark : [];
}

function applyCmTheme() {
  const theme = cmThemeFor(isDark());
  editorView.dispatch({ effects: editorThemeConfig.reconfigure(theme) });
  outputView.dispatch({ effects: outputThemeConfig.reconfigure(theme) });
}

// ---------------------------------------------------------------
// CodeMirror: Input Editor (Markdown with YAML frontmatter)
// ---------------------------------------------------------------
const savedContent = localStorage.getItem(STORAGE_KEY);
const initialContent = savedContent || DEFAULT_CONTENT;

const editorView = new EditorView({
  doc: initialContent,
  extensions: [
    basicSetup,
    markdown(),
    EditorState.tabSize.of(2),
    indentUnit.of('  '),
    editorThemeConfig.of(cmThemeFor(isDark())),
    keymap.of([{
      key: 'Mod-Enter',
      run: () => { doCompile(); return true; }
    }]),
    EditorView.updateListener.of(update => {
      if (update.docChanged) {
        try { localStorage.setItem(STORAGE_KEY, update.state.doc.toString()); }
        catch (_) { /* localStorage full or unavailable */ }
        // Mark output as stale (editor changed since last compile)
        outputIsStale = true;
        updateTabStatusDot();
        if (isReady) {
          scheduleCompile();
        } else {
          pendingCompile = true;
        }
      }
    }),
  ],
  parent: editorMount,
});

// If restoring saved content, clear the dropdown since it may not match any sample
if (savedContent) {
  sampleSelect.value = '';
}

// ---------------------------------------------------------------
// CodeMirror: Output View (YAML, read-only)
// ---------------------------------------------------------------
const outputView = new EditorView({
  doc: '',
  extensions: [
    basicSetup,
    yaml(),
    EditorState.readOnly.of(true),
    EditorView.editable.of(false),
    outputThemeConfig.of(cmThemeFor(isDark())),
  ],
  parent: outputMount,
});

// Listen for OS theme changes and update CodeMirror accordingly
darkMq.addEventListener('change', () => applyCmTheme());

// ---------------------------------------------------------------
// Sample selector + deep-link loading
// ---------------------------------------------------------------

/** Replace editor content and trigger compile */
function setEditorContent(text) {
  editorView.dispatch({
    changes: { from: 0, to: editorView.state.doc.length, insert: text }
  });
}

/** Load a built-in sample by key */
async function loadSample(key) {
  const sample = SAMPLES[key];
  if (!sample) return;

  // Sync dropdown
  sampleSelect.value = key;
  setHashQuietly(key);

  if (sample.content) {
    setEditorContent(sample.content);
    return;
  }

  // Fetch from URL
  setStatus('compiling', 'Fetching...');
  try {
    const text = await fetchContent(sample.url);
    sample.content = text; // cache on the sample object too
    setEditorContent(text);
  } catch (err) {
    setStatus('error', 'Fetch failed');
    errorText.textContent = err.message;
    errorBanner.classList.remove('d-none');
  }
}

/** Load content from an arbitrary URL (deep-link) */
async function loadFromUrl(url) {
  // Set dropdown to show it's a custom URL
  if (!sampleSelect.querySelector('option[value="__url"]')) {
    const opt = document.createElement('option');
    opt.value = '__url';
    opt.textContent = 'Custom URL';
    sampleSelect.appendChild(opt);
  }
  sampleSelect.value = '__url';
  setHashQuietly(url);

  setStatus('compiling', 'Fetching...');
  try {
    const text = await fetchContent(url);
    setEditorContent(text);
  } catch (err) {
    setStatus('error', 'Fetch failed');
    errorText.textContent = err.message;
    errorBanner.classList.remove('d-none');
  }
}

/** Parse the current hash and load accordingly */
async function loadFromHash() {
  const hash = getHashValue();
  if (!hash) return false;

  if (SAMPLES[hash]) {
    await loadSample(hash);
    return true;
  }

  // Treat as URL if it starts with http
  if (hash.startsWith('http://') || hash.startsWith('https://')) {
    await loadFromUrl(hash);
    return true;
  }

  return false;
}

sampleSelect.addEventListener('change', () => {
  const key = sampleSelect.value;
  if (key === '__url') return;
  loadSample(key);
});

window.addEventListener('hashchange', () => loadFromHash());

// ---------------------------------------------------------------
// Status (uses Primer Label component)
// ---------------------------------------------------------------
const STATUS_LABEL_MAP = {
  loading: 'Label--accent',
  ready: 'Label--success',
  compiling: 'Label--accent',
  error: 'Label--danger'
};

function setStatus(status, text) {
  // Swap Label modifier class
  Object.values(STATUS_LABEL_MAP).forEach(cls => statusBadge.classList.remove(cls));
  statusBadge.classList.add(STATUS_LABEL_MAP[status] || 'Label--secondary');
  statusBadge.setAttribute('data-status', status);
  statusText.textContent = text;

  // Pulse animation for loading/compiling states
  if (status === 'loading' || status === 'compiling') {
    statusDot.style.animation = 'pulse 1.2s ease-in-out infinite';
  } else {
    statusDot.style.animation = '';
  }
}

// ---------------------------------------------------------------
// Compile
// ---------------------------------------------------------------
function scheduleCompile() {
  if (compileTimer) clearTimeout(compileTimer);
  compileTimer = setTimeout(doCompile, 400);
}

async function doCompile() {
  if (!isReady || isCompiling) return;
  if (compileTimer) {
    clearTimeout(compileTimer);
    compileTimer = null;
  }

  const md = editorView.state.doc.toString();
  if (!md.trim()) {
    outputMount.style.display = 'none';
    outputPlaceholder.classList.remove('d-none');
    outputPlaceholder.classList.add('d-flex');
    outputPlaceholder.textContent = 'Compiled YAML will appear here';
    currentYaml = '';
    return;
  }

  isCompiling = true;
  setStatus('compiling', 'Compiling...');
  if (fabCompile) fabCompile.classList.add('compiling');

  // Hide old banners
  errorBanner.classList.add('d-none');
  warningBanner.classList.add('d-none');

  try {
    const result = await compiler.compile(md);

    if (result.error) {
      setStatus('error', 'Error');
      lastCompileStatus = 'error';
      updateTabStatusDot();
      errorText.textContent = result.error;
      errorBanner.classList.remove('d-none');
    } else {
      setStatus('ready', 'Ready');
      lastCompileStatus = 'ok';
      outputIsStale = false;
      updateTabStatusDot();
      currentYaml = result.yaml;

      // Update output CodeMirror view
      outputView.dispatch({
        changes: { from: 0, to: outputView.state.doc.length, insert: result.yaml }
      });
      outputMount.style.display = 'block';
      outputPlaceholder.classList.add('d-none');
      outputPlaceholder.classList.remove('d-flex');

      if (result.warnings && result.warnings.length > 0) {
        warningText.textContent = result.warnings.join('\n');
        warningBanner.classList.remove('d-none');
      }
    }
  } catch (err) {
    setStatus('error', 'Error');
    lastCompileStatus = 'error';
    updateTabStatusDot();
    errorText.textContent = err.message || String(err);
    errorBanner.classList.remove('d-none');
  } finally {
    isCompiling = false;
    if (fabCompile) fabCompile.classList.remove('compiling');
  }
}

// ---------------------------------------------------------------
// Banner close
// ---------------------------------------------------------------
$('errorClose').addEventListener('click', () => errorBanner.classList.add('d-none'));
$('warningClose').addEventListener('click', () => warningBanner.classList.add('d-none'));

// ---------------------------------------------------------------
// Mobile: Tab-based layout
// ---------------------------------------------------------------
const mobileMq = window.matchMedia('(max-width: 767px)');

/** Check if currently in mobile layout */
function isMobileLayout() {
  return mobileMq.matches;
}

/** Switch the active mobile tab */
function switchTab(tab) {
  activeTab = tab;

  // Update tab button states
  tabBar.querySelectorAll('.tab-btn').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.panel === tab);
  });

  // Show/hide panels
  if (tab === 'editor') {
    panelEditor.style.display = '';
    panelOutput.style.display = 'none';
  } else {
    panelEditor.style.display = 'none';
    panelOutput.style.display = '';
  }
}

/** Update the status dot on the Output tab */
function updateTabStatusDot() {
  if (!tabStatusDot) return;
  if (lastCompileStatus === 'error') {
    tabStatusDot.setAttribute('data-stale', 'error');
  } else if (outputIsStale) {
    tabStatusDot.setAttribute('data-stale', 'true');
  } else {
    tabStatusDot.removeAttribute('data-stale');
  }
}

/** Apply or revert mobile layout depending on viewport width */
function applyResponsiveLayout() {
  if (isMobileLayout()) {
    // Enter mobile mode: show only the active tab's panel
    switchTab(activeTab);
  } else {
    // Exit mobile mode: show both panels, restore flex
    panelEditor.style.display = '';
    panelOutput.style.display = '';
    panelEditor.style.flex = '';
    panelOutput.style.flex = '';
  }
}

// Tab button click handlers
tabBar.addEventListener('click', (e) => {
  const btn = e.target.closest('.tab-btn');
  if (!btn || !isMobileLayout()) return;
  switchTab(btn.dataset.panel);
});

// FAB compile button
fabCompile.addEventListener('click', () => {
  doCompile();
});

// Swipe gesture support on panels container
let touchStartX = 0;
let touchStartY = 0;
let touchStartTime = 0;

panels.addEventListener('touchstart', (e) => {
  // Only handle swipe gestures in mobile tab mode and when not dragging the divider
  if (!isMobileLayout() || isDragging) return;
  touchStartX = e.touches[0].clientX;
  touchStartY = e.touches[0].clientY;
  touchStartTime = Date.now();
}, { passive: true });

panels.addEventListener('touchend', (e) => {
  if (!isMobileLayout() || isDragging) return;
  const dx = e.changedTouches[0].clientX - touchStartX;
  const dy = e.changedTouches[0].clientY - touchStartY;
  const dt = Date.now() - touchStartTime;

  // Require: horizontal distance > 50px, more horizontal than vertical, within 500ms
  if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.5 && dt < 500) {
    if (dx < 0 && activeTab === 'editor') {
      // Swipe left: go to Output
      switchTab('output');
    } else if (dx > 0 && activeTab === 'output') {
      // Swipe right: go to Editor
      switchTab('editor');
    }
  }
}, { passive: true });

// Listen for viewport changes (e.g., device rotation, window resize)
mobileMq.addEventListener('change', () => applyResponsiveLayout());

// Apply on initial load
applyResponsiveLayout();

// ---------------------------------------------------------------
// Draggable divider
// ---------------------------------------------------------------
divider.addEventListener('mousedown', (e) => {
  isDragging = true;
  divider.classList.add('dragging');
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
  e.preventDefault();
});

document.addEventListener('mousemove', (e) => {
  if (!isDragging) return;
  const rect = panels.getBoundingClientRect();
  const isMobile = window.innerWidth < 768;

  if (isMobile) {
    const fraction = (e.clientY - rect.top) / rect.height;
    const clamped = Math.max(0.2, Math.min(0.8, fraction));
    panelEditor.style.flex = `0 0 ${clamped * 100}%`;
    panelOutput.style.flex = `0 0 ${(1 - clamped) * 100}%`;
  } else {
    const fraction = (e.clientX - rect.left) / rect.width;
    const clamped = Math.max(0.2, Math.min(0.8, fraction));
    panelEditor.style.flex = `0 0 ${clamped * 100}%`;
    panelOutput.style.flex = `0 0 ${(1 - clamped) * 100}%`;
  }
});

document.addEventListener('mouseup', () => {
  if (isDragging) {
    isDragging = false;
    divider.classList.remove('dragging');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  }
});

// Touch support for mobile divider
divider.addEventListener('touchstart', (e) => {
  isDragging = true;
  divider.classList.add('dragging');
  e.preventDefault();
});

document.addEventListener('touchmove', (e) => {
  if (!isDragging) return;
  const touch = e.touches[0];
  const rect = panels.getBoundingClientRect();
  const isMobile = window.innerWidth < 768;

  if (isMobile) {
    const fraction = (touch.clientY - rect.top) / rect.height;
    const clamped = Math.max(0.2, Math.min(0.8, fraction));
    panelEditor.style.flex = `0 0 ${clamped * 100}%`;
    panelOutput.style.flex = `0 0 ${(1 - clamped) * 100}%`;
  } else {
    const fraction = (touch.clientX - rect.left) / rect.width;
    const clamped = Math.max(0.2, Math.min(0.8, fraction));
    panelEditor.style.flex = `0 0 ${clamped * 100}%`;
    panelOutput.style.flex = `0 0 ${(1 - clamped) * 100}%`;
  }
});

document.addEventListener('touchend', () => {
  if (isDragging) {
    isDragging = false;
    divider.classList.remove('dragging');
  }
});

// ---------------------------------------------------------------
// Initialize compiler
// ---------------------------------------------------------------
async function init() {
  // Hide the loading overlay immediately — the editor is already visible
  loadingOverlay.classList.add('hidden');

  // Show compiler-loading status in the header badge
  setStatus('loading', 'Loading compiler...');

  // Show a helpful placeholder in the output panel while WASM downloads
  outputPlaceholder.textContent = 'Compiler loading... You can start editing!';

  // Kick off deep-link / sample loading (works before WASM is ready)
  loadFromHash();

  try {
    compiler = createWorkerCompiler({
      workerUrl: '/gh-aw/wasm/compiler-worker.js'
    });

    await compiler.ready;
    isReady = true;
    setStatus('ready', 'Ready');

    // Compile whatever the user has typed (or the default/deep-linked content)
    doCompile();
  } catch (err) {
    setStatus('error', 'Failed to load');
    outputPlaceholder.textContent = `Failed to load compiler: ${err.message}`;
  }
}

init();
