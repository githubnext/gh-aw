// @ts-check

function normalizeLabelNames(labels) {
  if (!Array.isArray(labels)) {
    return [];
  }
  return labels
    .map(label => {
      if (typeof label === "string") {
        return label;
      }
      if (label && typeof label.name === "string") {
        return label.name;
      }
      return "";
    })
    .filter(Boolean);
}

function normalizeAssigneeLogins(assignees) {
  if (!Array.isArray(assignees)) {
    return [];
  }
  return assignees
    .map(assignee => {
      if (typeof assignee === "string") {
        return assignee;
      }
      if (assignee && typeof assignee.login === "string") {
        return assignee.login;
      }
      return "";
    })
    .filter(Boolean);
}

function normalizeRequestedReviewers(reviewers) {
  if (!Array.isArray(reviewers)) {
    return [];
  }
  return reviewers
    .map(reviewer => {
      if (typeof reviewer === "string") {
        return reviewer;
      }
      if (reviewer && typeof reviewer.login === "string") {
        return reviewer.login;
      }
      return "";
    })
    .filter(Boolean);
}

function normalizeRequestedTeams(teams) {
  if (!Array.isArray(teams)) {
    return [];
  }
  return teams
    .map(team => {
      if (typeof team === "string") {
        return team;
      }
      if (team && typeof team.slug === "string") {
        return team.slug;
      }
      if (team && typeof team.name === "string") {
        return team.name;
      }
      return "";
    })
    .filter(Boolean);
}

function normalizeReviews(reviews) {
  if (!Array.isArray(reviews)) {
    return [];
  }
  return reviews
    .map(review => ({
      ...(review?.id != null ? { id: review.id } : {}),
      ...(review?.user?.login ? { user: review.user.login } : {}),
      ...(review?.state ? { state: review.state } : {}),
      ...(review?.submitted_at ? { submitted_at: review.submitted_at } : {}),
    }))
    .filter(review => Object.keys(review).length > 0);
}

function extractIssueStateFromData(issue) {
  return {
    title: typeof issue?.title === "string" ? issue.title : "",
    body: typeof issue?.body === "string" ? issue.body : "",
    state: typeof issue?.state === "string" ? issue.state : "",
    labels: normalizeLabelNames(issue?.labels),
    assignees: normalizeAssigneeLogins(issue?.assignees),
  };
}

function mergeIssueState(baseState, issue) {
  const nextState = {
    title: "",
    body: "",
    state: "",
    labels: [],
    assignees: [],
    ...(baseState || {}),
  };
  if (!issue || typeof issue !== "object") {
    return nextState;
  }
  if ("title" in issue && typeof issue.title === "string") {
    nextState.title = issue.title;
  }
  if ("body" in issue && typeof issue.body === "string") {
    nextState.body = issue.body;
  }
  if ("state" in issue && typeof issue.state === "string") {
    nextState.state = issue.state;
  }
  if ("labels" in issue) {
    nextState.labels = normalizeLabelNames(issue.labels);
  }
  if ("assignees" in issue) {
    nextState.assignees = normalizeAssigneeLogins(issue.assignees);
  }
  return nextState;
}

async function fetchIssueState(github, repoParts, issueNumber) {
  const { data: issue } = await github.rest.issues.get({
    owner: repoParts.owner,
    repo: repoParts.repo,
    issue_number: issueNumber,
  });
  return extractIssueStateFromData(issue);
}

function extractReviewStateFromData(pullRequest, reviews) {
  return {
    requested_reviewers: normalizeRequestedReviewers(pullRequest?.requested_reviewers),
    requested_team_reviewers: normalizeRequestedTeams(pullRequest?.requested_teams ?? pullRequest?.requested_team_reviewers),
    reviews: normalizeReviews(reviews),
  };
}

async function fetchPullRequestReviewState(github, repoParts, pullRequestNumber) {
  const [{ data: pullRequest }, { data: reviews }] = await Promise.all([
    github.rest.pulls.get({
      owner: repoParts.owner,
      repo: repoParts.repo,
      pull_number: pullRequestNumber,
    }),
    github.rest.pulls.listReviews({
      owner: repoParts.owner,
      repo: repoParts.repo,
      pull_number: pullRequestNumber,
      per_page: 100,
    }),
  ]);
  return extractReviewStateFromData(pullRequest, reviews);
}

function attachExecutionState(result, beforeState, afterState) {
  return {
    ...result,
    ...(beforeState ? { before_state: beforeState } : {}),
    ...(afterState ? { after_state: afterState } : {}),
  };
}

module.exports = {
  attachExecutionState,
  extractIssueStateFromData,
  extractReviewStateFromData,
  fetchIssueState,
  fetchPullRequestReviewState,
  mergeIssueState,
  normalizeLabelNames,
};
