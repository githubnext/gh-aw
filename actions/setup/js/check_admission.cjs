// @ts-check
"use strict";

const { performance } = require("node:perf_hooks");
const { randomInt } = require("node:crypto");
const { AdmissionError, FAILURE, readAdmissionConfig, buildAdmissionIdentity, parseAdmissionDecision, readBoundedBody } = require("./admission_protocol.cjs");

const OUTAGE = "ERR_ADMISSION: Admission service unavailable; continuing without a service decision.";

/**
 * The operation must honor its signal. The race also prevents a noncooperative
 * transport from supplying a late success after the monotonic deadline.
 * @template T
 * @param {(signal: AbortSignal) => Promise<T>} operation
 * @param {number} deadline
 * @param {() => number} now
 * @param {number} [maximum]
 * @returns {Promise<T>}
 */
async function bounded(operation, deadline, now, maximum = 10000) {
  const remaining = Math.min(maximum, deadline - now());
  if (remaining <= 0) throw new AdmissionError(true);
  const controller = new AbortController();
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let timer;
  try {
    const result = await Promise.race([
      operation(controller.signal),
      new Promise((_, reject) => {
        timer = setTimeout(() => {
          controller.abort();
          reject(new AdmissionError(true));
        }, remaining);
      }),
    ]);
    if (now() >= deadline) throw new AdmissionError(true);
    return result;
  } finally {
    clearTimeout(timer);
    controller.abort();
  }
}

/** @param {Response} response */
function discard(response) {
  void response.body?.cancel().catch(() => {});
}

/** @param {number} status */
function isTransient(status) {
  return status === 408 || status === 429 || (status >= 500 && status <= 599);
}

/**
 * @typedef {{traceId: string, parentSpanId: string | undefined, operationSpanId: string, attempts: Array<{spanId: string, startMs: number, endMs: number}>, startMs: number, endMs: number, endpoints: ReturnType<typeof import("./send_otlp_span.cjs").parseOTLPEndpoints>}} AdmissionTrace
 * @typedef {{result: "skipped"} | ({result: "ignored"} & AdmissionTrace) | ({result: "admitted"} & ReturnType<typeof parseAdmissionDecision> & AdmissionTrace)} AdmissionResult
 */

/**
 * @param {{env?: NodeJS.ProcessEnv, fetchImpl?: typeof fetch, now?: () => number, jitter?: () => number, mask?: (token: string) => void}} [options]
 * @returns {Promise<AdmissionResult>}
 */
async function checkAdmission({ env = process.env, fetchImpl = fetch, now = () => performance.now(), jitter = () => randomInt(250), mask = () => {} } = {}) {
  const config = readAdmissionConfig(env);
  if (!config) return { result: "skipped" };
  const body = buildAdmissionIdentity(env);
  const requestUrl = env.ACTIONS_ID_TOKEN_REQUEST_URL;
  const bootstrapToken = env.ACTIONS_ID_TOKEN_REQUEST_TOKEN;
  if (!requestUrl || !bootstrapToken) throw new AdmissionError();
  let oidcUrl;
  try {
    oidcUrl = new URL(requestUrl);
  } catch {
    throw new AdmissionError();
  }
  if (oidcUrl.protocol !== "https:" || oidcUrl.username || oidcUrl.password || oidcUrl.hash) throw new AdmissionError();
  oidcUrl.searchParams.set("audience", config.audience);

  const { generateTraceId, generateSpanId, isValidTraceId, isValidSpanId, parseOTLPEndpoints, buildTraceparent } = require("./send_otlp_span.cjs");
  const endpoints = parseOTLPEndpoints();
  const existingTraceId = env.GITHUB_AW_OTEL_TRACE_ID || "";
  const traceId = isValidTraceId(existingTraceId) ? existingTraceId : generateTraceId();
  const parentSpanId = isValidTraceId(env.GITHUB_AW_OTEL_TRACE_ID || "") && isValidSpanId(env.GITHUB_AW_OTEL_PARENT_SPAN_ID || "") ? env.GITHUB_AW_OTEL_PARENT_SPAN_ID : undefined;
  const operationSpanId = generateSpanId();
  const startMs = Date.now();
  const deadline = now() + 60000;
  let token = "";
  let expiresAt = 0;
  /** @type {Array<{spanId: string, startMs: number, endMs: number}>} */
  const attempts = [];
  /** @type {{maxAiCredits: number, threatDetectionMaxAiCredits?: number, evalsMaxAiCredits?: number} | undefined} */
  let decision;
  try {
    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        if (attempt > 0) {
          await bounded(
            signal =>
              new Promise((resolve, reject) => {
                const timer = setTimeout(resolve, 250 * 2 ** (attempt - 1) + jitter());
                signal.addEventListener(
                  "abort",
                  () => {
                    clearTimeout(timer);
                    reject(new AdmissionError(true));
                  },
                  { once: true }
                );
              }),
            deadline,
            now
          );
        }
        if (!token || expiresAt <= Date.now() + 10000) {
          token = await bounded(
            async signal => {
              const response = await fetchImpl(oidcUrl, { headers: { authorization: `bearer ${bootstrapToken}`, accept: "application/json" }, redirect: "manual", signal });
              if (response.status !== 200) {
                discard(response);
                throw new AdmissionError(isTransient(response.status));
              }
              let result;
              try {
                result = JSON.parse(await readBoundedBody(response, 65536));
              } catch (error) {
                if (error instanceof AdmissionError) throw error;
                if (error instanceof SyntaxError) throw new AdmissionError();
                throw new AdmissionError(true);
              }
              if (typeof result?.value !== "string" || !/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(result.value)) throw new AdmissionError();
              try {
                mask(result.value);
              } catch {
                throw new AdmissionError();
              }
              try {
                const payload = JSON.parse(Buffer.from(result.value.split(".")[1], "base64url").toString("utf8"));
                if (!Number.isFinite(payload.exp) || payload.exp * 1000 <= Date.now()) throw new AdmissionError();
                expiresAt = payload.exp * 1000;
              } catch {
                throw new AdmissionError();
              }
              return result.value;
            },
            deadline,
            now
          );
        }
        const spanId = generateSpanId();
        const attemptStartMs = Date.now();
        try {
          decision = await bounded(
            async signal => {
              /** @type {Record<string, string>} */
              const headers = { authorization: `bearer ${token}`, "content-type": "application/json", "cache-control": "no-store" };
              headers.traceparent = buildTraceparent(traceId, spanId);
              const response = await fetchImpl(config.endpoint, { method: "POST", headers, body, redirect: "manual", signal });
              if (response.status !== 200) {
                discard(response);
                throw new AdmissionError(isTransient(response.status));
              }
              if (!/^application\/json(?:\s*;\s*charset\s*=\s*(?:"utf-8"|utf-8))?\s*$/i.test(response.headers.get("content-type") || "")) {
                discard(response);
                throw new AdmissionError();
              }
              return parseAdmissionDecision(await readBoundedBody(response, 4096));
            },
            deadline,
            now
          );
        } finally {
          attempts.push({ spanId, startMs: attemptStartMs, endMs: Date.now() });
        }
        break;
      } catch (error) {
        if (error instanceof AdmissionError && !error.transient) throw error;
        if (attempt === 2 || now() >= deadline) break;
      }
    }
  } finally {
    token = "";
  }
  const traceContext = { traceId, parentSpanId, operationSpanId, attempts, startMs, endMs: Date.now(), endpoints };
  if (!decision) {
    if (config.onOutage === "block") throw new AdmissionError();
    return { result: "ignored", ...traceContext };
  }
  return { result: "admitted", ...decision, ...traceContext };
}

/** @param {{setOutput: (key: string, value: string) => void, exportVariable: (key: string, value: string) => void, setSecret: (value: string) => void, warning: (value: string) => void, setFailed: (value: string) => void}} core */
async function main(core) {
  try {
    const decision = await checkAdmission({ mask: token => core.setSecret(token) });
    if (decision.result === "ignored") core.warning(OUTAGE);
    if (decision.result === "admitted") {
      core.setOutput("max-ai-credits", String(decision.maxAiCredits));
      if (decision.threatDetectionMaxAiCredits !== undefined) core.setOutput("max-ai-credits-threat-detection", String(decision.threatDetectionMaxAiCredits));
      if (decision.evalsMaxAiCredits !== undefined) core.setOutput("max-ai-credits-evals", String(decision.evalsMaxAiCredits));
    }
    if (decision.result !== "skipped") {
      core.exportVariable("GITHUB_AW_OTEL_TRACE_ID", decision.traceId);
      core.setOutput("trace-id", decision.traceId);
      if (decision.endpoints.length) {
        const { buildOTLPPayload, buildGitHubActionsResourceAttributes, sendOTLPToAllEndpoints } = require("./send_otlp_span.cjs");
        const resourceAttributes = buildGitHubActionsResourceAttributes({ repository: process.env.GITHUB_REPOSITORY || "", runId: process.env.GITHUB_RUN_ID || "", staged: false });
        for (const attempt of decision.attempts) {
          const payload = buildOTLPPayload({
            ...attempt,
            traceId: decision.traceId,
            parentSpanId: decision.operationSpanId,
            spanName: "gh-aw.admission.http",
            serviceName: "gh-aw",
            kind: 3,
            attributes: [],
            resourceAttributes,
          });
          await sendOTLPToAllEndpoints(decision.endpoints, payload, { maxRetries: 0 });
        }
        await sendOTLPToAllEndpoints(
          decision.endpoints,
          buildOTLPPayload({
            traceId: decision.traceId,
            spanId: decision.operationSpanId,
            parentSpanId: decision.parentSpanId,
            startMs: decision.startMs,
            endMs: decision.endMs,
            spanName: "gh-aw.admission",
            serviceName: "gh-aw",
            attributes: [],
            resourceAttributes,
          }),
          { maxRetries: 0 }
        );
      }
    }
    core.setOutput("result", decision.result);
  } catch {
    core.setFailed(FAILURE);
  }
}

module.exports = { checkAdmission, main, bounded, OUTAGE };
