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
const themeToggle = $('themeToggle');
const divider = $('divider');
const panelEditor = $('panelEditor');
const panelOutput = $('panelOutput');
const panels = $('panels');

// ---------------------------------------------------------------
// State
// ---------------------------------------------------------------
let compiler = null;
let isReady = false;
let isCompiling = false;
let compileTimer = null;
let currentYaml = '';

// ---------------------------------------------------------------
// Theme (uses Primer's data-color-mode)
// ---------------------------------------------------------------
const editorThemeConfig = new Compartment();
const outputThemeConfig = new Compartment();

function getPreferredTheme() {
  const saved = localStorage.getItem('gh-aw-playground-theme');
  if (saved) return saved;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function cmThemeFor(theme) {
  return theme === 'dark' ? oneDark : [];
}

function setTheme(theme) {
  document.documentElement.setAttribute('data-color-mode', theme);
  localStorage.setItem('gh-aw-playground-theme', theme);
  const sunIcon = themeToggle.querySelector('.icon-sun');
  const moonIcon = themeToggle.querySelector('.icon-moon');
  if (theme === 'dark') {
    sunIcon.style.display = 'block';
    moonIcon.style.display = 'none';
  } else {
    sunIcon.style.display = 'none';
    moonIcon.style.display = 'block';
  }

  // Update CodeMirror themes
  const cmTheme = cmThemeFor(theme);
  editorView.dispatch({ effects: editorThemeConfig.reconfigure(cmTheme) });
  outputView.dispatch({ effects: outputThemeConfig.reconfigure(cmTheme) });
}

// ---------------------------------------------------------------
// CodeMirror: Input Editor (Markdown with YAML frontmatter)
// ---------------------------------------------------------------
const editorView = new EditorView({
  doc: DEFAULT_CONTENT,
  extensions: [
    basicSetup,
    markdown(),
    EditorState.tabSize.of(2),
    indentUnit.of('  '),
    editorThemeConfig.of(cmThemeFor(getPreferredTheme())),
    keymap.of([{
      key: 'Mod-Enter',
      run: () => { doCompile(); return true; }
    }]),
    EditorView.updateListener.of(update => {
      if (update.docChanged && isReady) {
        scheduleCompile();
      }
    }),
  ],
  parent: editorMount,
});

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
    outputThemeConfig.of(cmThemeFor(getPreferredTheme())),
  ],
  parent: outputMount,
});

// ---------------------------------------------------------------
// Apply initial theme + listen for changes
// ---------------------------------------------------------------
setTheme(getPreferredTheme());

themeToggle.addEventListener('click', () => {
  const current = document.documentElement.getAttribute('data-color-mode');
  setTheme(current === 'dark' ? 'light' : 'dark');
});

// Listen for OS theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
  if (!localStorage.getItem('gh-aw-playground-theme')) {
    setTheme(e.matches ? 'dark' : 'light');
  }
});

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

  // Hide old banners
  errorBanner.classList.add('d-none');
  warningBanner.classList.add('d-none');

  try {
    const result = await compiler.compile(md);

    if (result.error) {
      setStatus('error', 'Error');
      errorText.textContent = result.error;
      errorBanner.classList.remove('d-none');
    } else {
      setStatus('ready', 'Ready');
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
    errorText.textContent = err.message || String(err);
    errorBanner.classList.remove('d-none');
  } finally {
    isCompiling = false;
  }
}

// ---------------------------------------------------------------
// Banner close
// ---------------------------------------------------------------
$('errorClose').addEventListener('click', () => errorBanner.classList.add('d-none'));
$('warningClose').addEventListener('click', () => warningBanner.classList.add('d-none'));

// ---------------------------------------------------------------
// Draggable divider
// ---------------------------------------------------------------
let isDragging = false;

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
  try {
    compiler = createWorkerCompiler({
      workerUrl: '/gh-aw/wasm/compiler-worker.js'
    });

    await compiler.ready;
    isReady = true;
    setStatus('ready', 'Ready');
    loadingOverlay.classList.add('hidden');

    // Load from hash deep-link, or compile default content
    const loaded = await loadFromHash();
    if (!loaded) {
      doCompile();
    }
  } catch (err) {
    setStatus('error', 'Failed to load');
    loadingOverlay.querySelector('.f4').textContent = 'Failed to load compiler';
    loadingOverlay.querySelector('.f6').textContent = err.message;
    loadingOverlay.querySelector('.loading-spinner').style.display = 'none';
  }
}

init();
