// Setup Activation Action - Post Cleanup
// Removes the /tmp/gh-aw/ directory created during the main action step.
// Runs in the post-job phase so that temporary files are erased after the
// workflow job completes, regardless of success or failure.
//
// Files inside /tmp/gh-aw/ may be owned by root (written by Docker containers
// or privileged scripts), so we use `sudo rm -rf` — GitHub-hosted runners have
// passwordless sudo.  We fall back to fs.rmSync for self-hosted runners that
// don't have sudo but do have direct write access.

const { spawnSync } = require("child_process");
const fs = require("fs");

const tmpDir = "/tmp/gh-aw";

console.log(`Cleaning up ${tmpDir}...`);

// Try sudo rm -rf first (handles root-owned files on GitHub-hosted runners)
const result = spawnSync("sudo", ["rm", "-rf", tmpDir], { stdio: "inherit" });

if (result.status === 0) {
  console.log(`Cleaned up ${tmpDir}`);
} else {
  // Fall back to fs.rmSync for environments without sudo
  try {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    console.log(`Cleaned up ${tmpDir}`);
  } catch (err) {
    // Log but do not fail — cleanup is best-effort
    console.error(`Warning: failed to clean up ${tmpDir}: ${err.message}`);
  }
}
