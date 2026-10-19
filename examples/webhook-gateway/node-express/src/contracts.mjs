import { readFileSync } from "node:fs";
import { canonical, decimal, hash, httpsURL, nonempty, object, requireThat, scalar, shape } from "./core.mjs";

const catalog = name => JSON.parse(readFileSync(new URL(`../catalogs/${name}`, import.meta.url)));
export const notifications = catalog("github_notification_webhooks.json");
export const providers = catalog("provider_issue_webhooks.json");
export const resources = catalog("webhook_events.json");
export const definitions = {
  ...resources,
  ...providers,
  ...Object.fromEntries(Object.entries(notifications.events).map(([event, definition]) => [event, { ...definition, filters: { ...notifications.common_filters, ...definition.filters } }])),
};
export const commandActions = {
  issues: ["opened", "edited", "reopened"],
  issue_comment: ["created", "edited"],
  pull_request_comment: ["created", "edited"],
  pull_request: ["opened", "edited", "reopened"],
  pull_request_review_comment: ["created", "edited"],
  discussion: ["created", "edited"],
  discussion_comment: ["created", "edited"],
};
export const contextMapping = {
  envelope: "github.event.client_payload",
  aw_context: "github.event.client_payload.aw_context",
  event: "github.event.client_payload.aw_context.webhook.event",
  deliveries: "github.event.client_payload.aw_context.webhook.deliveries",
  items: "aw_context.items",
};
const list = value => Array.isArray(value) && value.length > 0 && new Set(value).size === value.length;
const sameSet = (left, right) => list(left) && canonical([...left].sort()) === canonical([...right].sort());

function conditionValid(condition, definition, types) {
  shape(condition, ["filter", "path", "operator", "value_type", "values"]);
  const selector = definition.filters[condition.filter];
  requireThat(selector && canonical(selector.path) === canonical(condition.path) && selector.type === condition.value_type && condition.operator === "equals", "unknown-condition");
  requireThat(
    list(condition.values) && condition.values.every(value => scalar(value, selector.type) && (typeof value !== "string" || !value.includes("${{")) && (!selector.enum || selector.enum.includes(value))),
    "invalid-condition-values"
  );
  requireThat(!selector.actions || types.every(type => selector.actions.includes(type)), "condition-unavailable-for-activity");
}

export function validateSubscription(subscription) {
  shape(subscription, ["schema_version", "event", "types", "conditions", "dispatch"], ["native_events", "provider", "activity_normalization", "command_filter", "repos", "orgs", "any_of"]);
  requireThat(subscription.schema_version === 2 && Array.isArray(subscription.conditions), "unsupported-subscription");
  const dispatch = subscription.dispatch;
  shape(dispatch, ["repository", "method", "event_type", "workflow", "payload_projection", "context", "batch", "max_payload_bytes", "max_request_bytes"]);
  requireThat(dispatch.repository === "${{ github.repository }}" && dispatch.method === "repository_dispatch" && /^\.github\/workflows\/[A-Za-z0-9_.-]+\.lock\.yml$/.test(dispatch.workflow), "invalid-dispatch-contract");
  const workflowId = dispatch.workflow.slice(".github/workflows/".length, -".lock.yml".length);
  requireThat(dispatch.event_type === `gh-aw-webhook-${hash(workflowId).slice(0, 24)}`, "invalid-workflow-route");
  requireThat(
    canonical(dispatch.payload_projection) === canonical({ policy: "resource-url-types-filters/v2" }) &&
      canonical(dispatch.context) === canonical(contextMapping) &&
      canonical(dispatch.batch) === canonical({ mode: "same-event", deduplicate_by: "delivery_id" }) &&
      dispatch.max_payload_bytes === 65535 &&
      dispatch.max_request_bytes === 65535,
    "unsupported-dispatch-contract"
  );
  const command = ["slash_command", "label_command"].includes(subscription.event);
  if (subscription.event === "label_command") {
    // No generation-aware provider compare-and-remove primitive is specified.
    requireThat(false, "label-consumption-unsupported", 422);
  }
  const definition = definitions[subscription.event];
  requireThat(command || definition, "unsupported-family", 422);
  requireThat(list(subscription.types) && subscription.types.every(type => (command ? Object.values(commandActions).flat() : definition.actions).includes(type)), "unsupported-activities");
  if (subscription.repos !== undefined) {
    requireThat(
      !definition?.provider &&
        subscription.event !== "projects_v2_item" &&
        list(subscription.repos) &&
        subscription.repos.every(repo => typeof repo === "string" && /^[A-Za-z0-9_-]+\/(?:[A-Za-z0-9._-]+|\*)$/.test(repo) && !/\/\.{1,2}$/.test(repo)),
      "invalid-repository-selectors"
    );
  }
  if (subscription.orgs !== undefined) {
    requireThat(!definition?.provider && list(subscription.orgs) && subscription.orgs.every(org => typeof org === "string" && /^[A-Za-z0-9*](?:[A-Za-z0-9*-]{0,37}[A-Za-z0-9*])?$/.test(org)), "invalid-organization-selectors");
  }
  if (command) {
    requireThat(!subscription.provider && !subscription.activity_normalization && !subscription.any_of && subscription.conditions.length === 0, "invalid-command-subscription");
    const filter = subscription.command_filter;
    shape(filter, ["policy", "names", "events", "strategy"]);
    requireThat(
      filter.policy === "slash-command/v1" &&
        list(filter.names) &&
        filter.names.length <= 25 &&
        filter.names.every(name => typeof name === "string" && /^(?:\*|[A-Za-z0-9][A-Za-z0-9._-]*\*?)$/.test(name)) &&
        list(filter.events) &&
        filter.events.every(event => Object.hasOwn(commandActions, event)) &&
        ["inline", "centralized"].includes(filter.strategy),
      "invalid-command-filter"
    );
    requireThat(
      sameSet(subscription.native_events, [...new Set(filter.events.map(event => (event === "pull_request_comment" ? "issue_comment" : event)))]) &&
        sameSet(subscription.types, [...new Set(filter.events.flatMap(event => commandActions[event]))]),
      "invalid-command-native-events"
    );
    return;
  }
  requireThat(subscription.command_filter === undefined, "unexpected-command-filter");
  for (const condition of subscription.conditions) conditionValid(condition, definition, subscription.types);
  if (subscription.any_of !== undefined) {
    requireThat(subscription.event === "projects_v2_item" && Array.isArray(subscription.any_of) && subscription.any_of.length > 0, "invalid-condition-alternatives");
    for (const group of subscription.any_of) {
      requireThat(
        Array.isArray(group) &&
          group.length === 2 &&
          sameSet(
            group.map(condition => condition.filter),
            ["field.from.name", "field.to.name"]
          ),
        "invalid-transition-pair"
      );
      for (const condition of group) conditionValid(condition, definition, subscription.types);
    }
    requireThat(
      subscription.conditions.some(condition => condition.filter === "field.name" && canonical(condition.values) === '["Status"]') &&
        subscription.conditions.some(condition => condition.filter === "field.type" && canonical(condition.values) === '["single_select"]'),
      "invalid-status-conditions"
    );
  }
  if (definition.provider) {
    requireThat(subscription.provider === definition.provider, "invalid-provider");
    shape(subscription.activity_normalization, ["policy"], subscription.event === "azure_devops_issue" ? ["closed_states"] : []);
    requireThat(subscription.activity_normalization.policy === `${definition.provider}-issue/v1`, "unknown-normalization-policy");
    const terminal = subscription.activity_normalization.closed_states;
    requireThat(terminal === undefined || (list(terminal) && terminal.every(nonempty)), "invalid-terminal-states");
    requireThat(subscription.event !== "azure_devops_issue" || !subscription.types.some(type => ["closed", "reopened"].includes(type)) || terminal, "terminal-states-required");
    requireThat(sameSet(subscription.native_events, [...new Set(subscription.types.map(type => definition.native_events[type]))]), "invalid-provider-native-events");
    const scope = { jira_issue: "project", linear_issue: "organization-id", azure_devops_issue: "project_id" }[subscription.event];
    requireThat(
      subscription.conditions.some(condition => condition.filter === scope),
      "provider-scope-required"
    );
  } else {
    requireThat(subscription.provider === undefined && subscription.activity_normalization === undefined, "unexpected-provider-policy");
    const native = definition.native_event;
    requireThat(native ? sameSet(subscription.native_events, [native]) : subscription.native_events === undefined, "invalid-github-native-events");
    if (subscription.event === "projects_v2_item")
      requireThat(
        subscription.conditions.some(condition => condition.filter === "project_id"),
        "project-scope-required"
      );
  }
}

function endpoint(value, testHTTP) {
  const url = new URL(value);
  requireThat(
    (url.protocol === "https:" || (testHTTP && url.protocol === "http:" && ["127.0.0.1", "[::1]"].includes(url.hostname))) &&
      !url.username &&
      !url.password &&
      !value.includes("?") &&
      !value.includes("#") &&
      !value.endsWith("/") &&
      value === `${url.origin}${url.pathname === "/" ? "" : url.pathname}`,
    "invalid-api-base"
  );
}

export function validateConfiguration(config, env, { testHTTP = false } = {}) {
  shape(config, ["sources", "destinations", "github"]);
  shape(config.github, ["apiUrl", "serverUrl", "appId", "botId", "privateKeyEnv"]);
  endpoint(config.github.apiUrl, testHTTP);
  endpoint(config.github.serverUrl, false);
  decimal(config.github.appId);
  decimal(config.github.botId);
  requireThat(config.github.appId === env.GH_AW_GATEWAY_APP_ID && config.github.botId === env.GH_AW_GATEWAY_APP_BOT_ID && config.github.privateKeyEnv === "GH_AW_GATEWAY_APP_PRIVATE_KEY", "central-identity-mismatch");
  requireThat(Array.isArray(config.sources) && Array.isArray(config.destinations), "invalid-configuration");
  requireThat(new Set(config.sources.map(source => source.id)).size === config.sources.length && new Set(config.destinations.map(destination => destination.id)).size === config.destinations.length, "duplicate-enrollment");
  for (const source of config.sources) {
    shape(source, ["id", "provider", "approved", "revision", "credentialEnv", "scope"]);
    requireThat(
      /^[A-Za-z0-9_-]{1,100}$/.test(source.id) &&
        ["github", "jira", "linear", "azure-devops"].includes(source.provider) &&
        typeof source.approved === "boolean" &&
        nonempty(source.revision) &&
        /^[A-Z][A-Z0-9_]*$/.test(source.credentialEnv) &&
        !source.credentialEnv.startsWith("GH_AW_GATEWAY_"),
      "invalid-source-enrollment"
    );
    const scope = source.scope;
    if (source.provider === "github") {
      shape(scope, [], ["repositoryId", "repository", "ownerId", "organization", "project"]);
      requireThat(scope.repository || scope.project, "github-source-scope-required");
      if (scope.repository) {
        decimal(scope.repositoryId);
        decimal(scope.ownerId);
        requireThat(/^[A-Za-z0-9_-]+\/[A-Za-z0-9._-]+$/.test(scope.repository) && !/\/\.{1,2}$/.test(scope.repository), "invalid-source-repository");
      }
      if (scope.organization) {
        shape(scope.organization, ["id", "login"]);
        decimal(scope.organization.id);
        requireThat(/^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$/.test(scope.organization.login), "invalid-source-organization");
      }
      if (scope.project) {
        shape(scope.project, ["nodeId", "ownerKind", "owner", "number"]);
        decimal(scope.project.number);
        requireThat(nonempty(scope.project.nodeId) && ["orgs", "users"].includes(scope.project.ownerKind) && /^[A-Za-z0-9_-]+$/.test(scope.project.owner), "invalid-project-binding");
      }
    } else if (source.provider === "jira") {
      shape(scope, ["origin", "project"]);
      endpoint(scope.origin, false);
      requireThat(new URL(scope.origin).origin === scope.origin && nonempty(scope.project), "invalid-jira-scope");
    } else if (source.provider === "linear") {
      shape(scope, ["organizationId", "workspace"], ["teamId"]);
      requireThat(nonempty(scope.organizationId) && /^[A-Za-z0-9_-]+$/.test(scope.workspace) && (scope.teamId === undefined || nonempty(scope.teamId)), "invalid-linear-scope");
    } else {
      shape(scope, ["collectionId", "projectId", "collectionUrl"], ["projectSegment"]);
      const url = httpsURL(scope.collectionUrl);
      requireThat(
        nonempty(scope.collectionId) && nonempty(scope.projectId) && !url.hash && /^\/[A-Za-z0-9_-]+$/.test(url.pathname) && (url.hostname === "dev.azure.com" || /^[A-Za-z0-9-]+\.visualstudio\.com$/.test(url.hostname)),
        "invalid-azure-scope"
      );
      requireThat(scope.projectSegment === undefined || /^(?:[A-Za-z0-9_.-]|%[0-9A-Fa-f]{2})+$/.test(scope.projectSegment), "invalid-azure-project-segment");
    }
  }
  for (const destination of config.destinations) {
    shape(destination, [
      "id",
      "approved",
      "revision",
      "repository",
      "repositoryId",
      "installationId",
      "accountId",
      "accountLogin",
      "installationPermissions",
      "defaultBranch",
      "workflowRevision",
      "artifactSha256",
      "gateVerified",
      "sourceIds",
      "subscriptions",
    ]);
    requireThat(
      nonempty(destination.id) &&
        nonempty(destination.revision) &&
        typeof destination.approved === "boolean" &&
        typeof destination.gateVerified === "boolean" &&
        /^[A-Za-z0-9_-]+\/[A-Za-z0-9._-]+$/.test(destination.repository) &&
        !/\/\.{1,2}$/.test(destination.repository) &&
        nonempty(destination.defaultBranch) &&
        /^[0-9a-f]{40}$/.test(destination.workflowRevision) &&
        /^[0-9a-f]{64}$/.test(destination.artifactSha256),
      "invalid-destination-enrollment"
    );
    for (const id of [destination.repositoryId, destination.installationId, destination.accountId]) decimal(id);
    requireThat(
      nonempty(destination.accountLogin) &&
        list(destination.sourceIds) &&
        destination.sourceIds.every(id => config.sources.some(source => source.id === id)) &&
        Array.isArray(destination.subscriptions) &&
        destination.subscriptions.length > 0,
      "invalid-destination-binding"
    );
    requireThat(
      object(destination.installationPermissions) && destination.installationPermissions.contents === "write" && Object.values(destination.installationPermissions).every(value => ["read", "write"].includes(value)),
      "invalid-approved-installation-permissions"
    );
    const events = new Set();
    const workflows = new Set();
    for (const subscription of destination.subscriptions) {
      validateSubscription(subscription);
      requireThat(!events.has(subscription.event), "ambiguous-subscription-family");
      events.add(subscription.event);
      workflows.add(subscription.dispatch.workflow);
    }
    requireThat(workflows.size === 1, "one-workflow-per-destination-binding");
  }
  return config;
}

export const approvalFingerprint = (config, source, destination) => hash(canonical({ github: config.github, source, destination }));
export const subscriptionsDigest = subscriptions => hash(canonical(subscriptions));
