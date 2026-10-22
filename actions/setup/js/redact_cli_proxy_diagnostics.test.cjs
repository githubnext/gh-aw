// @ts-check
const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { MAX_DIAGNOSTIC_BYTES, readCliProxyDiagnostics, redactCliProxyDiagnostics } = require("./redact_cli_proxy_diagnostics.cjs");

describe("CLI proxy console diagnostic sanitizer", () => {
  const root = path.join(__dirname, `.redact-cli-proxy-diagnostics-${process.pid}`);
  const input = path.join(root, "diagnostics.log");
  const script = path.join(__dirname, "redact_cli_proxy_diagnostics.cjs");

  beforeAll(() => fs.mkdirSync(root));
  afterAll(() => fs.rmSync(root, { recursive: true, force: true }));
  afterEach(() => vi.restoreAllMocks());

  function publish(content, env = {}) {
    fs.writeFileSync(input, content);
    return spawnSync(process.execPath, [script, input], {
      encoding: "utf8",
      env: { PATH: process.env.PATH, ...env },
    });
  }

  it("normalizes NUL bytes before matching environment credentials and collecting masks", () => {
    const secret = "synthetic-proxy-credential";
    const output = redactCliProxyDiagnostics(`synthetic-proxy-\0credential\n::add-\0mask::runtime-\0credential\nruntime-credential\nruntime-\0credential`, { GH_TOKEN: secret });
    expect(output).not.toContain("\0");
    expect(output).not.toContain(secret);
    expect(output).not.toContain("runtime-credential");
    expect(output).not.toContain("::add-mask::");
  });

  it.each([
    ['synthetic"password', 'token="synthetic"password"'],
    ['synthetic"password', 'token=synthetic\\"password'],
    ['synthetic"password', 'token="synthetic\\"password"'],
    ['synthetic"password', "token=synthetic\\u0022password"],
    ['synthetic"\\password', "token=synthetic\\u0022\\u005cpassword"],
  ])("preserves physical lines when redacting %s from %s", (secret, spelling) => {
    const content = Array.from({ length: 32 }, (_, index) => `line-${index + 1}: ${spelling}`).join("\n") + "\n";
    const redacted = redactCliProxyDiagnostics(content, { GH_TOKEN: secret });
    expect(redacted.trimEnd().split("\n")).toHaveLength(32);
    expect(redacted).not.toContain(secret);
    expect(redacted).not.toContain("password");
    const result = publish(content, { GH_TOKEN: secret });
    expect(result.status).toBe(0);
    expect(result.stderr).toBe("");
    expect(result.stdout.split("\n")).toHaveLength(20);
    expect(result.stdout).toContain("line-13:");
    expect(result.stdout).toContain("line-32:");
    expect(result.stdout).not.toContain("line-12:");
    expect(result.stdout).not.toContain("password");
  });

  it("collects masks before truncation and preserves built-in and supplied arbitrary-secret coverage", () => {
    const builtin = `ghs_${"a".repeat(40)}`;
    const content = `::add-mask::runtime-secret\n${"safe line\n".repeat(30)}runtime-secret ${builtin} x$ first-line second-line custom-value\n`;
    const result = publish(content, {
      GITHUB_TOKEN: "x$",
      GH_AW_SECRET_NAMES: "CUSTOM, MULTILINE",
      SECRET_CUSTOM: "custom-value",
      MULTILINE: " first-line \nsecond-line",
    });
    expect(result.status).toBe(0);
    expect(result.stdout.split("\n")).toHaveLength(20);
    for (const secret of ["runtime-secret", builtin, "x$", "first-line", "second-line", "custom-value"]) expect(result.stdout).not.toContain(secret);
    expect(result.stdout).toContain("safe line");
  });

  it("accepts the exact byte limit", () => {
    const result = publish("a".repeat(MAX_DIAGNOSTIC_BYTES));
    expect(result.status).toBe(0);
    expect(result.stdout).toHaveLength(MAX_DIAGNOSTIC_BYTES);
    expect(result.stderr).toBe("");
  });

  it("rejects an oversized file before reading it", () => {
    fs.writeFileSync(input, "a".repeat(MAX_DIAGNOSTIC_BYTES + 1));
    const read = vi.spyOn(fs, "readFileSync");
    expect(() => readCliProxyDiagnostics(input)).toThrow("Unable to read CLI proxy diagnostics");
    expect(read).not.toHaveBeenCalled();
  });

  it("counts UTF-8 bytes rather than characters and emits no partial diagnostics on failure", () => {
    const result = publish("é".repeat(MAX_DIAGNOSTIC_BYTES / 2 + 1));
    expect(result.status).toBe(1);
    expect(result.stdout).toBe("");
    expect(result.stderr).toBe("");
  });

  it("rejects non-regular files before reading", () => {
    const read = vi.spyOn(fs, "readFileSync");
    expect(() => readCliProxyDiagnostics(root)).toThrow("Unable to read CLI proxy diagnostics");
    expect(read).not.toHaveBeenCalled();
  });

  it("fails closed without leaking a missing input path", () => {
    const result = spawnSync(process.execPath, [script, path.join(root, "synthetic-credential")], { encoding: "utf8" });
    expect(result.status).toBe(1);
    expect(result.stdout).toBe("");
    expect(result.stderr).toBe("");
  });
});
