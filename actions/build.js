#!/usr/bin/env node

const esbuild = require('esbuild');
const fs = require('fs');
const path = require('path');

// Get all action directories
const actionsDir = __dirname;
const entries = fs.readdirSync(actionsDir, { withFileTypes: true });

const actionDirs = entries
  .filter(entry => entry.isDirectory() && fs.existsSync(path.join(actionsDir, entry.name, 'src', 'index.js')))
  .map(entry => entry.name);

console.log(`Building ${actionDirs.length} actions...`);

// Build each action
for (const actionName of actionDirs) {
  const srcPath = path.join(actionsDir, actionName, 'src', 'index.js');
  const outPath = path.join(actionsDir, actionName, 'index.js');
  
  console.log(`  Building ${actionName}...`);
  
  esbuild.buildSync({
    entryPoints: [srcPath],
    bundle: true,
    platform: 'node',
    target: 'node20',
    outfile: outPath,
    format: 'cjs',
    minify: false,
    sourcemap: false,
  });
  
  console.log(`    ✓ ${actionName}/index.js`);
}

console.log(`✨ All actions built successfully`);
