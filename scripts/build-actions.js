#!/usr/bin/env node

/**
 * Build Actions - Bundle JavaScript actions from source files
 * 
 * This script builds custom GitHub Actions from source files by:
 * 1. Reading source JavaScript files from actions/{action-name}/src/
 * 2. Embedding required dependencies from pkg/workflow/js/
 * 3. Bundling into actions/{action-name}/index.js
 * 4. Validating action.yml files
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const ACTIONS_DIR = path.join(__dirname, '..', 'actions');
const PKG_JS_DIR = path.join(__dirname, '..', 'pkg', 'workflow', 'js');

/**
 * Get list of action directories
 */
function getActionDirs() {
  if (!fs.existsSync(ACTIONS_DIR)) {
    console.error('Error: actions/ directory does not exist');
    process.exit(1);
  }
  
  const entries = fs.readdirSync(ACTIONS_DIR, { withFileTypes: true });
  return entries
    .filter(entry => entry.isDirectory())
    .map(entry => entry.name);
}

/**
 * Validate action.yml file
 */
function validateActionYml(actionPath) {
  const ymlPath = path.join(actionPath, 'action.yml');
  
  if (!fs.existsSync(ymlPath)) {
    throw new Error(`action.yml not found in ${actionPath}`);
  }
  
  // Read and basic validation
  const content = fs.readFileSync(ymlPath, 'utf8');
  
  // Check required fields
  const requiredFields = ['name', 'description', 'runs'];
  for (const field of requiredFields) {
    if (!content.includes(`${field}:`)) {
      throw new Error(`Missing required field '${field}' in ${ymlPath}`);
    }
  }
  
  // Check that it's a node20 action
  if (!content.includes("using: 'node20'")) {
    throw new Error(`Action must use 'node20' runtime in ${ymlPath}`);
  }
  
  return true;
}

/**
 * Get dependencies for a specific action
 * 
 * TODO: Implement automatic dependency resolution by calling Go functions
 * that use FindJavaScriptDependencies() from pkg/workflow/bundler.go
 * For now, we use a manual mapping that must be updated when dependencies change.
 */
function getActionDependencies(actionName) {
  // Manual dependency mapping - update when adding new actions or changing dependencies
  
  const dependencyMap = {
    'setup-safe-outputs': [
      'safe_outputs_mcp_server.cjs',
      'safe_outputs_bootstrap.cjs',
      'safe_outputs_tools_loader.cjs',
      'safe_outputs_config.cjs',
      'safe_outputs_handlers.cjs',
      'safe_outputs_tools.json',
      'mcp_server_core.cjs',
      'mcp_logger.cjs',
      'messages.cjs',
    ],
    'setup-safe-inputs': [
      'safe_inputs_mcp_server.cjs',
      'safe_inputs_bootstrap.cjs',
      'safe_inputs_config_loader.cjs',
      'safe_inputs_tool_factory.cjs',
      'safe_inputs_validation.cjs',
      'mcp_server_core.cjs',
      'mcp_logger.cjs',
    ],
  };
  
  return dependencyMap[actionName] || [];
}

/**
 * Read file content
 */
function readFile(filePath) {
  if (!fs.existsSync(filePath)) {
    throw new Error(`File not found: ${filePath}`);
  }
  return fs.readFileSync(filePath, 'utf8');
}

/**
 * Build a single action
 */
function buildAction(actionName) {
  console.log(`\n📦 Building action: ${actionName}`);
  
  const actionPath = path.join(ACTIONS_DIR, actionName);
  const srcPath = path.join(actionPath, 'src', 'index.js');
  const outputPath = path.join(actionPath, 'index.js');
  
  // Validate action.yml
  console.log('  ✓ Validating action.yml');
  validateActionYml(actionPath);
  
  // Read source file
  if (!fs.existsSync(srcPath)) {
    throw new Error(`Source file not found: ${srcPath}`);
  }
  console.log('  ✓ Reading source file');
  let source = readFile(srcPath);
  
  // Get dependencies
  const dependencies = getActionDependencies(actionName);
  console.log(`  ✓ Found ${dependencies.length} dependencies`);
  
  // Read all dependency files
  const files = {};
  for (const dep of dependencies) {
    const depPath = path.join(PKG_JS_DIR, dep);
    try {
      const content = readFile(depPath);
      files[dep] = content;
      console.log(`    - ${dep}`);
    } catch (error) {
      console.warn(`    ⚠ Warning: Could not read ${dep}: ${error.message}`);
    }
  }
  
  // Generate FILES object with embedded content
  const filesJson = JSON.stringify(files, null, 2)
    .split('\n')
    .map(line => '  ' + line)
    .join('\n');
  
  // Replace the FILES placeholder
  source = source.replace(
    /const FILES = \{[^}]*\};/s,
    `const FILES = ${filesJson.trim()};`
  );
  
  // Write output
  fs.writeFileSync(outputPath, source, 'utf8');
  console.log(`  ✓ Built ${outputPath}`);
  console.log(`  ✓ Embedded ${Object.keys(files).length} files`);
  
  return true;
}

/**
 * Clean generated files
 */
function cleanActions() {
  console.log('\n🧹 Cleaning generated action files');
  
  const actionDirs = getActionDirs();
  
  for (const actionName of actionDirs) {
    const indexPath = path.join(ACTIONS_DIR, actionName, 'index.js');
    if (fs.existsSync(indexPath)) {
      fs.unlinkSync(indexPath);
      console.log(`  ✓ Removed ${actionName}/index.js`);
    }
  }
}

/**
 * Validate all actions
 */
function validateActions() {
  console.log('\n✅ Validating all actions');
  
  const actionDirs = getActionDirs();
  let valid = true;
  
  for (const actionName of actionDirs) {
    const actionPath = path.join(ACTIONS_DIR, actionName);
    try {
      validateActionYml(actionPath);
      console.log(`  ✓ ${actionName}/action.yml is valid`);
    } catch (error) {
      console.error(`  ✗ ${actionName}/action.yml: ${error.message}`);
      valid = false;
    }
  }
  
  return valid;
}

/**
 * Main function
 */
function main() {
  const args = process.argv.slice(2);
  const command = args[0] || 'build';
  
  try {
    switch (command) {
      case 'build':
        console.log('Building all actions...');
        const actionDirs = getActionDirs();
        for (const actionName of actionDirs) {
          buildAction(actionName);
        }
        console.log('\n✨ All actions built successfully\n');
        break;
        
      case 'clean':
        cleanActions();
        console.log('\n✨ Cleanup complete\n');
        break;
        
      case 'validate':
        const valid = validateActions();
        if (!valid) {
          process.exit(1);
        }
        console.log('\n✨ All actions valid\n');
        break;
        
      default:
        console.error(`Unknown command: ${command}`);
        console.error('Usage: node build-actions.js [build|clean|validate]');
        process.exit(1);
    }
  } catch (error) {
    console.error(`\n❌ Error: ${error.message}`);
    process.exit(1);
  }
}

main();
