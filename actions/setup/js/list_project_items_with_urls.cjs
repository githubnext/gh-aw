// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * List project items with full content URLs
 * 
 * This tool extends the GitHub MCP list_project_items functionality by including
 * the content.url field for linked issues and pull requests, which is not provided
 * by the standard GitHub MCP server tools.
 * 
 * @param {Object} params - Parameters
 * @param {string} params.project - Project URL (e.g., https://github.com/orgs/myorg/projects/123)
 * @param {number} [params.first=100] - Number of items to fetch per page
 * @returns {Promise<Array>} Project items with full URLs
 */
async function listProjectItemsWithUrls({ project, first = 100 }) {
  core.info(`Listing project items with URLs for: ${project}`);

  // Parse project URL
  const match = project.match(/github\.com\/(users|orgs)\/([^/]+)\/projects\/(\d+)/);
  if (!match) {
    throw new Error(`Invalid project URL: "${project}". Expected format: https://github.com/orgs/myorg/projects/123`);
  }

  const [, scope, ownerLogin, projectNumber] = match;
  const projectNumberInt = parseInt(projectNumber, 10);

  // Get project ID first
  const projectQuery =
    scope === "orgs"
      ? `query($login: String!, $number: Int!) {
          organization(login: $login) {
            projectV2(number: $number) {
              id
              title
              url
            }
          }
        }`
      : `query($login: String!, $number: Int!) {
          user(login: $login) {
            projectV2(number: $number) {
              id
              title
              url
            }
          }
        }`;

  const projectResult = await github.graphql(projectQuery, {
    login: ownerLogin,
    number: projectNumberInt,
  });

  const projectData = scope === "orgs" ? projectResult?.organization?.projectV2 : projectResult?.user?.projectV2;

  if (!projectData) {
    throw new Error(`Project #${projectNumberInt} not found for ${scope} ${ownerLogin}`);
  }

  const projectId = projectData.id;
  core.info(`Found project: ${projectData.title} (${projectData.url})`);

  // Fetch all project items with full content information including URLs
  const allItems = [];
  let hasNextPage = true;
  let endCursor = null;

  while (hasNextPage) {
    const itemsQuery = `query($projectId: ID!, $first: Int!, $after: String) {
      node(id: $projectId) {
        ... on ProjectV2 {
          items(first: $first, after: $after) {
            nodes {
              id
              type
              content {
                __typename
                ... on Issue {
                  id
                  number
                  title
                  url
                  state
                  repository {
                    name
                    owner {
                      login
                    }
                  }
                }
                ... on PullRequest {
                  id
                  number
                  title
                  url
                  state
                  repository {
                    name
                    owner {
                      login
                    }
                  }
                }
                ... on DraftIssue {
                  id
                  title
                }
              }
              fieldValues(first: 20) {
                nodes {
                  ... on ProjectV2ItemFieldTextValue {
                    text
                    field {
                      ... on ProjectV2FieldCommon {
                        name
                      }
                    }
                  }
                  ... on ProjectV2ItemFieldNumberValue {
                    number
                    field {
                      ... on ProjectV2FieldCommon {
                        name
                      }
                    }
                  }
                  ... on ProjectV2ItemFieldDateValue {
                    date
                    field {
                      ... on ProjectV2FieldCommon {
                        name
                      }
                    }
                  }
                  ... on ProjectV2ItemFieldSingleSelectValue {
                    name
                    field {
                      ... on ProjectV2FieldCommon {
                        name
                      }
                    }
                  }
                  ... on ProjectV2ItemFieldIterationValue {
                    title
                    startDate
                    duration
                    field {
                      ... on ProjectV2FieldCommon {
                        name
                      }
                    }
                  }
                }
              }
            }
            pageInfo {
              hasNextPage
              endCursor
            }
          }
        }
      }
    }`;

    const result = await github.graphql(itemsQuery, {
      projectId,
      first,
      after: endCursor,
    });

    const items = result.node.items.nodes || [];
    allItems.push(...items);

    hasNextPage = result.node.items.pageInfo.hasNextPage;
    endCursor = result.node.items.pageInfo.endCursor;

    core.info(`Fetched ${items.length} items (total so far: ${allItems.length})`);
  }

  // Transform items to a more usable format
  const transformedItems = allItems.map(item => {
    const transformed = {
      id: item.id,
      type: item.type,
    };

    // Add content information with URL
    if (item.content) {
      transformed.content = {
        type: item.content.__typename,
      };

      if (item.content.__typename === "Issue" || item.content.__typename === "PullRequest") {
        transformed.content.id = item.content.id;
        transformed.content.number = item.content.number;
        transformed.content.title = item.content.title;
        transformed.content.url = item.content.url; // <-- THE MISSING URL!
        transformed.content.state = item.content.state;
        
        if (item.content.repository) {
          transformed.content.repository = {
            owner: item.content.repository.owner.login,
            name: item.content.repository.name,
          };
        }
      } else if (item.content.__typename === "DraftIssue") {
        transformed.content.id = item.content.id;
        transformed.content.title = item.content.title;
      }
    }

    // Add field values
    if (item.fieldValues?.nodes) {
      transformed.fields = {};
      item.fieldValues.nodes.forEach(fieldValue => {
        if (!fieldValue?.field?.name) return;
        
        const fieldName = fieldValue.field.name;
        
        if (fieldValue.text !== undefined) {
          transformed.fields[fieldName] = fieldValue.text;
        } else if (fieldValue.number !== undefined) {
          transformed.fields[fieldName] = fieldValue.number;
        } else if (fieldValue.date !== undefined) {
          transformed.fields[fieldName] = fieldValue.date;
        } else if (fieldValue.name !== undefined) {
          transformed.fields[fieldName] = fieldValue.name;
        } else if (fieldValue.title !== undefined) {
          transformed.fields[fieldName] = {
            title: fieldValue.title,
            startDate: fieldValue.startDate,
            duration: fieldValue.duration,
          };
        }
      });
    }

    return transformed;
  });

  core.info(`✓ Retrieved ${transformedItems.length} project items with full content URLs`);
  return transformedItems;
}

/**
 * Main execution function
 */
async function main() {
  try {
    const input = core.getInput("params", { required: true });
    const params = JSON.parse(input);

    const items = await listProjectItemsWithUrls(params);
    
    // Output results as JSON
    core.setOutput("items", JSON.stringify(items));
    core.setOutput("count", items.length);
    
    // Also log a summary
    const issueCount = items.filter(i => i.content?.type === "Issue").length;
    const prCount = items.filter(i => i.content?.type === "PullRequest").length;
    const draftCount = items.filter(i => i.content?.type === "DraftIssue").length;
    
    core.info(`Summary: ${issueCount} issues, ${prCount} PRs, ${draftCount} drafts`);
  } catch (error) {
    core.setFailed(`Failed to list project items: ${getErrorMessage(error)}`);
    throw error;
  }
}

module.exports = { main, listProjectItemsWithUrls };

// Run if called directly
if (require.main === module) {
  main();
}
