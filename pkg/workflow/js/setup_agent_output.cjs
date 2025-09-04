function main() {
  const fs = require("fs");
  const crypto = require("crypto");

  // Generate a random filename for the output file
  const randomId = crypto.randomBytes(8).toString("hex");
  const outputFile = `/tmp/aw_output_${randomId}.txt`;

  // Ensure the /tmp directory exists and create empty output file
  fs.mkdirSync("/tmp", { recursive: true });
  fs.writeFileSync(outputFile, "", { mode: 0o644 });

  // Verify the file was created and is writable
  if (!fs.existsSync(outputFile)) {
    throw new Error(`Failed to create output file: ${outputFile}`);
  }

  // Set the environment variable for subsequent steps
  core.exportVariable("GITHUB_AW_SAFE_OUTPUTS", outputFile);
  console.log("Created agentic output file:", outputFile);

  // Also set as step output for reference
  core.setOutput("output_file", outputFile);
}

main();
