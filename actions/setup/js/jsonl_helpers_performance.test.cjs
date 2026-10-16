import { describe, expect, it } from "vitest";
import { execFileSync } from "node:child_process";
import path from "node:path";

const helper = path.join(__dirname, "jsonl_helpers.cjs");
const budgetMs = 15000;

function measure(body, heapMb = 64) {
  const script = `
    const { performance } = require("node:perf_hooks");
    const { iterateJsonlEvents, streamJsonlEvents } = require(${JSON.stringify(helper)});
    (async () => {
      const start = performance.now();
      const result = await (async () => { ${body} })();
      process.stdout.write(JSON.stringify({ ...result, elapsedMs: performance.now() - start }));
    })().catch(error => { console.error(error); process.exitCode = 1; });
  `;
  return JSON.parse(execFileSync(process.execPath, [`--max-old-space-size=${heapMb}`, "-e", script], { encoding: "utf8", timeout: budgetMs + 5000 }));
}

describe("JSONL performance regressions", () => {
  it(
    "streams a million short records within a 64 MB heap and bounded time",
    () => {
      const result = measure(`
      const batch = (JSON.stringify({ id: 7, text: "x".repeat(64) }) + "\\n").repeat(256);
      function* chunks() {
        for (let index = 0; index < 4096; index++) yield batch;
      }
      let records = 0;
      let checksum = 0;
      for await (const event of streamJsonlEvents(chunks())) {
        if (event.type !== "json" || event.value.text.length !== 64) throw new Error("Unexpected record");
        records++;
        checksum += event.value.id;
      }
      return { records, checksum, bytes: Buffer.byteLength(batch) * 4096 };
    `);
      expect(result.records).toBe(1048576);
      expect(result.checksum).toBe(7 * 1048576);
      expect(result.bytes).toBeGreaterThan(64 * 1024 * 1024);
      expect(result.elapsedMs).toBeLessThan(budgetMs);
    },
    budgetMs + 10000
  );

  it(
    "frames valid and corrupt 8 MB records from small chunks without quadratic copying",
    () => {
      const result = measure(
        `
      const fragment = "x".repeat(256);
      function* chunks() {
        for (let record = 0; record < 4; record++) {
          yield '{"text":"';
          for (let index = 0; index < 32768; index++) yield fragment;
          yield record % 2 === 0 ? '"}\\r\\n' : '"}x\\n';
        }
        yield '{"last":true}';
      }
      let records = 0;
      let errors = 0;
      let last = false;
      let payloadBytes = 0;
      for await (const event of streamJsonlEvents(chunks())) {
        if (event.type === "invalid") {
          if (event.reason !== "syntax") throw new Error("Unexpected rejection");
          errors++;
        } else if (event.value.last) {
          last = true;
        } else {
          if (event.value.text.length !== 8 * 1024 * 1024) throw new Error("Truncated record");
          payloadBytes += event.value.text.length;
          records++;
        }
      }
      return { records, errors, last, payloadBytes };
    `,
        128
      );
      expect(result).toMatchObject({ records: 2, errors: 2, last: true, payloadBytes: 16 * 1024 * 1024 });
      expect(result.elapsedMs).toBeLessThan(budgetMs);
    },
    budgetMs + 10000
  );

  it(
    "prefilters sustained non-JSON noise and recovers from syntax errors with bounded resources",
    () => {
      const result = measure(`
      const originalParse = JSON.parse;
      let parseCalls = 0;
      JSON.parse = (...args) => { parseCalls++; return originalParse(...args); };
      try {
        const batch = 'diagnostic: not JSON\\n{"bad":}\\n{"id":3}\\n';
        const input = batch.repeat(100000);
        let records = 0;
        let errors = 0;
        let checksum = 0;
        let lastLine = 0;
        for (const event of iterateJsonlEvents(input)) {
          lastLine = event.lineNumber;
          if (event.type === "invalid") errors++;
          else { records++; checksum += event.value.id; }
        }
        const eventParseCalls = parseCalls;
        let callbacks = 0;
        let callbackRecords = 0;
        for (const event of iterateJsonlEvents(batch.repeat(10000), {
          onError(error) {
            if (error.message !== "Invalid JSONL record") throw new Error("Unexpected error");
            callbacks++;
          }
        })) {
          if (event.type === "json") callbackRecords++;
        }
        return { records, errors, checksum, lastLine, eventParseCalls, callbacks, callbackRecords };
      } finally {
        JSON.parse = originalParse;
      }
    `);
      expect(result).toMatchObject({ records: 100000, errors: 200000, checksum: 300000, lastLine: 300000, eventParseCalls: 200000 });
      expect(result.callbacks).toBe(20000);
      expect(result.callbackRecords).toBe(10000);
      expect(result.elapsedMs).toBeLessThan(budgetMs);
    },
    budgetMs + 10000
  );
});
