// @ts-check

const { ERR_VALIDATION } = require("./error_codes.cjs");
const { matchesCommandName, parseSlashCommand } = require("./slash_command_matcher.cjs");

const COMMAND_SOURCE_ACTIONS = {
  issues: ["opened", "edited", "reopened"],
  issue_comment: ["created", "edited"],
  pull_request_comment: ["created", "edited"],
  pull_request: ["opened", "edited", "reopened"],
  pull_request_review_comment: ["created", "edited"],
  discussion: ["created", "edited"],
  discussion_comment: ["created", "edited"],
};
const LABEL_SOURCE_EVENTS = ["issues", "pull_request", "discussion"];
const COMMAND_SOURCE_URLS = {
  issues: { path: "issues", anchor: /^$/ },
  issue_comment: { path: "issues", anchor: /^#issuecomment-[1-9][0-9]*$/ },
  pull_request_comment: { path: "pull", anchor: /^#issuecomment-[1-9][0-9]*$/ },
  pull_request: { path: "pull", anchor: /^$/ },
  pull_request_review_comment: { path: "pull", anchor: /^#discussion_r[1-9][0-9]*$/ },
  discussion: { path: "discussions", anchor: /^$/ },
  discussion_comment: { path: "discussions", anchor: /^#discussioncomment-[1-9][0-9]*$/ },
};

function isGatewayCommand(event) {
  return event === "slash_command" || event === "label_command";
}

function validGatewayCommandSubscription(subscription) {
  const filter = subscription.command_filter;
  const label = subscription.event === "label_command";
  const events = label ? LABEL_SOURCE_EVENTS : Object.keys(COMMAND_SOURCE_ACTIONS);
  if (
    !filter ||
    typeof filter !== "object" ||
    Array.isArray(filter) ||
    Object.keys(filter).some(key => !["policy", "names", "events", "strategy", ...(label ? ["remove_label"] : [])].includes(key)) ||
    filter.policy !== (label ? "label-command/v1" : "slash-command/v1") ||
    !Array.isArray(filter.names) ||
    filter.names.length === 0 ||
    filter.names.length > (label ? 50 : 25) ||
    new Set(filter.names).size !== filter.names.length ||
    !filter.names.every(name => typeof name === "string" && name.trim() !== "" && !name.includes("${{") && (label || /^(\*|[^/*][^/*]*\*?)$/.test(name))) ||
    !Array.isArray(filter.events) ||
    filter.events.length === 0 ||
    new Set(filter.events).size !== filter.events.length ||
    !filter.events.every(event => events.includes(event)) ||
    !["inline", label ? "decentralized" : "centralized"].includes(filter.strategy) ||
    (label && filter.remove_label !== true) ||
    subscription.conditions.length !== 0 ||
    subscription.any_of !== undefined ||
    subscription.provider !== undefined ||
    subscription.activity_normalization !== undefined
  ) {
    return false;
  }
  const native = [...new Set(filter.events.map(event => (event === "pull_request_comment" ? "issue_comment" : event)))];
  return (
    Array.isArray(subscription.native_events) &&
    native.length === subscription.native_events.length &&
    native.every(event => subscription.native_events.includes(event)) &&
    subscription.types.every(type => (label ? type === "labeled" : filter.events.some(event => COMMAND_SOURCE_ACTIONS[event].includes(type))))
  );
}

function matchesGatewayCommandSubscription(subscription, payload) {
  const filter = subscription.command_filter;
  const label = subscription.event === "label_command";
  const projected = payload.filters;
  const key = label ? "label" : "command";
  if (!projected || Object.keys(projected).length !== 2 || projected.event?.length !== 1 || projected[key]?.length !== 1 || !filter.events.includes(projected.event[0])) return false;
  const value = projected[key][0];
  const action = payload.type[0];
  return label ? action === "labeled" && filter.names.includes(value) : COMMAND_SOURCE_ACTIONS[projected.event[0]].includes(action) && parseSlashCommand(`/${value}`) === value && filter.names.some(name => matchesCommandName(name, value));
}

function resolveGatewayCommandResource(event, url, payload, env) {
  const reject = () => {
    throw new Error(`${ERR_VALIDATION}: Webhook validation failed: gateway command requires a canonical GitHub item/comment URL matching its declared source event`);
  };
  let base;
  try {
    base = new URL(env.GITHUB_SERVER_URL || "https://github.com");
  } catch {
    return reject();
  }
  const sourceEvent = payload.filters?.event?.[0];
  if (!Object.hasOwn(COMMAND_SOURCE_URLS, sourceEvent) || (event === "label_command" && !LABEL_SOURCE_EVENTS.includes(sourceEvent))) return reject();
  const { path, anchor } = COMMAND_SOURCE_URLS[sourceEvent];
  const prefix = base.pathname.replace(/\/$/, "");
  if (url.origin !== base.origin || !url.pathname.startsWith(`${prefix}/`)) return reject();
  const match = url.pathname.slice(prefix.length).match(/^\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+)\/(issues|pull|discussions)\/([1-9][0-9]*)$/);
  if (!match || match[3] !== path || !anchor.test(url.hash)) return reject();
  return { item_type: event, item_number: match[4], repo: `${match[1]}/${match[2]}`, source: `${url.origin}${url.pathname}`, url: url.href };
}

module.exports = { isGatewayCommand, validGatewayCommandSubscription, matchesGatewayCommandSubscription, resolveGatewayCommandResource, COMMAND_SOURCE_ACTIONS };
