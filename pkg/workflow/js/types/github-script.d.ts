// Type definitions for GitHub Actions github-script action
// These globals are provided by the github-script action environment

import * as core from '@actions/core';
import * as github from '@actions/github';

declare global {
  /**
   * GitHub API client instance provided by github-script action
   */
  const github: ReturnType<typeof github.getOctokit>;
  
  /**
   * GitHub Actions context object provided by github-script action
   */
  const context: typeof github.context;
  
  /**
   * Actions core utilities provided by github-script action
   */
  const core: typeof core;
  
  /**
   * Console object for logging (available in Node.js environment)
   */
  const console: Console;
  
  /**
   * Process object for environment variables and utilities
   */
  const process: NodeJS.Process;
  
  /**
   * Require function for CommonJS modules
   */
  const require: NodeRequire;
  
  /**
   * Global exports object for CommonJS modules
   */
  var exports: any;
  
  /**
   * Global module object for CommonJS modules
   */
  var module: NodeJS.Module;
}

export {};
