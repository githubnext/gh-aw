// ================================================================
// gh-aw Playground - Application Logic
// ================================================================

import { createWorkerCompiler } from '/gh-aw/wasm/compiler-loader.js';
import { attachHoverTooltips } from './hover-tooltips.js';

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
const editorTextarea = $('editorTextarea');
const outputPlaceholder = $('outputPlaceholder');
const outputCode = $('outputCode');
const outputPre = $('outputPre');
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
// Input Editor (<textarea>)
// ---------------------------------------------------------------
const savedContent = localStorage.getItem(STORAGE_KEY);
const initialContent = savedContent || DEFAULT_CONTENT;
editorTextarea.value = initialContent;

// Tab inserts 2 spaces (preserving undo); Shift-Tab dedents; Mod-Enter triggers compile
editorTextarea.addEventListener('keydown', (e) => {
  if (e.key === 'Tab' && !e.shiftKey) {
    e.preventDefault();
    // execCommand preserves the browser undo stack
    document.execCommand('insertText', false, '  ');
  }
  if (e.key === 'Tab' && e.shiftKey) {
    e.preventDefault();
    const start = editorTextarea.selectionStart;
    const end = editorTextarea.selectionEnd;
    const val = editorTextarea.value;
    // Find the start of the current line
    const lineStart = val.lastIndexOf('\n', start - 1) + 1;
    const lineEnd = val.indexOf('\n', start);
    const line = val.substring(lineStart, lineEnd === -1 ? val.length : lineEnd);
    const spaces = line.match(/^ {1,2}/);
    if (spaces) {
      const removed = spaces[0].length;
      // Select the leading spaces and delete them via execCommand to preserve undo
      editorTextarea.selectionStart = lineStart;
      editorTextarea.selectionEnd = lineStart + removed;
      document.execCommand('delete', false);
      // Restore adjusted selection
      const newStart = Math.max(lineStart, start - removed);
      const newEnd = Math.max(lineStart, end - removed);
      editorTextarea.selectionStart = newStart;
      editorTextarea.selectionEnd = newEnd;
    }
  }
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault();
    if (isReady) {
      doCompile();
    } else {
      pendingCompile = true;
    }
  }
});

// Save to localStorage and schedule auto-compile on input
editorTextarea.addEventListener('input', () => {
  try { localStorage.setItem(STORAGE_KEY, editorTextarea.value); }
  catch (_) { /* localStorage full or unavailable */ }
  if (isReady) {
    scheduleCompile();
  } else {
    pendingCompile = true;
  }
});

// Attach hover tooltips to the textarea
attachHoverTooltips(editorTextarea);

// If restoring saved content, clear the dropdown since it may not match any sample
if (savedContent) {
  sampleSelect.value = '';
}

// ---------------------------------------------------------------
// Sample selector + deep-link loading
// ---------------------------------------------------------------

/** Replace editor content and trigger compile */
function setEditorContent(text) {
  editorTextarea.value = text;
  editorTextarea.dispatchEvent(new Event('input'));
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

  const md = editorTextarea.value;
  if (!md.trim()) {
    outputPre.style.display = 'none';
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

      // Update output display
      outputCode.textContent = result.yaml;
      outputPre.style.display = 'block';
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
