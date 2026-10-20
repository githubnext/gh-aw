// @ts-check

const { ERR_VALIDATION } = require("./error_codes.cjs");
const { events: GITHUB_NOTIFICATION_EVENTS } = require("./github_notification_webhooks.json");

function isGitHubNotification(event) {
  return typeof event === "string" && Object.hasOwn(GITHUB_NOTIFICATION_EVENTS, event);
}

function validGitHubNotificationSubscription(subscription) {
  const definition = GITHUB_NOTIFICATION_EVENTS[subscription.event];
  return (
    subscription.provider === undefined &&
    subscription.activity_normalization === undefined &&
    subscription.command_filter === undefined &&
    subscription.any_of === undefined &&
    Array.isArray(subscription.native_events) &&
    subscription.native_events.length === 1 &&
    subscription.native_events[0] === definition.native_event &&
    new Set(subscription.types).size === subscription.types.length &&
    subscription.types.every(type => definition.actions.includes(type))
  );
}

function resolveGitHubNotificationResource(event, url, env) {
  const reject = () => {
    throw new Error(`${ERR_VALIDATION}: Webhook validation failed: GitHub notification requires a canonical item/comment URL matching its event family`);
  };
  let base;
  try {
    base = new URL(env.GITHUB_SERVER_URL || "https://github.com");
  } catch {
    return reject();
  }
  const definition = GITHUB_NOTIFICATION_EVENTS[event];
  const prefix = base.pathname.replace(/\/$/, "");
  if (url.origin !== base.origin || !url.pathname.startsWith(`${prefix}/`)) return reject();
  const match = url.pathname.slice(prefix.length).match(/^\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+)\/(issues|pull|discussions|commit)\/([1-9][0-9]*|[a-f0-9]{40})$/);
  if (!match || match[3] !== definition.path || !/^(?!\.{1,2}$).+$/.test(match[2])) return reject();
  if (definition.path === "commit" ? !/^[a-f0-9]{40}$/.test(match[4]) : !/^[1-9][0-9]*$/.test(match[4])) return reject();
  const validAnchor = definition.anchor ? url.hash.startsWith(`#${definition.anchor}`) && /^[1-9][0-9]*$/.test(url.hash.slice(definition.anchor.length + 1)) : url.hash === "";
  if (!validAnchor) return reject();
  return { item_type: event, item_number: match[4], repo: `${match[1]}/${match[2]}`, source: `${url.origin}${url.pathname}`, url: url.href };
}

module.exports = { GITHUB_NOTIFICATION_EVENTS, isGitHubNotification, validGitHubNotificationSubscription, resolveGitHubNotificationResource };
