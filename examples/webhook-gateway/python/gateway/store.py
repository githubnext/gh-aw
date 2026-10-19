"""SQLite admission ledger and conservative dispatch state machine."""

from __future__ import annotations

import base64
import hashlib
import hmac
import os
import re
import sqlite3
import time
from contextlib import contextmanager

from .common import Rejected, decode, encode, integer, require, text
from .contracts import envelope, project, source_authorized
from .github import AppDispatcher, DispatchError
from .policy import current, policy_digest, validate_policy

MAX_BODY_BYTES = 25 * 1024 * 1024
MAX_RECEIPTS = 100000
MAX_PENDING = 1000


def authenticate(source, headers, raw, now):
    provider = source["provider"]
    secret = os.environ.get(source["secret_env"])
    require(secret, "source_credential_unavailable", 503)
    if provider == "azure-devops":
        expected = "Basic " + base64.b64encode(secret.encode()).decode()
        require(hmac.compare_digest(headers.get("authorization", "").encode(), expected.encode()), "provider_authentication_failed", 401)
    else:
        header, prefix = {
            "github": ("x-hub-signature-256", "sha256="),
            "jira": ("x-hub-signature", "sha256="),
            "linear": ("linear-signature", ""),
        }[provider]
        supplied = headers.get(header, "")
        expected = prefix + hmac.new(secret.encode(), raw, hashlib.sha256).hexdigest()
        require(hmac.compare_digest(supplied.encode(), expected.encode()), "provider_authentication_failed", 401)
    body = decode(raw)
    require(type(body) is dict, "webhook_body_must_be_object")
    if provider == "linear":
        timestamp = body.get("webhookTimestamp")
        require(integer(timestamp) and abs(now * 1000 - timestamp) <= 60000, "linear_timestamp_outside_window", 401)
    header = {"github": "x-github-delivery", "jira": "x-atlassian-webhook-identifier",
              "linear": "linear-delivery"}.get(provider)
    delivery_id = headers.get(header) if header else body.get("id")
    require(text(delivery_id) and len(delivery_id.encode()) <= 1024
            and not re.search(r"[\x00-\x1f\x7f]", delivery_id), "missing_or_invalid_delivery_id")
    native = headers.get("x-github-event", "") if provider == "github" else ""
    require(provider != "github" or re.fullmatch(r"[a-z0-9_]{1,100}", native), "invalid_github_native_event")
    return body, delivery_id, native


class Store:
    def __init__(self, path, clock=time.time):
        self.path = str(path)
        self.clock = clock
        require(self.path != ":memory:", "durable_database_required")
        fd = os.open(self.path, os.O_CREAT | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
        os.close(fd)
        require(not os.path.islink(self.path) and os.stat(self.path).st_mode & 0o077 == 0, "database_permissions_must_be_private", 503)
        with self.connection() as db:
            db.execute("PRAGMA journal_mode=WAL")
            db.executescript("""
                CREATE TABLE IF NOT EXISTS policy (id INTEGER PRIMARY KEY CHECK(id=1), digest TEXT NOT NULL, config BLOB NOT NULL);
                CREATE TABLE IF NOT EXISTS receipts (
                    source TEXT NOT NULL, delivery_id TEXT NOT NULL, digest TEXT NOT NULL,
                    original_digest TEXT NOT NULL, created REAL NOT NULL,
                    PRIMARY KEY(source, delivery_id), UNIQUE(source, original_digest)
                );
                CREATE TABLE IF NOT EXISTS jobs (
                    id INTEGER PRIMARY KEY, source TEXT NOT NULL, destination TEXT NOT NULL,
                    policy_digest TEXT NOT NULL, request BLOB NOT NULL,
                    state TEXT NOT NULL DEFAULT 'pending', attempts INTEGER NOT NULL DEFAULT 0,
                    due REAL NOT NULL, error TEXT
                );
            """)

    @contextmanager
    def connection(self):
        try:
            db = sqlite3.connect(self.path, timeout=1, isolation_level=None)
            db.row_factory = sqlite3.Row
            db.execute("PRAGMA synchronous=FULL")
            try:
                yield db
            finally:
                db.close()
        except sqlite3.OperationalError:
            raise Rejected("durable_store_unavailable", 503) from None

    @contextmanager
    def transaction(self):
        with self.connection() as db:
            db.execute("BEGIN IMMEDIATE")
            try:
                yield db
                db.commit()
            except BaseException:
                db.rollback()
                raise

    def install(self, config):
        validate_policy(config)
        with self.transaction() as db:
            db.execute("INSERT OR REPLACE INTO policy VALUES (1, ?, ?)", (policy_digest(config), encode(config)))

    def policy(self, db):
        row = db.execute("SELECT digest, config FROM policy WHERE id=1").fetchone()
        require(row, "administrative_policy_required", 503)
        return row["digest"], decode(row["config"])

    def admit(self, source_id, headers, raw):
        require(len(raw) <= MAX_BODY_BYTES, "ingress_body_too_large", 413)
        # Authentication precedes any source fields being interpreted as authority.
        with self.transaction() as db:
            generation, config = self.policy(db)
            source = config["sources"].get(source_id)
            require(source, "unknown_source", 404)
            body, delivery_id, native = authenticate(source, headers, raw, self.clock())
            require(source["expires_at"] > self.clock(), "source_approval_expired_or_revoked", 403)
            source_authorized(source, body)
            original_digest = hashlib.sha256(raw).hexdigest()
            digest = hashlib.sha256(raw + b"\0" + native.encode()).hexdigest()
            previous = db.execute("SELECT digest FROM receipts WHERE source=? AND delivery_id=?", (source_id, delivery_id)).fetchone()
            if previous:
                require(hmac.compare_digest(previous["digest"], digest), "conflicting_delivery_identity", 409)
                return {"status": "duplicate", "admitted": 0}
            require(not db.execute("SELECT 1 FROM receipts WHERE source=? AND original_digest=?", (source_id, original_digest)).fetchone(), "reidentified_replay", 409)
            require(db.execute("SELECT count(*) FROM receipts").fetchone()[0] < MAX_RECEIPTS, "receipt_budget_exhausted", 503)
            jobs = []
            for binding in config["bindings"]:
                if binding["source"] != source_id:
                    continue
                destination = config["destinations"][binding["destination"]]
                current(source, destination, self.clock())
                for sub in destination["subscriptions"]:
                    payload = project(sub, native, body, source, config["github"])
                    if payload is not None:
                        jobs.append((source_id, binding["destination"], generation, encode(envelope(sub, delivery_id, payload)), self.clock()))
            require(jobs or any(b["source"] == source_id for b in config["bindings"]), "source_destination_not_approved", 403)
            pending = db.execute("SELECT count(*) FROM jobs WHERE state IN ('pending','inflight')").fetchone()[0]
            require(pending + len(jobs) <= MAX_PENDING, "admission_budget_exhausted", 503)
            db.execute("INSERT INTO receipts VALUES (?, ?, ?, ?, ?)", (source_id, delivery_id, digest, original_digest, self.clock()))
            db.executemany("INSERT INTO jobs(source,destination,policy_digest,request,due) VALUES (?,?,?,?,?)", jobs)
            return {"status": "admitted" if jobs else "not_matched", "admitted": len(jobs)}

    def work_once(self, dispatcher_factory=AppDispatcher):
        # Commit inflight before any network operation. A crash leaves an explicit
        # unresolved outcome; another process never automatically reclaims it.
        with self.transaction() as db:
            job = db.execute("SELECT * FROM jobs WHERE state='pending' AND due<=? ORDER BY id LIMIT 1", (self.clock(),)).fetchone()
            if job is None:
                return False
            db.execute("UPDATE jobs SET state='inflight', attempts=attempts+1 WHERE id=?", (job["id"],))
        with self.transaction() as db:
            generation, config = self.policy(db)
            try:
                require(generation == job["policy_digest"], "approval_generation_changed", 403)
                source = config["sources"].get(job["source"])
                destination = config["destinations"].get(job["destination"])
                require(source and destination and {"source": job["source"], "destination": job["destination"]} in config["bindings"], "source_destination_not_approved", 403)
                current(source, destination, self.clock())
                # This transaction serializes local policy withdrawal and all
                # workers with the bounded credential/dispatch sequence.
                dispatcher_factory(config["github"]).dispatch(
                    destination, decode(job["request"]),
                    lambda: current(source, destination, self.clock()),
                )
                state, error, due = "sent", None, self.clock()
            except DispatchError as failure:
                error, due = failure.code, self.clock()
                if failure.delivered:
                    state = "sent_cleanup_failed"
                elif failure.uncertain:
                    state = "uncertain"
                elif failure.retry and job["attempts"] + 1 < 5:
                    state, due = "pending", self.clock() + min(2 ** (job["attempts"] + 1), 300)
                else:
                    state = "failed"
            except Rejected as failure:
                state, error, due = "blocked", failure.code, self.clock()
            db.execute("UPDATE jobs SET state=?,error=?,due=? WHERE id=?", (state, error, due, job["id"]))
        return True

    def status(self):
        with self.connection() as db:
            return [dict(row) for row in db.execute("SELECT id,state,attempts,error FROM jobs ORDER BY id")]
