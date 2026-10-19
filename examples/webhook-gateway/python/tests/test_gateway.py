from __future__ import annotations

import base64
import concurrent.futures
import copy
import hashlib
import hmac
import http.client
import json
import os
import shutil
import subprocess
import tempfile
import threading
import time
import unittest
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest.mock import patch

from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding, rsa

from gateway.common import MAX_SAFE_INTEGER, Rejected, canonical_url, decode, encode
from gateway.contracts import (
    CATALOGS, CONTEXT, NOTIFICATIONS, definitions, envelope, normalize, project,
    validate_subscription,
)
from gateway.github import AppDispatcher, DispatchError, Transport
from gateway.policy import read_policy, validate_policy
from gateway.server import handler
from gateway.store import Store

ROOT = Path(__file__).resolve().parents[4]
NOW = 1800000000
REVISION = "a" * 40
SECRET = "synthetic-webhook-secret-not-a-real-credential"
OPAQUE_TOKEN = "synthetic opaque installation credential"


def condition(event, name, values):
    definition = definitions(event)[name]
    return {
        "filter": name, "path": definition["path"], "operator": "equals",
        "value_type": definition["type"], "values": values,
    }


def subscription(event="issues", types=None, conditions=None):
    sub = {
        "schema_version": 2, "event": event,
        "types": types or ["opened"], "conditions": conditions or [],
        "dispatch": {
            "repository": "${{ github.repository }}", "method": "repository_dispatch",
            "event_type": "gh-aw-webhook-" + hashlib.sha256(b"example").hexdigest()[:24],
            "workflow": ".github/workflows/example.lock.yml",
            "payload_projection": {"policy": "resource-url-types-filters/v2"},
            "context": CONTEXT, "batch": {"mode": "same-event", "deduplicate_by": "delivery_id"},
            "max_payload_bytes": 65535, "max_request_bytes": 65535,
        },
    }
    if event in NOTIFICATIONS:
        sub["native_events"] = [NOTIFICATIONS[event]["native_event"]]
    if event in CATALOGS["providers"]:
        definition = CATALOGS["providers"][event]
        sub["provider"] = definition["provider"]
        sub["activity_normalization"] = {"policy": definition["provider"] + "-issue/v1"}
        sub["native_events"] = sorted({definition["native_events"][x] for x in sub["types"]})
    return sub


def github_body():
    return {
        "action": "opened",
        "repository": {"id": 100, "full_name": "source/repo",
                       "owner": {"id": 40, "login": "source", "type": "Organization"}},
        "organization": {"id": 40, "login": "source"},
        "sender": {"login": "alice"},
        "issue": {"number": 42, "html_url": "https://github.com/source/repo/issues/42",
                  "body": "unforwarded private source content", "labels": [{"name": "ready"}, {"name": "private"}]},
    }


def config_for(sub=None):
    sub = sub or subscription()
    artifact = b"# gh-aw-manifest: " + encode({"webhook_subscriptions": [sub]}) + b"\n# synthetic lock fixture\n"
    return {
        "schema_version": 1,
        "github": {
            "api_url": "https://api.github.com", "server_url": "https://github.com",
            "app_id": 123, "bot_id": 456, "app_slug": "cao", "private_key_env": "TEST_CAO_PRIVATE_KEY",
        },
        "sources": {"src": {
            "provider": "github", "approved": True, "expires_at": NOW + 3600,
            "secret_env": "TEST_SOURCE_SECRET",
            "repository": {"id": 100, "full_name": "source/repo", "owner_id": 40, "organization": "source"},
        }},
        "destinations": {"dst": {
            "approved": True, "expires_at": NOW + 3600, "repository": "destination/repo",
            "repository_id": 789, "installation_id": 321, "account_id": 50,
            "account_login": "destination", "default_branch": "main", "revision": REVISION,
            "artifact_sha256": hashlib.sha256(artifact).hexdigest(),
            "gate_verified": True, "identity_verified": True, "subscriptions": [sub],
        }},
        "bindings": [{"source": "src", "destination": "dst"}],
    }, artifact


def signed(body, delivery="original-delivery", native="issues", provider="github"):
    raw = encode(body)
    digest = hmac.new(SECRET.encode(), raw, hashlib.sha256).hexdigest()
    headers = {
        "github": {"x-hub-signature-256": "sha256=" + digest, "x-github-delivery": delivery, "x-github-event": native},
        "jira": {"x-hub-signature": "sha256=" + digest, "x-atlassian-webhook-identifier": delivery},
        "linear": {"linear-signature": digest, "linear-delivery": delivery},
        "azure-devops": {"authorization": "Basic " + base64.b64encode(SECRET.encode()).decode()},
    }[provider]
    return headers, raw


class LocalTransport(Transport):
    """Test-only transport injection: production has no insecure-HTTP switch."""

    def __init__(self, address):
        super().__init__("https://api.github.com")
        self.base = address


class FakeGitHub:
    def __init__(self, artifact, key):
        self.artifact = artifact
        self.key = key
        self.calls = []
        self.dispatches = []
        self.failures = {}
        self.installation_override = {}
        self.token_override = {}
        self.metadata_override = {}
        self.commit = REVISION
        self.before_dispatch = None
        outer = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_GET(self):
                self.respond()

            def do_POST(self):
                self.respond()

            def do_DELETE(self):
                self.respond()

            def respond(self):
                body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
                credential = self.headers.get("Authorization", "").removeprefix("Bearer ")
                outer.calls.append((self.command, self.path, credential, decode(body) if body else None))
                status, result = outer.response(self.command, self.path, credential, body)
                raw = encode(result) if result is not None else b""
                self.send_response(status)
                self.send_header("Content-Length", str(len(raw)))
                if status == 302:
                    self.send_header("Location", "/credential-leak-target")
                self.end_headers()
                self.wfile.write(raw)

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.start()
        self.address = "http://127.0.0.1:" + str(self.server.server_address[1])

    def close(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def response(self, method, path, credential, body):
        if (method, path) in self.failures:
            # Deliberately credential-shaped upstream body must never surface.
            return self.failures[(method, path)], {"message": SECRET + OPAQUE_TOKEN}
        if path.endswith("/installation") or path.endswith("/access_tokens"):
            head, claims, signature = credential.split(".")
            self.key.public_key().verify(
                base64.urlsafe_b64decode(signature + "=="),
                (head + "." + claims).encode(), padding.PKCS1v15(), hashes.SHA256(),
            )
            jwt = decode(base64.urlsafe_b64decode(claims + "=="))
            assert jwt == {"iss": "123", "iat": NOW - 60, "exp": NOW + 540}
        else:
            assert credential == OPAQUE_TOKEN
        if path == "/repos/destination/repo/installation":
            return 200, {
                "id": 321, "app_id": 123, "suspended_at": None,
                "account": {"id": 50, "login": "destination", "type": "Organization"},
                "permissions": {"contents": "write", "metadata": "read"},
                **self.installation_override,
            }
        if path == "/app/installations/321/access_tokens":
            assert decode(body) == {"repository_ids": [789], "permissions": {"contents": "write"}}
            return 201, {
                "token": OPAQUE_TOKEN,
                "expires_at": datetime.fromtimestamp(NOW + 3600, timezone.utc).isoformat(),
                "permissions": {"contents": "write", "metadata": "read"},
                "repository_selection": "selected",
                "repositories": [{"id": 789, "full_name": "destination/repo"}],
                **self.token_override,
            }
        if path == "/repos/destination/repo":
            return 200, {"id": 789, "full_name": "destination/repo", "default_branch": "main", **self.metadata_override}
        if path == "/repos/destination/repo/commits/main":
            return 200, {"sha": self.commit}
        if path.startswith("/repos/destination/repo/contents/"):
            assert path.endswith("?ref=" + REVISION)
            return 200, {"encoding": "base64", "type": "file", "content": base64.b64encode(self.artifact).decode()}
        if path == "/repos/destination/repo/dispatches":
            if self.before_dispatch:
                self.before_dispatch()
            self.dispatches.append(decode(body))
            return 204, None
        if path == "/installation/token":
            return 204, None
        return 404, None


class GatewayTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        cls.pem = cls.key.private_bytes(
            serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8, serialization.NoEncryption(),
        ).decode()

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.env = patch.dict(os.environ, {"TEST_SOURCE_SECRET": SECRET, "TEST_CAO_PRIVATE_KEY": self.pem})
        self.env.start()
        self.addCleanup(self.env.stop)
        self.store = Store(Path(self.temp.name) / "gateway.sqlite", clock=lambda: NOW)
        self.config, self.artifact = config_for()
        self.store.install(self.config)

    def fake(self):
        fake = FakeGitHub(self.artifact, self.key)
        self.addCleanup(fake.close)
        return fake

    def factory(self, fake):
        return lambda config: AppDispatcher(config, LocalTransport(fake.address), clock=lambda: NOW)

    def dispatch(self, fake, request=None):
        return self.factory(fake)(self.config["github"]).dispatch(
            self.config["destinations"]["dst"], request or {},
            lambda: None,  # Synthetic tests explicitly supply an approved admission.
        )

    def admit(self, body=None, **kwargs):
        headers, raw = signed(body or github_body(), **kwargs)
        return self.store.admit("src", headers, raw)

    def assert_rejected(self, code, operation):
        with self.assertRaises(Rejected) as raised:
            operation()
        self.assertEqual(code, raised.exception.code)
        self.assertNotIn(SECRET, str(raised.exception))
        self.assertNotIn(OPAQUE_TOKEN, str(raised.exception))

    def test_http_ingress_to_scoped_dispatch_and_cleanup(self):
        fake = self.fake()
        ingress = ThreadingHTTPServer(("127.0.0.1", 0), handler(self.store))
        thread = threading.Thread(target=ingress.serve_forever)
        thread.start()
        self.addCleanup(thread.join)
        self.addCleanup(ingress.server_close)
        self.addCleanup(ingress.shutdown)
        connection = http.client.HTTPConnection("127.0.0.1", ingress.server_address[1])
        connection.request("GET", "/healthz")
        self.assertEqual(connection.getresponse().read(), b'{"status":"running"}')
        headers, raw = signed(github_body())
        connection.request("POST", "/ingress/src", raw, {**headers, "Content-Type": "application/json"})
        response = connection.getresponse()
        self.assertEqual(response.status, 200)
        self.assertEqual(decode(response.read())["admitted"], 1)
        connection.close()
        self.assertTrue(self.store.work_once(self.factory(fake)))
        self.assertEqual(self.store.status(), [{"id": 1, "state": "sent", "attempts": 1, "error": None}])
        self.assertEqual([x[0] for x in fake.calls], ["GET", "POST", "GET", "GET", "GET", "POST", "DELETE"])
        self.assertEqual(fake.calls[-2][2], fake.calls[-1][2])
        payload = fake.dispatches[0]["client_payload"]["aw_context"]["webhook"]["deliveries"][0]["payload"]
        self.assertEqual(payload, {"url": "https://github.com/source/repo/issues/42", "type": ["opened"]})
        serialized = encode(fake.dispatches[0])
        self.assertNotIn(b"private", serialized)
        with self.store.connection() as db:
            self.assertNotIn(b"unforwarded", db.execute("SELECT request FROM jobs").fetchone()[0])

    def test_raw_body_signature_not_reserialized_json(self):
        headers, raw = signed(github_body())
        self.assert_rejected("provider_authentication_failed", lambda: self.store.admit("src", headers, raw + b" "))
        self.assertEqual(self.store.status(), [])

    def test_authenticated_unapproved_and_unbound_sources(self):
        self.config["sources"]["src"]["approved"] = False
        self.store.install(self.config)
        self.assert_rejected("source_not_approved", self.admit)
        self.config["sources"]["src"]["approved"] = True
        self.config["bindings"] = []
        self.store.install(self.config)
        self.assert_rejected("source_destination_not_approved", self.admit)

    def test_source_identity_and_org_authority(self):
        for mutate in (
            lambda b: b["repository"].update(id=999),
            lambda b: b["repository"]["owner"].update(type="User"),
            lambda b: b["organization"].update(id=999),
        ):
            body = github_body()
            mutate(body)
            with self.assertRaises(Rejected):
                self.admit(body)
        self.assertEqual(self.store.status(), [])

    def test_same_delivery_concurrent_and_restart(self):
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            outcomes = list(pool.map(lambda _: self.admit(), range(12)))
        self.assertEqual(sum(x["admitted"] for x in outcomes), 1)
        self.store = Store(self.store.path, clock=lambda: NOW)
        self.assertEqual(self.admit()["status"], "duplicate")
        body = github_body()
        body["issue"]["body"] = "changed"
        self.assert_rejected("conflicting_delivery_identity", lambda: self.admit(body))
        self.assert_rejected("reidentified_replay", lambda: self.admit(delivery="different-id"))

    def test_dispatch_concurrent_workers_only_send_once(self):
        self.admit()
        fake = self.fake()
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            results = list(pool.map(lambda _: self.store.work_once(self.factory(fake)), range(4)))
        self.assertEqual(sum(results), 1)
        self.assertEqual(len(fake.dispatches), 1)

    def test_admission_fences_policy_withdrawal_and_expiry(self):
        self.admit()
        fake = self.fake()
        self.config["destinations"]["dst"]["approved"] = False
        self.store.install(self.config)
        self.store.work_once(self.factory(fake))
        self.assertEqual(self.store.status()[0]["state"], "blocked")
        self.assertEqual(fake.calls, [])
        self.config["destinations"]["dst"]["approved"] = True
        self.config["sources"]["src"]["expires_at"] = NOW - 1
        self.store.install(self.config)
        self.assert_rejected("source_approval_expired_or_revoked", self.admit)

    def test_expired_gate_or_identity_approval_never_admits(self):
        for key in ("gate_verified", "identity_verified", "approved"):
            config = copy.deepcopy(self.config)
            config["destinations"]["dst"][key] = False
            self.store.install(config)
            self.assert_rejected("destination_approval_expired_or_revoked", self.admit)

    def test_mint_failure_retries_without_credential_fallback(self):
        self.admit()
        fake = self.fake()
        fake.failures[("POST", "/app/installations/321/access_tokens")] = 503
        self.store.work_once(self.factory(fake))
        self.assertEqual(self.store.status()[0]["state"], "pending")
        self.assertEqual(fake.dispatches, [])
        self.assertEqual(len(fake.calls), 2)
        fake.failures.clear()
        self.store.clock = lambda: NOW + 10
        self.store.work_once(self.factory(fake))
        self.assertEqual(self.store.status()[0]["state"], "sent")

    def test_authorization_expiring_during_preparation_blocks_dispatch(self):
        self.admit()
        fake = self.fake()
        transport = LocalTransport(fake.address)
        original = transport.request
        clock = [NOW]
        self.store.clock = lambda: clock[0]

        def expires(method, path, credential, body=None):
            response = original(method, path, credential, body)
            if "/contents/" in path:
                clock[0] = NOW + 3601
            return response

        transport.request = expires
        self.store.work_once(lambda config: AppDispatcher(config, transport, clock=lambda: NOW))
        self.assertEqual(self.store.status()[0]["state"], "blocked")
        self.assertEqual(self.store.status()[0]["error"], "source_approval_expired_or_revoked")
        self.assertEqual(fake.dispatches, [])
        self.assertEqual(fake.calls[-1][:2], ("DELETE", "/installation/token"))

    def test_policy_changes_are_serialized_with_inflight_dispatch(self):
        self.admit()
        fake = self.fake()
        entered, release = threading.Event(), threading.Event()

        def pause():
            entered.set()
            self.assertTrue(release.wait(2))

        fake.before_dispatch = pause
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
            worker = pool.submit(self.store.work_once, self.factory(fake))
            self.assertTrue(entered.wait(2))
            config = copy.deepcopy(self.config)
            config["sources"]["src"]["approved"] = False
            install = pool.submit(self.store.install, config)
            time.sleep(0.05)
            self.assertFalse(install.done())
            release.set()
            worker.result()
            install.result()
        self.assertEqual(len(fake.dispatches), 1)
        self.assert_rejected("source_not_approved", self.admit)

    def test_dispatch_failure_unknown_outcome_not_retried(self):
        self.admit()
        fake = self.fake()
        fake.failures[("POST", "/repos/destination/repo/dispatches")] = 503
        self.store.work_once(self.factory(fake))
        self.assertEqual(self.store.status()[0]["state"], "uncertain")
        self.assertFalse(self.store.work_once(self.factory(fake)))
        self.assertEqual(fake.calls[-1][:2], ("DELETE", "/installation/token"))

    def test_dispatch_network_failure_is_uncertain(self):
        self.admit()
        fake = self.fake()
        transport = LocalTransport(fake.address)
        original = transport.request

        def failing(method, path, credential, body=None):
            if path.endswith("/dispatches"):
                raise DispatchError("github_transport_failed", retry=True)
            return original(method, path, credential, body)

        transport.request = failing
        self.store.work_once(lambda config: AppDispatcher(config, transport, clock=lambda: NOW))
        self.assertEqual(self.store.status()[0]["error"], "dispatch_outcome_unknown")
        self.assertEqual(self.store.status()[0]["state"], "uncertain")

    def test_crash_after_dispatch_is_not_automatically_replayed(self):
        self.admit()

        class Crash:
            def dispatch(self, *_):
                raise SystemExit("synthetic process interruption")

        with self.assertRaises(SystemExit):
            self.store.work_once(lambda _: Crash())
        restarted = Store(self.store.path, clock=lambda: NOW)
        self.assertEqual(restarted.status()[0]["state"], "inflight")
        self.assertFalse(restarted.work_once(lambda _: Crash()))

    def test_redirect_never_forwards_credentials(self):
        self.admit()
        fake = self.fake()
        fake.failures[("GET", "/repos/destination/repo/installation")] = 302
        self.store.work_once(self.factory(fake))
        self.assertEqual(len(fake.calls), 1)
        self.assertEqual(self.store.status()[0]["state"], "failed")

    def test_upstream_error_body_and_credentials_do_not_reach_status_or_errors(self):
        self.admit()
        fake = self.fake()
        fake.failures[("POST", "/app/installations/321/access_tokens")] = 401
        self.store.work_once(self.factory(fake))
        status = encode(self.store.status())
        self.assertNotIn(SECRET.encode(), status)
        self.assertNotIn(OPAQUE_TOKEN.encode(), status)
        self.assertEqual(self.store.status()[0]["state"], "failed")
        self.assertEqual(fake.dispatches, [])

    def test_retry_budget_is_bounded(self):
        self.admit()
        fake = self.fake()
        fake.failures[("POST", "/app/installations/321/access_tokens")] = 503
        for step in range(5):
            self.store.clock = lambda step=step: NOW + step * 100
            self.assertTrue(self.store.work_once(self.factory(fake)))
        self.assertEqual(self.store.status()[0]["attempts"], 5)
        self.assertEqual(self.store.status()[0]["state"], "failed")
        self.assertFalse(self.store.work_once(self.factory(fake)))

    def test_excessive_token_scope_is_revoked_never_dispatched(self):
        fake = self.fake()
        for override in (
            {"permissions": {"contents": "write", "issues": "write"}},
            {"repository_selection": "all"},
            {"repositories": [{"id": 999, "full_name": "other/repo"}]},
            {"expires_at": "2000-01-01T00:00:00Z"},
            {"expires_at": "2030-01-01T00:00:00"},
        ):
            with self.subTest(override=override):
                fake.token_override = override
                self.assert_rejected(
                    "invalid_token_expiry" if override.get("expires_at") == "2030-01-01T00:00:00"
                    else "installation_token_scope_or_expiry_mismatch",
                    lambda: self.dispatch(fake),
                )
                self.assertEqual(fake.calls[-1][:2], ("DELETE", "/installation/token"))
        self.assertEqual(fake.dispatches, [])

    def test_installation_and_default_revision_must_still_match(self):
        fake = self.fake()
        fake.installation_override = {"app_id": 999}
        self.assert_rejected("installation_binding_mismatch", lambda: self.dispatch(fake))
        fake.installation_override = {}
        fake.commit = "b" * 40
        self.assert_rejected("destination_revision_changed", lambda: self.dispatch(fake))
        self.assertEqual(fake.dispatches, [])
        self.assertEqual(fake.calls[-1][:2], ("DELETE", "/installation/token"))

    def test_token_cleanup_failure_reported_without_redispatch(self):
        self.admit()
        fake = self.fake()
        fake.failures[("DELETE", "/installation/token")] = 500
        self.store.work_once(self.factory(fake))
        self.assertEqual(self.store.status()[0]["state"], "sent_cleanup_failed")
        self.assertEqual(self.store.status()[0]["error"], "dispatch_token_revocation_failed")
        self.assertFalse(self.store.work_once(self.factory(fake)))
        self.assertEqual(len(fake.dispatches), 1)

    def test_json_duplicate_nonfinite_and_malformed_are_rejected(self):
        for raw in (b'{"action":"opened","action":"closed"}', b'{"x":NaN}', b'{"x":"\\ud800"}', b"{", b"null", b"[]", b'{"__proto__":{}}'):
            digest = hmac.new(SECRET.encode(), raw, hashlib.sha256).hexdigest()
            headers = {"x-hub-signature-256": "sha256=" + digest, "x-github-event": "issues", "x-github-delivery": "bad"}
            with self.subTest(raw=raw), self.assertRaises(Rejected):
                self.store.admit("src", headers, raw)
        self.assertEqual(self.store.status(), [])

    def test_unbounded_python_ids_and_boolean_integer_confusion(self):
        for value in (MAX_SAFE_INTEGER + 1, True, 1.0, 0, -1, "42"):
            body = github_body()
            body["issue"]["number"] = value
            with self.subTest(value=value):
                self.assert_rejected("unsafe_integer_id", lambda: self.admit(body))
        sub = subscription(conditions=[condition("issues", "number", [MAX_SAFE_INTEGER + 1])])
        self.assert_rejected("invalid_condition_values", lambda: validate_subscription(sub))

    def test_matching_types_and_minimal_literal_projection(self):
        sub = subscription(conditions=[condition("issues", "labels", ["ready", "not-present"])])
        validate_subscription(sub)
        payload = project(sub, "issues", github_body(), self.config["sources"]["src"], self.config["github"])
        self.assertEqual(payload["filters"], {"labels": ["ready"]})
        self.assertNotIn("private", encode(payload).decode())
        sub["conditions"].append(condition("issues", "number", [True]))
        self.assert_rejected("invalid_condition_values", lambda: validate_subscription(sub))

    def test_repos_orgs_intersect_independent_approval(self):
        sub = subscription()
        sub["repos"], sub["orgs"] = ["source/*"], ["sour*"]
        validate_subscription(sub)
        self.assertIsNotNone(project(sub, "issues", github_body(), self.config["sources"]["src"], self.config["github"]))
        sub["orgs"] = ["not-source"]
        self.assertIsNone(project(sub, "issues", github_body(), self.config["sources"]["src"], self.config["github"]))

    def test_all_notification_families_identity_and_anchors(self):
        for event, definition in NOTIFICATIONS.items():
            with self.subTest(event=event):
                body = github_body()
                action = definition["actions"][0]
                body["action"] = action
                body["pull_request"] = {"number": 42}
                body["discussion"] = {"number": 42}
                body["comment"] = {"id": 123, "commit_id": "c" * 40}
                body["review"] = {"id": 123}
                body["thread"] = {"comments": [{"id": 123}]}
                if event == "pull_request_comment":
                    body["issue"]["pull_request"] = {}
                number = "c" * 40 if event == "commit_comment" else "42"
                url = "https://github.com/source/repo/" + definition["path"] + "/" + number
                if definition["anchor"]:
                    url += "#" + definition["anchor"] + "123"
                target = body
                for segment in definition["url_path"][:-1]:
                    target = target[int(segment)] if type(target) is list else target[segment]
                target[definition["url_path"][-1]] = url
                sub = subscription(event, [action])
                validate_subscription(sub)
                payload = project(sub, definition["native_event"], body, self.config["sources"]["src"], self.config["github"])
                self.assertEqual(payload, {"url": url, "type": [action]})
                target[definition["url_path"][-1]] = url + "?unapproved=1"
                with self.assertRaises(Rejected):
                    project(sub, definition["native_event"], body, self.config["sources"]["src"], self.config["github"])

    def test_pr_comment_evidence_cannot_be_guessed_from_url(self):
        body = github_body()
        body["action"] = "created"
        body["comment"] = {"id": 123, "html_url": "https://github.com/source/repo/pull/42#issuecomment-123"}
        body["issue"]["pull_request"] = None
        self.assert_rejected("malformed_pull_request_evidence", lambda: project(subscription("pull_request_comment", ["created"]), "issue_comment", body, self.config["sources"]["src"], self.config["github"]))

    def test_project_transition_pair_not_cross_combined(self):
        sub = subscription("projects_v2_item", ["edited"], [
            condition("projects_v2_item", "project_id", ["PVT_example"]),
            condition("projects_v2_item", "field.name", ["Status"]),
            condition("projects_v2_item", "field.type", ["single_select"]),
        ])
        sub["any_of"] = [
            [condition("projects_v2_item", "field.from.name", ["Todo"]), condition("projects_v2_item", "field.to.name", ["Ready"])],
            [condition("projects_v2_item", "field.from.name", ["In progress"]), condition("projects_v2_item", "field.to.name", ["Done"])],
        ]
        validate_subscription(sub)
        source = {"provider": "github", "projects": {"PVT_example": {"owner_kind": "orgs", "owner": "source", "owner_id": 40, "number": 7}}}
        body = {"action": "edited", "organization": {"id": 40, "login": "source"},
                "projects_v2_item": {"project_node_id": "PVT_example", "id": 123},
                "changes": {"field_value": {"field_name": "Status", "field_type": "single_select", "from": {"name": "Todo"}, "to": {"name": "Done"}}}}
        self.assertIsNone(project(sub, "projects_v2_item", body, source, self.config["github"]))
        body["changes"]["field_value"]["to"]["name"] = "Ready"
        payload = project(sub, "projects_v2_item", body, source, self.config["github"])
        self.assertEqual(payload["url"], "https://api.github.com/orgs/source/projectsV2/7/items/123")
        self.assertEqual(payload["filters"]["field.to.name"], ["Ready"])
        self.assertEqual(len(payload["filters"]), 5)
        body["changes"]["field_value"]["project_number"] = 99
        self.assert_rejected("project_number_mismatch", lambda: project(sub, "projects_v2_item", body, source, self.config["github"]))

    def test_both_security_alert_families_minimal(self):
        for event, kind, severity in (("code_scanning_alert", "code-scanning", "error"), ("dependabot_alert", "dependabot", "high")):
            body = github_body()
            body["action"] = "created"
            url = "https://api.github.com/repos/source/repo/" + kind + "/alerts/42"
            body["alert"] = {"number": 42, "url": url, "rule": {"severity": severity}, "security_advisory": {"severity": severity}}
            sub = subscription(event, ["created"], [condition(event, "severity", [severity])])
            validate_subscription(sub)
            self.assertEqual(project(sub, event, body, self.config["sources"]["src"], self.config["github"]),
                             {"url": url, "type": ["created"], "filters": {"severity": [severity]}})

    def test_slash_commands_position_zero_and_no_arguments_forwarded(self):
        sub = subscription("slash_command", ["created", "edited"])
        sub["native_events"] = ["issue_comment"]
        sub["command_filter"] = {"policy": "slash-command/v1", "names": ["review*"], "events": ["issue_comment"], "strategy": "inline"}
        validate_subscription(sub)
        body = github_body()
        body["action"] = "created"
        body["comment"] = {"id": 123, "html_url": "https://github.com/source/repo/issues/42#issuecomment-123"}
        for content in (" /review", "\n/review", "text /review", "/Review", "/review/invalid"):
            body["comment"]["body"] = content
            self.assertIsNone(project(sub, "issue_comment", body, self.config["sources"]["src"], self.config["github"]))
        body["comment"]["body"] = "/review-fix private command arguments"
        payload = project(sub, "issue_comment", body, self.config["sources"]["src"], self.config["github"])
        self.assertEqual(payload["filters"], {"event": ["issue_comment"], "command": ["review-fix"]})
        self.assertNotIn("private", encode(payload).decode())

    def test_label_and_user_project_limitations_are_explicit(self):
        self.assert_rejected("unsupported_label_consumption", lambda: validate_subscription(subscription("label_command", ["labeled"])))
        config = copy.deepcopy(self.config)
        config["sources"]["src"]["projects"] = {"PVT_test": {"owner_kind": "users", "owner": "source", "owner_id": 40, "number": 7}}
        self.assert_rejected("unsupported_user_owned_project", lambda: validate_policy(config))

    def test_jira_raw_auth_lifecycle_and_selected_projection(self):
        sub = subscription("jira_issue", ["closed"], [condition("jira_issue", "project", ["APP"])])
        config, _ = config_for(sub)
        config["sources"]["src"] = {"provider": "jira", "approved": True, "expires_at": NOW + 3600, "secret_env": "TEST_SOURCE_SECRET", "origin": "https://example.atlassian.net", "project": "APP"}
        self.store.install(config)
        body = {"webhookEvent": "jira:issue_updated",
                "issue": {"id": "10001", "self": "https://example.atlassian.net/rest/api/3/issue/10001",
                          "fields": {"project": {"key": "APP"}, "labels": ["ready"]}},
                "changelog": {"items": [{"fieldtype": "jira", "field": "resolution", "from": None, "to": "1"},
                                         {"fieldtype": "jira", "field": "labels", "fromString": None, "toString": "ready"}]}}
        headers, raw = signed(body, provider="jira")
        self.assertEqual(self.store.admit("src", headers, raw)["admitted"], 1)
        with self.store.connection() as db:
            payload = decode(db.execute("SELECT request FROM jobs").fetchone()[0])["client_payload"]["aw_context"]["webhook"]["deliveries"][0]["payload"]
        self.assertEqual(payload["type"], ["closed"])
        self.assertEqual(payload["filters"], {"project": ["APP"]})
        self.assertEqual(normalize("jira_issue", body, sub), ["closed", "edited", "labeled"])
        body["changelog"]["items"] = [{"fieldtype": "custom", "field": "resolution", "from": None, "to": "1"}]
        self.assertEqual(normalize("jira_issue", body, sub), ["edited"])

    def test_linear_signature_signed_timestamp_and_evidence(self):
        sub = subscription("linear_issue", ["closed", "labeled", "unlabeled"], [condition("linear_issue", "organization-id", ["org-id"])])
        config, _ = config_for(sub)
        config["sources"]["src"] = {"provider": "linear", "approved": True, "expires_at": NOW + 3600, "secret_env": "TEST_SOURCE_SECRET", "organization_id": "org-id", "workspace": "workspace"}
        self.store.install(config)
        body = {"type": "Issue", "action": "update", "organizationId": "org-id", "webhookTimestamp": NOW * 1000,
                "url": "https://linear.app/workspace/issue/APP-42/example",
                "data": {"id": "issue-uuid", "identifier": "APP-42", "completedAt": "2027-01-15T08:00:00Z", "canceledAt": None, "labelIds": ["new"]},
                "updatedFrom": {"completedAt": None, "labelIds": ["old"]}}
        self.assertEqual(normalize("linear_issue", body, sub), ["closed", "edited", "labeled", "unlabeled"])
        headers, raw = signed(body, provider="linear")
        self.assertEqual(self.store.admit("src", headers, raw)["admitted"], 1)
        body["webhookTimestamp"] = (NOW - 61) * 1000
        headers, raw = signed(body, provider="linear")
        self.assert_rejected("linear_timestamp_outside_window", lambda: self.store.admit("src", headers, raw))
        del body["data"]["canceledAt"]
        self.assertNotIn("closed", normalize("linear_issue", body, sub))
        body["url"] = "https://linear.app/issue/APP-42"
        with self.assertRaises(Rejected):
            project(sub, "", body, config["sources"]["src"], config["github"])

    def test_azure_basic_auth_update_id_and_terminal_projection(self):
        sub = subscription("azure_devops_issue", ["closed", "edited", "labeled"], [condition("azure_devops_issue", "project_id", ["project-id"])])
        sub["activity_normalization"]["closed_states"] = ["Done", "Closed"]
        config, _ = config_for(sub)
        config["sources"]["src"] = {"provider": "azure-devops", "approved": True, "expires_at": NOW + 3600, "secret_env": "TEST_SOURCE_SECRET", "collection_id": "collection-id", "project_id": "project-id", "collection_url": "https://dev.azure.com/organization"}
        self.store.install(config)
        body = {"id": "stable-native-delivery", "eventType": "workitem.updated",
                "resourceContainers": {"collection": {"id": "collection-id"}, "project": {"id": "project-id"}},
                "resource": {"id": 999, "workItemId": 42, "revision": {},
                             "url": "https://dev.azure.com/organization/_apis/wit/workItems/42",
                             "fields": {"System.State": {"oldValue": "Active", "newValue": "Done"},
                                        "System.Tags": {"oldValue": "old", "newValue": "old; new"}}}}
        headers, raw = signed(body, provider="azure-devops")
        self.assertEqual(self.store.admit("src", headers, raw)["admitted"], 1)
        payload = project(sub, "", body, config["sources"]["src"], config["github"])
        self.assertEqual(payload["filters"]["closed-states"], ["Done"])
        self.assertTrue(payload["url"].endswith("/42"))
        edited = copy.deepcopy(sub)
        edited["types"] = ["edited"]
        self.assertNotIn("closed-states", project(edited, "", body, config["sources"]["src"], config["github"])["filters"])
        for revision in (
            {"id": 99}, {"fields": {"System.Id": 99}},
            {"url": "https://dev.azure.com/other/_apis/wit/workItems/42"},
        ):
            body["resource"]["revision"] = revision
            with self.subTest(revision=revision), self.assertRaises(Rejected):
                project(sub, "", body, config["sources"]["src"], config["github"])
        body["resource"]["revision"] = {"id": 42, "fields": {"System.Id": 42}, "url": body["resource"]["url"]}
        self.assertEqual(project(sub, "", body, config["sources"]["src"], config["github"])["url"], payload["url"])
        del body["resource"]["workItemId"]
        self.assert_rejected("unsafe_integer_id", lambda: project(sub, "", body, config["sources"]["src"], config["github"]))

    def test_exact_utf8_request_boundary_not_character_count(self):
        sub = subscription()
        payload = {"url": "https://github.com/source/repo/issues/42", "type": ["opened"]}
        baseline = len(encode(envelope(sub, "x", payload)))
        delivery = "x" * (65535 - baseline + 1)
        self.assertEqual(len(encode(envelope(sub, delivery, payload))), 65535)
        self.assert_rejected("oversized_dispatch", lambda: envelope(sub, delivery + "x", payload))
        unicode_delivery = "\u00e9" * ((65535 - baseline) // 2)
        self.assertLessEqual(len(encode(envelope(sub, unicode_delivery, payload))), 65535)
        self.assert_rejected("oversized_dispatch", lambda: envelope(sub, unicode_delivery + "\u00e9\u00e9", payload))

    def test_artifact_approval_import_and_unknown_projection_fail_closed(self):
        artifact = Path(self.temp.name) / "example.lock.yml"
        artifact.write_bytes(self.artifact)
        config = copy.deepcopy(self.config)
        destination = config["destinations"]["dst"]
        del destination["subscriptions"]
        destination["artifact"] = str(artifact)
        policy = Path(self.temp.name) / "policy.json"
        policy.write_bytes(encode(config))
        self.assertEqual(read_policy(policy), self.config)
        artifact.write_bytes(self.artifact + b"# changed")
        self.assert_rejected("artifact_digest_mismatch", lambda: read_policy(policy))
        sub = subscription()
        sub["dispatch"]["payload_projection"]["policy"] = "resource-url-types/v1"
        self.assert_rejected("unsupported_dispatch_contract", lambda: validate_subscription(sub))

    def test_enterprise_api_and_html_prefixes_remain_distinct(self):
        github = {**self.config["github"], "server_url": "https://github.example/enterprise", "api_url": "https://github.example/api/v3"}
        body = github_body()
        body["issue"]["html_url"] = "https://github.example/enterprise/source/repo/issues/42"
        payload = project(subscription(), "issues", body, self.config["sources"]["src"], github)
        self.assertEqual(payload["url"], body["issue"]["html_url"])
        body["issue"]["html_url"] = "https://github.example/source/repo/issues/42"
        self.assert_rejected("source_url_identity_mismatch", lambda: project(subscription(), "issues", body, self.config["sources"]["src"], github))

    def test_whatwg_noncanonical_forms_and_empty_delimiters_reject(self):
        for value in (
            "https://example.atlassian.net/rest/api/3/issue/42?",
            "https://example.atlassian.net/rest/api/3/issue/42#",
            "HTTPS://example.atlassian.net/rest/api/3/issue/42",
            "https://EXAMPLE.atlassian.net/rest/api/3/issue/42",
            "https://example.atlassian.net:443/rest/api/3/issue/42",
            "https://example.atlassian.net:0443/rest/api/3/issue/42",
            "https://example.atlassian.net:0800/rest/api/3/issue/42",
            "https://%65xample.atlassian.net/rest/api/3/issue/42",
            "https://example.atlassian.net/rest/./api/3/issue/42",
            "https://example.atlassian.net/rest/%2e/api/3/issue/42",
            "https://example.atlassian.net/rest/../api/3/issue/42",
            "https://user:pass@example.atlassian.net/rest/api/3/issue/42",
            "https://127.1/rest/api/3/issue/42",
            "https://0177.0.0.1/rest/api/3/issue/42",
            "https://0x7f000001/rest/api/3/issue/42",
            "https://127.0.0.1./rest/api/3/issue/42",
            "https://[0:0:0:0:0:0:0:1]/rest/api/3/issue/42",
            "https://example.atlassian.net/{prefix}/rest/api/3/issue/42",
        ):
            with self.subTest(value=value):
                self.assert_rejected("invalid_resource_url", lambda: canonical_url(value))
        for value in (
            "https://127.0.0.1/rest/api/3/issue/42",
            "https://[::1]/rest/api/3/issue/42",
            "https://example.atlassian.net:800/rest/api/3/issue/42",
            "https://linear.app/workspace/issue/APP-42/encoded%2eslug",
        ):
            with self.subTest(value=value):
                self.assertEqual(canonical_url(value).geturl(), value)

    def test_admission_and_receipt_budgets_do_not_acknowledge_undurable_jobs(self):
        with patch("gateway.store.MAX_PENDING", 0):
            self.assert_rejected("admission_budget_exhausted", self.admit)
        with self.store.connection() as db:
            self.assertEqual(db.execute("SELECT count(*) FROM receipts").fetchone()[0], 0)
        with patch("gateway.store.MAX_RECEIPTS", 0):
            self.assert_rejected("receipt_budget_exhausted", self.admit)
        self.assertEqual(self.store.status(), [])

    def test_http_fail_closed_duplicate_headers_and_authentication(self):
        ingress = ThreadingHTTPServer(("127.0.0.1", 0), handler(self.store))
        thread = threading.Thread(target=ingress.serve_forever)
        thread.start()
        self.addCleanup(thread.join)
        self.addCleanup(ingress.server_close)
        self.addCleanup(ingress.shutdown)
        headers, raw = signed(github_body())
        connection = http.client.HTTPConnection("127.0.0.1", ingress.server_address[1])
        connection.putrequest("POST", "/ingress/src")
        connection.putheader("Content-Type", "application/json")
        connection.putheader("Content-Length", str(len(raw)))
        connection.putheader("Content-Length", str(len(raw)))
        connection.endheaders(raw)
        response = connection.getresponse()
        self.assertEqual(response.status, 400)
        self.assertEqual(decode(response.read())["error"], "duplicate_http_header")
        connection.close()
        connection = http.client.HTTPConnection("127.0.0.1", ingress.server_address[1])
        connection.request("POST", "/ingress/src", raw, {"Content-Type": "application/json", **headers, "x-hub-signature-256": "bad"})
        response = connection.getresponse()
        self.assertEqual(response.status, 401)
        self.assertNotIn(SECRET.encode(), response.read())
        connection.close()
        self.assertEqual(self.store.status(), [])

    @unittest.skipUnless(shutil.which("node"), "Destination interoperability requires Node.js")
    def test_projected_dispatch_is_accepted_by_real_destination_validator(self):
        sub = subscription(conditions=[condition("issues", "labels", ["ready"])])
        payload = project(sub, "issues", github_body(), self.config["sources"]["src"], self.config["github"])
        request = envelope(sub, "original-delivery", payload)
        script = """
          const {validateWebhook}=require(process.argv[1]);
          let input='';process.stdin.on('data',chunk=>input+=chunk);
          process.stdin.on('end',()=>{const v=JSON.parse(input);
            process.stdout.write(JSON.stringify(validateWebhook(
              {action:v.request.event_type,client_payload:v.request.client_payload},
              {GITHUB_EVENT_NAME:'repository_dispatch',GH_AW_WEBHOOK_SUBSCRIPTIONS:JSON.stringify([v.sub])})));});
        """
        result = subprocess.run(
            ["node", "-e", script, str(ROOT / "actions/setup/js/validate_webhook.cjs")],
            input=encode({"sub": sub, "request": request}), capture_output=True, check=True,
        )
        self.assertFalse(decode(result.stdout)["skipped"])


if __name__ == "__main__":
    unittest.main()
