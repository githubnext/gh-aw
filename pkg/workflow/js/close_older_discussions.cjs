const { getCloseOlderDiscussionMessage } = require("./messages_close_discussion.cjs"),
  MAX_CLOSE_COUNT = 10,
  GRAPHQL_DELAY_MS = 500;
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}
async function searchOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, excludeNumber) {
  let searchQuery = `repo:${owner}/${repo} is:open`;
  if ((titlePrefix && (searchQuery += ` in:title "${titlePrefix.replace(/"/g, '\\"')}"`), labels && labels.length > 0)) for (const label of labels) searchQuery += ` label:"${label.replace(/"/g, '\\"')}"`;
  const result = await github.graphql(
    "\n    query($searchTerms: String!, $first: Int!) {\n      search(query: $searchTerms, type: DISCUSSION, first: $first) {\n        nodes {\n          ... on Discussion {\n            id\n            number\n            title\n            url\n            category {\n              id\n            }\n            labels(first: 100) {\n              nodes {\n                name\n              }\n            }\n            closed\n          }\n        }\n      }\n    }",
    { searchTerms: searchQuery, first: 50 }
  );
  return result && result.search && result.search.nodes
    ? result.search.nodes
        .filter(d => {
          if (!d || d.number === excludeNumber || d.closed) return !1;
          if (titlePrefix && d.title && !d.title.startsWith(titlePrefix)) return !1;
          if (labels && labels.length > 0) {
            const discussionLabels = d.labels?.nodes?.map(l => l.name) || [];
            if (!labels.every(label => discussionLabels.includes(label))) return !1;
          }
          return !(categoryId && (!d.category || d.category.id !== categoryId));
        })
        .map(d => ({ id: d.id, number: d.number, title: d.title, url: d.url }))
    : [];
}
async function addDiscussionComment(github, discussionId, message) {
  return (
    await github.graphql("\n    mutation($dId: ID!, $body: String!) {\n      addDiscussionComment(input: { discussionId: $dId, body: $body }) {\n        comment { \n          id \n          url\n        }\n      }\n    }", {
      dId: discussionId,
      body: message,
    })
  ).addDiscussionComment.comment;
}
async function closeDiscussionAsOutdated(github, discussionId) {
  return (await github.graphql("\n    mutation($dId: ID!) {\n      closeDiscussion(input: { discussionId: $dId, reason: OUTDATED }) {\n        discussion { \n          id\n          url\n        }\n      }\n    }", { dId: discussionId }))
    .closeDiscussion.discussion;
}
async function closeOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, newDiscussion, workflowName, runUrl) {
  const searchCriteria = [];
  (titlePrefix && searchCriteria.push(`title prefix: "${titlePrefix}"`),
    labels && labels.length > 0 && searchCriteria.push(`labels: [${labels.join(", ")}]`),
    core.info(`Searching for older discussions with ${searchCriteria.join(" and ")}`));
  const olderDiscussions = await searchOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, newDiscussion.number);
  if (0 === olderDiscussions.length) return (core.info("No older discussions found to close"), []);
  core.info(`Found ${olderDiscussions.length} older discussion(s) to close`);
  const discussionsToClose = olderDiscussions.slice(0, 10);
  olderDiscussions.length > 10 && core.warning(`Found ${olderDiscussions.length} older discussions, but only closing the first 10`);
  const closedDiscussions = [];
  for (let i = 0; i < discussionsToClose.length; i++) {
    const discussion = discussionsToClose[i];
    try {
      const closingMessage = getCloseOlderDiscussionMessage({ newDiscussionUrl: newDiscussion.url, newDiscussionNumber: newDiscussion.number, workflowName, runUrl });
      (core.info(`Adding closing comment to discussion #${discussion.number}`),
        await addDiscussionComment(github, discussion.id, closingMessage),
        core.info(`Closing discussion #${discussion.number} as outdated`),
        await closeDiscussionAsOutdated(github, discussion.id),
        closedDiscussions.push({ number: discussion.number, url: discussion.url }),
        core.info(`✓ Closed discussion #${discussion.number}: ${discussion.url}`));
    } catch (error) {
      core.error(`✗ Failed to close discussion #${discussion.number}: ${error instanceof Error ? error.message : String(error)}`);
    }
    i < discussionsToClose.length - 1 && (await delay(500));
  }
  return closedDiscussions;
}
module.exports = { closeOlderDiscussions, searchOlderDiscussions, addDiscussionComment, closeDiscussionAsOutdated, MAX_CLOSE_COUNT: 10, GRAPHQL_DELAY_MS: 500 };
