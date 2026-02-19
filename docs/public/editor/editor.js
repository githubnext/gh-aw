// ================================================================
// gh-aw Playground - Application Logic
// ================================================================

import { createWorkerCompiler } from '/gh-aw/wasm/compiler-loader.js';

// ---------------------------------------------------------------
// Default workflow content
// ---------------------------------------------------------------
const DEFAULT_CONTENT = [
  '---',
  'name: hello-world',
  'description: A simple hello world workflow',
  'on:',
  '  workflow_dispatch:',
  'engine: copilot',
  '---',
  '',
  '# Mission',
  '',
  'Say hello to the world! Check the current date and time, and greet the user warmly.',
  ''
].join('\n');

// ---------------------------------------------------------------
// DOM Elements
// ---------------------------------------------------------------
const $ = (id) => document.getElementById(id);

const editor = $('editor');
const outputPre = $('outputPre');
const outputPlaceholder = $('outputPlaceholder');
const outputContainer = $('outputContainer');
const compileBtn = $('compileBtn');
const statusBadge = $('statusBadge');
const statusText = $('statusText');
const statusDot = $('statusDot');
const loadingOverlay = $('loadingOverlay');
const errorBanner = $('errorBanner');
const errorText = $('errorText');
const warningBanner = $('warningBanner');
const warningText = $('warningText');
const lineNumbersInner = $('lineNumbersInner');
const themeToggle = $('themeToggle');
const toggleTrack = $('toggleTrack');
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
let autoCompile = true;
let compileTimer = null;
let currentYaml = '';

// ---------------------------------------------------------------
// Theme (uses Primer's data-color-mode)
// ---------------------------------------------------------------
function getPreferredTheme() {
  const saved = localStorage.getItem('gh-aw-playground-theme');
  if (saved) return saved;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
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
}

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
// Keyboard shortcut hint (Mac vs other)
// ---------------------------------------------------------------
const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
document.querySelectorAll('.kbd-hint-mac').forEach(el => el.style.display = isMac ? 'inline' : 'none');
document.querySelectorAll('.kbd-hint-other').forEach(el => el.style.display = isMac ? 'none' : 'inline');

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
// Line numbers
// ---------------------------------------------------------------
function updateLineNumbers() {
  const lines = editor.value.split('\n').length;
  let html = '';
  for (let i = 1; i <= lines; i++) {
    html += '<div>' + i + '</div>';
  }
  lineNumbersInner.innerHTML = html;
}

function syncLineNumberScroll() {
  const lineNumbers = $('lineNumbers');
  lineNumbers.scrollTop = editor.scrollTop;
}

// ---------------------------------------------------------------
// Editor setup
// ---------------------------------------------------------------
editor.value = DEFAULT_CONTENT;
updateLineNumbers();

editor.addEventListener('input', () => {
  updateLineNumbers();
  if (autoCompile && isReady) {
    scheduleCompile();
  }
});

editor.addEventListener('scroll', syncLineNumberScroll);

// Tab key inserts 2 spaces
editor.addEventListener('keydown', (e) => {
  if (e.key === 'Tab') {
    e.preventDefault();
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    const val = editor.value;
    editor.value = val.substring(0, start) + '  ' + val.substring(end);
    editor.selectionStart = editor.selectionEnd = start + 2;
    editor.dispatchEvent(new Event('input'));
  }

  // Ctrl+Enter or Cmd+Enter to compile
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault();
    doCompile();
  }
});

// ---------------------------------------------------------------
// Auto-compile toggle
// ---------------------------------------------------------------
$('autoCompileToggle').addEventListener('click', () => {
  autoCompile = !autoCompile;
  if (autoCompile) {
    toggleTrack.classList.add('active');
  } else {
    toggleTrack.classList.remove('active');
  }
});

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

  const md = editor.value;
  if (!md.trim()) {
    outputPre.classList.add('d-none');
    outputPlaceholder.classList.remove('d-none');
    outputPlaceholder.textContent = 'Compiled YAML will appear here';
    currentYaml = '';
    return;
  }

  isCompiling = true;
  setStatus('compiling', 'Compiling...');
  compileBtn.disabled = true;

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
      outputPre.textContent = result.yaml;
      outputPre.classList.remove('d-none');
      outputPlaceholder.classList.add('d-none');

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
    compileBtn.disabled = !isReady;
  }
}

compileBtn.addEventListener('click', doCompile);

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
    compileBtn.disabled = false;
    loadingOverlay.classList.add('hidden');

    // Auto-compile the default content
    if (autoCompile) {
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
