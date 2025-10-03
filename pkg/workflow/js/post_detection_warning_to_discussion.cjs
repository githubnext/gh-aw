/**
 * Posts a warning comment to a discussion when threat detection finds issues
 * This script runs in the detection job when threats are detected.
 */

const discussionId = process.env.DISCUSSION_ID;
const body = '⚠️ **Security Alert**: Threat detection found a potential issue. Please review the [detection logs](../actions/runs/${{ github.run_id }}) for details.';

const mutation = `
  mutation($discussionId: ID!, $body: String!) {
    addDiscussionComment(input: {
      discussionId: $discussionId,
      body: $body
    }) {
      comment {
        id
      }
    }
  }
`;

try {
  await github.graphql(mutation, {
    discussionId,
    body
  });
  core.info('Posted warning comment to discussion');
} catch (error) {
  core.warning(`Failed to post comment to discussion: ${error.message}`);
}
