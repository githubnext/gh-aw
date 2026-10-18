"use strict";

const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const http = require("node:http");
const dns = require("node:dns").promises;
const { spawnSync } = require("node:child_process");

const CHECKS = ["allowed-api", "allowed-web", "denied-domain", "dns", "home-read", "home-write", "agent-write", "localhost"];

function curl(url) {
  const result = spawnSync("curl", ["--silent", "--show-error", "--fail", "--max-time", "10", "--output", "/dev/null", "--write-out", "%{http_connect} %{http_code}", url], {
    encoding: "utf8",
    timeout: 15000,
    maxBuffer: 4096,
  });
  const match = /^(\d{3}) (\d{3})$/.exec(result.stdout || "");
  return { exitCode: result.status, connectStatus: match ? Number(match[1]) : null, httpStatus: match ? Number(match[2]) : null };
}

function networkCheck(name, result, denied = false) {
  let status = "INCONCLUSIVE";
  let detail = "Request did not establish either access or an explicit proxy denial";
  if (denied) {
    if (result.connectStatus === 403 && result.httpStatus === 0 && Number.isInteger(result.exitCode) && result.exitCode > 0) {
      status = "PASS";
      detail = "Proxy rejected CONNECT to the non-allowed domain";
    } else if (Number.isInteger(result.httpStatus) && result.httpStatus > 0) {
      status = "FAIL";
      detail = "Observed an HTTP response from the non-allowed domain";
    }
  } else if (result.exitCode === 0 && Number.isInteger(result.httpStatus) && result.httpStatus >= 200 && result.httpStatus < 400) {
    status = "PASS";
    detail = "Allowed domain returned a successful HTTP response";
  } else if (result.connectStatus === 403 || (Number.isInteger(result.httpStatus) && result.httpStatus > 0)) {
    status = "FAIL";
    detail = "Allowed domain did not return a successful HTTP response";
  }
  return { name, status, detail, exitCode: result.exitCode, connectStatus: result.connectStatus, httpStatus: result.httpStatus };
}

function fileRoundTrip(directory) {
  const temporary = fs.mkdtempSync(path.join(directory, ".firewall-policy-"));
  const marker = path.join(temporary, "marker");
  try {
    fs.writeFileSync(marker, "firewall-policy-check", { flag: "wx", mode: 0o600 });
    if (fs.readFileSync(marker, "utf8") !== "firewall-policy-check") throw new Error("Marker mismatch");
  } finally {
    if (fs.existsSync(marker)) fs.unlinkSync(marker);
    fs.rmdirSync(temporary);
  }
}

async function localhostCheck() {
  const server = http.createServer((_request, response) => response.end("firewall-policy-check"));
  try {
    await new Promise((resolve, reject) => {
      server.once("error", reject);
      server.listen(0, "127.0.0.1", resolve);
    });
    await new Promise((resolve, reject) => {
      const request = http.get({ hostname: "127.0.0.1", port: server.address().port, path: "/", timeout: 5000 }, response => {
        let body = "";
        response.setEncoding("utf8");
        response.on("data", chunk => (body += chunk));
        response.on("error", reject);
        response.on("end", () => (response.statusCode === 200 && body === "firewall-policy-check" ? resolve() : reject(new Error("Local response mismatch"))));
      });
      request.on("timeout", () => request.destroy(new Error("Local request timed out")));
      request.on("error", reject);
    });
  } finally {
    if (server.listening) await new Promise(resolve => server.close(resolve));
  }
}

async function lookup() {
  let timer;
  try {
    await Promise.race([
      dns.lookup("github.com"),
      new Promise((_resolve, reject) => {
        timer = setTimeout(() => reject(new Error("DNS timed out")), 10000);
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
}

function outcome(checks) {
  if (checks.some(check => check.status === "FAIL")) return "FAIL";
  return checks.every(check => check.status === "PASS") ? "PASS" : "INCONCLUSIVE";
}

async function runChecks({ request = curl, resolveDNS = lookup, home = os.homedir(), agent = "/tmp/gh-aw/agent", hostname = "/etc/hostname", local = localhostCheck } = {}) {
  const checks = [networkCheck("allowed-api", request("https://api.github.com")), networkCheck("allowed-web", request("https://github.com")), networkCheck("denied-domain", request("https://example.com"), true)];
  for (const [name, operation] of [
    ["dns", resolveDNS],
    [
      "home-read",
      () => {
        fs.readdirSync(home);
        if (!fs.readFileSync(hostname, "utf8").trim()) throw new Error("Empty hostname");
      },
    ],
    ["home-write", () => fileRoundTrip(home)],
    ["agent-write", () => fileRoundTrip(agent)],
    ["localhost", local],
  ]) {
    try {
      await operation();
      checks.push({ name, status: "PASS", detail: "Assertion completed" });
    } catch (error) {
      checks.push({ name, status: name === "dns" ? "INCONCLUSIVE" : "FAIL", detail: `Assertion failed (${typeof error.code === "string" && /^[A-Z0-9_]+$/.test(error.code) ? error.code : "CHECK_ERROR"})` });
    }
  }
  return { scope: "bounded-firewall-policy", status: outcome(checks), checks };
}

function verifyReport(report) {
  if (report?.scope !== "bounded-firewall-policy" || report.status !== "PASS" || !Array.isArray(report.checks) || report.checks.length !== CHECKS.length) {
    throw new Error("Firewall validation failed or is incomplete; no passing evidence");
  }
  for (const [index, name] of CHECKS.entries()) {
    if (report.checks[index]?.name !== name || report.checks[index].status !== "PASS") throw new Error(`Missing passing assertion: ${name}`);
  }
  for (const index of [0, 1, 2]) {
    if (networkCheck(CHECKS[index], report.checks[index], index === 2).status !== "PASS") throw new Error(`Invalid network evidence: ${CHECKS[index]}`);
  }
}

async function main(args) {
  if (args.length === 2 && args[0] === "--verify") {
    verifyReport(JSON.parse(fs.readFileSync(args[1], "utf8")));
    console.log("PASS: all eight bounded firewall assertions completed; this is not a sandbox security assessment");
    return 0;
  }
  if (args.length !== 1 || args[0].startsWith("--")) throw new Error("Usage: node scripts/firewall-policy-check.cjs [--verify] REPORT.json");
  const report = await runChecks();
  fs.writeFileSync(args[0], JSON.stringify(report, null, 2) + "\n", { flag: "wx", mode: 0o600 });
  console.log(JSON.stringify(report));
  return report.status === "PASS" ? 0 : report.status === "FAIL" ? 1 : 2;
}

module.exports = { CHECKS, networkCheck, fileRoundTrip, localhostCheck, runChecks, verifyReport };
if (require.main === module) {
  main(process.argv.slice(2))
    .then(code => {
      process.exitCode = code;
    })
    .catch(() => {
      console.error("Firewall validation failed or is incomplete: unable to execute checks or verify evidence");
      process.exitCode = 1;
    });
}
