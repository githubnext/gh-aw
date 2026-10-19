import { commandActions, definitions, notifications, providers } from "./contracts.mjs";
import { canonical, httpsURL, nonempty, numericId, object, requireThat, valuesAt } from "./core.mjs";

const strings = value => Array.isArray(value) && value.every(nonempty);
const matchesName = (name, token) => name === "*" || (name.endsWith("*") ? token.startsWith(name.slice(0, -1)) : name === token);
const patternsMatch = (value, patterns) => patterns.some(pattern => new RegExp(`^${pattern.toLowerCase().replaceAll("*", ".*")}$`).test(value.toLowerCase()));
const read = (value, path) => valuesAt(value, path)[0];

function organization(source, payload) {
  const enrolled = source.scope.organization;
  if (!enrolled) return undefined;
  requireThat(numericId(payload.organization?.id) === enrolled.id && payload.organization.login.toLowerCase() === enrolled.login.toLowerCase(), "source-organization-mismatch", 403);
  return enrolled.login;
}

function repository(source, payload) {
  const scope = source.scope;
  requireThat(
    scope.repository &&
      numericId(payload.repository?.id) === scope.repositoryId &&
      typeof payload.repository?.full_name === "string" &&
      payload.repository.full_name.toLowerCase() === scope.repository.toLowerCase() &&
      numericId(payload.repository.owner?.id) === scope.ownerId,
    "source-repository-mismatch",
    403
  );
  if (scope.organization) {
    requireThat(scope.ownerId === scope.organization.id && scope.repository.split("/")[0].toLowerCase() === scope.organization.login.toLowerCase(), "source-owner-mismatch", 403);
  }
  return payload.repository.full_name;
}

function githubReference(event, source, payload, github) {
  const definition = notifications.events[event];
  const repo = repository(source, payload);
  if (definition.issue_kind === "issue") {
    requireThat(object(payload.issue) && !Object.hasOwn(payload.issue, "pull_request"), "contradictory-issue-kind");
  } else if (definition.issue_kind === "pull_request") {
    requireThat(object(payload.issue?.pull_request), "missing-pull-request-evidence");
  }
  let parent;
  if (definition.issue_kind) parent = numericId(payload.issue.number);
  else if (event.startsWith("pull_request")) parent = numericId(payload.pull_request?.number);
  else if (event.startsWith("discussion")) parent = numericId(payload.discussion?.number);
  else {
    parent = payload.comment?.commit_id;
    requireThat(typeof parent === "string" && /^[0-9a-f]{40}$/.test(parent), "invalid-source-commit");
  }
  let suffix = "";
  if (definition.anchor) {
    const item = event === "pull_request_review" ? payload.review : event === "pull_request_review_thread" ? payload.thread?.comments?.[0] : payload.comment;
    suffix = `#${definition.anchor}${numericId(item?.id)}`;
  }
  const original = read(payload, definition.url_path);
  const url = httpsURL(original);
  requireThat(original === `${github.serverUrl}/${repo}/${definition.path}/${parent}${suffix}`, "source-url-identity-mismatch");
  return url.href;
}

function jiraActivities(payload) {
  if (payload.webhookEvent === "jira:issue_created") return ["opened"];
  if (payload.webhookEvent === "jira:issue_deleted") return ["deleted"];
  const types = ["edited"];
  for (const change of Array.isArray(payload.changelog?.items) ? payload.changelog.items : []) {
    if (!object(change) || change.fieldtype !== "jira") continue;
    if (change.field === "resolution" && Object.hasOwn(change, "from") && Object.hasOwn(change, "to")) {
      if (change.from === null && nonempty(change.to)) types.push("closed");
      if (nonempty(change.from) && change.to === null) types.push("reopened");
    }
    const labels = payload.issue?.fields?.labels;
    if (change.field === "labels" && strings(labels)) {
      if (change.fromString === null && nonempty(change.toString) && labels.length === 1 && labels[0] === change.toString) types.push("labeled");
      if (nonempty(change.fromString) && change.toString === null && labels.length === 0) types.push("unlabeled");
    }
  }
  return types;
}

function labelTypes(before, after) {
  if (!strings(before) || !strings(after)) return [];
  return [...(after.some(value => !before.includes(value)) ? ["labeled"] : []), ...(before.some(value => !after.includes(value)) ? ["unlabeled"] : [])];
}

function linearActivities(payload) {
  if (payload.action === "create") return ["opened"];
  if (payload.action === "remove") return ["deleted"];
  const types = ["edited"];
  const before = payload.updatedFrom;
  if (!object(before)) return types;
  const timestamp = value => nonempty(value) && /^\d{4}-\d{2}-\d{2}T/.test(value) && Number.isFinite(Date.parse(value));
  const terminal = value => (timestamp(value.completedAt) || timestamp(value.canceledAt) ? true : value.completedAt === null && value.canceledAt === null ? false : undefined);
  if (["completedAt", "canceledAt"].some(key => Object.hasOwn(before, key))) {
    const was = terminal({ ...payload.data, ...before });
    const now = terminal(payload.data);
    if (was === false && now === true) types.push("closed");
    if (was === true && now === false) types.push("reopened");
  }
  return [...types, ...labelTypes(before.labelIds, payload.data.labelIds)];
}

function azureActivities(payload, states) {
  const simple = { "workitem.created": "opened", "workitem.deleted": "deleted", "workitem.restored": "restored" }[payload.eventType];
  if (simple) return [simple];
  const types = ["edited"];
  const state = payload.resource?.fields?.["System.State"];
  if (nonempty(state?.oldValue) && nonempty(state?.newValue)) {
    if (!states.includes(state.oldValue) && states.includes(state.newValue)) types.push("closed");
    if (states.includes(state.oldValue) && !states.includes(state.newValue)) types.push("reopened");
  }
  const tags = payload.resource?.fields?.["System.Tags"];
  const split = value =>
    value
      .split(";")
      .map(tag => tag.trim())
      .filter(Boolean);
  if (typeof tags?.oldValue === "string" && typeof tags?.newValue === "string") types.push(...labelTypes(split(tags.oldValue), split(tags.newValue)));
  return types;
}

function verifyAzureLinks(payload, scope, id) {
  const collection = new URL(scope.collectionUrl);
  const links = [payload.resource?.url, payload.resource?._links?.self?.href, payload.resource?._links?.workItem?.href, payload.resource?.revision?.url, payload.resource?.revision?._links?.self?.href].filter(value => value !== undefined);
  for (const link of links) {
    const url = httpsURL(link);
    requireThat(!url.hash && url.origin === collection.origin, "contradictory-work-item-link");
    const match = url.pathname.match(/^(.*)\/_apis\/wit\/work[Ii]tems\/([1-9][0-9]*)(?:\/(?:updates|revisions)\/[1-9][0-9]*)?$/);
    const bases = [collection.pathname, `${collection.pathname}/${scope.projectId}`, ...(scope.projectSegment ? [`${collection.pathname}/${scope.projectSegment}`] : [])];
    requireThat(match && bases.includes(match[1]) && match[2] === id, "contradictory-work-item-link");
  }
}

function conditionsProjection(subscription, payload) {
  for (const group of subscription.any_of ?? [[]]) {
    const conditions = [...subscription.conditions, ...group];
    const filters = Object.create(null);
    let matches = true;
    for (const condition of conditions) {
      const actual = valuesAt(payload, condition.path);
      const selected = [...new Set(actual.filter(value => condition.values.includes(value)))];
      const prior = filters[condition.filter];
      filters[condition.filter] = (prior ? selected.filter(value => prior.includes(value)) : selected).sort((a, b) => canonical(a).localeCompare(canonical(b)));
      if (!filters[condition.filter].length) matches = false;
    }
    if (matches) return filters;
  }
  return undefined;
}

export function project(subscription, source, payload, nativeEvent, github) {
  const event = subscription.event;
  const definition = definitions[event];
  const expectedProvider = definition?.provider ?? "github";
  if (source.provider !== expectedProvider) return undefined;
  let url, types;
  let repo, org;
  if (source.provider === "github") {
    const semantic = event === "slash_command" ? (nativeEvent === "issue_comment" && object(payload.issue?.pull_request) ? "pull_request_comment" : nativeEvent) : event;
    const sourceDefinition = notifications.events[semantic];
    const expectedNative = sourceDefinition?.native_event ?? event;
    if (expectedNative !== nativeEvent) return undefined;
    if (nativeEvent === "issue_comment") {
      if (semantic === "issue_comment" && object(payload.issue?.pull_request)) return undefined;
      if (semantic === "pull_request_comment" && !object(payload.issue?.pull_request)) return undefined;
    }
    if (!subscription.types.includes(payload.action)) return undefined;
    org = organization(source, payload);
    if (event === "projects_v2_item") {
      const binding = source.scope.project;
      requireThat(binding && payload.projects_v2_item?.project_node_id === binding.nodeId, "source-project-mismatch", 403);
      const item = numericId(payload.projects_v2_item.id);
      if (binding.ownerKind === "orgs") requireThat(org && binding.owner.toLowerCase() === org.toLowerCase(), "source-project-owner-mismatch", 403);
      url = `${github.apiUrl}/${binding.ownerKind}/${binding.owner}/projectsV2/${binding.number}/items/${item}`;
      // An API mock may use HTTP, but resource references remain canonical HTTPS.
      httpsURL(url);
    } else if (event.endsWith("_alert")) {
      repo = repository(source, payload);
      const number = numericId(payload.alert?.number);
      const family = event === "code_scanning_alert" ? "code-scanning" : "dependabot";
      url = httpsURL(payload.alert.url).href;
      requireThat(url === `${github.apiUrl}/repos/${repo}/${family}/alerts/${number}`, "source-alert-url-mismatch");
    } else {
      repo = repository(source, payload);
      if (event === "slash_command") {
        const filter = subscription.command_filter;
        if (!filter.events.includes(semantic) || !commandActions[semantic]?.includes(payload.action)) return undefined;
        const body = semantic === "issues" ? payload.issue?.body : semantic === "pull_request" ? payload.pull_request?.body : semantic === "discussion" ? payload.discussion?.body : payload.comment?.body;
        const token = typeof body === "string" ? body.match(/^\/([a-zA-Z0-9][a-zA-Z0-9._-]*)(?=$|\s)/)?.[1] : undefined;
        if (!token || !filter.names.some(name => matchesName(name, token))) return undefined;
        url = githubReference(semantic, source, payload, github);
        if (!selectors(subscription, repo, org)) return undefined;
        return { url, type: [payload.action], filters: { event: [semantic], command: [token] } };
      }
      url = githubReference(event, source, payload, github);
    }
    if (!selectors(subscription, repo, org)) return undefined;
    types = [payload.action];
  } else {
    const native = source.provider === "jira" ? payload.webhookEvent : source.provider === "linear" ? payload.action : payload.eventType;
    if (!subscription.native_events.includes(native) || (source.provider === "linear" && payload.type !== "Issue")) return undefined;
    const scope = source.scope;
    if (source.provider === "jira") {
      requireThat(typeof payload.issue?.id === "string" && /^[1-9][0-9]*$/.test(payload.issue.id) && payload.issue.fields?.project?.key === scope.project, "source-jira-scope-mismatch", 403);
      url = httpsURL(payload.issue.self).href;
      requireThat(new URL(url).origin === scope.origin && new RegExp(`^/rest/api/[23]/issue/${payload.issue.id}$`).test(new URL(url).pathname) && !new URL(url).hash, "source-jira-url-mismatch");
      types = jiraActivities(payload);
    } else if (source.provider === "linear") {
      requireThat(nonempty(payload.data?.id) && payload.organizationId === scope.organizationId && (!scope.teamId || payload.data.teamId === scope.teamId), "source-linear-scope-mismatch", 403);
      requireThat(typeof payload.data.identifier === "string" && /^[A-Z][A-Z0-9_-]*-[1-9][0-9]*$/.test(payload.data.identifier), "missing-linear-url-binding");
      const base = `https://linear.app/${scope.workspace}/issue/${payload.data.identifier}`;
      const links = [...(Object.hasOwn(payload, "url") ? [payload.url] : []), ...(Object.hasOwn(payload.data, "url") ? [payload.data.url] : [])];
      requireThat(links.length > 0, "missing-linear-url-binding");
      for (const link of links) {
        const candidate = httpsURL(link).href;
        requireThat(!new URL(candidate).hash && (candidate === base || (candidate.startsWith(`${base}/`) && /^(?:[A-Za-z0-9_-]|%[0-9a-fA-F]{2})+$/.test(candidate.slice(base.length + 1)))), "source-linear-url-mismatch");
      }
      url = links[0];
      types = linearActivities(payload);
    } else {
      requireThat(payload.resourceContainers?.collection?.id === scope.collectionId && payload.resourceContainers?.project?.id === scope.projectId, "source-azure-scope-mismatch", 403);
      const snapshot = native !== "workitem.updated" || (payload.resourceVersion === "1.0" && !payload.resource?.revision);
      const id = numericId(payload.resource?.workItemId ?? (snapshot ? payload.resource?.id : undefined));
      if (snapshot && payload.resource?.workItemId !== undefined && payload.resource?.id !== undefined) requireThat(numericId(payload.resource.id) === id, "contradictory-work-item-identity");
      if (object(payload.resource?.revision) && payload.resource.revision.id !== undefined) requireThat(numericId(payload.resource.revision.id) === id, "contradictory-work-item-identity");
      verifyAzureLinks(payload, scope, id);
      url = `${scope.collectionUrl}${scope.projectSegment ? `/${scope.projectSegment}` : ""}/_apis/wit/workItems/${id}`;
      types = azureActivities(payload, subscription.activity_normalization.closed_states ?? []);
    }
    types = [...new Set(types.filter(type => subscription.types.includes(type)))].sort();
    requireThat(!(types.includes("closed") && types.includes("reopened")), "contradictory-lifecycle-evidence");
    if (!types.length) return undefined;
  }
  const filters = conditionsProjection(subscription, payload);
  if (!filters) return undefined;
  if (event === "azure_devops_issue" && types.some(type => ["closed", "reopened"].includes(type))) {
    const state = payload.resource.fields["System.State"];
    filters["closed-states"] = [types.includes("closed") ? state.newValue : state.oldValue];
  }
  return { url, type: types, ...(Object.keys(filters).length ? { filters } : {}) };
}

function selectors(subscription, repo, organizationName) {
  if (subscription.repos && (!repo || !subscription.repos.some(pattern => (pattern.endsWith("/*") ? repo.toLowerCase().startsWith(pattern.toLowerCase().slice(0, -1)) : repo.toLowerCase() === pattern.toLowerCase())))) return false;
  if (subscription.orgs && (!organizationName || !patternsMatch(organizationName, subscription.orgs))) return false;
  return true;
}
