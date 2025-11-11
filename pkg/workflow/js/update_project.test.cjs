import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  getInput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    issues: {
      addLabels: vi.fn().mockResolvedValue({}),
    },
  },
  graphql: vi.fn(),
};

const mockContext = {
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("update_project.cjs", () => {
  let updateProjectScript;
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = (data) => {
    tempFilePath = path.join(
      "/tmp",
      `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`
    );
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "update_project.cjs");
    updateProjectScript = fs.readFileSync(scriptPath, "utf8");
    updateProjectScript = updateProjectScript.replace("export {};", "");
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  describe("generateCampaignId", () => {
    it("should generate campaign ID with slug and timestamp", async () => {
      // We can't directly test the function since it's not exported,
      // but we can observe its behavior through the main function
      const output = {
        items: [
          {
            type: "update_project",
            project: "Bug Bash Q1 2025",
          },
        ],
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          // Get repository ID
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          // Find existing project
          repository: {
            projectsV2: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          // Create project
          createProjectV2: {
            projectV2: {
              id: "project123",
              title: "Bug Bash Q1 2025",
              url: "https://github.com/testowner/testrepo/projects/1",
              number: 1,
            },
          },
        })
        .mockResolvedValueOnce({
          // Link project to repo
          linkProjectV2ToRepository: {
            repository: { id: "repo123" },
          },
        });

      setAgentOutput(output);

      // Execute the script
      await eval(`(async () => { ${updateProjectScript} })()`);

      // Verify campaign ID was logged
      const campaignIdLog = mockCore.info.mock.calls.find((call) =>
        call[0].startsWith("Campaign ID:")
      );
      expect(campaignIdLog).toBeDefined();
      expect(campaignIdLog[0]).toMatch(/Campaign ID: bug-bash-q1-2025-[a-z0-9]{8}/);
    });
  });

  describe("create new project", () => {
    it("should create a new project when it doesn't exist", async () => {
      const output = {
        items: [
          {
            type: "update_project",
            project: "New Campaign",
          },
        ],
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          // Get repository ID
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          // Find existing project (none found)
          repository: {
            projectsV2: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          // Create project
          createProjectV2: {
            projectV2: {
              id: "project123",
              title: "New Campaign",
              url: "https://github.com/testowner/testrepo/projects/1",
              number: 1,
            },
          },
        })
        .mockResolvedValueOnce({
          // Link project to repo
          linkProjectV2ToRepository: {
            repository: { id: "repo123" },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);

      // Wait for async operations
      // No need to wait with eval

      // Verify project creation
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("createProjectV2"),
        expect.objectContaining({
          ownerId: "repo123",
          title: "New Campaign",
          shortDescription: expect.stringContaining("Campaign ID:"),
        })
      );

      // Verify project linking
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("linkProjectV2ToRepository"),
        expect.objectContaining({
          projectId: "project123",
          repositoryId: "repo123",
        })
      );

      // Verify outputs were set
      expect(mockCore.setOutput).toHaveBeenCalledWith("project-id", "project123");
      expect(mockCore.setOutput).toHaveBeenCalledWith("project-number", 1);
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "project-url",
        "https://github.com/testowner/testrepo/projects/1"
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "campaign-id",
        expect.stringMatching(/new-campaign-[a-z0-9]{8}/)
      );
    });

    it("should use custom campaign ID when provided", async () => {
      const output = {
        type: "update_project",
        project: "Custom Campaign",
        campaign_id: "custom-id-2025",
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          createProjectV2: {
            projectV2: {
              id: "project456",
              title: "Custom Campaign",
              url: "https://github.com/testowner/testrepo/projects/2",
              number: 2,
            },
          },
        })
        .mockResolvedValueOnce({
          linkProjectV2ToRepository: {
            repository: { id: "repo123" },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      // Verify custom campaign ID was used
      expect(mockCore.info).toHaveBeenCalledWith("Campaign ID: custom-id-2025");
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "campaign-id",
        "custom-id-2025"
      );
    });
  });

  describe("find existing project", () => {
    it("should find existing project by title", async () => {
      const output = {
        type: "update_project",
        project: "Existing Campaign",
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          // Find existing project by title
          repository: {
            projectsV2: {
              nodes: [
                {
                  id: "existing-project-123",
                  title: "Existing Campaign",
                  number: 5,
                },
              ],
            },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      expect(mockCore.info).toHaveBeenCalledWith(
        "✓ Found existing project: Existing Campaign (#5)"
      );
      
      // Should not create a new project
      expect(mockGithub.graphql).not.toHaveBeenCalledWith(
        expect.stringContaining("createProjectV2"),
        expect.anything()
      );
    });

    it("should find existing project by number", async () => {
      const output = {
        type: "update_project",
        project: "7", // Project number as string
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [
                {
                  id: "project-by-number",
                  title: "Some Project",
                  number: 7,
                },
              ],
            },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      expect(mockCore.info).toHaveBeenCalledWith(
        "✓ Found existing project: 7 (#7)"
      );
    });
  });

  describe("add issue to project", () => {
    it("should add issue to project board", async () => {
      const output = {
        type: "update_project",
        project: "Bug Tracking",
        issue: 42,
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [
                { id: "project123", title: "Bug Tracking", number: 1 },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          // Get issue ID
          repository: {
            issue: { id: "issue-id-42" },
          },
        })
        .mockResolvedValueOnce({
          // Check if item exists on board
          node: {
            items: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          // Add item to board
          addProjectV2ItemById: {
            item: { id: "item123" },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      // Verify issue was queried
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("issue(number: $number)"),
        expect.objectContaining({
          owner: "testowner",
          repo: "testrepo",
          number: 42,
        })
      );

      // Verify item was added to board
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("addProjectV2ItemById"),
        expect.objectContaining({
          projectId: "project123",
          contentId: "issue-id-42",
        })
      );

      // Verify campaign label was added
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 42,
        labels: [expect.stringMatching(/campaign:bug-tracking-[a-z0-9]{8}/)],
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith("item-id", "item123");
    });

    it("should skip adding issue if already on board", async () => {
      const output = {
        type: "update_project",
        project: "Bug Tracking",
        issue: 42,
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [
                { id: "project123", title: "Bug Tracking", number: 1 },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          repository: {
            issue: { id: "issue-id-42" },
          },
        })
        .mockResolvedValueOnce({
          // Item already exists on board
          node: {
            items: {
              nodes: [
                {
                  id: "existing-item",
                  content: { id: "issue-id-42" },
                },
              ],
            },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");

      // Should not add item again
      expect(mockGithub.graphql).not.toHaveBeenCalledWith(
        expect.stringContaining("addProjectV2ItemById"),
        expect.anything()
      );
    });
  });

  describe("add pull request to project", () => {
    it("should add PR to project board", async () => {
      const output = {
        type: "update_project",
        project: "PR Review Board",
        pull_request: 99,
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [
                { id: "project789", title: "PR Review Board", number: 3 },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          // Get PR ID
          repository: {
            pullRequest: { id: "pr-id-99" },
          },
        })
        .mockResolvedValueOnce({
          node: {
            items: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          addProjectV2ItemById: {
            item: { id: "pr-item-99" },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      // Verify PR was queried (not issue)
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("pullRequest(number: $number)"),
        expect.objectContaining({
          number: 99,
        })
      );

      // Verify campaign label was added to PR
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 99,
        labels: [expect.stringMatching(/campaign:pr-review-board-[a-z0-9]{8}/)],
      });
    });
  });

  describe("update custom fields", () => {
    it("should update text field on project item", async () => {
      const output = {
        type: "update_project",
        project: "Field Test",
        issue: 10,
        fields: {
          Status: "In Progress",
        },
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [{ id: "project999", title: "Field Test", number: 10 }],
            },
          },
        })
        .mockResolvedValueOnce({
          repository: {
            issue: { id: "issue-id-10" },
          },
        })
        .mockResolvedValueOnce({
          node: {
            items: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          addProjectV2ItemById: {
            item: { id: "item-10" },
          },
        })
        .mockResolvedValueOnce({
          // Get project fields
          node: {
            fields: {
              nodes: [
                {
                  id: "field-status",
                  name: "Status",
                },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          // Update field value
          updateProjectV2ItemFieldValue: {
            projectV2Item: { id: "item-10" },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      expect(mockCore.info).toHaveBeenCalledWith(
        '✓ Updated field "Status" = "In Progress"'
      );
    });

    it("should handle single select field with options", async () => {
      const output = {
        type: "update_project",
        project: "Priority Board",
        issue: 15,
        fields: {
          Priority: "High",
        },
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [
                { id: "priority-project", title: "Priority Board", number: 5 },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          repository: {
            issue: { id: "issue-id-15" },
          },
        })
        .mockResolvedValueOnce({
          node: {
            items: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          addProjectV2ItemById: {
            item: { id: "item-15" },
          },
        })
        .mockResolvedValueOnce({
          // Get project fields with options
          node: {
            fields: {
              nodes: [
                {
                  id: "field-priority",
                  name: "Priority",
                  options: [
                    { id: "option-low", name: "Low" },
                    { id: "option-medium", name: "Medium" },
                    { id: "option-high", name: "High" },
                  ],
                },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          updateProjectV2ItemFieldValue: {
            projectV2Item: { id: "item-15" },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      // Verify field was updated with correct option ID
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("updateProjectV2ItemFieldValue"),
        expect.objectContaining({
          fieldId: "field-priority",
          value: { singleSelectOptionId: "option-high" },
        })
      );
    });

    it("should warn when field does not exist", async () => {
      const output = {
        type: "update_project",
        project: "Test Project",
        issue: 20,
        fields: {
          NonExistentField: "Some Value",
        },
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [
                { id: "test-project", title: "Test Project", number: 1 },
              ],
            },
          },
        })
        .mockResolvedValueOnce({
          repository: {
            issue: { id: "issue-id-20" },
          },
        })
        .mockResolvedValueOnce({
          node: {
            items: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          addProjectV2ItemById: {
            item: { id: "item-20" },
          },
        })
        .mockResolvedValueOnce({
          node: {
            fields: {
              nodes: [
                {
                  id: "field-status",
                  name: "Status",
                },
              ],
            },
          },
        });

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      expect(mockCore.warning).toHaveBeenCalledWith(
        'Field "NonExistentField" not found in project'
      );
    });
  });

  describe("error handling", () => {
    it("should handle campaign label add failure gracefully", async () => {
      const output = {
        type: "update_project",
        project: "Label Test",
        issue: 50,
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [{ id: "project-label", title: "Label Test", number: 2 }],
            },
          },
        })
        .mockResolvedValueOnce({
          repository: {
            issue: { id: "issue-id-50" },
          },
        })
        .mockResolvedValueOnce({
          node: {
            items: {
              nodes: [],
            },
          },
        })
        .mockResolvedValueOnce({
          addProjectV2ItemById: {
            item: { id: "item-50" },
          },
        });

      // Mock label addition to fail
      mockGithub.rest.issues.addLabels.mockRejectedValueOnce(
        new Error("Label creation failed")
      );

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      // Should warn but not fail
      expect(mockCore.warning).toHaveBeenCalledWith(
        "Failed to add campaign label: Label creation failed"
      );

      // Should still complete successfully
      expect(mockCore.info).toHaveBeenCalledWith(
        "✓ Project management completed successfully"
      );
    });

    it("should throw error on project creation failure", async () => {
      const output = {
        type: "update_project",
        project: "Fail Project",
      };

      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: { id: "repo123" },
        })
        .mockResolvedValueOnce({
          repository: {
            projectsV2: {
              nodes: [],
            },
          },
        })
        .mockRejectedValueOnce(new Error("GraphQL error: Insufficient permissions"));

      setAgentOutput(output);

      await eval(`(async () => { ${updateProjectScript} })()`);
      // No need to wait with eval

      expect(mockCore.error).toHaveBeenCalledWith(
        expect.stringContaining("Failed to manage project:")
      );
    });
  });
});
