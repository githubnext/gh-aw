---
# Pattern: Integration Bot
# Complexity: Intermediate
# Use Case: Integrate with external tools like Slack, Notion, or custom APIs
name: Integration Bot
description: Integrates GitHub events with external tools and services
on:
  issues:
    types: [opened, closed]
  pull_request:
    types: [opened, closed]
  discussion:
    types: [created]
  # TODO: Add other triggers as needed
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
engine: copilot
network:
  # TODO: Add external domains you need to access
  allowed:
    - "hooks.slack.com"
    - "api.notion.com"
    - "api.example.com"
tools:
  github:
    mode: remote
    toolsets: [issues, pull_requests, discussions]
  bash:
    - "curl *"
    - "jq *"
# Note: Integration bots typically don't use GitHub safe-outputs
# as they integrate with external APIs via network calls
timeout-minutes: 10
strict: true
---

# Integration Bot

Integrate GitHub events with external tools like Slack, Notion, project management systems, or custom APIs.

## Integration Types

# TODO: Choose which integrations to implement

### 1. Slack Notifications

Send notifications to Slack channels:

```markdown
**Events**: Issues, PRs, discussions, releases
**Actions**:
- Post to channel when issue created
- Thread updates on issue comments
- Notify on PR merge
- Alert on high-priority issues

**Setup Required**:
- Slack webhook URL (store in repository secret)
- Channel configuration
```

### 2. Notion Database Sync

Sync GitHub items to Notion database:

```markdown
**Events**: Issues, PRs created/updated
**Actions**:
- Create Notion page for each issue
- Update status when issue changes
- Sync labels as Notion tags
- Link to GitHub items

**Setup Required**:
- Notion API key (repository secret)
- Database ID
- Property mappings
```

### 3. Project Management Tools

Integrate with Jira, Linear, Asana, etc.:

```markdown
**Events**: Issues, PRs, commits
**Actions**:
- Create tickets in external system
- Sync status changes
- Link commits to tickets
- Update ticket on PR merge

**Setup Required**:
- API credentials
- Project/board IDs
- Field mappings
```

### 4. Analytics/Monitoring

Send data to analytics platforms:

```markdown
**Events**: Workflow runs, deployments, metrics
**Actions**:
- Track deployment frequency
- Monitor success rates
- Alert on anomalies
- Dashboard updates

**Setup Required**:
- Analytics API endpoint
- Metrics definitions
```

### 5. Custom Webhooks

Call custom webhook endpoints:

```markdown
**Events**: Any GitHub event
**Actions**:
- POST event data to webhook
- Transform data as needed
- Handle webhook responses
- Retry on failures

**Setup Required**:
- Webhook URL
- Authentication method
- Data transformation rules
```

## Implementation Steps

### Step 1: Get Event Data

```bash
# Extract relevant data from GitHub event
# Use GitHub tools to get event details
EVENT_TYPE="(issues, pull_request, etc.)"
REPOSITORY="(repository name)"

# TODO: Extract data based on event type
if [ "$EVENT_TYPE" = "issues" ]; then
  ISSUE_NUMBER="(issue number)"
  ISSUE_TITLE="(issue title)"
  ISSUE_URL="(issue URL)"
  ISSUE_STATE="(open/closed)"
  AUTHOR="(author username)"
  
elif [ "$EVENT_TYPE" = "pull_request" ]; then
  PR_NUMBER="(PR number)"
  PR_TITLE="(PR title)"
  PR_URL="(PR URL)"
  PR_STATE="(open/closed)"
  AUTHOR="(author username)"
fi
```

### Step 2: Format Message/Payload

```python
#!/usr/bin/env python3
"""
Message Formatter
TODO: Customize for your integration
"""
import json
import os

event_type = os.environ.get('GITHUB_EVENT_NAME')

# TODO: Format message based on integration type

def format_slack_message(event_data):
    """Format message for Slack"""
    if event_type == 'issues':
        return {
            "text": f"New Issue: {event_data['title']}",
            "blocks": [
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": f"*New Issue*: <{event_data['url']}|#{event_data['number']} {event_data['title']}>"
                    }
                },
                {
                    "type": "context",
                    "elements": [
                        {
                            "type": "mrkdwn",
                            "text": f"Created by {event_data['author']} in {event_data['repository']}"
                        }
                    ]
                },
                {
                    "type": "actions",
                    "elements": [
                        {
                            "type": "button",
                            "text": {"type": "plain_text", "text": "View Issue"},
                            "url": event_data['url']
                        }
                    ]
                }
            ]
        }
    return {}

def format_notion_page(event_data):
    """Format Notion database entry"""
    if event_type == 'issues':
        return {
            "parent": {"database_id": os.environ.get('NOTION_DATABASE_ID')},
            "properties": {
                "Name": {
                    "title": [{"text": {"content": event_data['title']}}]
                },
                "Status": {
                    "select": {"name": "Open" if event_data['state'] == 'open' else "Closed"}
                },
                "GitHub": {
                    "url": event_data['url']
                },
                "Number": {
                    "number": event_data['number']
                },
                "Author": {
                    "rich_text": [{"text": {"content": event_data['author']}}]
                }
            }
        }
    return {}

def format_webhook_payload(event_data):
    """Format custom webhook payload"""
    # TODO: Customize payload structure
    return {
        "event_type": event_type,
        "repository": event_data['repository'],
        "timestamp": event_data.get('timestamp'),
        "data": event_data
    }

# Load event data
event_data = {
    'type': event_type,
    'number': os.environ.get('ISSUE_NUMBER') or os.environ.get('PR_NUMBER'),
    'title': os.environ.get('ISSUE_TITLE') or os.environ.get('PR_TITLE'),
    'url': os.environ.get('ISSUE_URL') or os.environ.get('PR_URL'),
    'state': os.environ.get('ISSUE_STATE') or os.environ.get('PR_STATE'),
    'author': os.environ.get('AUTHOR'),
    'repository': os.environ.get('REPOSITORY')
}

# Format for target integration
# TODO: Choose your integration type
slack_message = format_slack_message(event_data)

# Save formatted message
with open('/tmp/formatted-message.json', 'w') as f:
    json.dump(slack_message, f, indent=2)

print("Message formatted for integration")
```

### Step 3: Send to External Service

```bash
# TODO: Choose your integration method

# Example 1: Slack Webhook
send_to_slack() {
  local webhook_url="$1"  # From repository secret
  local message_file="$2"
  
  curl -X POST "$webhook_url" \
    -H "Content-Type: application/json" \
    -d @"$message_file"
  
  if [ $? -eq 0 ]; then
    echo "✓ Sent to Slack successfully"
  else
    echo "✗ Failed to send to Slack"
    return 1
  fi
}

# Example 2: Notion API
send_to_notion() {
  local api_key="$1"  # From repository secret
  local database_id="$2"
  local payload_file="$3"
  
  curl -X POST "https://api.notion.com/v1/pages" \
    -H "Authorization: Bearer $api_key" \
    -H "Content-Type: application/json" \
    -H "Notion-Version: 2022-06-28" \
    -d @"$payload_file"
  
  if [ $? -eq 0 ]; then
    echo "✓ Created Notion page successfully"
  else
    echo "✗ Failed to create Notion page"
    return 1
  fi
}

# Example 3: Custom Webhook
send_to_webhook() {
  local webhook_url="$1"
  local api_key="$2"  # Optional
  local payload_file="$3"
  
  curl -X POST "$webhook_url" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $api_key" \
    -d @"$payload_file" \
    --retry 3 \
    --retry-delay 2
  
  if [ $? -eq 0 ]; then
    echo "✓ Webhook called successfully"
  else
    echo "✗ Webhook call failed"
    return 1
  fi
}

# TODO: Call your integration function
# send_to_slack "$SLACK_WEBHOOK_URL" "/tmp/formatted-message.json"
# send_to_notion "$NOTION_API_KEY" "$NOTION_DATABASE_ID" "/tmp/formatted-message.json"
# send_to_webhook "$WEBHOOK_URL" "$API_KEY" "/tmp/formatted-message.json"
```

### Step 4: Handle Response

```bash
# Capture and handle the response
response=$(curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d @/tmp/formatted-message.json \
  -w "\n%{http_code}" -s)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
  echo "✓ Integration successful"
  echo "Response: $body"
else
  echo "✗ Integration failed with status $http_code"
  echo "Error: $body"
  
  # TODO: Add error handling logic
  # - Retry with exponential backoff
  # - Log error for investigation
  # - Alert team
fi
```

### Step 5: Error Handling & Retry

```python
#!/usr/bin/env python3
"""
Retry Logic with Exponential Backoff
"""
import time
import requests

def call_webhook_with_retry(url, payload, max_retries=3):
    """Call webhook with retry logic"""
    
    for attempt in range(max_retries):
        try:
            response = requests.post(
                url,
                json=payload,
                timeout=10
            )
            
            if response.status_code in [200, 201, 204]:
                print(f"✓ Success on attempt {attempt + 1}")
                return response
            
            if response.status_code >= 500:
                # Server error, retry
                if attempt < max_retries - 1:
                    wait_time = 2 ** attempt  # Exponential backoff
                    print(f"Server error, retrying in {wait_time}s...")
                    time.sleep(wait_time)
                    continue
            else:
                # Client error, don't retry
                print(f"✗ Client error: {response.status_code}")
                print(response.text)
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"✗ Request failed: {e}")
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt
                print(f"Retrying in {wait_time}s...")
                time.sleep(wait_time)
            else:
                print(f"✗ Failed after {max_retries} attempts")
                return None
    
    return None
```

## Customization Guide

### Configure Network Access

```yaml
# TODO: Add domains you need to access
network:
  allowed:
    - "hooks.slack.com"              # Slack
    - "api.notion.com"               # Notion
    - "api.linear.app"               # Linear
    - "*.atlassian.net"              # Jira
    - "api.github.com"               # GitHub API
    - "your-webhook.example.com"     # Custom webhook
```

### Store Secrets Securely

```bash
# TODO: Add secrets to repository settings

# In GitHub repository settings > Secrets and variables > Actions
# Add these secrets:
# - SLACK_WEBHOOK_URL
# - NOTION_API_KEY
# - NOTION_DATABASE_ID
# - CUSTOM_WEBHOOK_URL
# - API_KEY

# Access in workflow:
SLACK_WEBHOOK="(from secrets)"
```

### Add Rate Limiting

```python
# TODO: Implement rate limiting

import time
from datetime import datetime, timedelta

class RateLimiter:
    def __init__(self, max_calls, period_seconds):
        self.max_calls = max_calls
        self.period = timedelta(seconds=period_seconds)
        self.calls = []
    
    def wait_if_needed(self):
        now = datetime.now()
        
        # Remove old calls outside the period
        self.calls = [call for call in self.calls if now - call < self.period]
        
        # Check if we're at the limit
        if len(self.calls) >= self.max_calls:
            # Wait until oldest call expires
            oldest = min(self.calls)
            wait_until = oldest + self.period
            wait_seconds = (wait_until - now).total_seconds()
            
            if wait_seconds > 0:
                print(f"Rate limit reached, waiting {wait_seconds:.1f}s")
                time.sleep(wait_seconds)
        
        self.calls.append(now)

# Usage
limiter = RateLimiter(max_calls=10, period_seconds=60)  # 10 calls per minute
limiter.wait_if_needed()
call_api()
```

## Example Integrations

### Slack Notification Example

```json
{
  "text": "New Issue: Fix authentication bug",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*New Issue*: <https://github.com/org/repo/issues/123|#123 Fix authentication bug>"
      }
    },
    {
      "type": "context",
      "elements": [
        {
          "type": "mrkdwn",
          "text": "Created by @user in org/repo • Priority: High • Label: bug"
        }
      ]
    },
    {
      "type": "actions",
      "elements": [
        {
          "type": "button",
          "text": {"type": "plain_text", "text": "View Issue"},
          "url": "https://github.com/org/repo/issues/123"
        },
        {
          "type": "button",
          "text": {"type": "plain_text", "text": "Assign to Me"},
          "url": "https://github.com/org/repo/issues/123"
        }
      ]
    }
  ]
}
```

### Notion Sync Example

```json
{
  "parent": {"database_id": "abc123"},
  "properties": {
    "Name": {"title": [{"text": {"content": "Fix authentication bug"}}]},
    "Status": {"select": {"name": "Open"}},
    "Priority": {"select": {"name": "High"}},
    "GitHub": {"url": "https://github.com/org/repo/issues/123"},
    "Number": {"number": 123},
    "Labels": {"multi_select": [{"name": "bug"}, {"name": "security"}]},
    "Assignee": {"people": []},
    "Created": {"date": {"start": "2025-12-29"}}
  }
}
```

## Advanced Features

### Bidirectional Sync

```markdown
Sync changes both ways:
- GitHub → External tool (this workflow)
- External tool → GitHub (separate workflow or webhook)

**Example**: Update GitHub issue when Notion page changes
```

### Conditional Integration

```bash
# Only integrate based on conditions
if [[ "$ISSUE_TITLE" == *"[urgent]"* ]]; then
  send_to_slack "$URGENT_CHANNEL_WEBHOOK"
elif [[ "$LABELS" == *"bug"* ]]; then
  send_to_slack "$BUG_CHANNEL_WEBHOOK"
fi
```

### Aggregate and Batch

```python
# Instead of sending each event immediately, aggregate and send in batches
# Useful for high-frequency events

def batch_send(items, batch_size=10):
    for i in range(0, len(items), batch_size):
        batch = items[i:i+batch_size]
        send_batch_to_integration(batch)
        time.sleep(1)  # Rate limiting
```

## Related Examples

- **Production examples**:
  - `.github/workflows/notion-issue-summary.md` - Notion integration

## Tips

- **Test webhooks**: Use tools like webhook.site to test payloads
- **Handle secrets carefully**: Never log or expose secrets
- **Implement retries**: Networks are unreliable, always retry
- **Rate limit**: Respect API rate limits
- **Log responses**: Keep logs for debugging
- **Monitor failures**: Alert on repeated failures
- **Document setup**: External integrations need setup docs

## Security Considerations

- Store API keys and tokens in repository secrets
- Use HTTPS endpoints only
- Validate webhook responses
- Don't expose sensitive data in external tools
- Implement request signing where supported
- Monitor for unauthorized access

---

**Pattern Info**:
- Complexity: Intermediate
- Trigger: GitHub events (issues, PRs, discussions)
- Safe Outputs: None (integrates with external APIs)
- Tools: GitHub (issues, pull_requests, discussions), bash (curl)
- Network: Requires external domain access
