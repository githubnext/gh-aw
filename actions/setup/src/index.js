// Setup Activation Action
// Copies activation job files to the agent environment

const core = require("@actions/core");
const fs = require("fs");
const path = require("path");
const tmp = require("tmp");

// Embedded activation files will be inserted here during build
const FILES = {
  // This will be populated by the build script
};

async function run() {
  try {
    const requestedDestination = core.getInput("destination") || "/tmp/gh-aw/actions/activation";

    // Use tmp library to create secure temporary files
    // This ensures files are created with secure permissions and are inaccessible to other users
    let destination;

    if (requestedDestination.startsWith("/tmp")) {
      // For /tmp paths, use tmp library to create a secure directory
      const tmpDir = tmp.dirSync({ mode: 0o700, prefix: "gh-aw-", unsafeCleanup: false });
      destination = tmpDir.name;
      core.info(`Created secure temporary directory: ${destination}`);
    } else {
      // For other paths, create directory with secure permissions
      destination = requestedDestination;
      if (!fs.existsSync(destination)) {
        fs.mkdirSync(destination, { recursive: true, mode: 0o700 });
        core.info(`Created directory: ${destination}`);
      }
    }

    core.info(`Copying activation files to ${destination}`);

    let fileCount = 0;

    // Copy each embedded file using tmp library for secure file creation
    for (const [filename, content] of Object.entries(FILES)) {
      const filePath = path.join(destination, filename);

      // Use tmp library to create secure temporary file
      const tmpFile = tmp.fileSync({ mode: 0o600, dir: destination, name: filename, keep: true });
      fs.writeFileSync(tmpFile.name, content, { encoding: "utf8" });
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
