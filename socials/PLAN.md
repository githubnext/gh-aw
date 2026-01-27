# Social Media Campaign Plan

## Overview

This directory contains scripts and content for promoting the gh-aw blog series across multiple social media platforms. The campaign will roll out blog posts sequentially, one per day, with engagement tracking and analytics.

## Platforms

- **X (Twitter)** - GitHub Next account
- **Bluesky** - GitHub Next account  
- **Mastodon** - GitHub Next account
- **LinkedIn** - Personal accounts (team members)
- **Future**: Open to additional platforms

## Directory Structure

```
/socials/
├── PLAN.md              # This file - campaign planning and documentation
├── scripts.sh           # Main automation script for posting and tracking
├── config.env           # Configuration and API credentials (gitignored)
├── content/             # Social media content for each blog post
│   ├── 01-welcome.md
│   ├── 02-meet-workflows.md
│   ├── 03-continuous-simplicity.md
│   └── ...
├── published/           # Metadata about published posts
│   ├── 2026-01-12.json  # Post metadata with IDs and URLs
│   └── ...
└── analytics/           # Engagement data and reports
    ├── daily/           # Daily snapshots
    └── summary.json     # Aggregate analytics
```

## Content Strategy

### Blog Post Schedule

The social campaign schedule intentionally does **not** align with blog publication dates.

- **Start date:** 2026-01-21
- **Cadence:** daily (one blog entry per day)
- **Strategy:** start with the intro + “Meet the Workflows”, then roll out the workflow posts one-by-one day-by-day (this shifts later posts).

Planned schedule (date -> content file):

- 2026-01-21 -> `01-welcome.md`
- 2026-01-22 -> `02-meet-workflows.md`

Meet the Workflows series (one per day):

- 2026-01-23 -> `03-meet-workflows-continuous-simplicity.md`
- 2026-01-24 -> `04-meet-workflows-continuous-refactoring.md`
- 2026-01-25 -> `05-meet-workflows-continuous-style.md`
- 2026-01-26 -> `06-meet-workflows-continuous-improvement.md`
- 2026-01-27 -> `07-meet-workflows-testing-validation.md`
- 2026-01-28 -> `08-meet-workflows-security-compliance.md`
- 2026-01-29 -> `09-meet-workflows-quality-hygiene.md`
- 2026-01-30 -> `10-meet-workflows-issue-management.md`
- 2026-01-31 -> `11-meet-workflows-operations-release.md`
- 2026-02-01 -> `12-meet-workflows-tool-infrastructure.md`
- 2026-02-02 -> `13-meet-workflows-organization.md`
- 2026-02-03 -> `14-meet-workflows-multi-phase.md`
- 2026-02-04 -> `15-meet-workflows-interactive-chatops.md`
- 2026-02-05 -> `16-meet-workflows-documentation.md`
- 2026-02-06 -> `17-meet-workflows-campaigns.md`
- 2026-02-07 -> `18-meet-workflows-advanced-analytics.md`
- 2026-02-08 -> `19-meet-workflows-metrics-analytics.md`
- 2026-02-09 -> `20-meet-workflows-creative-culture.md`

Remaining posts (shifted later due to daily Meet the Workflows roll-out):

- 2026-02-10 -> `21-twelve-lessons.md`
- 2026-02-11 -> `22-design-patterns.md`
- 2026-02-12 -> `23-operational-patterns.md`
- 2026-02-13 -> `24-imports-sharing.md`
- 2026-02-14 -> `25-security-lessons.md`
- 2026-02-15 -> `26-how-workflows-work.md`
- 2026-02-16 -> `27-authoring-workflows.md`

### Content Format

Each content file contains:
- **Platform-specific variants** (character limits, formatting)
- **Hashtags** (#AI #Automation #GitHub #DevOps)
- **Visual suggestions** (images, screenshots)
- **Engagement hooks** (questions, CTAs)
- **Timing recommendations** (best time to post)

## Script Functionality

### Core Features (`scripts.sh`)

1. **Post Publication**
   - Read next scheduled content file
   - Post to all configured platforms
   - Save post IDs and URLs to `published/`
   - Handle rate limits and retries

2. **Engagement Tracking**
   - Fetch metrics for previous posts (likes, shares, replies)
   - Store daily snapshots in `analytics/daily/`
   - Update aggregate metrics in `analytics/summary.json`

3. **Status Reporting**
   - Generate daily summary of campaign progress
   - Identify high-performing posts
   - Flag issues (failed posts, low engagement)

4. **Scheduling Logic**
   - Determine which content to post based on date
   - Skip weekends (optional)
   - Handle manual overrides

### API Requirements

The script requires API credentials for:
- **X API** (OAuth 2.0 or API keys)
- **Bluesky** (App password)
- **Mastodon** (Access token)
- **LinkedIn** (OAuth 2.0)

Store credentials in `config.env` (add to `.gitignore`).

### Error Handling

- Retry failed posts with exponential backoff
- Log errors to `errors.log`
- Send notifications for critical failures
- Continue on partial failures (some platforms succeed)

## Agentic Workflow Integration

### Daily Workflow Trigger

The campaign will be driven by a daily agentic workflow (to be created):

```yaml
name: Daily Social Media Campaign
on:
  schedule:
    - cron: '0 14 0 * *'  # 2 PM UTC daily
  workflow_dispatch:       # Manual trigger
```

### Workflow Responsibilities

1. Run `socials/scripts.sh post` to publish scheduled content
2. Run `socials/scripts.sh track` to collect engagement data
3. Run `socials/scripts.sh report` to generate daily summary
4. Create issue if errors detected
5. Update campaign dashboard

## Metrics and Analytics

### Key Metrics

For each post, track:
- **Impressions/Views**: How many people saw the post
- **Engagement Rate**: Likes + shares + replies / impressions
- **Click-through Rate**: Clicks on blog link / impressions
- **Best Performing Platform**: Which platform drove most traffic
- **Peak Engagement Time**: When most interactions occurred

### Reporting

- **Daily**: Snapshot of yesterday's metrics
- **Weekly**: Comparison of posts, trend analysis
- **Campaign End**: Full retrospective with insights

## Content Creation Guidelines

### Social Media Best Practices

1. **Keep it concise**: Lead with the hook, not the context
2. **Use visuals**: Include screenshots, diagrams, or graphics
3. **Add hashtags**: 3-5 relevant tags per post
4. **Include CTA**: "Read more →", "What's your experience?", etc.
5. **Time it right**: Post during peak engagement hours (10 AM - 2 PM ET)

### Platform-Specific Adaptations

- **X**: 280 chars, thread if needed, use images
- **Bluesky**: 300 chars, conversational tone, rich embeds
- **Mastodon**: 500 chars, technical detail OK, use CW if long
- **LinkedIn**: Longer form (1300 chars), professional tone, native articles

### Content Template

Each `content/*.md` file should include:

```markdown
# Post Title

Blog URL: [url]
Publish Date: [YYYY-MM-DD]
Primary Theme: [theme]

## X (Twitter)
[280 char post text]

Thread (optional):
1/ [first tweet]
2/ [second tweet]

## Bluesky
[300 char post text]

## Mastodon
[500 char post text]

## LinkedIn
[1300 char post text - professional angle]

## Common Elements
- Hashtags: #AI #GitHub #Automation #DevOps
- Visual: [description or path]
- CTA: [call to action]
- Best Time: 10 AM ET

## Engagement Strategy
- Reply to: [expected questions]
- Monitor: [keywords, mentions]
- Amplify: [repost if X likes in Y hours]
```

## Campaign Timeline

### Pre-Launch (Week 0)
- [ ] Create all content files
- [ ] Test scripts on test accounts
- [ ] Set up API credentials
- [ ] Configure monitoring

### Launch (Week 1-2)
- [ ] Daily posts for main blog series
- [ ] Monitor engagement closely
- [ ] Adjust timing based on metrics
- [ ] Respond to comments

### Mid-Campaign (Week 3-4)  
- [ ] Continue daily posts
- [ ] Share engagement highlights
- [ ] Create follow-up content based on questions
- [ ] Cross-promote high-performing posts

### Post-Campaign (Week 5+)
- [ ] Final analytics report
- [ ] Document lessons learned
- [ ] Plan next campaign iteration
- [ ] Archive successful content

## Success Criteria

- **Reach**: 10K+ impressions per post average
- **Engagement**: 2%+ engagement rate
- **Traffic**: 500+ blog visitors from social
- **Community**: 50+ meaningful conversations
- **Growth**: 200+ new followers across platforms

## Next Steps

1. Create `config.env` with API credentials
2. Implement `scripts.sh` core functionality
3. Write content files for all blog posts (start with the first two)
4. Test on staging/test accounts
5. Create agentic workflow for daily automation
6. Launch campaign!

## Notes

- Content files should be reviewed by team before posting
- Consider A/B testing different post formats
- Engage authentically - don't just broadcast
- Celebrate community contributions and discussions
- Be prepared to adjust strategy based on what works
