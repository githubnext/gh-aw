// ================================================================
// gh-aw Playground - Application Logic
// ================================================================

import { createWorkerCompiler } from '/gh-aw/wasm/compiler-loader.js';

// ---------------------------------------------------------------
// Default workflow content
// ---------------------------------------------------------------
const DEFAULT_CONTENT = `---
name: hello-world
description: A simple hello world workflow
on:
  workflow_dispatch:
engine: copilot
---

# Mission

Say hello to the world! Check the current date and time, and greet the user warmly.
`;

// ---------------------------------------------------------------
// DOM Elements
// ---------------------------------------------------------------
const $ = (id) => document.getElementById(id);

const editor = $('editor');
const outputPre = $('outputPre');
const outputPlaceholder = $('outputPlaceholder');
const compileBtn = $('compileBtn');
const copyBtn = $('copyBtn');
const statusBadge = $('statusBadge');
const statusText = $('statusText');
const loadingOverlay = $('loadingOverlay');
const errorBanner = $('errorBanner');
const errorText = $('errorText');
const warningBanner = $('warningBanner');
const warningText = $('warningText');
const lineNumbers = $('lineNumbers');
const lineNumbersInner = $('lineNumbersInner');
const themeToggle = $('themeToggle');
const toggleTrack = $('toggleTrack');
const divider = $('divider');
const panelEditor = $('panelEditor');
const panelOutput = $('panelOutput');
const panels = $('panels');
const tabBar = $('tabBar');
const tabAdd = $('tabAdd');

// ---------------------------------------------------------------
// State
// ---------------------------------------------------------------
let compiler = null;
let isReady = false;
let isCompiling = false;
let autoCompile = true;
let compileTimer = null;
let currentYaml = '';

// File tabs state: ordered list of { name, content }
const MAIN_FILE = 'workflow.md';
let files = [{ name: MAIN_FILE, content: DEFAULT_CONTENT }];
let activeTab = MAIN_FILE;

// ---------------------------------------------------------------
// Theme
// ---------------------------------------------------------------
function getPreferredTheme() {
  const saved = localStorage.getItem('gh-aw-playground-theme');
  if (saved) return saved;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function setTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  localStorage.setItem('gh-aw-playground-theme', theme);
  const sunIcon = themeToggle.querySelector('.icon-sun');
  const moonIcon = themeToggle.querySelector('.icon-moon');
  sunIcon.style.display = theme === 'dark' ? 'block' : 'none';
  moonIcon.style.display = theme === 'dark' ? 'none' : 'block';
}

setTheme(getPreferredTheme());

themeToggle.addEventListener('click', () => {
  const current = document.documentElement.getAttribute('data-theme');
  setTheme(current === 'dark' ? 'light' : 'dark');
});

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
// Status
// ---------------------------------------------------------------
function setStatus(status, text) {
  statusBadge.setAttribute('data-status', status);
  statusText.textContent = text;
}

// ---------------------------------------------------------------
// Line numbers
// ---------------------------------------------------------------
function updateLineNumbers() {
  const lines = editor.value.split('\n').length;
  let html = '';
  for (let i = 1; i <= lines; i++) html += '<div>' + i + '</div>';
  lineNumbersInner.innerHTML = html;
}

function syncLineNumberScroll() {
  lineNumbers.scrollTop = editor.scrollTop;
}

// ---------------------------------------------------------------
// File tabs
// ---------------------------------------------------------------
function getFile(name) {
  return files.find(f => f.name === name);
}

function renderTabs() {
  tabBar.querySelectorAll('.tab').forEach(el => el.remove());

  for (const file of files) {
    const tab = document.createElement('div');
    tab.className = 'tab' + (file.name === activeTab ? ' active' : '');
    tab.dataset.name = file.name;

    const label = document.createElement('span');
    label.textContent = file.name;
    tab.appendChild(label);

    if (file.name !== MAIN_FILE) {
      const close = document.createElement('button');
      close.className = 'tab-close';
      close.title = 'Remove file';
      close.innerHTML = '<svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor"><path d="M3.72 3.72a.75.75 0 011.06 0L8 6.94l3.22-3.22a.749.749 0 011.275.326.749.749 0 01-.215.734L9.06 8l3.22 3.22a.749.749 0 01-.326 1.275.749.749 0 01-.734-.215L8 9.06l-3.22 3.22a.751.751 0 01-1.042-.018.751.751 0 01-.018-1.042L6.94 8 3.72 4.78a.75.75 0 010-1.06z"/></svg>';
      close.addEventListener('click', (e) => {
        e.stopPropagation();
        removeTab(file.name);
      });
      tab.appendChild(close);
    }

    tab.addEventListener('click', () => switchTab(file.name));
    tabBar.insertBefore(tab, tabAdd);
  }
}

function switchTab(name) {
  const current = getFile(activeTab);
  if (current) current.content = editor.value;

  activeTab = name;
  const file = getFile(name);
  if (file) {
    editor.value = file.content;
    updateLineNumbers();
  }
  renderTabs();
}

function addTab() {
  const name = prompt('File path (e.g. shared/my-tools.md):');
  if (!name || !name.trim()) return;

  const trimmed = name.trim();
  if (getFile(trimmed)) { switchTab(trimmed); return; }

  const defaultImportContent = `---
# Shared workflow component
# This file can define: tools, steps, engine, mcp-servers, etc.
tools:
  - name: example_tool
    description: An example tool
---

# Instructions

Add your shared workflow instructions here.
`;

  files.push({ name: trimmed, content: defaultImportContent });
  switchTab(trimmed);
}

function removeTab(name) {
  if (name === MAIN_FILE) return;
  files = files.filter(f => f.name !== name);
  if (activeTab === name) {
    switchTab(MAIN_FILE);
  } else {
    renderTabs();
  }
  if (autoCompile && isReady) scheduleCompile();
}

tabAdd.addEventListener('click', addTab);

// ---------------------------------------------------------------
// Editor setup
// ---------------------------------------------------------------
editor.value = DEFAULT_CONTENT;
updateLineNumbers();
renderTabs();

editor.addEventListener('input', () => {
  updateLineNumbers();
  const file = getFile(activeTab);
  if (file) file.content = editor.value;
  if (autoCompile && isReady) scheduleCompile();
});

editor.addEventListener('scroll', syncLineNumberScroll);

editor.addEventListener('keydown', (e) => {
  if (e.key === 'Tab') {
    e.preventDefault();
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    editor.value = editor.value.substring(0, start) + '  ' + editor.value.substring(end);
    editor.selectionStart = editor.selectionEnd = start + 2;
    editor.dispatchEvent(new Event('input'));
  }
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
  toggleTrack.classList.toggle('active', autoCompile);
});

// ---------------------------------------------------------------
// Compile
// ---------------------------------------------------------------
function scheduleCompile() {
  if (compileTimer) clearTimeout(compileTimer);
  compileTimer = setTimeout(doCompile, 400);
}

function getImportFiles() {
  const importFiles = {};
  for (const file of files) {
    if (file.name !== MAIN_FILE) importFiles[file.name] = file.content;
  }
  return Object.keys(importFiles).length > 0 ? importFiles : undefined;
}

async function doCompile() {
  if (!isReady || isCompiling) return;
  if (compileTimer) { clearTimeout(compileTimer); compileTimer = null; }

  // Save current editor content
  const currentFile = getFile(activeTab);
  if (currentFile) currentFile.content = editor.value;

  // Get the main workflow content
  const mainFile = getFile(MAIN_FILE);
  const md = mainFile ? mainFile.content : '';
  if (!md.trim()) {
    outputPre.style.display = 'none';
    outputPlaceholder.style.display = 'flex';
    outputPlaceholder.textContent = 'Compiled YAML will appear here';
    currentYaml = '';
    copyBtn.disabled = true;
    return;
  }

  isCompiling = true;
  setStatus('compiling', 'Compiling...');
  compileBtn.disabled = true;
  errorBanner.classList.remove('visible');
  warningBanner.classList.remove('visible');

  try {
    const importFiles = getImportFiles();
    const result = await compiler.compile(md, importFiles);

    if (result.error) {
      setStatus('error', 'Error');
      errorText.textContent = result.error;
      errorBanner.classList.add('visible');
    } else {
      setStatus('ready', 'Ready');
      currentYaml = result.yaml;
      outputPre.textContent = result.yaml;
      outputPre.style.display = 'block';
      outputPlaceholder.style.display = 'none';
      copyBtn.disabled = false;

      if (result.warnings && result.warnings.length > 0) {
        warningText.textContent = result.warnings.join('\n');
        warningBanner.classList.add('visible');
      }
    }
  } catch (err) {
    setStatus('error', 'Error');
    errorText.textContent = err.message || String(err);
    errorBanner.classList.add('visible');
  } finally {
    isCompiling = false;
    compileBtn.disabled = !isReady;
  }
}

compileBtn.addEventListener('click', doCompile);

// ---------------------------------------------------------------
// Copy YAML
// ---------------------------------------------------------------
function showCopyFeedback() {
  const feedback = $('copyFeedback');
  feedback.classList.add('show');
  setTimeout(() => feedback.classList.remove('show'), 2000);
}

copyBtn.addEventListener('click', async () => {
  if (!currentYaml) return;
  try {
    await navigator.clipboard.writeText(currentYaml);
  } catch {
    const ta = document.createElement('textarea');
    ta.value = currentYaml;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  }
  showCopyFeedback();
});

// ---------------------------------------------------------------
// Banner close
// ---------------------------------------------------------------
$('errorClose').addEventListener('click', () => errorBanner.classList.remove('visible'));
$('warningClose').addEventListener('click', () => warningBanner.classList.remove('visible'));

// ---------------------------------------------------------------
// Draggable divider
// ---------------------------------------------------------------
let isDragging = false;

function resizePanels(clientX, clientY) {
  const rect = panels.getBoundingClientRect();
  const isMobile = window.innerWidth < 768;
  const pos = isMobile ? clientY - rect.top : clientX - rect.left;
  const size = isMobile ? rect.height : rect.width;
  const clamped = Math.max(0.2, Math.min(0.8, pos / size));
  panelEditor.style.flex = `0 0 ${clamped * 100}%`;
  panelOutput.style.flex = `0 0 ${(1 - clamped) * 100}%`;
}

divider.addEventListener('mousedown', (e) => {
  isDragging = true;
  divider.classList.add('dragging');
  const isMobile = window.innerWidth < 768;
  document.body.style.cursor = isMobile ? 'row-resize' : 'col-resize';
  document.body.style.userSelect = 'none';
  e.preventDefault();
});

document.addEventListener('mousemove', (e) => {
  if (isDragging) resizePanels(e.clientX, e.clientY);
});

document.addEventListener('mouseup', () => {
  if (isDragging) {
    isDragging = false;
    divider.classList.remove('dragging');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  }
});

divider.addEventListener('touchstart', (e) => {
  isDragging = true;
  divider.classList.add('dragging');
  e.preventDefault();
});

document.addEventListener('touchmove', (e) => {
  if (isDragging) resizePanels(e.touches[0].clientX, e.touches[0].clientY);
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

    if (autoCompile) doCompile();
  } catch (err) {
    setStatus('error', 'Failed to load');
    loadingOverlay.querySelector('.loading-text').textContent = 'Failed to load compiler';
    loadingOverlay.querySelector('.loading-subtext').textContent = err.message;
    loadingOverlay.querySelector('.loading-spinner').style.display = 'none';
  }
}

init();
