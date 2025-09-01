import { describe, it, expect, beforeEach, vi } from 'vitest';
import fs from 'fs';
import path from 'path';

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn()
  }
};

const mockGithub = {
  rest: {
    issues: {
      createComment: vi.fn()
    }
  }
};

const mockContext = {
  eventName: 'issues',
  runId: 12345,
  repo: {
    owner: 'testowner',
    repo: 'testrepo'
  },
  payload: {
    issue: {
      number: 123
    },
    repository: {
      html_url: 'https://github.com/testowner/testrepo'
    }
  }
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe('create_comment.cjs', () => {
  let createCommentScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();
    
    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    
    // Reset context to default state
    global.context.eventName = 'issues';
    global.context.payload.issue = { number: 123 };
    
    // Read the script content
    const scriptPath = path.join(process.cwd(), 'pkg/workflow/js/create_comment.cjs');
    createCommentScript = fs.readFileSync(scriptPath, 'utf8');
  });

  it('should skip when no agent output is provided', async () => {
    // Remove the output content environment variable
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('No GITHUB_AW_AGENT_OUTPUT environment variable found');
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should skip when agent output is empty', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = '   ';
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('Agent output content is empty');
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should skip when not in issue or PR context', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'add-issue-comment',
        body: 'Test comment content'
      }]
    });
    global.context.eventName = 'push'; // Not an issue or PR event
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('Not running in issue or pull request context, skipping comment creation');
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should create comment on issue successfully', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'add-issue-comment',
        body: 'Test comment content'
      }]
    });
    global.context.eventName = 'issues';
    
    const mockComment = {
      id: 456,
      html_url: 'https://github.com/testowner/testrepo/issues/123#issuecomment-456'
    };
    
    mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);
    
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: 'testowner',
      repo: 'testrepo',
      issue_number: 123,
      body: expect.stringContaining('Test comment content')
    });
    
    expect(mockCore.setOutput).toHaveBeenCalledWith('comment_id', 456);
    expect(mockCore.setOutput).toHaveBeenCalledWith('comment_url', mockComment.html_url);
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should create comment on pull request successfully', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'add-issue-comment',
        body: 'Test PR comment content'
      }]
    });
    global.context.eventName = 'pull_request';
    global.context.payload.pull_request = { number: 789 };
    delete global.context.payload.issue; // Remove issue from payload
    
    const mockComment = {
      id: 789,
      html_url: 'https://github.com/testowner/testrepo/issues/789#issuecomment-789'
    };
    
    mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);
    
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: 'testowner',
      repo: 'testrepo',
      issue_number: 789,
      body: expect.stringContaining('Test PR comment content')
    });
    
    consoleSpy.mockRestore();
  });

  it('should include run information in comment body', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'add-issue-comment',
        body: 'Test content'
      }]
    });
    global.context.eventName = 'issues';
    global.context.payload.issue = { number: 123 }; // Make sure issue context is properly set
    
    const mockComment = {
      id: 456,
      html_url: 'https://github.com/testowner/testrepo/issues/123#issuecomment-456'
    };
    
    mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);
    
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    expect(mockGithub.rest.issues.createComment.mock.calls).toHaveLength(1);
    
    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
    expect(callArgs.body).toContain('Test content');
    expect(callArgs.body).toContain('Generated by Agentic Workflow Run');
    expect(callArgs.body).toContain('[12345]');
    expect(callArgs.body).toContain('https://github.com/testowner/testrepo/actions/runs/12345');
    
    consoleSpy.mockRestore();
  });
});