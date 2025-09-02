import { describe, it, expect, beforeEach, vi } from 'vitest';
import fs from 'fs';
import path from 'path';

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn()
  }
};

const mockGithub = {
  rest: {
    issues: {
      update: vi.fn()
    }
  }
};

const mockContext = {
  eventName: 'issues',
  repo: {
    owner: 'testowner',
    repo: 'testrepo'
  },
  payload: {
    issue: {
      number: 123
    }
  }
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe('update_issue.cjs', () => {
  let updateIssueScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();
    
    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_UPDATE_STATUS;
    delete process.env.GITHUB_AW_UPDATE_TITLE;
    delete process.env.GITHUB_AW_UPDATE_BODY;
    delete process.env.GITHUB_AW_UPDATE_TARGET;
    
    // Set default values
    process.env.GITHUB_AW_UPDATE_STATUS = 'false';
    process.env.GITHUB_AW_UPDATE_TITLE = 'false';
    process.env.GITHUB_AW_UPDATE_BODY = 'false';
    
    // Read the script
    const scriptPath = path.join(__dirname, 'update_issue.cjs');
    updateIssueScript = fs.readFileSync(scriptPath, 'utf8');
  });

  it('should skip when no agent output is provided', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('No GITHUB_AW_AGENT_OUTPUT environment variable found');
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should skip when agent output is empty', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = '   ';
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('Agent output content is empty');
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should skip when not in issue context for triggering target', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        title: 'Updated title'
      }]
    });
    process.env.GITHUB_AW_UPDATE_TITLE = 'true';
    global.context.eventName = 'push'; // Not an issue event
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('Target is "triggering" but not running in issue context, skipping issue update');
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should update issue title successfully', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        title: 'Updated issue title'
      }]
    });
    process.env.GITHUB_AW_UPDATE_TITLE = 'true';
    global.context.eventName = 'issues';
    
    const mockIssue = {
      number: 123,
      title: 'Updated issue title',
      html_url: 'https://github.com/testowner/testrepo/issues/123'
    };
    
    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: 'testowner',
      repo: 'testrepo',
      issue_number: 123,
      title: 'Updated issue title'
    });
    
    expect(mockCore.setOutput).toHaveBeenCalledWith('issue_number', 123);
    expect(mockCore.setOutput).toHaveBeenCalledWith('issue_url', mockIssue.html_url);
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should update issue status successfully', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        status: 'closed'
      }]
    });
    process.env.GITHUB_AW_UPDATE_STATUS = 'true';
    global.context.eventName = 'issues';
    
    const mockIssue = {
      number: 123,
      html_url: 'https://github.com/testowner/testrepo/issues/123'
    };
    
    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: 'testowner',
      repo: 'testrepo',
      issue_number: 123,
      state: 'closed'
    });
    
    consoleSpy.mockRestore();
  });

  it('should update multiple fields successfully', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        title: 'New title',
        body: 'New body content',
        status: 'open'
      }]
    });
    process.env.GITHUB_AW_UPDATE_TITLE = 'true';
    process.env.GITHUB_AW_UPDATE_BODY = 'true';
    process.env.GITHUB_AW_UPDATE_STATUS = 'true';
    global.context.eventName = 'issues';
    
    const mockIssue = {
      number: 123,
      html_url: 'https://github.com/testowner/testrepo/issues/123'
    };
    
    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: 'testowner',
      repo: 'testrepo',
      issue_number: 123,
      title: 'New title',
      body: 'New body content',
      state: 'open'
    });
    
    consoleSpy.mockRestore();
  });

  it('should handle explicit issue number with target "*"', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        issue_number: 456,
        title: 'Updated title'
      }]
    });
    process.env.GITHUB_AW_UPDATE_TITLE = 'true';
    process.env.GITHUB_AW_UPDATE_TARGET = '*';
    global.context.eventName = 'push'; // Not an issue event, but should work with explicit target
    
    const mockIssue = {
      number: 456,
      html_url: 'https://github.com/testowner/testrepo/issues/456'
    };
    
    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: 'testowner',
      repo: 'testrepo',
      issue_number: 456,
      title: 'Updated title'
    });
    
    consoleSpy.mockRestore();
  });

  it('should skip when no valid updates are provided', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        title: 'New title'
      }]
    });
    // All update flags are false
    process.env.GITHUB_AW_UPDATE_STATUS = 'false';
    process.env.GITHUB_AW_UPDATE_TITLE = 'false';
    process.env.GITHUB_AW_UPDATE_BODY = 'false';
    global.context.eventName = 'issues';
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('No valid updates to apply for this item');
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });

  it('should validate status values', async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [{
        type: 'update-issue',
        status: 'invalid'
      }]
    });
    process.env.GITHUB_AW_UPDATE_STATUS = 'true';
    global.context.eventName = 'issues';
    
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);
    
    expect(consoleSpy).toHaveBeenCalledWith('Invalid status value: invalid. Must be \'open\' or \'closed\'');
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    
    consoleSpy.mockRestore();
  });
});
