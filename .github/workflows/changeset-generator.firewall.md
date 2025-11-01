---
name: Changeset Generator
on:
  pull_request:
    types: [labeled]
  workflow_dispatch:
  reaction: "rocket"
if: github.event.pull_request.base.ref == github.event.repository.default_branch && (github.event_name == 'workflow_dispatch' || github.event.label.name == 'changeset')
permissions:
  contents: read
  pull-requests: read
  issues: read
engine: copilot
safe-outputs:
  push-to-pull-request-branch:
    commit-title-suffix: " [skip-ci]"
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
            core.info('📥 Checking for Llama Guard 3:1b model...');
            try {
              // Check if model is already available
              const modelsOutput = await exec.getExecOutput('ollama', ['list']);
              const modelExists = modelsOutput.stdout.includes('llama-guard3');
              
              if (modelExists) {
                core.info('✅ Llama Guard 3 model already available');
              } else {
                core.info('📥 Downloading Llama Guard 3:1b model...');
                core.info('This may take several minutes...');
                const startTime = Date.now();
                await exec.exec('ollama', ['pull', 'llama-guard3:1b']);
                
                const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                core.info(`✅ Model downloaded successfully in ${elapsed}s`);
                
                // Verify model is now available
                const verifyOutput = await exec.getExecOutput('ollama', ['list']);
                if (!verifyOutput.stdout.includes('llama-guard3')) {
                  throw new Error('Llama Guard 3 model not found after download');
                }
              }
              core.info('✅ Llama Guard 3 model ready');
            } catch (error) {
              core.setFailed(`Failed to prepare model: ${error instanceof Error ? error.message : String(error)}`);
              throw error;
            }
            
            // ===== SCAN SAFE OUTPUTS =====
            core.info('🔍 Starting Llama Guard 3 threat scan...');
            const scanDir = '/tmp/gh-aw/threat-detection';
            
            let threatsDetected = false;
            const results = [];
            
            // ===== SCAN AGENT OUTPUT ITEMS =====
            const agentOutputPath = path.join(scanDir, 'agent_output.json');
            core.info(`\n📄 Scanning Agent Output Items: ${agentOutputPath}`);
            
            if (fs.existsSync(agentOutputPath)) {
              try {
                const agentOutputContent = fs.readFileSync(agentOutputPath, 'utf8');
                const agentOutput = JSON.parse(agentOutputContent);
                
                if (agentOutput.items && Array.isArray(agentOutput.items)) {
                  core.info(`Found ${agentOutput.items.length} safe output items to scan`);
                  
                  for (let i = 0; i < agentOutput.items.length; i++) {
                    const item = agentOutput.items[i];
                    const itemName = `Agent Output Item #${i + 1} (${item.type || 'unknown'})`;
                    core.info(`\n📋 Scanning ${itemName}...`);
                    
                    try {
                      // Convert item to string for analysis
                      const itemContent = JSON.stringify(item, null, 2);
                      const itemSize = (itemContent.length / 1024).toFixed(2);
                      core.info(`Item size: ${itemSize} KB`);
                      
                      // Truncate very large items
                      const maxChars = 8000;
                      const content = itemContent.length > maxChars 
                        ? itemContent.substring(0, maxChars) + '\n\n[Content truncated for scanning]'
                        : itemContent;
                      
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
                            messages: [{ role: 'user', content: content }],
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
                      
                      // Result must be "safe" or contain "S8" (Intellectual Property, which we allow)
                      const outputLower = output.toLowerCase();
                      const isSafe = outputLower.trim() === 'safe' || outputLower.includes('s8');
                      
                      results.push({
                        file: itemName,
                        path: agentOutputPath,
                        itemIndex: i,
                        itemType: item.type,
                        safe: isSafe,
                        response: output.trim()
                      });
                      
                      if (!isSafe) {
                        threatsDetected = true;
                        core.warning(`⚠️  Potential threat detected in ${itemName}`);
                      }
                    } catch (error) {
                      core.error(`Error scanning ${itemName}: ${error instanceof Error ? error.message : String(error)}`);
                      results.push({
                        file: itemName,
                        path: agentOutputPath,
                        itemIndex: i,
                        safe: false,
                        error: error instanceof Error ? error.message : String(error)
                      });
                      threatsDetected = true;
                    }
                  }
                } else {
                  core.info('No items array found in agent_output.json');
                }
              } catch (error) {
                core.error(`Error reading agent_output.json: ${error instanceof Error ? error.message : String(error)}`);
                results.push({
                  file: 'Agent Output',
                  path: agentOutputPath,
                  safe: false,
                  error: error instanceof Error ? error.message : String(error)
                });
                threatsDetected = true;
              }
            } else {
              core.info(`⚠️  Agent output file not found, skipping: ${agentOutputPath}`);
            }
            
            // ===== SCAN CODE PATCH =====
            const patchPath = path.join(scanDir, 'aw.patch');
            core.info(`\n📄 Scanning Code Patch: ${patchPath}`);
            
            if (fs.existsSync(patchPath)) {
              try {
                const patchContent = fs.readFileSync(patchPath, 'utf8');
                const patchSize = (patchContent.length / 1024).toFixed(2);
                core.info(`Patch size: ${patchSize} KB`);
                
                // Truncate very large patches
                const maxChars = 8000;
                const content = patchContent.length > maxChars 
                  ? patchContent.substring(0, maxChars) + '\n\n[Content truncated for scanning]'
                  : patchContent;
                
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
                      messages: [{ role: 'user', content: content }],
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
                
                // Result must be "safe" or contain "S8" (Intellectual Property, which we allow)
                const outputLower = output.toLowerCase();
                const isSafe = outputLower.trim() === 'safe' || outputLower.includes('s8');
                
                results.push({
                  file: 'Code Patch',
                  path: patchPath,
                  safe: isSafe,
                  response: output.trim()
                });
                
                if (!isSafe) {
                  threatsDetected = true;
                  core.warning(`⚠️  Potential threat detected in Code Patch`);
                }
              } catch (error) {
                core.error(`Error scanning Code Patch: ${error instanceof Error ? error.message : String(error)}`);
                results.push({
                  file: 'Code Patch',
                  path: patchPath,
                  safe: false,
                  error: error instanceof Error ? error.message : String(error)
                });
                threatsDetected = true;
              }
            } else {
              core.info(`⚠️  Patch file not found, skipping: ${patchPath}`);
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
timeout_minutes: 20
network:
  firewall: true
tools:
  bash:
    - "*"
  edit:
imports:
  - shared/changeset-format.md
  - shared/jqschema.md
steps:
  - name: Setup changeset directory
    run: |
      mkdir -p .changeset
      git config user.name "github-actions[bot]"
      git config user.email "github-actions[bot]@users.noreply.github.com"
---

# Changeset Generator

You are the Changeset Generator agent - responsible for automatically creating changeset files when a pull request becomes ready for review.

## Mission

When a pull request is marked as ready for review, analyze the changes and create a properly formatted changeset file that documents the changes according to the changeset specification.

## Current Context

- **Repository**: ${{ github.repository }}
- **Pull Request Number**: ${{ github.event.pull_request.number }}
- **Pull Request Content**: "${{ needs.activation.outputs.text }}"

**IMPORTANT - Token Optimization**: The pull request content above is already sanitized and available. DO NOT use `pull_request_read` or similar GitHub API tools to fetch PR details - you already have everything you need in the context above. Using API tools wastes 40k+ tokens per call.

## Task

Your task is to:

1. **Analyze the Pull Request**: Review the pull request title and description above to understand what has been modified.

2. **Use the repository name as the package identifier** (gh-aw)

3. **Determine the Change Type**:
   - **major**: Major breaking changes (X.0.0) - Very unlikely, probably should be **minor**
   - **minor**: Breaking changes in the CLI (0.X.0) - indicated by "BREAKING CHANGE" or major API changes
   - **patch**: Bug fixes, docs, refactoring, internal changes, tooling, new shared workflows (0.0.X)
   
   **Important**: Internal changes, tooling, and documentation are always "patch" level.

4. **Generate the Changeset File**:
   - Create file in `.changeset/` directory (already created by pre-step)
   - Use format from the changeset format reference above
   - Filename: `<type>-<short-description>.md` (e.g., `patch-fix-bug.md`)

5. **Commit and Push Changes**:
   - Git is already configured by pre-step
   - Add and commit the changeset file using git commands:
     ```bash
     git add .changeset/<filename> && git commit -m "Add changeset"
     ```
   - **CRITICAL**: You MUST call the `push_to_pull_request_branch` tool to push your changes:
     ```javascript
     push_to_pull_request_branch({
       message: "Add changeset for this pull request"
     })
     ```
   - The `branch` parameter is optional - it will automatically detect the current PR branch
   - This tool call is REQUIRED for your changes to be pushed to the pull request
   - **WARNING**: If you don't call this tool, your changeset file will NOT be pushed and the job will be skipped

## Guidelines

- **Be Accurate**: Analyze the PR content carefully to determine the correct change type
- **Be Clear**: The changeset description should clearly explain what changed
- **Be Concise**: Keep descriptions brief but informative
- **Follow Conventions**: Use the exact changeset format specified above
- **Single Package Default**: If unsure about package structure, default to "gh-aw"
- **Smart Naming**: Use descriptive filenames that indicate the change (e.g., `patch-fix-rendering-bug.md`)

