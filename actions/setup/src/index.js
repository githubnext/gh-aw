// Setup Activation Action
// Copies activation job files to the agent environment

const core = require('@actions/core');
const fs = require('fs');
const path = require('path');

async function run() {
  try {
    const destination = core.getInput('destination') || '/tmp/gh-aw/actions/activation';
    
    core.info(`Copying activation files to ${destination}`);
    
    // Create destination directory if it doesn't exist
    if (!fs.existsSync(destination)) {
      fs.mkdirSync(destination, { recursive: true });
      core.info(`Created directory: ${destination}`);
    }
    
    // The source directory for scripts - relative to this action's directory
    // When running in GitHub Actions, __dirname is the action's root directory
    const jsSourceDir = path.join(__dirname, 'js');
    const shSourceDir = path.join(__dirname, 'sh');
    
    let fileCount = 0;
    
    // Copy JavaScript files
    if (fs.existsSync(jsSourceDir)) {
      const jsFiles = fs.readdirSync(jsSourceDir).filter(f => f.endsWith('.cjs') || f.endsWith('.json'));
      for (const filename of jsFiles) {
        const sourcePath = path.join(jsSourceDir, filename);
        const destPath = path.join(destination, filename);
        const content = fs.readFileSync(sourcePath, 'utf8');
        fs.writeFileSync(destPath, content, 'utf8');
        core.info(`Copied: ${filename}`);
        fileCount++;
      }
    }
    
    // Copy shell scripts
    if (fs.existsSync(shSourceDir)) {
      const shFiles = fs.readdirSync(shSourceDir).filter(f => f.endsWith('.sh'));
      for (const filename of shFiles) {
        const sourcePath = path.join(shSourceDir, filename);
        const destPath = path.join(destination, filename);
        const content = fs.readFileSync(sourcePath, 'utf8');
        fs.writeFileSync(destPath, content, 'utf8');
        core.info(`Copied: ${filename}`);
        fileCount++;
      }
    }
    
    core.setOutput('files-copied', fileCount.toString());
    core.info(`✓ Successfully copied ${fileCount} files`);
    
  } catch (error) {
    core.setFailed(`Action failed: ${error.message}`);
  }
}

run();
