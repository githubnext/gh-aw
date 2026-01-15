// Setup Activation Action
// Copies activation job files to the agent environment

const core = require("@actions/core");
const fs = require("fs");
const path = require("path");

// Embedded activation files will be inserted here during build
const FILES = {
  // This will be populated by the build script
};

async function run() {
  try {
    const destination = core.getInput("destination") || "/tmp/gh-aw/actions/activation";

    core.info(`Copying activation files to ${destination}`);

    // Create destination directory with secure permissions if it doesn't exist
    // Note: mode parameter is ignored on Windows; relies on default NTFS permissions
    if (!fs.existsSync(destination)) {
      fs.mkdirSync(destination, { recursive: true, mode: 0o700 });
      core.info(`Created directory: ${destination}`);
    }

    let fileCount = 0;

    // Copy each embedded file
    for (const [filename, content] of Object.entries(FILES)) {
      const filePath = path.join(destination, filename);
      // Create file with secure permissions (readable/writable only by owner)
      // Note: mode parameter is ignored on Windows; relies on default NTFS permissions
      fs.writeFileSync(filePath, content, { encoding: "utf8", mode: 0o600 });
      core.info(`Copied: ${filename}`);
      fileCount++;
    }

    core.setOutput("files-copied", fileCount.toString());
    core.info(`✓ Successfully copied ${fileCount} files`);
  } catch (error) {
    core.setFailed(`Action failed: ${error.message}`);
  }
}

run();
