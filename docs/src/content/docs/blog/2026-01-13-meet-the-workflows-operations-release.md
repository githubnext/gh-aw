---
title: "Meet the Workflows in Peli's Agent Factory: Operations & Release"
description: "A curated tour of operations and release workflows that ship software"
authors:
  - dsyme
  - peli
date: 2026-01-13
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/
  label: "Security & Compliance Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-creative-culture/
  label: "Creative & Culture Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Welcome to another stop in Peli's Agent Factory!

We've covered a lot of ground: workflows that triage incoming activity, maintain code quality, track metrics and performance, and enforce [security boundaries](/gh-aw/blog/2026-01-13-meet-the-workflows-security-compliance/). These foundational layers help us manage vulnerabilities, validate network access, and prevent credential exposure. We've built the infrastructure to run AI agents safely.

Now comes the moment of truth: actually shipping software to users. All the quality checks, security scans, and metrics tracking culminate in one critical process - the release. Operations and release workflows handle the orchestration of building, testing, generating release notes, and publishing. These workflows can't afford to be experimental; they need to be rock-solid reliable, well-tested, and yes, even a bit boring. Let's explore how automation makes shipping predictable and stress-free.

## 🚀 Operations & Release Workflows

The agents that help us actually ship software:

- **[Release](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/release.md)** - Orchestrates builds, tests, and release note generation
- **[Daily Workflow Updater](https://github.com/githubnext/gh-aw/tree/main/.github/workflows/daily-workflow-updater.md)** - Keeps actions and dependencies current (because dependency updates never stop)

Shipping software is stressful enough without worrying about whether you formatted your release notes correctly. The Release workflow handles the entire orchestration - building, testing, generating coherent release notes from commits, and publishing. What's interesting here is the **reliability** requirement: these workflows can't afford to be creative or experimental. They need to be deterministic, well-tested, and boring (in a good way).

The Daily Workflow Updater taught us that maintenance is a perfect use case for agents - it's repetitive, necessary, and nobody enjoys doing it manually. These workflows handle the toil so we can focus on the interesting problems.

## Time for a Palette Cleanser

After all this serious infrastructure talk, we discovered something delightful: agents don't have to be all business.

Continue reading: [Creative & Culture Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-creative-culture/)

---

*This is part 5 of a 16-part series exploring the workflows in Peli's Agent Factory.*
