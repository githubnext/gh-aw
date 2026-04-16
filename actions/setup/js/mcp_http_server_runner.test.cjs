// @ts-check
import { afterEach, describe, it, expect, vi } from "vitest";
import http from "http";
import { runHttpServer } from "./mcp_http_server_runner.cjs";

/**
 * Helper: create a minimal mock transport that records requests
 */
function makeMockTransport() {
  return {
    /** @type {Array<{req: unknown, res: unknown, body: unknown}>} */
    calls: [],
    /** @param {unknown} req @param {unknown} res @param {unknown} body */
    async handleRequest(req, res, body) {
      this.calls.push({ req, res, body });
      /** @type {any} */ res.writeHead(200, { "Content-Type": "application/json" });
      /** @type {any} */ res.end(JSON.stringify({ jsonrpc: "2.0", id: 1, result: {} }));
    },
  };
}

/**
 * Helper: create a minimal mock logger (silent)
 */
function makeMockLogger() {
  return {
    debug: vi.fn(),
    debugError: vi.fn(),
  };
}

/**
 * Perform an HTTP request against a running server and collect the response.
 *
 * @param {http.Server} server
 * @param {Object} opts
 * @param {string} opts.method
 * @param {string} opts.path
 * @param {string} [opts.body]
 * @param {Record<string,string>} [opts.headers]
 * @returns {Promise<{statusCode: number, headers: http.IncomingHttpHeaders, body: string}>}
 */
function request(server, { method, path, body, headers = {} }) {
  return new Promise((resolve, reject) => {
    const addr = /** @type {import('net').AddressInfo} */ server.address();
    const req = http.request(
      {
        hostname: "127.0.0.1",
        port: addr.port,
        path,
        method,
        headers: { "Content-Type": "application/json", ...headers },
      },
      res => {
        let data = "";
        res.on("data", c => (data += c));
        res.on("end", () => resolve({ statusCode: res.statusCode || 0, headers: res.headers, body: data }));
      }
    );
    req.on("error", reject);
    if (body) req.write(body);
    req.end();
  });
}

describe("mcp_http_server_runner.cjs - runHttpServer", () => {
  /** @type {http.Server | null} */
  let currentServer = null;

  afterEach(async () => {
    if (currentServer) {
      // Close all keep-alive connections before stopping the server to prevent
      // lingering open sockets from keeping the Node.js event loop alive.
      currentServer.closeAllConnections();
      await new Promise(r => currentServer.close(r));
      currentServer = null;
    }
  });

  it("sets CORS headers on every response", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
    });

    // server.listen(0) picks a random port
    await new Promise(r => currentServer.once("listening", r));

    const res = await request(currentServer, { method: "POST", path: "/", body: '{"jsonrpc":"2.0","id":1,"method":"ping"}' });
    expect(res.headers["access-control-allow-origin"]).toBe("*");
    expect(res.headers["access-control-allow-methods"]).toContain("POST");
  });

  it("responds 200 to OPTIONS preflight without calling transport", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
    });
    await new Promise(r => currentServer.once("listening", r));

    const res = await request(currentServer, { method: "OPTIONS", path: "/" });
    expect(res.statusCode).toBe(200);
    expect(transport.calls).toHaveLength(0);
  });

  it("responds to GET /health with payload from getHealthPayload", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok", server: "test-server", version: "2.0.0", tools: 42 }),
      logger,
      serverLabel: "Test",
    });
    await new Promise(r => currentServer.once("listening", r));

    const res = await request(currentServer, { method: "GET", path: "/health" });
    expect(res.statusCode).toBe(200);
    const payload = JSON.parse(res.body);
    expect(payload.status).toBe("ok");
    expect(payload.server).toBe("test-server");
    expect(payload.version).toBe("2.0.0");
    expect(payload.tools).toBe(42);
  });

  it("responds 405 for non-POST methods other than GET /health and OPTIONS", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
    });
    await new Promise(r => currentServer.once("listening", r));

    const res = await request(currentServer, { method: "PUT", path: "/" });
    expect(res.statusCode).toBe(405);
    const body = JSON.parse(res.body);
    expect(body.error).toBe("Method not allowed");
  });

  it("responds 400 with JSON-RPC error for invalid JSON body", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
    });
    await new Promise(r => currentServer.once("listening", r));

    const res = await request(currentServer, { method: "POST", path: "/", body: "{ not valid json" });
    expect(res.statusCode).toBe(400);
    const body = JSON.parse(res.body);
    expect(body.jsonrpc).toBe("2.0");
    expect(body.error.code).toBe(-32700);
    expect(body.error.message).toContain("Parse error");
  });

  it("delegates valid POST requests to transport.handleRequest with parsed body", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
    });
    await new Promise(r => currentServer.once("listening", r));

    const payload = { jsonrpc: "2.0", id: 5, method: "tools/list" };
    await request(currentServer, { method: "POST", path: "/", body: JSON.stringify(payload) });
    expect(transport.calls).toHaveLength(1);
    expect(transport.calls[0].body).toEqual(payload);
  });

  it("calls configureServer callback with the http.Server instance before binding", async () => {
    const transport = makeMockTransport();
    const logger = makeMockLogger();
    let capturedServer = null;
    currentServer = await runHttpServer({
      transport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
      configureServer: s => {
        capturedServer = s;
        s.timeout = 0;
      },
    });
    await new Promise(r => currentServer.once("listening", r));

    expect(capturedServer).toBe(currentServer);
    expect(currentServer.timeout).toBe(0);
  });

  it("returns 500 when transport.handleRequest throws", async () => {
    const logger = makeMockLogger();
    const throwingTransport = {
      async handleRequest(_req, _res, _body) {
        throw new Error("boom");
      },
    };
    currentServer = await runHttpServer({
      transport: throwingTransport,
      port: 0,
      getHealthPayload: () => ({ status: "ok" }),
      logger,
      serverLabel: "Test",
    });
    await new Promise(r => currentServer.once("listening", r));

    const res = await request(currentServer, { method: "POST", path: "/", body: '{"jsonrpc":"2.0","id":1,"method":"ping"}' });
    expect(res.statusCode).toBe(500);
    const body = JSON.parse(res.body);
    expect(body.error.code).toBe(-32603);
  });
});

describe("mcp_http_server_runner.cjs - logStartupError", () => {
  it("re-throws the error after logging", () => {
    const { logStartupError } = require("./mcp_http_server_runner.cjs");
    const err = new Error("test-error");
    expect(() => logStartupError(err, "test-ns", { Port: 3000 })).toThrow("test-error");
  });
});
