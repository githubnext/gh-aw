---
title: "Meet the Workflows in Peli's Agent Factory: Security & Compliance"
description: "A curated tour of security and compliance workflows that enforce safe boundaries"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/
  label: "Metrics & Analytics Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-operations-release/
  label: "Operations & Release Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Great to have you back at [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

In our previous posts, we've seen how workflows handle triage, maintain quality, and track [metrics and analytics](/gh-aw/blog/2026-01-13-meet-the-workflows-metrics-analytics/). These workflows help us understand agent performance, identify cost optimization opportunities, and monitor the entire ecosystem. We learned that you can't optimize what you don't measure.

But measurement alone isn't enough - we also need *boundaries*. When AI agents can access APIs, modify code, and interact with external services, security becomes paramount. How do we ensure agents only access authorized resources? How do we track vulnerabilities and enforce compliance deadlines? How do we prevent credential exposure? That's where security and compliance workflows become our essential guardrails - the watchful guardians that let us sleep soundly at night.

## 🔒 Security & Compliance Workflows

These agents are our security guards, keeping watch and enforcing the rules:

- **[Security Compliance](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/security-compliance.md?plain=1)** - Runs vulnerability campaigns with deadline tracking  
  [→ View security reports](https://github.com/search?q=repo%3Agithubnext%2Fgh-aw+label%3Asecurity+is%3Aissue&type=issues)
- **[Firewall](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/firewall.md?plain=1)** - Tests network security and validates rules  
  [→ View firewall reports](https://github.com/search?q=repo%3Agithubnext%2Fgh-aw+firewall+is%3Aissue&type=issues)
- **[Daily Secrets Analysis](https://github.com/githubnext/gh-aw/tree/532a0412680638e5e93b6e8c5ea9b8074fe6be22/.github/workflows/daily-secrets-analysis.md?plain=1)** - Scans for exposed credentials (yes, it happens)  
  [→ View secrets analysis](https://github.com/search?q=repo%3Agithubnext%2Fgh-aw+in%3Atitle+%22%5Bdaily+secrets%5D%22+is%3Aissue&type=issues)

Security workflows were where we got serious about trust boundaries. The Security Compliance agent manages entire vulnerability remediation campaigns with deadline tracking - perfect for those "audit in 3 weeks" panic moments. We learned that AI agents need guardrails just like humans need seat belts. 

The Firewall workflow validates that our agents can't access unauthorized resources, because an AI agent with unrestricted network access is... let's just say we sleep better with these safeguards. These workflows prove that automation and security aren't at odds - when done right, automated security is more consistent than manual reviews.

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

## Next Up: From Safety to Shipping

With guardrails in place, we could focus on the ultimate goal: getting software into users' hands reliably and repeatedly.

Continue reading: [Operations & Release Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-operations-release/)

---

*This is part 4 of a 16-part series exploring the workflows in Peli's Agent Factory.*
