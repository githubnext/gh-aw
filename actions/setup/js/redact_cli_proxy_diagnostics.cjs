// @ts-check
const fs = require("node:fs");
const { redactArtifactContent } = require("./redact_secrets.cjs");
const { collectAddMaskedValues } = require("./add_mask_redaction.cjs");

/**
 * @param {string} content
 * @param {NodeJS.ProcessEnv} env
 * @returns {string}
 */
function redactCliProxyDiagnostics(content, env) {
  const names = new Set(["GH_TOKEN", "GITHUB_TOKEN", ...Object.keys(env).filter(name => name.startsWith("SECRET_"))]);
  for (const name of (env.GH_AW_SECRET_NAMES || "")
    .split(",")
    .map(name => name.trim())
    .filter(Boolean)) {
    names.add(name);
    names.add(`SECRET_${name}`);
  }
  const values = collectAddMaskedValues(content);
  for (const name of names) {
    const value = env[name];
    if (!value) continue;
    // Match short, whitespace-padded and individual multiline secret values too.
    values.push(
      value,
      ...value
        .split(/\r\n|\r|\n/)
        .flatMap(line => [line, line.trim()])
        .filter(Boolean)
    );
  }
  return redactArtifactContent(content, [], values).content;
}

if (require.main === module) {
  try {
    const content = fs.readFileSync(process.argv[2], "utf8");
    const redacted = redactCliProxyDiagnostics(content, process.env);
    process.stdout.write(redacted.trimEnd().split("\n").slice(-20).join("\n"));
  } catch {
    // The caller withholds both streams and emits a fixed, payload-free error.
    process.exitCode = 1;
  }
}

module.exports = { redactCliProxyDiagnostics };
