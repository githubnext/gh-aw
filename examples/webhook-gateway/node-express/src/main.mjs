import { readFileSync } from "node:fs";
import { createServer } from "node:https";
import { Store } from "./store.mjs";
import { createGateway } from "./server.mjs";
import { GatewayError, parseJSON, requireThat } from "./core.mjs";

process.umask(0o077);
let server;
let store;
try {
  requireThat(process.env.GATEWAY_CONFIG && process.env.GATEWAY_DATABASE && process.env.GATEWAY_TLS_CERT && process.env.GATEWAY_TLS_KEY, "config-database-and-tls-files-required", 503);
  store = new Store(process.env.GATEWAY_DATABASE);
  const gateway = createGateway({ store, loadConfiguration: () => parseJSON(readFileSync(process.env.GATEWAY_CONFIG)) });
  server = createServer({ cert: readFileSync(process.env.GATEWAY_TLS_CERT), key: readFileSync(process.env.GATEWAY_TLS_KEY), minVersion: "TLSv1.2", requestTimeout: 30000 }, gateway.app);
  const port = Number(process.env.PORT ?? 8443);
  requireThat(Number.isInteger(port) && port > 0 && port <= 65535, "invalid-listen-port");
  let stopping = false;
  let timer;
  let working = false;
  const work = async () => {
    if (stopping) return;
    working = true;
    try {
      await gateway.drainOne();
    } catch (error) {
      process.stderr.write(`${JSON.stringify({ event: "worker_failed", code: error instanceof GatewayError ? error.code : "durable-store-failed" })}\n`);
      process.exitCode = 1;
      stopping = true;
      server.close();
    } finally {
      working = false;
      if (!stopping) timer = setTimeout(work, 250);
      else store.close();
    }
  };
  const stop = () => {
    stopping = true;
    clearTimeout(timer);
    server.close(() => {
      if (!working) store.close();
    });
  };
  process.once("SIGTERM", stop);
  process.once("SIGINT", stop);
  server.on("error", () => {
    process.stderr.write('{"event":"server_failed","code":"listen-failed"}\n');
    process.exitCode = 1;
    stop();
  });
  server.listen(port, process.env.HOST ?? "127.0.0.1", () => {
    process.stderr.write('{"event":"server_ready"}\n');
    work();
  });
} catch (error) {
  process.stderr.write(`${JSON.stringify({ event: "startup_failed", code: error instanceof GatewayError ? error.code : "startup-configuration-unreadable" })}\n`);
  process.exitCode = 1;
  store?.close();
}
