"""Server-only RS256 App credentials, pinned-origin HTTP, and scoped dispatch."""

from __future__ import annotations

import base64
import os
import time
import urllib.error
import urllib.request
from datetime import datetime

from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding, rsa

from .common import Rejected, decode, encode, object_at, require

MAX_API_RESPONSE = 2 * 1024 * 1024


class DispatchError(Rejected):
    def __init__(self, code, retry=False, uncertain=False, delivered=False):
        super().__init__(code, 502)
        self.retry = retry
        self.uncertain = uncertain
        self.delivered = delivered


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class Transport:
    """No proxies, redirects, resource URL fetches, or arbitrary URL input."""

    def __init__(self, base):
        require(base.startswith("https://"), "https_github_api_required")
        self.base = base.rstrip("/")
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), NoRedirect(),
        )

    def request(self, method, path, credential, body=None):
        require(path.startswith("/") and not path.startswith("//"), "invalid_github_api_path")
        request = urllib.request.Request(
            self.base + path,
            data=encode(body) if body is not None else None,
            method=method,
            headers={
                "Authorization": "Bearer " + credential,
                "Accept": "application/vnd.github+json",
                "Content-Type": "application/json",
                "X-GitHub-Api-Version": "2022-11-28",
                "User-Agent": "gh-aw-python-gateway-reference",
            },
        )
        try:
            with self.opener.open(request, timeout=5) as response:
                raw = response.read(MAX_API_RESPONSE + 1)
                require(len(raw) <= MAX_API_RESPONSE, "oversized_github_response")
                return response.status, decode(raw) if raw else None
        except urllib.error.HTTPError as error:
            # Never read or interpolate credential-bearing upstream error bodies.
            status = error.code
            error.close()
            return status, None
        except (urllib.error.URLError, OSError, TimeoutError):
            raise DispatchError("github_transport_failed", retry=True) from None


def b64(value):
    return base64.urlsafe_b64encode(value).rstrip(b"=")


def app_jwt(config, now):
    key = os.environ.get(config["private_key_env"])
    require(key, "dispatch_private_key_unavailable", 503)
    try:
        private = serialization.load_pem_private_key(key.encode(), password=None)
        require(isinstance(private, rsa.RSAPrivateKey) and private.key_size >= 2048, "invalid_dispatch_private_key", 503)
        header = b64(encode({"alg": "RS256", "typ": "JWT"}))
        claims = b64(encode({"iss": str(config["app_id"]), "iat": int(now) - 60, "exp": int(now) + 540}))
        message = header + b"." + claims
        signature = private.sign(message, padding.PKCS1v15(), hashes.SHA256())
        return (message + b"." + b64(signature)).decode("ascii")
    except (ValueError, TypeError, UnicodeError):
        raise Rejected("invalid_dispatch_private_key", 503) from None


def permissions(value):
    return type(value) is dict and value.get("contents") == "write" and (
        set(value) <= {"contents", "metadata"}
    ) and ("metadata" not in value or value["metadata"] == "read")


class AppDispatcher:
    def __init__(self, config, transport=None, clock=time.time):
        self.config = config
        self.transport = transport or Transport(config["api_url"])
        self.clock = clock

    def call(self, method, path, credential, expected, body=None):
        status, response = self.transport.request(method, path, credential, body)
        if status != expected:
            raise DispatchError("github_api_status_rejected", retry=status == 429 or status >= 500)
        return response

    def dispatch(self, destination, request, authorize):
        from hashlib import sha256
        from urllib.parse import quote

        require(callable(authorize), "admission_authorization_required", 403)
        authorize()
        jwt = app_jwt(self.config, self.clock())
        repository = destination["repository"]
        installation = self.call("GET", "/repos/" + repository + "/installation", jwt, 200)
        require(type(installation) is dict
                and type(installation.get("id")) is int and installation["id"] == destination["installation_id"]
                and type(installation.get("app_id")) is int and installation["app_id"] == self.config["app_id"]
                and installation.get("suspended_at", "missing") is None
                and type(object_at(installation, ["account"]).get("id")) is int
                and object_at(installation, ["account"]).get("id") == destination["account_id"]
                and object_at(installation, ["account"]).get("login") == destination["account_login"]
                and object_at(installation, ["account"]).get("type") in {"Organization", "User"}
                and permissions(installation.get("permissions")), "installation_binding_mismatch", 403)
        token = None
        delivered = False
        dispatch_started = False
        try:
            authorize()
            result = self.call(
                "POST", "/app/installations/" + str(destination["installation_id"]) + "/access_tokens",
                jwt, 201, {"repository_ids": [destination["repository_id"]], "permissions": {"contents": "write"}},
            )
            require(type(result) is dict and type(result.get("token")) is str and result["token"]
                    and "\r" not in result["token"] and "\n" not in result["token"], "invalid_installation_token", 403)
            token = result["token"]
            try:
                expiration = datetime.fromisoformat(result["expires_at"].replace("Z", "+00:00"))
                require(expiration.tzinfo is not None, "invalid_token_expiry", 403)
                expiry = expiration.timestamp()
            except (KeyError, ValueError, TypeError, AttributeError):
                raise Rejected("invalid_token_expiry", 403) from None
            require(expiry > self.clock() + 30 and permissions(result.get("permissions"))
                    and result.get("repository_selection") == "selected"
                    and type(result.get("repositories")) is list
                    and len(result["repositories"]) == 1
                    and type(result["repositories"][0]) is dict
                    and result["repositories"][0].get("id") == destination["repository_id"]
                    and type(result["repositories"][0].get("id")) is int
                    and result["repositories"][0].get("full_name") == repository,
                    "installation_token_scope_or_expiry_mismatch", 403)
            metadata = self.call("GET", "/repos/" + repository, token, 200)
            require(type(metadata) is dict and type(metadata.get("id")) is int
                    and metadata["id"] == destination["repository_id"]
                    and metadata.get("full_name") == repository
                    and metadata.get("default_branch") == destination["default_branch"], "destination_repository_mismatch", 403)
            commit = self.call("GET", "/repos/" + repository + "/commits/" + quote(destination["default_branch"], safe=""), token, 200)
            require(type(commit) is dict and commit.get("sha") == destination["revision"], "destination_revision_changed", 403)
            workflow = destination["subscriptions"][0]["dispatch"]["workflow"]
            artifact = self.call(
                "GET", "/repos/" + repository + "/contents/" + workflow + "?ref=" + destination["revision"], token, 200,
            )
            try:
                require(type(artifact) is dict and artifact.get("encoding") == "base64"
                        and artifact.get("type") == "file", "invalid_destination_artifact", 403)
                raw = base64.b64decode("".join(artifact["content"].split()), validate=True)
            except (KeyError, TypeError, ValueError, AttributeError):
                raise Rejected("invalid_destination_artifact", 403) from None
            require(sha256(raw).hexdigest() == destination["artifact_sha256"], "destination_artifact_changed", 403)
            require(expiry > self.clock() + 5, "installation_token_expired", 403)
            authorize()
            dispatch_started = True
            status, _ = self.transport.request("POST", "/repos/" + repository + "/dispatches", token, request)
            if status != 204:
                # Even an upstream 5xx may follow acceptance: never automatically resend.
                raise DispatchError("dispatch_not_acknowledged", uncertain=status >= 500, retry=status == 429)
            delivered = True
        except DispatchError as error:
            if dispatch_started and error.code == "github_transport_failed":
                raise DispatchError("dispatch_outcome_unknown", uncertain=True) from None
            raise
        finally:
            if token:
                try:
                    status, _ = self.transport.request("DELETE", "/installation/token", token)
                    if status != 204:
                        raise DispatchError("dispatch_token_revocation_failed", uncertain=dispatch_started and not delivered, delivered=delivered)
                except (Rejected, OSError):
                    raise DispatchError("dispatch_token_revocation_failed", uncertain=dispatch_started and not delivered, delivered=delivered) from None
