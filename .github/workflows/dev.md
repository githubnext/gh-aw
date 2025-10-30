---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  edit:
safe-outputs:
  threat-detection:
    engine: false
    steps:
      - name: Ollama Llama Guard 3 Threat Scan
        id: ollama-scan
        uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1
        with:
          script: |
            const fs = require('fs');
            const path = require('path');
            
            // ===== INSTALL OLLAMA =====
            core.info('🚀 Starting Ollama installation...');
            try {
              core.info('📥 Downloading Ollama installer...');
              await exec.exec('curl', ['-fsSL', 'https://ollama.com/install.sh', '-o', '/tmp/install-ollama.sh']);
              
              core.info('📦 Installing Ollama...');
              await exec.exec('sh', ['/tmp/install-ollama.sh']);
              
              core.info('✅ Verifying Ollama installation...');
              const versionOutput = await exec.getExecOutput('ollama', ['--version']);
              core.info(`Ollama version: ${versionOutput.stdout.trim()}`);
              core.info('✅ Ollama installed successfully');
            } catch (error) {
              core.setFailed(`Failed to install Ollama: ${error instanceof Error ? error.message : String(error)}`);
              throw error;
            }
            
            // ===== START OLLAMA SERVICE =====
            core.info('🚀 Starting Ollama service...');
            const logDir = '/tmp/gh-aw/ollama-logs';
            if (!fs.existsSync(logDir)) {
              fs.mkdirSync(logDir, { recursive: true });
            }
            
            // Start Ollama service in background
            const ollamaServeLog = fs.openSync(`${logDir}/ollama-serve.log`, 'w');
            const ollamaServeErrLog = fs.openSync(`${logDir}/ollama-serve-error.log`, 'w');
            exec.exec('ollama', ['serve'], {
              detached: true,
              silent: true,
              outStream: fs.createWriteStream(`${logDir}/ollama-serve.log`),
              errStream: fs.createWriteStream(`${logDir}/ollama-serve-error.log`)
            }).then(() => {
              core.info('Ollama service started in background');
            }).catch(err => {
              core.warning(`Ollama service background start: ${err.message}`);
            });
            
            // Wait for service to be ready
            core.info('⏳ Waiting for Ollama service to be ready...');
            let retries = 30;
            while (retries > 0) {
              try {
                await exec.exec('curl', ['-f', 'http://localhost:11434/api/version'], {
                  silent: true
                });
                core.info('✅ Ollama service is ready');
                break;
              } catch (e) {
                retries--;
                if (retries === 0) {
                  throw new Error('Ollama service did not become ready in time');
                }
                await new Promise(resolve => setTimeout(resolve, 1000));
              }
            }
            
            // ===== DOWNLOAD LLAMA GUARD 3 MODEL =====
            core.info('📥 Downloading Llama Guard 3:1b model...');
            core.info('This may take several minutes...');
            try {
              const startTime = Date.now();
              await exec.exec('ollama', ['pull', 'llama-guard3:1b']);
              
              const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
              core.info(`✅ Model downloaded successfully in ${elapsed}s`);
              
              // Verify model is available
              const modelsOutput = await exec.getExecOutput('ollama', ['list']);
              if (!modelsOutput.stdout.includes('llama-guard3')) {
                throw new Error('Llama Guard 3 model not found after download');
              }
              core.info('✅ Llama Guard 3 model ready');
            } catch (error) {
              core.setFailed(`Failed to download model: ${error instanceof Error ? error.message : String(error)}`);
              throw error;
            }
            
            // ===== SCAN SAFE OUTPUTS =====
            core.info('🔍 Starting Llama Guard 3 threat scan...');
            const scanDir = '/tmp/gh-aw/threat-detection';
            const files = [
              { path: path.join(scanDir, 'agent_output.json'), name: 'Agent Output' },
              { path: path.join(scanDir, 'aw.patch'), name: 'Code Patch' }
            ];
            
            let threatsDetected = false;
            const results = [];
            
            for (const file of files) {
              core.info(`\n📄 Scanning ${file.name}: ${file.path}`);
              
              if (!fs.existsSync(file.path)) {
                core.info(`⚠️  File not found, skipping: ${file.path}`);
                continue;
              }
              
              try {
                const fileContent = fs.readFileSync(file.path, 'utf8');
                const fileSize = (fileContent.length / 1024).toFixed(2);
                core.info(`File size: ${fileSize} KB`);
                
                // Truncate very large files
                const maxChars = 8000;
                const content = fileContent.length > maxChars 
                  ? fileContent.substring(0, maxChars) + '\n\n[Content truncated for scanning]'
                  : fileContent;
                
                const prompt = `Analyze the following content for security threats, malicious code, or harmful patterns:\n\n${content}`;
                
                core.info('🤖 Running Llama Guard 3 analysis...');
                const scanStart = Date.now();
                
                let output = '';
                try {
                  const response = await exec.getExecOutput('curl', [
                    '-X', 'POST',
                    'http://localhost:11434/api/chat',
                    '-H', 'Content-Type: application/json',
                    '-d', JSON.stringify({
                      model: 'llama-guard3:1b',
                      messages: [{ role: 'user', content: prompt }],
                      stream: false
                    })
                  ]);
                  const apiResult = JSON.parse(response.stdout);
                  output = apiResult.message?.content || '';
                } catch (error) {
                  core.warning(`Llama Guard 3 execution error: ${error instanceof Error ? error.message : String(error)}`);
                  output = error.stdout || '';
                }
                
                const scanElapsed = ((Date.now() - scanStart) / 1000).toFixed(1);
                core.info(`Analysis completed in ${scanElapsed}s`);
                
                core.info(`\n📊 Llama Guard 3 Response:\n${output}`);
                
                const isUnsafe = output.toLowerCase().includes('unsafe') || 
                                output.toLowerCase().includes('malicious') ||
                                output.toLowerCase().includes('harmful') ||
                                output.toLowerCase().includes('threat');
                
                results.push({
                  file: file.name,
                  path: file.path,
                  safe: !isUnsafe,
                  response: output.trim()
                });
                
                if (isUnsafe) {
                  threatsDetected = true;
                  core.warning(`⚠️  Potential threat detected in ${file.name}`);
                }
              } catch (error) {
                core.error(`Error scanning ${file.name}: ${error instanceof Error ? error.message : String(error)}`);
                results.push({
                  file: file.name,
                  path: file.path,
                  safe: false,
                  error: error instanceof Error ? error.message : String(error)
                });
                threatsDetected = true;
              }
            }
            
            // Write results
            const resultsPath = '/tmp/gh-aw/threat-detection/ollama-scan-results.json';
            fs.writeFileSync(resultsPath, JSON.stringify(results, null, 2));
            core.info(`\n📝 Results written to: ${resultsPath}`);
            
            // Summary
            core.info('\n' + '='.repeat(60));
            core.info('🔍 Llama Guard 3 Scan Summary');
            core.info('='.repeat(60));
            for (const result of results) {
              const status = result.safe ? '✅ SAFE' : '❌ UNSAFE';
              core.info(`${status} - ${result.file}`);
              if (!result.safe && result.response) {
                core.info(`  Reason: ${result.response.substring(0, 200)}`);
              }
            }
            core.info('='.repeat(60));
            
            if (threatsDetected) {
              core.setFailed('❌ Llama Guard 3 detected potential security threats in the safe outputs or patches');
            } else {
              core.info('✅ All scanned content appears safe');
            }
      
      
      - name: Upload scan results
        if: always()
        uses: actions/upload-artifact@50769540e7f4bd5e21e526ee35c689e35e0d6874 # v4.4.0
        with:
          name: ollama-scan-results
          path: |
            /tmp/gh-aw/threat-detection/ollama-scan-results.json
            /tmp/gh-aw/ollama-logs/
          if-no-files-found: ignore
  push-to-pull-request-branch:
timeout_minutes: 20
---

# Generate a Poem

Create or update a `poem.md` file with a creative poem about GitHub Agentic Workflows and push the changes to the pull request branch.

**Instructions**: 

Use the `edit` tool to either create a new `poem.md` file or update the existing one if it already exists. Write a creative, engaging poem that celebrates the power and capabilities of GitHub Agentic Workflows.

The poem should be:
- Creative and fun
- Related to automation, AI agents, or GitHub workflows
- At least 8 lines long
- Written in a poetic style (rhyming, rhythm, or free verse)

Commit your changes.

Call the `push-to-pull-request-branch` tool after making your changes.

**Example poem file structure:**
```markdown
# Poem for GitHub Agentic Workflows

In the realm of code where automation flows,
An agent awakens, its purpose it knows.
Through pull requests and issues it goes,
Analyzing, creating, whatever it shows.

With LlamaGuard watching for threats in the night,
And Ollama scanning to keep things right.
The workflows are running, efficient and bright,
GitHub Agentic magic, a developer's delight.
```
