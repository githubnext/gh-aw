import { DatabaseSync } from "node:sqlite";
import { chmodSync } from "node:fs";
import { requireThat } from "./core.mjs";

export class Store {
  constructor(filename) {
    this.db = new DatabaseSync(filename);
    if (filename !== ":memory:") chmodSync(filename, 0o600);
    this.db.exec(`
      PRAGMA journal_mode=WAL;
      PRAGMA synchronous=FULL;
      PRAGMA busy_timeout=5000;
      CREATE TABLE IF NOT EXISTS receipts (
        source TEXT NOT NULL, delivery TEXT NOT NULL, digest TEXT NOT NULL,
        PRIMARY KEY(source,delivery), UNIQUE(source,digest)
      );
      CREATE TABLE IF NOT EXISTS jobs (
        id INTEGER PRIMARY KEY, source TEXT NOT NULL, destination TEXT NOT NULL,
        delivery TEXT NOT NULL, family TEXT NOT NULL, approval TEXT NOT NULL, request TEXT NOT NULL,
        state TEXT NOT NULL DEFAULT 'pending', attempts INTEGER NOT NULL DEFAULT 0,
        next_at INTEGER NOT NULL DEFAULT 0, error TEXT,
        UNIQUE(source,destination,delivery,family)
      );
    `);
  }

  transaction(operation) {
    this.db.exec("BEGIN IMMEDIATE");
    try {
      const result = operation();
      this.db.exec("COMMIT");
      return result;
    } catch (error) {
      this.db.exec("ROLLBACK");
      throw error;
    }
  }

  admit(source, delivery, digest, jobs) {
    return this.transaction(() => {
      const existing = this.db.prepare("SELECT digest FROM receipts WHERE source=? AND delivery=?").get(source, delivery);
      if (existing) {
        requireThat(existing.digest === digest, "conflicting-delivery-replay", 409);
        return { duplicate: true, admitted: 0 };
      }
      const replay = this.db.prepare("SELECT delivery FROM receipts WHERE source=? AND digest=?").get(source, digest);
      if (replay) return { duplicate: true, admitted: 0 };
      requireThat(
        this.db.prepare("SELECT count(*) AS n FROM receipts").get().n < 100000 &&
          this.db.prepare("SELECT count(*) AS n FROM jobs").get().n + jobs.length <= 100000 &&
          this.db.prepare("SELECT count(*) AS n FROM jobs WHERE state IN ('pending','in_flight')").get().n + jobs.length <= 10000,
        "durable-admission-capacity-exhausted",
        503
      );
      this.db.prepare("INSERT INTO receipts(source,delivery,digest) VALUES(?,?,?)").run(source, delivery, digest);
      for (const job of jobs) {
        this.db
          .prepare("INSERT INTO jobs(source,destination,delivery,family,approval,request) VALUES(?,?,?,?,?,?)")
          .run(source, job.destination, delivery, job.request.client_payload.aw_context.webhook.event, job.approval, JSON.stringify(job.request));
      }
      return { duplicate: false, admitted: jobs.length };
    });
  }

  claim(now) {
    return this.transaction(() => {
      const job = this.db.prepare("SELECT * FROM jobs WHERE state='pending' AND next_at<=? ORDER BY id LIMIT 1").get(now);
      if (!job) return undefined;
      this.db.prepare("UPDATE jobs SET state='in_flight',attempts=attempts+1 WHERE id=?").run(job.id);
      return { ...job, attempts: job.attempts + 1, request: JSON.parse(job.request) };
    });
  }

  finish(job, state, error = null, next = 0) {
    requireThat(["pending", "dispatched", "blocked", "failed", "uncertain", "cleanup_failed", "uncertain_cleanup_failed", "dispatched_cleanup_failed"].includes(state), "invalid-job-transition", 500);
    this.db.prepare("UPDATE jobs SET state=?,error=?,next_at=? WHERE id=? AND state='in_flight'").run(state, error, next, job.id);
  }

  status() {
    // An interrupted in_flight claim is deliberately not reclaimed on restart.
    return this.db.prepare("SELECT id,source,destination,delivery,state,attempts,next_at,error FROM jobs ORDER BY id").all();
  }

  close() {
    this.db.close();
  }
}
