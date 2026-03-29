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
import { frontmatterHoverTooltip } from './hover-tooltips.js';

// ---------------------------------------------------------------
// Sample workflow registry
// ---------------------------------------------------------------

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
};

const DEFAULT_CONTENT = SAMPLES['hello-world'].content;

// ---------------------------------------------------------------
// Hash-based deep linking
//
// Supported formats:
//   #hello-world              — built-in sample key
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
let isDragging = false;

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
    frontmatterHoverTooltip,
    EditorView.updateListener.of(update => {
      if (update.docChanged) {
        try { localStorage.setItem(STORAGE_KEY, update.state.doc.toString()); }
        catch (_) { /* localStorage full or unavailable */ }
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
function loadSample(key) {
  const sample = SAMPLES[key];
  if (!sample) return;

  // Sync dropdown
  sampleSelect.value = key;
  setHashQuietly(key);

  setEditorContent(sample.content);
}

/** Parse the current hash and load accordingly */
function loadFromHash() {
  const hash = getHashValue();
  if (!hash) return false;

  if (SAMPLES[hash]) {
    loadSample(hash);
    return true;
  }

  return false;
}

sampleSelect.addEventListener('change', () => {
  const key = sampleSelect.value;
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
  const fraction = (e.clientX - rect.left) / rect.width;
  const clamped = Math.max(0.2, Math.min(0.8, fraction));
  panelEditor.style.flex = `0 0 ${clamped * 100}%`;
  panelOutput.style.flex = `0 0 ${(1 - clamped) * 100}%`;
});

document.addEventListener('mouseup', () => {
  if (isDragging) {
    isDragging = false;
    divider.classList.remove('dragging');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
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
