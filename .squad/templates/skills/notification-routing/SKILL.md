---
name: "notification-routing"
description: "Route agent notifications to specific channels by type — prevent alert fatigue from single-channel flooding"
domain: "communication"
confidence: "high"
source: "earned"
---

## Context

When a Squad grows beyond a few agents, notifications flood a single channel — failure alerts drown in daily
briefings, tech news buries security findings, and everything gets ignored. This is the pub-sub problem:
a single message queue for everything is a recipe for missed alerts.

The fix is **topic-based routing**: agents tag notifications with a channel type, and a routing function
sends them to the appropriate destination.

**Trigger symptoms:**
- Important alerts missed because they're buried in routine notifications
- Team members turning off notifications entirely (signal overwhelm)
- Onboarding friction: "where do I look for X?"

## Patterns

### Channel Config Schema

Define a `.squad/teams-channels.json` (or equivalent) mapping notification types to channel identifiers:

```json
{
  "teamId": "your-team-id",
  "channels": {
    "notifications": "squad-alerts",
    "tech-news":     "tech-news",
    "security":      "security-findings",
    "releases":      "release-announcements",
    "daily-digest":  "daily-digest"
  }
}
```

Place this in `.squad/` (git-tracked, shared across the team). For platforms that use channel IDs instead of
names (Teams, Slack), store the resolved ID alongside the name to avoid name-collision bugs:

```json
{
  "channels": {
    "notifications": { "name": "squad-alerts", "id": "channel-id-opaque-string" }
  }
}
```

### CHANNEL: Tag Convention

Agents prefix their output with `CHANNEL:<type>` to signal where the notification should go:

```
CHANNEL:security
Worf found 3 new CVEs in dependency scan: lodash@4.17.15, minimist@1.2.5
```

### Routing Dispatcher (shell pseudocode)

```bash
dispatch_notification() {
  local raw_output="$1"
  local channel="notifications"  # default

  if echo "$raw_output" | grep -qE '^CHANNEL:[a-z][a-z0-9-]*'; then
    channel=$(echo "$raw_output" | head -1 | cut -d: -f2)
    raw_output=$(echo "$raw_output" | tail -n +2)
  fi

  send_notification --channel "$channel" --message "$raw_output"
}
```

### Provider-Agnostic Adapter

The routing layer is provider-agnostic. Plug in your platform adapter:

```
.squad/notify-adapter.sh   # Teams / Slack / Discord / webhook -- swappable
```

The routing config and CHANNEL: tags never change. Only the adapter changes per deployment.

## Anti-Patterns

**Never send all notification types to one channel:**
```
send_notification --channel "general" --message "$anything"
```

**Never use display names as identifiers (name collision risk):**
```
send_to_team --name "Squad" --channel "notifications"
```

Resolve channel IDs once at setup. Use IDs at runtime.

## Distributed Systems Pattern

This is **pub-sub with topic routing** -- the same principle as Kafka topics, RabbitMQ routing keys, and
AWS SNS topic filtering. Route by type. Each consumer subscribes to the topics it cares about.