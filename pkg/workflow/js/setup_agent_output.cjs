function main() {
  const fs = require("fs");
  const crypto = require("crypto");

  // Create the safe outputs directory structure
  const safeOutputsDir = "/tmp/gh-aw/safe-outputs";
  const filesDir = `${safeOutputsDir}/files`;
  const outputFile = `${safeOutputsDir}/safe_outputs.jsonl`;

  // Ensure the safe outputs directory structure exists
  fs.mkdirSync(safeOutputsDir, { recursive: true });
  fs.mkdirSync(filesDir, { recursive: true });

  // We don't create the file, as the name is sufficiently random
  // and some engines (Claude) fails first Write to the file
  // if it exists and has not been read.

  // Set the environment variables for subsequent steps
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS_DIR", safeOutputsDir);
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS_FILES_DIR", filesDir);

  // Also set as step output for reference
  core.setOutput("output_file", outputFile);
  core.setOutput("output_dir", safeOutputsDir);
  core.setOutput("files_dir", filesDir);
}

main();
