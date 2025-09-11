#!/usr/bin/env node

/**
 * MCP Server for GitHub Agentic Workflow Safe Outputs
 * 
 * This TypeScript MCP server implements all safe output types as MCP tools.
 * When tools are invoked, it writes JSONL entries to the safe output file
 * configured via GITHUB_AW_SAFE_OUTPUT_CONFIG environment variable.
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  type CallToolResult,
  type Tool,
} from '@modelcontextprotocol/sdk/types.js';
import * as fs from 'fs';
import * as path from 'path';

// Safe output types supported by the system
interface SafeOutputConfig {
  'create-issue'?: {
    'title-prefix'?: string;
    labels?: string[];
    max?: number;
  };
  'add-issue-comment'?: {
    max?: number;
    target?: string;
  };
  'create-pull-request'?: {
    'title-prefix'?: string;
    labels?: string[];
    draft?: boolean;
    'if-no-changes'?: string;
  };
  'create-pull-request-review-comment'?: {
    max?: number;
    side?: string;
  };
  'create-repository-security-advisory'?: {
    max?: number;
  };
  'add-issue-label'?: {
    allowed?: string[];
    max?: number;
  };
  'update-issue'?: {
    status?: any;
    title?: any;
    body?: any;
    target?: string;
    max?: number;
  };
  'push-to-branch'?: {
    target?: string;
    'if-no-changes'?: string;
  };
  'missing-tool'?: {
    max?: number;
  };
  'create-discussion'?: {
    'title-prefix'?: string;
    'category-id'?: string;
    max?: number;
  };
  'allowed-domains'?: string[];
}

// Tool definitions for each safe output type
const SAFE_OUTPUT_TOOLS: Tool[] = [
  {
    name: 'create-issue',
    description: 'Create a GitHub issue with specified title, body, and optional labels',
    inputSchema: {
      type: 'object',
      properties: {
        title: { type: 'string', description: 'Issue title' },
        body: { type: 'string', description: 'Issue body content' },
        labels: { 
          type: 'array', 
          items: { type: 'string' }, 
          description: 'Optional labels to add to the issue' 
        }
      },
      required: ['title', 'body']
    }
  },
  {
    name: 'add-issue-comment',
    description: 'Add a comment to an issue or pull request',
    inputSchema: {
      type: 'object',
      properties: {
        body: { type: 'string', description: 'Comment body content' },
        issue_number: { 
          type: 'number', 
          description: 'Issue number to comment on (optional, defaults to triggering issue)' 
        }
      },
      required: ['body']
    }
  },
  {
    name: 'create-pull-request',
    description: 'Create a pull request with code changes and metadata',
    inputSchema: {
      type: 'object',
      properties: {
        title: { type: 'string', description: 'Pull request title' },
        body: { type: 'string', description: 'Pull request body content' },
        labels: { 
          type: 'array', 
          items: { type: 'string' }, 
          description: 'Optional labels to add to the pull request' 
        },
        draft: { type: 'boolean', description: 'Create as draft PR' }
      },
      required: ['title', 'body']
    }
  },
  {
    name: 'create-pull-request-review-comment',
    description: 'Create a review comment on a specific line of code in a pull request',
    inputSchema: {
      type: 'object',
      properties: {
        path: { type: 'string', description: 'File path relative to repository root' },
        line: { type: 'number', description: 'Line number for the comment' },
        start_line: { type: 'number', description: 'Starting line number for multi-line comments' },
        side: { type: 'string', description: 'Side of the diff (LEFT or RIGHT)' },
        body: { type: 'string', description: 'Review comment content' }
      },
      required: ['path', 'line', 'body']
    }
  },
  {
    name: 'create-repository-security-advisory',
    description: 'Create a repository security advisory with SARIF format',
    inputSchema: {
      type: 'object',
      properties: {
        file: { type: 'string', description: 'File path relative to repository root' },
        line: { type: 'number', description: 'Line number where the security issue occurs' },
        column: { type: 'number', description: 'Column number (optional)' },
        severity: { type: 'string', description: 'Severity level (error, warning, info, note)' },
        message: { type: 'string', description: 'Description of the security issue' },
        ruleIdSuffix: { type: 'string', description: 'Custom suffix for SARIF rule ID' }
      },
      required: ['file', 'line', 'severity', 'message']
    }
  },
  {
    name: 'add-issue-label',
    description: 'Add labels to an issue or pull request',
    inputSchema: {
      type: 'object',
      properties: {
        labels: { 
          type: 'array', 
          items: { type: 'string' }, 
          description: 'Labels to add' 
        },
        issue_number: { 
          type: 'number', 
          description: 'Issue number to label (optional, defaults to triggering issue)' 
        }
      },
      required: ['labels']
    }
  },
  {
    name: 'update-issue',
    description: 'Update issue properties like title, body, or status',
    inputSchema: {
      type: 'object',
      properties: {
        title: { type: 'string', description: 'New issue title' },
        body: { type: 'string', description: 'New issue body' },
        status: { type: 'string', description: 'Issue status (open or closed)' },
        issue_number: { 
          type: 'number', 
          description: 'Issue number to update (optional, defaults to triggering issue)' 
        }
      }
    }
  },
  {
    name: 'push-to-branch',
    description: 'Push changes to a branch with a commit message',
    inputSchema: {
      type: 'object',
      properties: {
        message: { type: 'string', description: 'Commit message for the changes' },
        pull_request_number: { 
          type: 'number', 
          description: 'Pull request number to push to (optional)' 
        }
      },
      required: ['message']
    }
  },
  {
    name: 'missing-tool',
    description: 'Report missing tools or functionality needed to complete tasks',
    inputSchema: {
      type: 'object',
      properties: {
        tool: { type: 'string', description: 'Name of the missing tool' },
        reason: { type: 'string', description: 'Reason why the tool is needed' },
        alternatives: { type: 'string', description: 'Suggested alternatives' },
        context: { type: 'string', description: 'Context where the tool is needed' }
      },
      required: ['tool', 'reason']
    }
  },
  {
    name: 'create-discussion',
    description: 'Create a GitHub discussion with specified title and body',
    inputSchema: {
      type: 'object',
      properties: {
        title: { type: 'string', description: 'Discussion title' },
        body: { type: 'string', description: 'Discussion body content' },
        category_id: { type: 'string', description: 'Discussion category ID' }
      },
      required: ['title', 'body']
    }
  }
];

class SafeOutputsMCPServer {
  private server: Server;
  private config: SafeOutputConfig;
  private outputFile: string;
  
  constructor() {
    this.server = new Server(
      {
        name: 'safe-outputs',
        version: '1.0.0'
      },
      {
        capabilities: {
          tools: {}
        }
      }
    );
    
    // Parse safe outputs configuration from environment
    this.config = this.loadSafeOutputsConfig();
    this.outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS || '';
    
    if (!this.outputFile) {
      throw new Error('GITHUB_AW_SAFE_OUTPUTS environment variable is required');
    }
    
    this.setupToolHandlers();
  }
  
  private loadSafeOutputsConfig(): SafeOutputConfig {
    const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
    if (!configEnv) {
      throw new Error('GITHUB_AW_SAFE_OUTPUTS_CONFIG environment variable is required');
    }
    
    try {
      return JSON.parse(configEnv);
    } catch (error) {
      throw new Error('Failed to parse GITHUB_AW_SAFE_OUTPUTS_CONFIG: ' + error);
    }
  }
  
  private setupToolHandlers(): void {
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      // Filter tools based on configuration - only return tools that are enabled
      const enabledTools = SAFE_OUTPUT_TOOLS.filter(tool => {
        return this.config[tool.name as keyof SafeOutputConfig] !== undefined;
      });
      
      return { tools: enabledTools };
    });
    
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;
      
      // Validate tool is enabled in configuration
      if (!(name in this.config)) {
        throw new Error(`Tool '${name}' is not enabled in safe outputs configuration`);
      }
      
      // Write JSONL entry to safe outputs file
      const entry = {
        type: name,
        ...args
      };
      
      try {
        await this.writeToOutputFile(entry);
        return {
          content: [{
            type: 'text',
            text: 'Successfully wrote ' + name + ' entry to safe outputs file'
          }]
        } as CallToolResult;
      } catch (error) {
        throw new Error('Failed to write to safe outputs file: ' + error);
      }
    });
  }
  
  private async writeToOutputFile(entry: any): Promise<void> {
    const jsonLine = JSON.stringify(entry) + '\n';
    await fs.promises.appendFile(this.outputFile, jsonLine, 'utf8');
  }
  
  async run(): Promise<void> {
    const transport = new StdioServerTransport();
    await this.server.connect(transport);
  }
}

// Main execution
if (require.main === module) {
  const server = new SafeOutputsMCPServer();
  server.run().catch(console.error);
}

export { SafeOutputsMCPServer };