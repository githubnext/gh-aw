"""Matching against authenticated originals, never hydrated or cached bodies."""

from __future__ import annotations

import hashlib
import json
import re
from datetime import datetime
from importlib.resources import files

from .common import (
    MAX_BYTES, at, base_url, canonical_url, decimal, encode, integer, object_at, positive,
    require, scalar, text, values_at,
)

CATALOGS = json.loads(files("gateway").joinpath("catalogs.json").read_text())
NOTIFICATIONS = CATALOGS["notifications"]["events"]
PROVIDERS = CATALOGS["providers"]
RESOURCES = CATALOGS["resources"]
COMMAND_ACTIONS = {
    "issues": ["opened", "edited", "reopened"],
    "issue_comment": ["created", "edited"],
    "pull_request_comment": ["created", "edited"],
    "pull_request": ["opened", "edited", "reopened"],
    "pull_request_review_comment": ["created", "edited"],
    "discussion": ["created", "edited"],
    "discussion_comment": ["created", "edited"],
}
CONTEXT = {
    "envelope": "github.event.client_payload",
    "aw_context": "github.event.client_payload.aw_context",
    "event": "github.event.client_payload.aw_context.webhook.event",
    "deliveries": "github.event.client_payload.aw_context.webhook.deliveries",
    "items": "aw_context.items",
}


def definitions(event):
    if event in NOTIFICATIONS:
        return {
            **CATALOGS["notifications"]["common_filters"],
            **NOTIFICATIONS[event]["filters"],
        }
    return (PROVIDERS.get(event) or RESOURCES.get(event) or {}).get("filters", {})


def validate_subscription(sub):
    require(type(sub) is dict, "invalid_subscription")
    require(set(sub) <= {
        "schema_version", "event", "types", "conditions", "any_of", "repos",
        "orgs", "provider", "native_events", "command_filter", "activity_normalization", "dispatch",
    }, "unknown_subscription_field")
    require(type(sub.get("schema_version")) is int and sub["schema_version"] == 2, "unsupported_schema")
    event = sub.get("event")
    require(event != "label_command", "unsupported_label_consumption")
    require(event in NOTIFICATIONS or event in PROVIDERS or event in RESOURCES or event == "slash_command", "unsupported_event")
    command = event == "slash_command"
    allowed = set(
        sum(COMMAND_ACTIONS.values(), []) if command
        else (NOTIFICATIONS.get(event) or PROVIDERS.get(event) or RESOURCES[event])["actions"]
    )
    types = sub.get("types")
    require(type(types) is list and types and all(type(x) is str and x in allowed for x in types)
            and len(set(types)) == len(types), "invalid_activity_types")
    require(type(sub.get("conditions")) is list, "invalid_conditions")
    groups = sub.get("any_of", [[]])
    require(type(groups) is list and groups and all(type(g) is list for g in groups)
            and ("any_of" not in sub or all(groups)), "invalid_alternatives")
    for condition in sub["conditions"] + [c for group in groups for c in group]:
        require(type(condition) is dict and set(condition) == {"filter", "path", "operator", "value_type", "values"}, "invalid_condition")
        definition = definitions(event).get(condition["filter"])
        require(definition and condition["path"] == definition["path"]
                and condition["value_type"] == definition["type"]
                and condition["operator"] == "equals", "unknown_condition")
        require(not definition.get("actions") or set(types) <= set(definition["actions"]), "condition_action_mismatch")
        vals = condition["values"]
        require(type(vals) is list and vals and all(scalar(x, definition["type"]) for x in vals)
                and len({encode(x) for x in vals}) == len(vals), "invalid_condition_values")
        require(all("${{" not in x for x in vals if type(x) is str), "nonliteral_condition")
        require(not definition.get("enum") or all(x in definition["enum"] for x in vals), "invalid_condition_enum")
    if event in NOTIFICATIONS:
        require(sub.get("native_events") == [NOTIFICATIONS[event]["native_event"]]
                and not any(k in sub for k in ("provider", "activity_normalization", "command_filter", "any_of")), "invalid_notification_subscription")
    elif event in PROVIDERS:
        definition = PROVIDERS[event]
        normalization = sub.get("activity_normalization", {})
        require(sub.get("provider") == definition["provider"]
                and normalization.get("policy") == definition["provider"] + "-issue/v1"
                and set(normalization) <= {"policy", "closed_states"}, "unsupported_normalization")
        expected = sorted({definition["native_events"][x] for x in types})
        require(sub.get("native_events") == expected, "invalid_native_events")
        scope = {"jira_issue": "project", "linear_issue": "organization-id", "azure_devops_issue": "project_id"}[event]
        require(scope in [c["filter"] for c in sub["conditions"]], "missing_provider_scope")
        if "closed_states" in normalization:
            states = normalization["closed_states"]
            require(event == "azure_devops_issue" and type(states) is list and states
                    and all(text(x) for x in states), "invalid_closed_states")
        require(event != "azure_devops_issue" or not ({"closed", "reopened"} & set(types))
                or normalization.get("closed_states"), "missing_closed_states")
        require("command_filter" not in sub, "unexpected_command_policy")
    elif command:
        policy = sub.get("command_filter", {})
        require(set(policy) == {"policy", "names", "events", "strategy"}
                and policy.get("policy") == "slash-command/v1"
                and policy.get("strategy") in {"inline", "centralized"}, "unsupported_command_policy")
        names, events = policy.get("names"), policy.get("events")
        require(type(names) is list and 0 < len(names) <= 25
                and all(type(x) is str and re.fullmatch(r"(?:\*|[A-Za-z0-9][A-Za-z0-9._-]*\*?)", x) for x in names)
                and names == sorted(set(names)), "invalid_command_names")
        require(type(events) is list and events and all(x in COMMAND_ACTIONS for x in events)
                and events == sorted(set(events)), "invalid_command_events")
        require(sub.get("native_events") == sorted({"issue_comment" if e == "pull_request_comment" else e for e in events})
                and not sub["conditions"] and not any(k in sub for k in ("any_of", "provider", "activity_normalization")), "invalid_command_subscription")
        require(set(types) == {a for e in events for a in COMMAND_ACTIONS[e]}, "invalid_command_activities")
    else:
        require(not any(k in sub for k in ("provider", "activity_normalization", "command_filter", "native_events")), "invalid_resource_subscription")
        require(event != "projects_v2_item" or "project_id" in [c["filter"] for c in sub["conditions"]], "missing_project_scope")
    for key, pattern in (
        ("repos", r"[A-Za-z0-9_-]+/(?:[A-Za-z0-9._-]+|\*)"),
        ("orgs", r"[A-Za-z0-9*](?:[A-Za-z0-9*-]{0,37}[A-Za-z0-9*])?"),
    ):
        if key in sub:
            vals = sub[key]
            require(event not in PROVIDERS and (key != "repos" or event != "projects_v2_item")
                    and type(vals) is list and vals
                    and all(type(x) is str and re.fullmatch(pattern, x) for x in vals), "invalid_source_selector")
    dispatch = sub.get("dispatch", {})
    require(set(dispatch) == {
        "repository", "method", "event_type", "workflow", "payload_projection",
        "context", "batch", "max_payload_bytes", "max_request_bytes",
    }, "invalid_dispatch_contract")
    workflow = dispatch.get("workflow", "")
    match = re.fullmatch(r"\.github/workflows/([A-Za-z0-9_.-]+)\.lock\.yml", workflow)
    require(match and match[1] not in {".", ".."}, "invalid_workflow_path")
    route = "gh-aw-webhook-" + hashlib.sha256(match[1].encode()).hexdigest()[:24]
    require(dispatch.get("event_type") == route and len(route) <= 100
            and dispatch.get("method") == "repository_dispatch"
            and dispatch.get("repository") == "${{ github.repository }}"
            and dispatch.get("payload_projection") == {"policy": "resource-url-types-filters/v2"}
            and dispatch.get("context") == CONTEXT
            and dispatch.get("batch") == {"mode": "same-event", "deduplicate_by": "delivery_id"}
            and type(dispatch.get("max_payload_bytes")) is int and dispatch["max_payload_bytes"] == MAX_BYTES
            and type(dispatch.get("max_request_bytes")) is int and dispatch["max_request_bytes"] == MAX_BYTES,
            "unsupported_dispatch_contract")


def semantic_event(native, body):
    if native == "issue_comment":
        issue = body.get("issue")
        require(type(issue) is dict, "missing_issue_identity")
        if "pull_request" in issue:
            require(type(issue["pull_request"]) is dict, "malformed_pull_request_evidence")
            return "pull_request_comment"
    return native


def notification_url(event, body, github):
    definition = NOTIFICATIONS[event]
    repo = at(body, ["repository", "full_name"])
    require(text(repo) and re.fullmatch(r"[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+", repo)
            and repo.split("/")[1] not in {".", ".."}, "invalid_source_repository")
    issue = body.get("issue")
    if definition.get("issue_kind"):
        require(type(issue) is dict, "missing_issue_identity")
        require(("pull_request" not in issue) if definition["issue_kind"] == "issue"
                else type(issue.get("pull_request")) is dict, "contradictory_issue_kind")
    if event == "commit_comment":
        number = at(body, ["comment", "commit_id"])
        require(type(number) is str and re.fullmatch(r"[a-f0-9]{40}", number), "invalid_commit")
    else:
        parent = "issue" if event in {"issues", "issue_comment", "pull_request_comment"} else (
            "discussion" if event in {"discussion", "discussion_comment"} else "pull_request"
        )
        number = positive(at(body, [parent, "number"]))
    anchor = ""
    if definition["anchor"]:
        path = ["review", "id"] if event == "pull_request_review" else (
            ["thread", "comments", "0", "id"] if event == "pull_request_review_thread" else ["comment", "id"]
        )
        anchor = "#" + definition["anchor"] + positive(at(body, path))
    expected = base_url(github["server_url"]) + "/" + repo + "/" + definition["path"] + "/" + number + anchor
    value = at(body, definition["url_path"])
    canonical_url(value, fragment=True)
    require(value == expected, "source_url_identity_mismatch")
    return value


def timestamp(value):
    if not text(value) or not re.match(r"^\d{4}-\d{2}-\d{2}T", value):
        return False
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).tzinfo is not None
    except ValueError:
        return False


def normalize(event, body, sub):
    if event not in PROVIDERS:
        return [body.get("action")]
    definition = PROVIDERS[event]
    native = body.get({"jira_issue": "webhookEvent", "linear_issue": "action", "azure_devops_issue": "eventType"}[event])
    require(native in definition["native_events"].values() and native in sub["native_events"], "unsupported_native_event")
    require(event != "linear_issue" or body.get("type") == "Issue", "unsupported_linear_entity")
    first = {"jira_issue": "jira:issue_created", "linear_issue": "create", "azure_devops_issue": "workitem.created"}[event]
    deleted = {"jira_issue": "jira:issue_deleted", "linear_issue": "remove", "azure_devops_issue": "workitem.deleted"}[event]
    if native == first:
        return ["opened"]
    if native == deleted:
        return ["deleted"]
    if native == "workitem.restored":
        return ["restored"]
    result = {"edited"}
    if event == "jira_issue":
        changes = at(body, ["changelog", "items"])
        labels = at(body, ["issue", "fields", "labels"])
        for change in changes if type(changes) is list else []:
            if type(change) is not dict or change.get("fieldtype") != "jira":
                continue
            if change.get("field") == "resolution" and "from" in change and "to" in change:
                if change["from"] is None and text(change["to"]):
                    result.add("closed")
                if text(change["from"]) and change["to"] is None:
                    result.add("reopened")
            if change.get("field") == "labels" and type(labels) is list and all(text(x) for x in labels):
                if "fromString" in change and change["fromString"] is None and text(change.get("toString")) and labels == [change["toString"]]:
                    result.add("labeled")
                if text(change.get("fromString")) and "toString" in change and change["toString"] is None and labels == []:
                    result.add("unlabeled")
    elif event == "linear_issue":
        before, after = body.get("updatedFrom"), body.get("data")
        if type(before) is dict and type(after) is dict:
            def terminal(obj):
                if any(timestamp(obj.get(k)) for k in ("completedAt", "canceledAt")):
                    return True
                if all(k in obj and obj[k] is None for k in ("completedAt", "canceledAt")):
                    return False
                return None
            if any(k in before for k in ("completedAt", "canceledAt")):
                was, now = terminal({**after, **before}), terminal(after)
                if was is False and now is True:
                    result.add("closed")
                if was is True and now is False:
                    result.add("reopened")
            add_labels(result, before.get("labelIds"), after.get("labelIds"))
    else:
        fields = at(body, ["resource", "fields"])
        fields = fields if type(fields) is dict else {}
        state, tags = fields.get("System.State"), fields.get("System.Tags")
        closed = sub["activity_normalization"].get("closed_states", [])
        if type(state) is dict and text(state.get("oldValue")) and text(state.get("newValue")):
            old, new = state["oldValue"], state["newValue"]
            if old not in closed and new in closed:
                result.add("closed")
            if old in closed and new not in closed:
                result.add("reopened")
        if type(tags) is dict and all(type(tags.get(k)) is str for k in ("oldValue", "newValue")):
            add_labels(result, [x.strip() for x in tags["oldValue"].split(";") if x.strip()],
                       [x.strip() for x in tags["newValue"].split(";") if x.strip()])
    require(not {"closed", "reopened"} <= result, "contradictory_lifecycle_evidence")
    return sorted(result)


def add_labels(result, before, after):
    if type(before) is list and type(after) is list and all(text(x) for x in before + after):
        if set(after) - set(before):
            result.add("labeled")
        if set(before) - set(after):
            result.add("unlabeled")


def source_authorized(source, body):
    require(source.get("approved") is True, "source_not_approved", 403)
    provider = source["provider"]
    if provider == "github":
        repo = source.get("repository")
        if repo:
            require(at(body, ["repository", "id"]) == repo["id"]
                    and type(at(body, ["repository", "id"])) is int
                    and at(body, ["repository", "full_name"]) == repo["full_name"]
                    and type(at(body, ["repository", "owner", "id"])) is int
                    and at(body, ["repository", "owner", "id"]) == repo["owner_id"],
                    "source_repository_not_approved", 403)
            if repo.get("organization"):
                require(at(body, ["repository", "owner", "type"]) == "Organization"
                        and type(at(body, ["organization", "id"])) is int
                        and at(body, ["organization", "id"]) == repo["owner_id"]
                        and at(body, ["organization", "login"]) == repo["organization"],
                        "source_organization_not_approved", 403)
    elif provider == "jira":
        require(at(body, ["issue", "fields", "project", "key"]) == source["project"], "source_project_not_approved", 403)
    elif provider == "linear":
        require(body.get("organizationId") == source["organization_id"]
                and (not source.get("team_id") or at(body, ["data", "teamId"]) == source["team_id"]), "source_organization_not_approved", 403)
    else:
        require(at(body, ["resourceContainers", "collection", "id"]) == source["collection_id"]
                and at(body, ["resourceContainers", "project", "id"]) == source["project_id"], "source_collection_not_approved", 403)


def resource_url(event, native, body, source, github):
    if event in NOTIFICATIONS:
        require(source.get("repository"), "repository_binding_required", 403)
        return notification_url(event, body, github)
    if event == "projects_v2_item":
        item = object_at(body, ["projects_v2_item"])
        node = item.get("project_node_id")
        require(text(node), "missing_project_identity")
        project = source.get("projects", {}).get(node)
        require(project, "project_binding_required", 403)
        owner = body.get("organization") if project["owner_kind"] == "orgs" else body.get("sender")
        require(type(owner) is dict and type(owner.get("id")) is int and owner.get("id") == project["owner_id"]
                and owner.get("login") == project["owner"], "project_owner_mismatch", 403)
        return base_url(github["api_url"]) + "/" + project["owner_kind"] + "/" + project["owner"] + "/projectsV2/" + positive(project["number"]) + "/items/" + positive(item.get("id"))
    if event in {"dependabot_alert", "code_scanning_alert"}:
        require(source.get("repository"), "repository_binding_required", 403)
        kind = "dependabot" if event == "dependabot_alert" else "code-scanning"
        expected = base_url(github["api_url"]) + "/repos/" + source["repository"]["full_name"] + "/" + kind + "/alerts/" + positive(at(body, ["alert", "number"]))
        require(at(body, ["alert", "url"]) == expected, "source_url_identity_mismatch")
        canonical_url(expected)
        return expected
    if event == "jira_issue":
        identity = decimal(at(body, ["issue", "id"]))
        value = at(body, ["issue", "self"])
        canonical_url(value)
        require(value in [source["origin"] + "/rest/api/" + version + "/issue/" + identity for version in ("2", "3")], "source_url_identity_mismatch")
        return value
    if event == "linear_issue":
        require(text(at(body, ["data", "id"])), "missing_linear_identity")
        value = body.get("url")
        url = canonical_url(value)
        identifier = at(body, ["data", "identifier"])
        require(type(identifier) is str and re.fullmatch(r"[A-Z][A-Z0-9_-]*-[1-9][0-9]*", identifier), "missing_linear_identifier")
        pattern = r"/" + re.escape(source["workspace"]) + r"/issue/" + re.escape(identifier) + r"(?:/(?:[A-Za-z0-9_-]|%[0-9a-fA-F]{2})+)?"
        require(url.netloc == "linear.app" and re.fullmatch(pattern, url.path), "source_url_identity_mismatch")
        if at(body, ["data", "url"]) is not None:
            require(at(body, ["data", "url"]) == value, "contradictory_linear_url")
        return value
    resource = object_at(body, ["resource"])
    snapshot = body.get("eventType") != "workitem.updated" or (
        body.get("resourceVersion") == "1.0" and "revision" not in resource
    )
    identity = resource.get("workItemId", resource.get("id") if snapshot else None)
    number = positive(identity)
    value = resource.get("url")
    canonical_url(value)
    require(value in [source["collection_url"] + "/" + prefix + "_apis/wit/workItems/" + number
                      for prefix in ("", source["project_id"] + "/")], "source_url_identity_mismatch")
    return value


def selectors_match(sub, source, body):
    if sub["event"] in PROVIDERS:
        return True
    if "repos" in sub:
        repo = source.get("repository", {}).get("full_name", "").lower()
        if not any(repo == x.lower() or (x.endswith("/*") and repo.startswith(x.lower()[:-1])) for x in sub["repos"]):
            return False
    if "orgs" in sub:
        if sub["event"] == "projects_v2_item":
            node = at(body, ["projects_v2_item", "project_node_id"])
            require(text(node), "missing_project_identity")
            project = source.get("projects", {}).get(node, {})
            org = project.get("owner") if project.get("owner_kind") == "orgs" else None
        else:
            org = source.get("repository", {}).get("organization")
        if not org or not any(re.fullmatch(".*".join(re.escape(p) for p in x.lower().split("*")), org.lower()) for x in sub["orgs"]):
            return False
    return True


def project_filters(sub, body):
    for group in sub.get("any_of", [[]]):
        conditions = sub["conditions"] + group
        projected = {}
        matches = True
        for condition in conditions:
            name = condition["filter"]
            originals = values_at(body, condition["path"])
            permitted = {
                encode(v): v for v in originals
                if scalar(v, condition["value_type"])
                and any(type(v) is type(x) and v == x for x in condition["values"])
            }
            if name in projected:
                permitted = {k: v for k, v in permitted.items() if v in projected[name]}
            if not permitted:
                matches = False
                break
            projected[name] = [permitted[k] for k in sorted(permitted)]
        if matches:
            return projected
    return None


def project(sub, native, body, source, github):
    event = sub["event"]
    if event in PROVIDERS:
        if source["provider"] != PROVIDERS[event]["provider"]:
            return None
        discriminator = body.get({"jira_issue": "webhookEvent", "linear_issue": "action", "azure_devops_issue": "eventType"}[event])
        if discriminator not in sub["native_events"] or (event == "linear_issue" and body.get("type") != "Issue"):
            return None
    else:
        if source["provider"] != "github":
            return None
        require(type(body.get("action")) is str, "invalid_native_action")
        if event == "slash_command":
            semantic = semantic_event(native, body)
            if semantic not in sub["command_filter"]["events"] or body.get("action") not in COMMAND_ACTIONS[semantic]:
                return None
        elif event in NOTIFICATIONS:
            if semantic_event(native, body) != event:
                return None
        elif native != event:
            return None
    if not selectors_match(sub, source, body):
        return None
    activities = sorted(set(normalize(event, body, sub)) & set(sub["types"]))
    if not activities:
        return None
    if event == "slash_command":
        semantic = semantic_event(native, body)
        path = ["comment", "body"] if "comment" in semantic else [
            "issue" if semantic == "issues" else semantic, "body"
        ]
        content = at(body, path)
        token = re.match(r"^/([a-zA-Z0-9][a-zA-Z0-9._-]*)(?=$|\s)", content) if type(content) is str else None
        if not token or not any(token[1] == name or (name.endswith("*") and token[1].startswith(name[:-1])) for name in sub["command_filter"]["names"]):
            return None
        filters = {"event": [semantic], "command": [token[1]]}
        url = resource_url(semantic, native, body, source, github)
    else:
        filters = project_filters(sub, body)
        if filters is None:
            return None
        url = resource_url(event, native, body, source, github)
        if event == "azure_devops_issue" and ({"closed", "reopened"} & set(activities)):
            side = "newValue" if "closed" in activities else "oldValue"
            filters["closed-states"] = [at(body, ["resource", "fields", "System.State", side])]
    payload = {"url": url, "type": activities}
    if filters:
        payload["filters"] = filters
    return payload


def envelope(sub, delivery_id, payload):
    client = {"schema_version": 2, "aw_context": {"webhook": {
        "event": sub["event"], "deliveries": [{"delivery_id": delivery_id, "payload": payload}],
    }}}
    result = {"event_type": sub["dispatch"]["event_type"], "client_payload": client}
    require(len(encode(client)) <= MAX_BYTES and len(encode(result)) <= MAX_BYTES, "oversized_dispatch", 413)
    return result
