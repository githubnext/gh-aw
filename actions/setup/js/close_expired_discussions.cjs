// @ts-check
// <reference types="@actions/github-script" />

const { executeExpiredEntityCleanup } = require("./expired_entity_main_flow.cjs");
const { addDiscussionComment, closeDiscussionAsOutdated, createClosedRecord, createExpiredEntityHandler } = require("./expired_entity_handler_factory.cjs");
const { getWorkflowMetadata } = require("./workflow_metadata_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

/**
 * Check if a discussion already has an expiration comment and fetch its closed state
 * @param {any} github - GitHub GraphQL instance
 * @param {string} discussionId - Discussion node ID
 * @returns {Promise<{hasComment: boolean, isClosed: boolean}>} Object with comment existence and closed state
 */
async function hasExpirationComment(github, discussionId) {
  const result = await github.graphql(
    `
    query($dId: ID!) {
      node(id: $dId) {
        ... on Discussion {
          closed
          comments(first: 100) {
            nodes {
              body
            }
          }
        }
      }
    }`,
    { dId: discussionId }
  );

  if (!result || !result.node) {
    return { hasComment: false, isClosed: false };
  }

  const isClosed = result.node.closed || false;
  const comments = result.node.comments?.nodes || [];
  const expirationCommentPattern = /<!--\s*gh-aw-closed\s*-->/;
  const hasComment = comments.some(comment => comment.body && expirationCommentPattern.test(comment.body));

  return { hasComment, isClosed };
}

async function main() {
  const { owner, repo } = resolveExecutionOwnerRepo();
  core.info(`Operating on repository: ${owner}/${repo}`);

  // Get workflow metadata for footer
  const { workflowName, workflowId, runUrl } = getWorkflowMetadata(owner, repo);

  await executeExpiredEntityCleanup(github, owner, repo, {
    entityType: "discussions",
    graphqlField: "discussions",
    resultKey: "discussions",
    entityLabel: "Discussion",
    summaryHeading: "Expired Discussions Cleanup",
    enableDedupe: true, // Discussions may have duplicates across pages
    includeSkippedHeading: true,
    processEntity: createExpiredEntityHandler({
      github,
      owner,
      repo,
      workflowName,
      workflowId,
      runUrl,
      entityNoun: "discussion",
      entityLabel: "Discussion",
      footerSuffix: "\n\n<!-- gh-aw-closed -->",
      beforeComment: async discussion => {
        core.info(`  Checking for existing expiration comment and closed state on discussion #${discussion.number}`);
        const { hasComment, isClosed } = await hasExpirationComment(github, discussion.id);

        if (isClosed) {
          core.warning(`  Discussion #${discussion.number} is already closed, skipping`);
          return {
            status: "skipped",
            record: createClosedRecord(discussion),
          };
        }

        if (hasComment) {
          core.warning(`  Discussion #${discussion.number} already has an expiration comment, skipping to avoid duplicate`);

          core.info(`  Attempting to close discussion #${discussion.number} without adding another comment`);
          await closeDiscussionAsOutdated(github, discussion.id);
          core.info(`  ✓ Discussion closed successfully`);

          return {
            status: "skipped",
            record: createClosedRecord(discussion),
          };
        }

        return undefined;
      },
      beforeCommentLog: discussion => `  Adding closing comment to discussion #${discussion.number}`,
      beforeCloseLog: discussion => `  Closing discussion #${discussion.number} as outdated`,
      addComment: (discussion, message) => addDiscussionComment(github, discussion.id, message),
      closeEntity: discussion => closeDiscussionAsOutdated(github, discussion.id),
    }),
  });
}

module.exports = { main };
