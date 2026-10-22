---
title: Human Review and Compliance Patterns
description: Patterns for human attribution, review, and evidence when agentic workflows propose code changes.
---

Organizations with change-management controls, including controls used in SOC 2
programs, may require a human creator or sponsor and an independent human
reviewer for code changes. The applicable requirement comes from the
organization's policy; using gh-aw or a particular authentication method does
not by itself establish compliance.

## Research, Issue, and Human Assignment

An agentic workflow can research a problem and publish its findings and proposed
implementation through [`create-issue`](/gh-aw/reference/safe-outputs/).
A human reviews the proposal and assigns the issue to a coding agent. The coding
agent creates a pull request, which a separate human reviews under the
repository's change-management rules.

This pattern records the human decision to initiate implementation separately
from the automated research. Keep the issue, assignment, pull request, and
review linked so reviewers can trace the proposal to the resulting change.
When human assignment is a required control, the research workflow should not
automatically assign the issue or create the implementation pull request.

Research and implementation can use different environments: for example, the
research workflow can run on Ubuntu while the coding agent uses a configured
Windows environment. The issue carries the findings and acceptance criteria
between them. See [ResearchPlanAssignOps](/gh-aw/patterns/research-plan-assign-ops/)
for the broader research and implementation pattern.

## Direct Pull Request Creation

A workflow using [`create-pull-request`](/gh-aw/reference/safe-outputs-pull-requests/)
can publish through the default GitHub Actions token, a GitHub App, or a
configured personal access token. These choices affect the publishing identity.
An automation identity does not identify a human sponsor by itself.

Where organizational policy permits it, a personal access token can associate
publication with a designated human account. That attribution is not evidence
that the person reviewed or approved the generated code. Record the sponsoring
decision and require independent review through the repository's rulesets or
branch protection. Follow the organization's credential ownership, scope, and
rotation requirements when choosing this approach.

## Review Evidence

Useful evidence includes the triggering event and workflow revision, the
research issue or change request, the human assignment or sponsoring decision,
the proposed diff, required checks, review approvals, and the merge record.
Retain workflow artifacts according to the organization's evidence policy;
artifact expiration can otherwise remove supporting execution details.

The distinction between publishing identity, human sponsorship, and code review
should be explicit in the control description. See
[Authentication](/gh-aw/reference/auth/) for supported credentials and
[Auditing Workflows](/gh-aw/reference/audit/) for run evidence.
