---
name: Changeset Generator
on:
  pull_request:
    types: [ready_for_review]
  workflow_dispatch:
  reaction: "rocket"
if: github.event.pull_request.base.ref == github.event.repository.default_branch
permissions:
  contents: read
  pull-requests: read
  issues: read
engine: copilot
safe-outputs:
  push-to-pull-request-branch:
    commit-title-suffix: " [skip-ci]"
  threat-detection:
    engine:
      id: custom
      steps: []
    steps:
      - name: Ollama LlamaGuard Threat Scan
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
            
            // ===== DOWNLOAD LLAMAGUARD MODEL =====
            core.info('📥 Downloading LlamaGuard-1b model...');
            core.info('This may take several minutes...');
            try {
              const startTime = Date.now();
              await exec.exec('ollama', ['pull', 'llamaguard']);
              
              const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
              core.info(`✅ Model downloaded successfully in ${elapsed}s`);
              
              // Verify model is available
              const modelsOutput = await exec.getExecOutput('ollama', ['list']);
              if (!modelsOutput.stdout.includes('llamaguard')) {
                throw new Error('LlamaGuard model not found after download');
              }
              core.info('✅ LlamaGuard model ready');
            } catch (error) {
              core.setFailed(`Failed to download model: ${error instanceof Error ? error.message : String(error)}`);
              throw error;
            }
            
            // ===== SCAN SAFE OUTPUTS =====
            core.info('🔍 Starting LlamaGuard threat scan...');
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
                const promptFile = `/tmp/gh-aw/ollama-prompt-${Date.now()}.txt`;
                fs.writeFileSync(promptFile, prompt);
                
                core.info('🤖 Running LlamaGuard analysis...');
                const scanStart = Date.now();
                
                let output = '';
                try {
                  const result = await exec.getExecOutput('sh', ['-c', `ollama run llamaguard < ${promptFile}`]);
                  output = result.stdout;
                } catch (error) {
                  core.warning(`LlamaGuard execution error: ${error instanceof Error ? error.message : String(error)}`);
                  output = error.stdout || '';
                }
                
                const scanElapsed = ((Date.now() - scanStart) / 1000).toFixed(1);
                core.info(`Analysis completed in ${scanElapsed}s`);
                fs.unlinkSync(promptFile);
                
                core.info(`\n📊 LlamaGuard Response:\n${output}`);
                
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
            core.info('🔍 LlamaGuard Scan Summary');
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
              core.setFailed('❌ LlamaGuard detected potential security threats in the safe outputs or patches');
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
   - Add and commit the changeset file to the current pull request branch
   - Use the push-to-pull-request-branch tool from safe-outputs to push changes
   - The changeset will be added directly to this pull request

## Guidelines

- **Be Accurate**: Analyze the PR content carefully to determine the correct change type
- **Be Clear**: The changeset description should clearly explain what changed
- **Be Concise**: Keep descriptions brief but informative
- **Follow Conventions**: Use the exact changeset format specified above
- **Single Package Default**: If unsure about package structure, default to "gh-aw"
- **Smart Naming**: Use descriptive filenames that indicate the change (e.g., `patch-fix-rendering-bug.md`)

