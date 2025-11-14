const mockCore = { 
  info: (...args) => console.log('INFO:', ...args),
  warning: (...args) => console.log('WARN:', ...args),
  error: (...args) => console.error('ERROR:', ...args),
  setFailed: (...args) => console.error('FAILED:', ...args),
  summary: { addRaw: () => mockCore.summary, write: async () => console.log('SUMMARY WRITTEN') }
};

const mockContext = { repo: { owner: 'test', repo: 'test' }, sha: 'abc123' };

const mockGithub = { 
  rest: { 
    repos: { 
      getContent: async (params) => {
        console.log('GET CONTENT:', params);
        return { data: { sha: 'test123' } };
      }, 
      listCommits: async (params) => {
        console.log('LIST COMMITS:', params);
        return { 
          data: [{
            sha: 'commit123',
            commit: { 
              committer: { date: '2024-01-01T10:00:00Z' }
            }
          }]
        };
      }
    } 
  } 
};

global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;
process.env.GH_AW_WORKFLOW_FILE = 'test.lock.yml';

const fs = require('fs');
const script = fs.readFileSync('check_workflow_timestamp.cjs', 'utf8');

(async () => {
  try {
    await eval('(async () => { ' + script + ' })()');
    console.log('Script completed successfully');
  } catch (e) {
    console.error('Script error:', e);
  }
})();
