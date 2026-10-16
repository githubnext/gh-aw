// @ts-check
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { redactCliProxyDiagnostics } = require("./redact_cli_proxy_diagnostics.cjs");

describe("CLI proxy diagnostic redaction", () => {
  let root;
  const custom = 'synthetic"\\credential%with-spaces';
  const builtin = `ghs_${"a".repeat(40)}`;
  const diagnostics = `connection refused: ${custom}\nbuiltin: ${builtin}\nretry with a different port\n`;

  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-cli-proxy-test-"));
  });

  afterEach(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  function executable(name, content) {
    fs.writeFileSync(path.join(root, name), `#!/bin/bash\n${content}\n`, { mode: 0o700 });
  }

  function runProxy(options = {}) {
    const source = fs.readFileSync(path.join(__dirname, "../sh/start_cli_proxy.sh"), "utf8");
    const tls = path.join(root, "tls");
    fs.writeFileSync(path.join(root, "start_cli_proxy.sh"), source.replaceAll("/tmp/gh-aw/difc-proxy-tls", tls).replaceAll("/tmp/gh-aw/mcp-logs", path.join(root, "mcp-logs")));
    fs.copyFileSync(path.join(__dirname, "../sh/proxy_env_lib.sh"), path.join(root, "proxy_env_lib.sh"));
    if (!options.missingSanitizer) fs.symlinkSync(path.join(__dirname, "redact_cli_proxy_diagnostics.cjs"), path.join(root, "redact_cli_proxy_diagnostics.cjs"));
    executable(
      "docker",
      `printf '%s\\n' "$*" >> "$TEST_CALLS"
case "$1" in
  run)
    printf '%s' "$TEST_DIAGNOSTICS"
    printf '%s' "$TEST_DIAGNOSTICS" >&2
    if [ "$TEST_RUN_FAILURE" = true ]; then exit 125; fi
    touch "$TEST_TLS_DIR/ca.crt"
    ;;
  logs)
    printf '%s' "$TEST_DIAGNOSTICS"
    printf '%s' "$TEST_DIAGNOSTICS" >&2
    exit 1
    ;;
  rm) printf 'removed container %s\\n' "$GH_TOKEN" ;;
esac`
    );
    executable("curl", '[ "$TEST_READY" = true ]');
    executable("sleep", ":");
    if (options.captureFailure) executable("mktemp", 'printf "%s" "$TEST_DIAGNOSTICS"; printf "%s" "$TEST_DIAGNOSTICS" >&2; exit 1');
    if (options.sanitizerFailure) {
      executable("node", 'printf "%s" "$TEST_DIAGNOSTICS"; printf "%s" "$TEST_DIAGNOSTICS" >&2; exit 1');
    } else if (options.redactorThrows) {
      const preload = path.join(root, "throw.cjs");
      fs.writeFileSync(preload, `require(${JSON.stringify(path.join(__dirname, "redact_secrets.cjs"))}).redactArtifactContent = () => { throw new Error(process.env.GH_TOKEN); };`);
      executable("node", `exec ${JSON.stringify(process.execPath)} --require ${JSON.stringify(preload)} "$@"`);
    }
    const result = spawnSync("bash", [path.join(root, "start_cli_proxy.sh")], {
      encoding: "utf8",
      env: {
        ...process.env,
        PATH: `${root}:${path.dirname(process.execPath)}:${process.env.PATH}`,
        TMPDIR: root,
        GH_TOKEN: custom,
        CLI_PROXY_IMAGE: options.noImage ? "" : "synthetic-proxy:latest",
        GITHUB_SERVER_URL: "https://github.com",
        TEST_DIAGNOSTICS: options.diagnostics || diagnostics,
        TEST_CALLS: path.join(root, "calls"),
        TEST_TLS_DIR: tls,
        TEST_READY: options.ready ? "true" : "false",
        TEST_RUN_FAILURE: options.runFailure ? "true" : "false",
      },
    });
    const output = result.stdout + result.stderr;
    expect(output).not.toContain(custom);
    expect(output).not.toContain(builtin);
    expect(fs.readdirSync(root).filter(name => name.startsWith("tmp."))).toEqual([]);
    return { ...result, output, calls: fs.existsSync(path.join(root, "calls")) ? fs.readFileSync(path.join(root, "calls"), "utf8") : "" };
  }

  it("redacts arbitrary supplied secrets, short values, multiline fragments and built-in credentials", () => {
    const env = {
      GH_TOKEN: custom,
      GITHUB_TOKEN: "short",
      GH_AW_SECRET_NAMES: " CUSTOM, MULTILINE ",
      SECRET_CUSTOM: "x$",
      MULTILINE: " first-line \nsecond-line\rthird-line",
    };
    const output = redactCliProxyDiagnostics(`${diagnostics}short x$ first-line second-line third-line`, env);
    for (const value of [custom, builtin, "short", "x$", "first-line", "second-line", "third-line"]) expect(output).not.toContain(value);
    expect(output).toContain("retry with a different port");
  });

  it("redacts escaped secret values and masks registered in diagnostics", () => {
    const output = redactCliProxyDiagnostics(`::add-mask::runtime-value\nruntime-value ${JSON.stringify(custom)}`, { GH_TOKEN: custom });
    expect(output).not.toContain("runtime-value");
    expect(output).not.toContain("::add-mask::");
    expect(output).not.toContain(JSON.stringify(custom).slice(1, -1));
  });

  it("redacts the complete log before limiting publication to twenty lines", () => {
    const result = runProxy({ diagnostics: `::add-mask::runtime-synthetic\n${"safe diagnostic\n".repeat(30)}runtime-synthetic\n${custom}\n${builtin}\n` });
    expect(result.status).toBe(1);
    expect(result.output).not.toContain("runtime-synthetic");
    expect(result.stdout.trimEnd().split("\n")).toHaveLength(20);
    expect(result.output).toContain("CLI proxy startup failed (exit status 1)");
  });

  it("preserves successful startup and only publishes sanitized output", () => {
    const result = runProxy({ ready: true });
    expect(result.status).toBe(0);
    expect(result.output).toContain("CLI proxy ready on port 18443");
    expect(result.output).toContain("connection refused: ***");
    expect(result.calls).not.toContain("logs awmg-cli-proxy");
  });

  it("sanitizes readiness failure logs, reports failure and removes the container", () => {
    const result = runProxy();
    expect(result.status).toBe(1);
    expect(result.output).toContain("CLI proxy failed to start within 30s");
    expect(result.output).toContain("retry with a different port");
    expect(result.calls).toContain("logs awmg-cli-proxy");
    expect(result.calls.match(/rm -f awmg-cli-proxy/g)).toHaveLength(2);
  });

  it("sanitizes docker run errors and preserves the original failure status", () => {
    const result = runProxy({ runFailure: true });
    expect(result.status).toBe(125);
    expect(result.output).toContain("connection refused: ***");
    expect(result.calls).not.toContain("logs awmg-cli-proxy");
  });

  it.each([false, true])("withholds partial sanitizer output on failure, even when ready=%s", ready => {
    const result = runProxy({ ready, sanitizerFailure: true });
    expect(result.status).toBe(1);
    expect(result.output).toBe("::error::CLI proxy diagnostics withheld because secret redaction failed\n");
  });

  it("withholds diagnostics when the shared redactor throws a secret-bearing exception", () => {
    const result = runProxy({ redactorThrows: true });
    expect(result.status).toBe(1);
    expect(result.output).toBe("::error::CLI proxy diagnostics withheld because secret redaction failed\n");
  });

  it("withholds diagnostics when the sanitizer cannot be loaded", () => {
    const result = runProxy({ missingSanitizer: true });
    expect(result.status).toBe(1);
    expect(result.output).toBe("::error::CLI proxy diagnostics withheld because secret redaction failed\n");
  });

  it("fails closed before starting Docker if private capture cannot be created", () => {
    const result = runProxy({ captureFailure: true });
    expect(result.status).toBe(1);
    expect(result.output).toBe("::error::CLI proxy diagnostics could not be captured safely\n");
    expect(result.calls).toBe("");
  });

  it("preserves the no-image skip", () => {
    const result = runProxy({ noImage: true });
    expect(result.status).toBe(0);
    expect(result.output).toContain("container image not specified, skipping proxy start");
    expect(result.calls).toBe("");
  });
});
