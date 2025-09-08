// Network permissions hook generator for Claude Code engine
// Writes a pre-generated Python hook script to the filesystem

const fs = require('fs');
const path = require('path');

async function main() {
  try {
    // Get the pre-generated Python script from environment variable
    const pythonScriptEnv = process.env.GITHUB_AW_PYTHON_SCRIPT;
    if (!pythonScriptEnv) {
      core.error('GITHUB_AW_PYTHON_SCRIPT environment variable not set');
      throw new Error('Missing required environment variable GITHUB_AW_PYTHON_SCRIPT');
    }

    // Parse the Python script JSON
    let pythonScript;
    try {
      pythonScript = JSON.parse(pythonScriptEnv);
    } catch (error) {
      core.error(`Failed to parse Python script JSON: ${error.message}`);
      throw error;
    }

    console.log(`Writing network permissions hook script (${pythonScript.length} characters)`);

    // Create the .claude/hooks directory
    const hooksDir = '.claude/hooks';
    await io.mkdirP(hooksDir);
    console.log(`Created directory: ${hooksDir}`);

    // Write the Python hook script
    const hookPath = path.join(hooksDir, 'network_permissions.py');
    fs.writeFileSync(hookPath, pythonScript, 'utf8');
    console.log(`Generated network permissions hook: ${hookPath}`);

    // Make the script executable
    fs.chmodSync(hookPath, '755');
    console.log(`Set executable permissions on: ${hookPath}`);

    console.log('✓ Network permissions hook generated successfully');
    
  } catch (error) {
    core.error(`Failed to generate network permissions hook: ${error.message}`);
    throw error;
  }
}

await main();