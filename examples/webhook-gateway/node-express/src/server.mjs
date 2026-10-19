import express from "express";
import { authenticate, deliveryIdentity, header } from "./auth.mjs";
import { approvalFingerprint, definitions, validateConfiguration } from "./contracts.mjs";
import { envelope, GatewayError, object, parseJSON, requireThat } from "./core.mjs";
import { dispatchApproved } from "./github.mjs";
import { project } from "./matching.mjs";

export function createGateway({ store, loadConfiguration, env = process.env, now = Date.now, fetchImpl = fetch, testHTTP = false, audit = event => process.stderr.write(`${JSON.stringify(event)}\n`) }) {
  const current = () => validateConfiguration(loadConfiguration(), env, { testHTTP });
  current();
  const app = express();
  app.disable("x-powered-by");
  app.set("trust proxy", false);
  app.get("/health", (_req, res) => {
    try {
      current();
      store.db.prepare("SELECT 1").get();
      res.json({ ready: true });
    } catch {
      audit({ event: "readiness_failed", code: "configuration-or-storage-unavailable" });
      res.status(503).json({ ready: false });
    }
  });
  app.post(
    "/ingress/:sourceId",
    (req, _res, next) => {
      try {
        requireThat(req.socket.encrypted || (testHTTP && ["127.0.0.1", "::1", "::ffff:127.0.0.1"].includes(req.socket.remoteAddress)), "https-required", 400);
        requireThat(!req.headers["content-encoding"] || req.headers["content-encoding"] === "identity", "compressed-webhooks-unsupported", 415);
        next();
      } catch (error) {
        next(error);
      }
    },
    express.raw({ type: "application/json", limit: "1mb", inflate: false }),
    (req, res, next) => {
      try {
        requireThat(Buffer.isBuffer(req.body), "json-content-type-required", 415);
        const configuration = current();
        const source = configuration.sources.find(item => item.id === req.params.sourceId);
        requireThat(source, "source-not-enrolled", 403);
        authenticate(req, source, env, req.body);
        requireThat(source.approved, "source-not-approved", 403);
        const payload = parseJSON(req.body);
        requireThat(object(payload), "webhook-object-required");
        const nativeEvent = source.provider === "github" ? header(req, "x-github-event") : undefined;
        if (source.provider === "github")
          requireThat(
            Object.entries(definitions).some(([event, definition]) => !definition.provider && (definition.native_event ?? event) === nativeEvent),
            "unsupported-native-event",
            422
          );
        const identity = deliveryIdentity(req, source, payload, now());
        const jobs = [];
        for (const destination of configuration.destinations) {
          if (!destination.approved || !destination.gateVerified || !destination.sourceIds.includes(source.id)) continue;
          for (const subscription of destination.subscriptions) {
            const projected = project(subscription, source, payload, nativeEvent, configuration.github);
            if (projected) jobs.push({ destination: destination.id, approval: approvalFingerprint(configuration, source, destination), request: envelope(subscription, identity.id, projected) });
          }
        }
        const result = store.admit(source.id, identity.id, identity.digest, jobs);
        res.status(200).json(result);
      } catch (error) {
        next(error);
      }
    }
  );
  app.use((_req, res) => res.status(404).json({ error: "route-not-found" }));
  app.use((error, _req, res, _next) => {
    const known = error instanceof GatewayError;
    const code = known ? error.code : error.type === "entity.too.large" ? "webhook-too-large" : "ingress-internal-failed";
    const status = known ? error.status : error.type === "entity.too.large" ? 413 : 503;
    audit({ event: "ingress_rejected", code });
    res.status(status).json({ error: code });
  });

  async function drainOne() {
    const job = store.claim(now());
    if (!job) return false;
    try {
      const configuration = current();
      const destination = configuration.destinations.find(item => item.id === job.destination);
      const assertCurrent = () => {
        const live = current();
        const liveSource = live.sources.find(item => item.id === job.source);
        const liveDestination = live.destinations.find(item => item.id === job.destination);
        requireThat(
          liveSource?.approved && liveDestination?.approved && liveDestination.gateVerified && liveDestination.sourceIds.includes(job.source) && approvalFingerprint(live, liveSource, liveDestination) === job.approval,
          "admission-approval-no-longer-current",
          403
        );
      };
      assertCurrent();
      await dispatchApproved({ github: configuration.github, destination, request: job.request, env, now, fetchImpl, assertCurrent });
      store.finish(job, "dispatched");
      audit({ event: "dispatch_acknowledged", job: job.id });
    } catch (error) {
      const safe = error instanceof GatewayError ? error : new GatewayError("dispatch-internal-failed", 500);
      let state = "failed";
      if (safe.code === "installation-token-revocation-failed") state = safe.dispatched ? "dispatched_cleanup_failed" : safe.uncertain ? "uncertain_cleanup_failed" : "cleanup_failed";
      else if (safe.uncertain) state = "uncertain";
      else if (safe.status === 403) state = "blocked";
      else if (safe.retryable && job.attempts < 3) state = "pending";
      store.finish(job, state, safe.code, state === "pending" ? now() + 1000 * 2 ** job.attempts : 0);
      audit({ event: "dispatch_not_completed", job: job.id, state, code: safe.code });
    }
    return true;
  }
  return { app, drainOne };
}
