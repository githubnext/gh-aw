// @ts-check
const fs = require("node:fs");
const { PROXY_ENV_NAMES } = require("./constants.cjs");
const { redactArtifactContent } = require("./redact_secrets.cjs");
const { collectAddMaskedValues } = require("./add_mask_redaction.cjs");

const MAX_DIAGNOSTIC_BYTES = 1024 * 1024;

/**
 * @param {string} content
 * @param {NodeJS.ProcessEnv} env
 * @returns {string}
 */
function redactCliProxyDiagnostics(content, env, credentialPatterns = []) {
  // Bash command substitution removes NUL bytes; normalize before matching.
  content = content.replaceAll("\0", "");
  const names = new Set(["GH_TOKEN", "GITHUB_TOKEN", ...PROXY_ENV_NAMES, ...Object.keys(env).filter(name => name.startsWith("SECRET_"))]);
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
  // Artifact redaction may JSON-quote ambiguous text. Redact each physical line
  // separately so that representation cannot hide lines from the publication cap.
  return content
    .split("\n")
    .map(line => {
      const patternMatches = credentialPatterns.flatMap(({ pattern, replacement }) =>
        Array.from(line.matchAll(pattern), match => ({
          start: match.index,
          end: match.index + match[0].length,
          value: match[0],
          replacement: match[0].replace(pattern, replacement),
        }))
      );
      return redactArtifactContent(line, [], values, { patternMatches }).content;
    })
    .join("\n");
}

/**
 * @param {string} filePath
 * @returns {string}
 */
function readCliProxyDiagnostics(filePath) {
  try {
    const stat = fs.statSync(filePath);
    if (!stat.isFile() || stat.size > MAX_DIAGNOSTIC_BYTES) {
      throw new Error("CLI proxy diagnostics exceed the input limit or are not a regular file");
    }
    return fs.readFileSync(filePath, "utf8");
  } catch (error) {
    throw new Error("Unable to read CLI proxy diagnostics", { cause: error });
  }
}

if (require.main === module) {
  try {
    const content = readCliProxyDiagnostics(process.argv[2]);
    const redacted = redactCliProxyDiagnostics(content, process.env);
    process.stdout.write(redacted.trimEnd().split("\n").slice(-20).join("\n"));
  } catch {
    // The caller withholds both streams and emits a fixed, payload-free error.
    process.exitCode = 1;
  }
}

module.exports = { MAX_DIAGNOSTIC_BYTES, readCliProxyDiagnostics, redactCliProxyDiagnostics };
