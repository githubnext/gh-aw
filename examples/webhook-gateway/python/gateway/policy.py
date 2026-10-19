"""Local administrative policy import; this is not a shared enrollment protocol."""

from __future__ import annotations

import hashlib
import re
from pathlib import Path

from .common import base_url, decode, encode, integer, require, text
from .contracts import validate_subscription


def validate_policy(config):
    require(type(config) is dict and set(config) == {
        "schema_version", "github", "sources", "destinations", "bindings",
    } and type(config["schema_version"]) is int and config["schema_version"] == 1, "invalid_local_policy")
    github = config["github"]
    require(type(github) is dict and set(github) == {
        "api_url", "server_url", "app_id", "bot_id", "app_slug", "private_key_env",
    }, "invalid_github_policy")
    base_url(github["api_url"])
    base_url(github["server_url"])
    require(integer(github["app_id"], True) and integer(github["bot_id"], True)
            and text(github["app_slug"])
            and re.fullmatch(r"[A-Z_][A-Z0-9_]*", github["private_key_env"]), "invalid_app_binding")
    require(type(config["sources"]) is dict and type(config["destinations"]) is dict
            and type(config["bindings"]) is list, "invalid_local_policy")
    for name, source in config["sources"].items():
        require(re.fullmatch(r"[A-Za-z0-9_-]{1,100}", name) and type(source) is dict, "invalid_source_binding")
        require(type(source.get("approved")) is bool and type(source.get("expires_at")) in (int, float)
                and source.get("provider") in {"github", "jira", "linear", "azure-devops"}
                and type(source.get("secret_env")) is str
                and re.fullmatch(r"[A-Z_][A-Z0-9_]*", source["secret_env"]), "invalid_source_binding")
        require(set(source) <= {
            "provider", "approved", "expires_at", "secret_env", "repository", "projects",
            "origin", "project", "organization_id", "team_id", "workspace", "collection_id", "project_id", "collection_url",
        }, "unknown_source_binding_field")
        provider = source["provider"]
        if provider == "github":
            require(source.get("repository") or source.get("projects"), "missing_github_source_binding")
            repo = source.get("repository")
            if repo:
                require(set(repo) == {"id", "full_name", "owner_id", "organization"}
                        and integer(repo["id"], True) and integer(repo["owner_id"], True)
                        and type(repo["full_name"]) is str
                        and re.fullmatch(r"[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+", repo["full_name"])
                        and repo["full_name"].split("/")[1] not in {".", ".."}
                        and (repo["organization"] is None or repo["organization"] == repo["full_name"].split("/")[0]),
                        "invalid_repository_enrollment")
            for node, project in source.get("projects", {}).items():
                require(text(node) and set(project) == {"owner_kind", "owner", "owner_id", "number"}, "invalid_project_enrollment")
                require(project["owner_kind"] == "orgs", "unsupported_user_owned_project")
                require(re.fullmatch(r"[A-Za-z0-9_-]+", project["owner"])
                        and integer(project["owner_id"], True) and integer(project["number"], True), "invalid_project_enrollment")
        elif provider == "jira":
            base_url(source.get("origin"))
            require(text(source.get("project")), "missing_jira_enrollment")
        elif provider == "linear":
            require(text(source.get("organization_id")) and text(source.get("workspace"))
                    and re.fullmatch(r"[A-Za-z0-9_-]+", source["workspace"])
                    and source["workspace"] != "issue", "missing_linear_enrollment")
        else:
            url = base_url(source.get("collection_url"))
            require(re.fullmatch(r"https://(?:dev\.azure\.com/[A-Za-z0-9_-]+|[A-Za-z0-9-]+\.visualstudio\.com/[A-Za-z0-9_-]+)", url)
                    and text(source.get("collection_id")) and text(source.get("project_id"))
                    and re.fullmatch(r"[A-Za-z0-9_-]+", source["project_id"]), "missing_azure_enrollment")
    for name, destination in config["destinations"].items():
        require(re.fullmatch(r"[A-Za-z0-9_-]{1,100}", name) and type(destination) is dict, "invalid_destination_binding")
        require(set(destination) == {
            "approved", "expires_at", "repository", "repository_id", "installation_id",
            "account_id", "account_login", "default_branch", "revision", "artifact_sha256",
            "gate_verified", "identity_verified", "subscriptions",
        }, "invalid_destination_binding")
        require(type(destination["approved"]) is bool
                and type(destination["expires_at"]) in (int, float)
                and type(destination["gate_verified"]) is bool
                and type(destination["identity_verified"]) is bool
                and all(integer(destination[k], True) for k in ("repository_id", "installation_id", "account_id"))
                and re.fullmatch(r"[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+", destination["repository"])
                and destination["repository"].split("/")[1] not in {".", ".."}
                and destination["account_login"] == destination["repository"].split("/")[0]
                and text(destination["default_branch"])
                and re.fullmatch(r"[a-f0-9]{40}", destination["revision"])
                and re.fullmatch(r"[a-f0-9]{64}", destination["artifact_sha256"]), "invalid_destination_identity")
        subscriptions = destination["subscriptions"]
        require(type(subscriptions) is list and subscriptions, "missing_compiled_subscriptions")
        for sub in subscriptions:
            validate_subscription(sub)
        require(len({s["event"] for s in subscriptions}) == len(subscriptions)
                and len({s["dispatch"]["workflow"] for s in subscriptions}) == 1, "ambiguous_compiled_route")
    seen = set()
    for binding in config["bindings"]:
        require(type(binding) is dict and set(binding) == {"source", "destination"}
                and binding["source"] in config["sources"]
                and binding["destination"] in config["destinations"], "invalid_source_destination_grant")
        pair = (binding["source"], binding["destination"])
        require(pair not in seen, "duplicate_source_destination_grant")
        seen.add(pair)
    return config


def read_policy(path):
    config = decode(Path(path).read_bytes())
    for destination in config.get("destinations", {}).values():
        artifact = Path(destination.pop("artifact"))
        raw = artifact.read_bytes()
        require(hashlib.sha256(raw).hexdigest() == destination["artifact_sha256"], "artifact_digest_mismatch")
        lines = [line[len(b"# gh-aw-manifest: "):] for line in raw.splitlines() if line.startswith(b"# gh-aw-manifest: ")]
        require(len(lines) == 1, "missing_or_ambiguous_manifest")
        manifest = decode(lines[0])
        require("subscriptions" not in destination, "caller_supplied_compiled_subscriptions")
        destination["subscriptions"] = manifest.get("webhook_subscriptions")
    return validate_policy(config)


def policy_digest(config):
    # Key order is intentionally part of this local approval generation.
    return hashlib.sha256(encode(config)).hexdigest()


def current(source, destination, now):
    require(source["approved"] is True and source["expires_at"] > now, "source_approval_expired_or_revoked", 403)
    require(destination["approved"] is True and destination["expires_at"] > now
            and destination["gate_verified"] is True and destination["identity_verified"] is True,
            "destination_approval_expired_or_revoked", 403)
