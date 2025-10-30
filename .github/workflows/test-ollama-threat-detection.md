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
      - name: Install Ollama
        id: install-ollama
        uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1
        with:
          script: |
            const fs = require('fs');
            const { execSync } = require('child_process');
            
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
      
      - name: Start Ollama service
        id: start-ollama
        run: |
          set -e
          
          echo "🚀 Starting Ollama service..."
          mkdir -p /tmp/gh-aw/ollama-logs
          
          # Start Ollama service in background
          nohup ollama serve > /tmp/gh-aw/ollama-logs/ollama-serve.log 2> /tmp/gh-aw/ollama-logs/ollama-serve-error.log &
          OLLAMA_PID=$!
          echo "Ollama service started with PID: $OLLAMA_PID"
          
          # Wait for service to be ready
          echo "⏳ Waiting for Ollama service to be ready..."
          for i in {1..30}; do
            if curl -f http://localhost:11434/api/version 2>/dev/null; then
              echo "✅ Ollama service is ready"
              break
            fi
            if [ $i -eq 30 ]; then
              echo "❌ Ollama service did not become ready in time"
              cat /tmp/gh-aw/ollama-logs/ollama-serve-error.log
              exit 1
            fi
            sleep 1
          done
      
      - name: Download LlamaGuard model
        id: download-model
        run: |
          set -e
          
          echo "📥 Downloading LlamaGuard-1b model..."
          echo "This may take several minutes..."
          
          START_TIME=$(date +%s)
          
          # Pull the llamaguard model with timeout
          timeout 600 ollama pull llamaguard || {
            echo "❌ Failed to download model within 10 minutes"
            exit 1
          }
          
          END_TIME=$(date +%s)
          ELAPSED=$((END_TIME - START_TIME))
          echo "✅ Model downloaded successfully in ${ELAPSED}s"
          
          # Verify model is available
          echo "✅ Verifying model availability..."
          ollama list
          
          if ! ollama list | grep -q "llamaguard"; then
            echo "❌ LlamaGuard model not found after download"
            exit 1
          fi
          
          echo "✅ LlamaGuard model ready"
      
      - name: Scan safe outputs with LlamaGuard
        id: scan-outputs
        uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1
        with:
          script: |
            const fs = require('fs');
            const { execSync } = require('child_process');
            const path = require('path');
            
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
                
                // Truncate very large files to avoid overwhelming the model
                const maxChars = 8000;
                const content = fileContent.length > maxChars 
                  ? fileContent.substring(0, maxChars) + '\n\n[Content truncated for scanning]'
                  : fileContent;
                
                // Create prompt for LlamaGuard
                const prompt = `Analyze the following content for security threats, malicious code, or harmful patterns:\n\n${content}`;
                
                // Create temporary file for prompt
                const promptFile = `/tmp/gh-aw/ollama-prompt-${Date.now()}.txt`;
                fs.writeFileSync(promptFile, prompt);
                
                core.info('🤖 Running LlamaGuard analysis...');
                const startTime = Date.now();
                
                // Run Ollama with llamaguard
                let output = '';
                try {
                  output = execSync(`ollama run llamaguard < ${promptFile}`, {
                    encoding: 'utf8',
                    timeout: 120000, // 2 minute timeout
                    maxBuffer: 1024 * 1024 * 10 // 10MB buffer
                  });
                } catch (error) {
                  core.warning(`LlamaGuard execution error: ${error instanceof Error ? error.message : String(error)}`);
                  output = error.stdout || '';
                }
                
                const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                core.info(`Analysis completed in ${elapsed}s`);
                
                // Clean up prompt file
                fs.unlinkSync(promptFile);
                
                // Parse LlamaGuard output
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
            
            // Write results to file
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
