#!/bin/bash
set -euo pipefail

# Social Media Campaign Automation Script
# Handles posting content and tracking engagement across multiple platforms

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTENT_DIR="$SCRIPT_DIR/content"
PUBLISHED_DIR="$SCRIPT_DIR/published"
ANALYTICS_DIR="$SCRIPT_DIR/analytics"
DAILY_ANALYTICS_DIR="$ANALYTICS_DIR/daily"
CONFIG_FILE="$SCRIPT_DIR/config.env"
LOG_FILE="$SCRIPT_DIR/campaign.log"
ERROR_LOG="$SCRIPT_DIR/errors.log"

# Global flags
DRY_RUN=${DRY_RUN:-0}

# Content limits
X_MAX_CHARS=${X_MAX_CHARS:-1000}

string_length() {
    local text="$1"

    if command -v python3 >/dev/null 2>&1; then
        printf '%s' "$text" | python3 -c 'import sys; print(len(sys.stdin.read()))'
        return 0
    fi

    # Fallback: byte-length in bash (may overcount emojis)
    echo "${#text}"
}

enforce_max_length() {
    local platform="$1"
    local max_chars="$2"
    local content="$3"

    local length
    length=$(string_length "$content")
    if [[ "$length" -gt "$max_chars" ]]; then
        error "$platform post exceeds ${max_chars} characters (got $length)"
        return 1
    fi
}

# Ensure required directories exist
mkdir -p "$PUBLISHED_DIR" "$ANALYTICS_DIR" "$DAILY_ANALYTICS_DIR"

# Load configuration (API keys, tokens)
if [[ -f "$CONFIG_FILE" ]]; then
    # shellcheck source=/dev/null
    source "$CONFIG_FILE"
else
    echo "Warning: config.env not found. API credentials will not be available."
fi

# Logging functions
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE" >&2
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "$ERROR_LOG" >&2
}

# Get today's date in YYYY-MM-DD format
TODAY=$(date +%Y-%m-%d)

# Campaign schedule (date -> content file)
# Start date for the campaign is 2026-01-21 (01-welcome) and then daily.
declare -A SCHEDULE=(
    ["2026-01-21"]="01-welcome.md"
    ["2026-01-22"]="02-meet-workflows.md"

    # Meet the Workflows series (one per day)
    ["2026-01-23"]="03-meet-workflows-continuous-simplicity.md"
    ["2026-01-24"]="04-meet-workflows-continuous-refactoring.md"
    ["2026-01-25"]="05-meet-workflows-continuous-style.md"
    ["2026-01-26"]="06-meet-workflows-continuous-improvement.md"
    ["2026-01-27"]="07-meet-workflows-testing-validation.md"
    ["2026-01-28"]="08-meet-workflows-security-compliance.md"
    ["2026-01-29"]="09-meet-workflows-quality-hygiene.md"
    ["2026-01-30"]="10-meet-workflows-issue-management.md"
    ["2026-01-31"]="11-meet-workflows-operations-release.md"
    ["2026-02-01"]="12-meet-workflows-tool-infrastructure.md"
    ["2026-02-02"]="13-meet-workflows-organization.md"
    ["2026-02-03"]="14-meet-workflows-multi-phase.md"
    ["2026-02-04"]="15-meet-workflows-interactive-chatops.md"
    ["2026-02-05"]="16-meet-workflows-documentation.md"
    ["2026-02-06"]="17-meet-workflows-campaigns.md"
    ["2026-02-07"]="18-meet-workflows-advanced-analytics.md"
    ["2026-02-08"]="19-meet-workflows-metrics-analytics.md"
    ["2026-02-09"]="20-meet-workflows-creative-culture.md"

    # Remaining blog series (shifted later due to daily Meet the Workflows roll-out)
    ["2026-02-10"]="21-twelve-lessons.md"
    ["2026-02-11"]="22-design-patterns.md"
    ["2026-02-12"]="23-operational-patterns.md"
    ["2026-02-13"]="24-imports-sharing.md"
    ["2026-02-14"]="25-security-lessons.md"
    ["2026-02-15"]="26-how-workflows-work.md"
    ["2026-02-16"]="27-authoring-workflows.md"
)

# Map content files to schedule
get_content_for_date() {
    local target_date="$1"
    local content_file="${SCHEDULE[$target_date]:-}"

    if [[ -n "$content_file" ]]; then
        echo "$CONTENT_DIR/$content_file"
    fi
}

# Parse content file to extract platform-specific posts
parse_content_file() {
    local content_file="$1"
    local platform="$2"
    
    if [[ ! -f "$content_file" ]]; then
        error "Content file not found: $content_file"
        return 1
    fi
    
    # Extract section for specific platform using awk
    # Use exact string comparison (not regex) so headings like "X (Twitter)" work.
    awk -v platform="## $platform" '
        $0 == platform { found=1; next }
        found && /^## / { exit }
        found { print }
    ' "$content_file" | sed '/^[[:space:]]*$/d'
}

# Post to X (Twitter)
post_to_x() {
    local content="$1"
    local date="$2"

    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        log "[DRY RUN] X post:"
        printf '%s\n' "$content" | tee -a "$LOG_FILE" >&2
        echo "dryrun-x-$date"
        return 0
    fi
    
    if [[ -z "${X_API_KEY:-}" ]]; then
        error "X_API_KEY not configured"
        return 1
    fi
    
    log "Posting to X: ${content:0:50}..."
    
    # Using Twitter API v2
    local response
    local payload
    payload=$(jq -n --arg text "$content" '{text: $text}')

    response=$(curl -s -X POST "https://api.twitter.com/2/tweets" \
        -H "Authorization: Bearer $X_API_KEY" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    local post_id
    post_id=$(echo "$response" | jq -r '.data.id // empty')
    
    if [[ -n "$post_id" ]]; then
        log "Posted to X successfully: $post_id"
        echo "$post_id"
        return 0
    else
        error "Failed to post to X: $response"
        return 1
    fi
}

# Post to Bluesky
post_to_bluesky() {
    local content="$1"
    local date="$2"

    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        log "[DRY RUN] Bluesky post:"
        printf '%s\n' "$content" | tee -a "$LOG_FILE" >&2
        echo "at://dryrun/bluesky/$date"
        return 0
    fi
    
    if [[ -z "${BLUESKY_HANDLE:-}" ]] || [[ -z "${BLUESKY_APP_PASSWORD:-}" ]]; then
        error "Bluesky credentials not configured"
        return 1
    fi
    
    log "Posting to Bluesky: ${content:0:50}..."
    
    # Create session
    local session
    local session_payload
    session_payload=$(jq -n --arg identifier "$BLUESKY_HANDLE" --arg password "$BLUESKY_APP_PASSWORD" '{identifier: $identifier, password: $password}')

    session=$(curl -s -X POST "https://bsky.social/xrpc/com.atproto.server.createSession" \
        -H "Content-Type: application/json" \
        -d "$session_payload")
    
    local access_token
    access_token=$(echo "$session" | jq -r '.accessJwt // empty')
    
    if [[ -z "$access_token" ]]; then
        error "Failed to authenticate with Bluesky"
        return 1
    fi
    
    # Create post
    local response
    local record_payload
    record_payload=$(jq -n \
        --arg repo "$BLUESKY_HANDLE" \
        --arg collection "app.bsky.feed.post" \
        --arg text "$content" \
        --arg createdAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '{repo: $repo, collection: $collection, record: {text: $text, createdAt: $createdAt}}')

    response=$(curl -s -X POST "https://bsky.social/xrpc/com.atproto.repo.createRecord" \
        -H "Authorization: Bearer $access_token" \
        -H "Content-Type: application/json" \
        -d "$record_payload")
    
    local post_uri
    post_uri=$(echo "$response" | jq -r '.uri // empty')
    
    if [[ -n "$post_uri" ]]; then
        log "Posted to Bluesky successfully: $post_uri"
        echo "$post_uri"
        return 0
    else
        error "Failed to post to Bluesky: $response"
        return 1
    fi
}

# Post to Mastodon
post_to_mastodon() {
    local content="$1"
    local date="$2"

    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        log "[DRY RUN] Mastodon post:"
        printf '%s\n' "$content" | tee -a "$LOG_FILE" >&2
        echo "dryrun-mastodon-$date"
        return 0
    fi
    
    if [[ -z "${MASTODON_INSTANCE:-}" ]] || [[ -z "${MASTODON_TOKEN:-}" ]]; then
        error "Mastodon credentials not configured"
        return 1
    fi
    
    log "Posting to Mastodon: ${content:0:50}..."
    
    local response
    response=$(curl -s -X POST "https://$MASTODON_INSTANCE/api/v1/statuses" \
        -H "Authorization: Bearer $MASTODON_TOKEN" \
        -F "status=$content")
    
    local post_id
    post_id=$(echo "$response" | jq -r '.id // empty')
    
    if [[ -n "$post_id" ]]; then
        local post_url="https://$MASTODON_INSTANCE/@${MASTODON_HANDLE}/$post_id"
        log "Posted to Mastodon successfully: $post_url"
        echo "$post_id"
        return 0
    else
        error "Failed to post to Mastodon: $response"
        return 1
    fi
}

# Post to LinkedIn
post_to_linkedin() {
    local content="$1"
    local date="$2"

    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        log "[DRY RUN] LinkedIn post:"
        printf '%s\n' "$content" | tee -a "$LOG_FILE" >&2
        echo "dryrun-linkedin-$date"
        return 0
    fi
    
    if [[ -z "${LINKEDIN_ACCESS_TOKEN:-}" ]] || [[ -z "${LINKEDIN_PERSON_URN:-}" ]]; then
        error "LinkedIn credentials not configured"
        return 1
    fi
    
    log "Posting to LinkedIn: ${content:0:50}..."
    
    local response
    local payload
    payload=$(jq -n --arg author "$LINKEDIN_PERSON_URN" --arg text "$content" '
        {
          author: $author,
          lifecycleState: "PUBLISHED",
          specificContent: {
            "com.linkedin.ugc.ShareContent": {
              shareCommentary: {text: $text},
              shareMediaCategory: "NONE"
            }
          },
          visibility: {"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"}
        }')

    response=$(curl -s -X POST "https://api.linkedin.com/v2/ugcPosts" \
        -H "Authorization: Bearer $LINKEDIN_ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -H "X-Restli-Protocol-Version: 2.0.0" \
        -d "$payload")
    
    local post_id
    post_id=$(echo "$response" | jq -r '.id // empty')
    
    if [[ -n "$post_id" ]]; then
        log "Posted to LinkedIn successfully: $post_id"
        echo "$post_id"
        return 0
    else
        error "Failed to post to LinkedIn: $response"
        return 1
    fi
}

# Main posting function
post_content() {
    local target_date="${1:-$TODAY}"
    local content_file
    content_file=$(get_content_for_date "$target_date")
    
    if [[ -z "$content_file" ]]; then
        log "No content scheduled for $target_date"
        return 0
    fi
    
    if [[ ! -f "$content_file" ]]; then
        log "Content scheduled for $target_date but not yet created: $content_file"
        return 0
    fi
    
    log "Starting campaign post for $target_date"
    log "Content file: $content_file"
    
    local metadata_file="$PUBLISHED_DIR/$target_date.json"
    local success_count=0
    local total_count=4
    local dry_run_json=false
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        dry_run_json=true
    fi
    
    # Initialize metadata
    cat > "$metadata_file" << EOF
{
    "date": "$target_date",
    "content_file": "$content_file",
    "published_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "dry_run": $dry_run_json,
    "posts": {}
}
EOF
    
    # Post to X
    local x_content
    x_content=$(parse_content_file "$content_file" "X (Twitter)")
    if [[ -n "$x_content" ]]; then
        if ! enforce_max_length "X" "$X_MAX_CHARS" "$x_content"; then
            true
        elif x_id=$(post_to_x "$x_content" "$target_date"); then
            jq --arg id "$x_id" --arg url "https://twitter.com/i/web/status/$x_id" --arg content "$x_content" \
                '.posts.x = {id: $id, url: $url, content: $content}' \
                "$metadata_file" > "$metadata_file.tmp" && mv "$metadata_file.tmp" "$metadata_file"
            ((success_count++))
        fi
    fi
    
    # Post to Bluesky  
    local bluesky_content
    bluesky_content=$(parse_content_file "$content_file" "Bluesky")
    if [[ -n "$bluesky_content" ]]; then
        if bluesky_uri=$(post_to_bluesky "$bluesky_content" "$target_date"); then
            jq --arg uri "$bluesky_uri" --arg content "$bluesky_content" '.posts.bluesky = {uri: $uri, content: $content}' \
                "$metadata_file" > "$metadata_file.tmp" && mv "$metadata_file.tmp" "$metadata_file"
            ((success_count++))
        fi
    fi
    
    # Post to Mastodon
    local mastodon_content
    mastodon_content=$(parse_content_file "$content_file" "Mastodon")
    if [[ -n "$mastodon_content" ]]; then
        if mastodon_id=$(post_to_mastodon "$mastodon_content" "$target_date"); then
            local mastodon_instance="${MASTODON_INSTANCE:-}"
            local mastodon_handle="${MASTODON_HANDLE:-}"
            local mastodon_url=""
            if [[ -n "$mastodon_instance" ]] && [[ -n "$mastodon_handle" ]]; then
                mastodon_url="https://$mastodon_instance/@${mastodon_handle}/$mastodon_id"
            fi

            jq --arg id "$mastodon_id" --arg url "$mastodon_url" --arg content "$mastodon_content" '.posts.mastodon = {id: $id, url: $url, content: $content}' \
                "$metadata_file" > "$metadata_file.tmp" && mv "$metadata_file.tmp" "$metadata_file"
            ((success_count++))
        fi
    fi
    
    # Post to LinkedIn
    local linkedin_content
    linkedin_content=$(parse_content_file "$content_file" "LinkedIn")
    if [[ -n "$linkedin_content" ]]; then
        if linkedin_id=$(post_to_linkedin "$linkedin_content" "$target_date"); then
            jq --arg id "$linkedin_id" --arg content "$linkedin_content" '.posts.linkedin = {id: $id, content: $content}' \
                "$metadata_file" > "$metadata_file.tmp" && mv "$metadata_file.tmp" "$metadata_file"
            ((success_count++))
        fi
    fi
    
    log "Posted to $success_count/$total_count platforms successfully"
    
    if [[ $success_count -eq 0 ]]; then
        error "Failed to post to any platform"
        return 1
    fi
    
    return 0
}

# Daily run: publish today's scheduled content, then track engagement
run_daily() {
    local target_date="${1:-$TODAY}"
    post_content "$target_date" || true
    track_recent_posts
}

run_campaign() {
    local start_date="${1:-}"
    local end_date="${2:-}"

    log "Running campaign"
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        log "[DRY RUN] No API calls will be made"
    fi

    local dates
    dates=$(printf '%s\n' "${!SCHEDULE[@]}" | sort)
    while IFS= read -r date; do
        if [[ -n "$start_date" ]] && [[ "$date" < "$start_date" ]]; then
            continue
        fi
        if [[ -n "$end_date" ]] && [[ "$date" > "$end_date" ]]; then
            continue
        fi
        post_content "$date" || true
    done <<< "$dates"

    track_recent_posts
}

# Track engagement for a specific post
track_post_engagement() {
    local date="$1"
    local metadata_file="$PUBLISHED_DIR/$date.json"
    
    if [[ ! -f "$metadata_file" ]]; then
        error "No published post found for $date"
        return 1
    fi
    
    log "Tracking engagement for $date"
    
    local analytics_file="$DAILY_ANALYTICS_DIR/$date-$(date +%Y%m%d).json"
    local engagement_data='{"date": "'$date'", "snapshot_time": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'", "platforms": {}}'
    
    # Track X engagement
    local x_id
    x_id=$(jq -r '.posts.x.id // empty' "$metadata_file")
    if [[ -n "$x_id" ]] && [[ -n "${X_API_KEY:-}" ]]; then
        local x_metrics
        x_metrics=$(curl -s "https://api.twitter.com/2/tweets/$x_id?tweet.fields=public_metrics" \
            -H "Authorization: Bearer $X_API_KEY")
        
        engagement_data=$(echo "$engagement_data" | jq --argjson metrics "$x_metrics" \
            '.platforms.x = $metrics.data.public_metrics')
    fi
    
    # Track Mastodon engagement
    local mastodon_id
    mastodon_id=$(jq -r '.posts.mastodon.id // empty' "$metadata_file")
    if [[ -n "$mastodon_id" ]] && [[ -n "${MASTODON_TOKEN:-}" ]]; then
        local mastodon_metrics
        mastodon_metrics=$(curl -s "https://$MASTODON_INSTANCE/api/v1/statuses/$mastodon_id" \
            -H "Authorization: Bearer $MASTODON_TOKEN")
        
        engagement_data=$(echo "$engagement_data" | jq --argjson metrics "$mastodon_metrics" \
            '.platforms.mastodon = {
                "reblogs": $metrics.reblogs_count,
                "favourites": $metrics.favourites_count,
                "replies": $metrics.replies_count
            }')
    fi
    
    echo "$engagement_data" > "$analytics_file"
    log "Engagement data saved to $analytics_file"
    
    # Update summary
    update_analytics_summary "$date" "$engagement_data"
}

# Update aggregate analytics summary
update_analytics_summary() {
    local date="$1"
    local engagement_data="$2"
    local summary_file="$ANALYTICS_DIR/summary.json"
    
    if [[ ! -f "$summary_file" ]]; then
        echo '{"posts": [], "totals": {}}' > "$summary_file"
    fi
    
    # Add or update post in summary
    jq --arg date "$date" --argjson data "$engagement_data" \
        '.posts = (.posts | map(select(.date != $date))) + [$data]' \
        "$summary_file" > "$summary_file.tmp" && mv "$summary_file.tmp" "$summary_file"
    
    log "Updated analytics summary"
}

# Track all recent posts
track_recent_posts() {
    log "Tracking engagement for recent posts"
    
    # Track posts from last 7 days
    for i in {0..6}; do
        local check_date
        check_date=$(date -d "$TODAY - $i days" +%Y-%m-%d 2>/dev/null || date -v-"$i"d +%Y-%m-%d)
        
        if [[ -f "$PUBLISHED_DIR/$check_date.json" ]]; then
            track_post_engagement "$check_date" || true
        fi
    done
}

# Generate daily report
generate_report() {
    log "Generating campaign report"
    
    local summary_file="$ANALYTICS_DIR/summary.json"
    if [[ ! -f "$summary_file" ]]; then
        log "No analytics data available yet"
        return 0
    fi
    
    echo "==================================="
    echo "Social Media Campaign Report"
    echo "Generated: $(date)"
    echo "==================================="
    echo ""
    
    # Show recent posts
    echo "Recent Posts:"
    jq -r '.posts | sort_by(.date) | reverse | .[:5] | .[] | 
        "  \(.date): " + 
        (if .platforms.x then "X: \(.platforms.x.like_count // 0) likes, \(.platforms.x.retweet_count // 0) retweets" else "" end)' \
        "$summary_file"
    
    echo ""
    echo "Full analytics available in: $ANALYTICS_DIR"
}

# Main command dispatcher
main() {
    # Parse global flags
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                DRY_RUN=1
                shift
                ;;
            --)
                shift
                break
                ;;
            *)
                break
                ;;
        esac
    done

    local command="${1:-help}"
    shift || true
    
    case "$command" in
        daily)
            run_daily "${1:-$TODAY}"
            ;;
        campaign)
            run_campaign "${1:-}" "${2:-}"
            ;;
        post)
            post_content "${1:-$TODAY}"
            ;;
        track)
            track_recent_posts
            ;;
        report)
            generate_report
            ;;
        status)
            log "Campaign status check"
            local published_count
            published_count=$(find "$PUBLISHED_DIR" -name "*.json" | wc -l)
            echo "Published posts: $published_count"
            echo "Latest: $(ls -t "$PUBLISHED_DIR"/*.json 2>/dev/null | head -n1 | xargs basename .json || echo 'none')"
            ;;
        help|*)
            cat << 'EOF'
Social Media Campaign Script

Usage:
    ./scripts.sh [--dry-run] daily [date]          - Post scheduled content and track engagement
    ./scripts.sh [--dry-run] campaign [start] [end] - Run the whole campaign schedule (optionally bounded)
    ./scripts.sh [--dry-run] post [date]           - Post content for today or specified date (YYYY-MM-DD)
    ./scripts.sh track           - Track engagement for recent posts (last 7 days)
    ./scripts.sh report          - Generate campaign analytics report
    ./scripts.sh status          - Show campaign status

Environment Variables (set in config.env):
    X_API_KEY                    - Twitter/X API bearer token
    BLUESKY_HANDLE               - Bluesky handle (user.bsky.social)
    BLUESKY_APP_PASSWORD         - Bluesky app password
    MASTODON_INSTANCE            - Mastodon instance (e.g., mastodon.social)
    MASTODON_TOKEN               - Mastodon access token
    MASTODON_HANDLE              - Mastodon username
    LINKEDIN_ACCESS_TOKEN        - LinkedIn OAuth token
    LINKEDIN_PERSON_URN          - LinkedIn person URN

Examples:
    ./scripts.sh --dry-run campaign              # Preview the full schedule (missing content tolerated)
    ./scripts.sh --dry-run daily 2026-01-21      # Preview day 1
    ./scripts.sh daily                   # Post today + track engagement
    ./scripts.sh post                    # Post today's scheduled content
    ./scripts.sh post 2026-01-21         # Post specific date's content
    ./scripts.sh track                   # Update engagement metrics
    ./scripts.sh report                  # View campaign analytics

Logs:
    campaign.log                 - General activity log
    errors.log                   - Error log
EOF
            ;;
    esac
}

# Run main function
main "$@"
