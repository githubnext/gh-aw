// Setup Activation Action - Post Cleanup
// Removes the /tmp/gh-aw/ directory created during the main action step.
// Runs in the post-job phase so that temporary files are erased after the
// workflow job completes, regardless of success or failure.

const fs = require("fs");

const tmpDir = "/tmp/gh-aw";

try {
  console.log(`Cleaning up ${tmpDir}...`);
  fs.rmSync(tmpDir, { recursive: true, force: true });
  console.log(`Cleaned up ${tmpDir}`);
} catch (err) {
  // Log but do not fail — cleanup is best-effort
  console.error(`Warning: failed to clean up ${tmpDir}: ${err.message}`);
}
