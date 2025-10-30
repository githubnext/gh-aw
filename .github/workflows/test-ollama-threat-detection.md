---
on:
  workflow_dispatch:

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: copilot

safe-outputs:
  create-issue:
    title-prefix: "[test] "
    labels: [test]
    max: 1
  threat-detection:
    steps:
      - name: Ollama LlamaGuard Threat Scan
        id: ollama-scan
        uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1
        with:
          script: |
            const fs = require('fs');
            const { execSync, spawn } = require('child_process');
            const path = require('path');
            
            // ===== INSTALL OLLAMA =====
            core.info('🚀 Starting Ollama installation...');
            try {
              core.info('📥 Downloading Ollama installer...');
              execSync('curl -fsSL https://ollama.com/install.sh -o /tmp/install-ollama.sh', {
                stdio: ['ignore', process.stdout, process.stderr]
              });
              
              core.info('📦 Installing Ollama...');
              execSync('sh /tmp/install-ollama.sh', {
                stdio: ['ignore', process.stdout, process.stderr]
              });
              
              core.info('✅ Verifying Ollama installation...');
              const version = execSync('ollama --version', { encoding: 'utf8' });
              core.info(`Ollama version: ${version.trim()}`);
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
            
            const ollamaProcess = spawn('ollama', ['serve'], {
              detached: true,
              stdio: ['ignore',
                fs.openSync(`${logDir}/ollama-serve.log`, 'w'),
                fs.openSync(`${logDir}/ollama-serve-error.log`, 'w')
              ]
            });
            ollamaProcess.unref();
            core.info(`Ollama service started with PID: ${ollamaProcess.pid}`);
            
            // Wait for service to be ready
            core.info('⏳ Waiting for Ollama service to be ready...');
            let retries = 30;
            while (retries > 0) {
              try {
                execSync('curl -f http://localhost:11434/api/version', {
                  stdio: 'pipe',
                  timeout: 2000
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
              execSync('ollama pull llamaguard', {
                stdio: ['ignore', process.stdout, process.stderr],
                timeout: 600000 // 10 minute timeout
              });
              
              const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
              core.info(`✅ Model downloaded successfully in ${elapsed}s`);
              
              // Verify model is available
              const models = execSync('ollama list', { encoding: 'utf8' });
              if (!models.includes('llamaguard')) {
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
                  output = execSync(`ollama run llamaguard < ${promptFile}`, {
                    encoding: 'utf8',
                    timeout: 120000,
                    maxBuffer: 1024 * 1024 * 10
                  });
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
---

# Test Ollama Threat Scanning

This is a test workflow to verify the Ollama LlamaGuard threat scanning configuration.

Create a simple test issue to demonstrate the workflow executes successfully with threat detection enabled.
