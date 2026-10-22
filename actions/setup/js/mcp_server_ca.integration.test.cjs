import fs from "fs";
import https from "https";
import net from "net";
import os from "os";
import path from "path";
import { execFileSync, spawn } from "child_process";
import { once } from "events";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { prepareMCPServerCAs } from "./mcp_server_ca.cjs";
import { extractOptionalServerNames } from "./start_mcp_gateway.cjs";

// Run against a native build of the pinned gateway with GH_AW_MCPG_BINARY=/path/to/awmg.
// Native tests use the same mount source/target mapping without requiring Docker.
describe.skipIf(!process.env.GH_AW_MCPG_BINARY)("per-server CA TLS against MCP Gateway", () => {
  let dir;
  let servers;
  let children;
  beforeEach(() => {
    dir = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "mcp-ca-tls-")));
    servers = [];
    children = [];
  });
  afterEach(async () => {
    for (const child of children) {
      if (child.exitCode === null && child.signalCode === null) {
        const exited = once(child, "exit");
        child.kill("SIGKILL");
        await exited;
      }
    }
    for (const server of servers) {
      server.closeAllConnections();
      await new Promise(resolve => server.close(resolve));
    }
    fs.rmSync(dir, { recursive: true, force: true });
  });

  async function backend(name, subjectAltName = "IP:127.0.0.1") {
    const cert = path.join(dir, `${name}.pem`);
    const key = path.join(dir, `${name}.key`);
    execFileSync(
      "openssl",
      ["req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1", "-keyout", key, "-out", cert, "-subj", `/CN=${name}`, "-addext", `subjectAltName=${subjectAltName}`, "-addext", "basicConstraints=critical,CA:TRUE"],
      {
        stdio: "ignore",
      }
    );
    let initialized = 0;
    const server = https.createServer({ key: fs.readFileSync(key), cert: fs.readFileSync(cert) }, (req, res) => {
      if (req.method !== "POST") {
        res.writeHead(405).end();
        return;
      }
      let body = "";
      req.on("data", chunk => {
        body += chunk;
      });
      req.on("end", () => {
        const message = JSON.parse(body);
        if (!("id" in message)) {
          res.writeHead(202).end();
          return;
        }
        let result = {};
        if (message.method === "initialize") {
          initialized += 1;
          result = { protocolVersion: message.params.protocolVersion, capabilities: { tools: {} }, serverInfo: { name, version: "1" } };
        } else if (message.method === "tools/list") {
          result = { tools: [{ name: "hello", description: "Hello", inputSchema: { type: "object", properties: {} } }] };
        }
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ jsonrpc: "2.0", id: message.id, result }));
      });
    });
    servers.push(server);
    await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
    return { cert, url: `https://127.0.0.1:${server.address().port}/mcp`, initialized: () => initialized };
  }

  async function gateway(mcpServers) {
    const reservation = net.createServer();
    await new Promise(resolve => reservation.listen(0, "127.0.0.1", resolve));
    const port = reservation.address().port;
    await new Promise(resolve => reservation.close(resolve));
    const config = { mcpServers, gateway: { port, domain: "localhost", agentId: "ca-regression", startupTimeout: 3 } };
    const args = prepareMCPServerCAs(config, { GITHUB_WORKSPACE: dir, RUNNER_TEMP: dir });
    extractOptionalServerNames(config);
    // Emulate the read-only Docker bind mapping for a native gateway process.
    for (let i = 1; i < args.length; i += 2) {
      const match = /^type=bind,source=(.*),target=(.*),readonly$/.exec(args[i]);
      expect(match).not.toBeNull();
      for (const server of Object.values(config.mcpServers)) {
        if (server.caCertFile === match[2]) server.caCertFile = match[1];
      }
    }
    const child = spawn(process.env.GH_AW_MCPG_BINARY, ["--config-stdin", "--routed", "--listen", `127.0.0.1:${port}`, "--log-dir", path.join(dir, `logs-${port}`), "--payload-dir", path.join(dir, `payloads-${port}`)], {
      env: { ...process.env, HTTP_PROXY: "", HTTPS_PROXY: "", ALL_PROXY: "", http_proxy: "", https_proxy: "", all_proxy: "", NO_PROXY: "127.0.0.1,localhost" },
      stdio: ["pipe", "pipe", "pipe"],
    });
    children.push(child);
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", data => {
      stdout += data;
    });
    child.stderr.on("data", data => {
      stderr += data;
    });
    child.stdin.end(JSON.stringify(config));
    return { child, stdout: () => stdout, stderr: () => stderr };
  }

  it("connects to two private CAs without leaking trust to an unconfigured or differently configured server", async () => {
    const first = await backend("first");
    const second = await backend("second");
    const wrongHostname = await backend("wrong-hostname", "DNS:mcp.invalid");
    const running = await gateway({
      first: { type: "http", url: first.url, caCertFile: first.cert },
      second: { type: "http", url: second.url, caCertFile: second.cert },
      wrong: { type: "http", url: first.url, caCertFile: second.cert, required: false, connectTimeout: 1 },
      unconfigured: { type: "http", url: first.url, required: false, connectTimeout: 1 },
      wrongHostname: { type: "http", url: wrongHostname.url, caCertFile: wrongHostname.cert, required: false, connectTimeout: 1 },
    });
    await expect.poll(() => running.stdout() + running.stderr(), { timeout: 15000 }).toContain('"mcpServers"');
    expect(first.initialized()).toBe(1);
    expect(second.initialized()).toBe(1);
    expect(wrongHostname.initialized()).toBe(0);
    expect(running.stderr()).toContain("certificate signed by unknown authority");
    expect(running.stderr()).toMatch(/doesn't contain any IP SANs|not valid for/);
    expect(running.stdout()).not.toContain("caCertFile");
    expect(running.child.exitCode).toBeNull();
  }, 20000);

  it.each(["malformed", "empty", "private-key"])(
    "rejects %s PEM without including contents in diagnostics",
    async kind => {
      const first = await backend("first");
      const file = path.join(dir, "invalid.pem");
      const contents = kind === "private-key" ? fs.readFileSync(path.join(dir, "first.key"), "utf8") : kind === "empty" ? "" : "certificate-content-must-not-be-logged";
      fs.writeFileSync(file, contents);
      const running = await gateway({ invalid: { type: "http", url: first.url, caCertFile: file, connectTimeout: 1 } });
      await expect.poll(() => running.stderr(), { timeout: 15000 }).toMatch(/custom CA certificate file/);
      expect(first.initialized()).toBe(0);
      expect(running.stderr()).toMatch(/custom CA certificate file/);
      if (contents) expect(running.stderr()).not.toContain(contents.trim());
    },
    20000
  );
});
