package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

const (
	inlineCopilotSDKDriverDir           = ".gh-aw/copilot-sdk"
	inlineCopilotSDKDriverWrapperPath   = inlineCopilotSDKDriverDir + "/inline-driver"
	inlineCopilotSDKDriverNodePath      = inlineCopilotSDKDriverDir + "/inline-driver.cjs"
	inlineCopilotSDKDriverPythonPath    = inlineCopilotSDKDriverDir + "/inline-driver.py"
	inlineCopilotSDKDriverGoPath        = inlineCopilotSDKDriverDir + "/inline-driver.go"
	inlineCopilotSDKDriverGoModPath     = inlineCopilotSDKDriverDir + "/go.mod"
	inlineCopilotSDKDriverJavaPath      = inlineCopilotSDKDriverDir + "/Main.java"
	inlineCopilotSDKDriverJavaPomPath   = inlineCopilotSDKDriverDir + "/pom.xml"
	inlineCopilotSDKDriverJavaClassPath = inlineCopilotSDKDriverDir + "/classpath.txt"
	inlineCopilotSDKDriverGoModuleName  = "gh-aw-inline-copilot-sdk-driver"
	inlineCopilotSDKDriverJavaMainClass = "Main"
)

func (d *InlineEngineDriver) sourcePath() string {
	if d == nil {
		return ""
	}
	switch d.Runtime {
	case "node":
		return inlineCopilotSDKDriverNodePath
	case "python":
		return inlineCopilotSDKDriverPythonPath
	case "go":
		return inlineCopilotSDKDriverGoPath
	case "java":
		return inlineCopilotSDKDriverJavaPath
	default:
		return ""
	}
}

func (d *InlineEngineDriver) wrapperScript() string {
	if d == nil {
		return ""
	}

	sourcePath := d.sourcePath()
	switch d.Runtime {
	case "node":
		return "#!/usr/bin/env bash\nset -euo pipefail\nexec node \"${GITHUB_WORKSPACE}/" + sourcePath + "\" \"$@\"\n"
	case "python":
		return "#!/usr/bin/env bash\nset -euo pipefail\nexec python3 \"${GITHUB_WORKSPACE}/" + sourcePath + "\" \"$@\"\n"
	case "go":
		return "#!/usr/bin/env bash\nset -euo pipefail\ncd \"${GITHUB_WORKSPACE}/" + inlineCopilotSDKDriverDir + "\"\nexec go run \"" + inlineCopilotSDKDriverGoPath[strings.LastIndex(inlineCopilotSDKDriverGoPath, "/")+1:] + "\" \"$@\"\n"
	case "java":
		return "#!/usr/bin/env bash\nset -euo pipefail\nif [ -f \"${GITHUB_WORKSPACE}/" + inlineCopilotSDKDriverJavaClassPath + "\" ]; then\n  CLASSPATH_CONTENT=$(cat \"${GITHUB_WORKSPACE}/" + inlineCopilotSDKDriverJavaClassPath + "\")\n  exec java -cp \"$CLASSPATH_CONTENT\" \"${GITHUB_WORKSPACE}/" + sourcePath + "\" \"$@\"\nfi\nexec java \"${GITHUB_WORKSPACE}/" + sourcePath + "\" \"$@\"\n"
	default:
		return ""
	}
}

func (d *InlineEngineDriver) additionalFiles() map[string]string {
	if d == nil {
		return nil
	}

	switch d.Runtime {
	case "go":
		return map[string]string{
			inlineCopilotSDKDriverGoModPath: fmt.Sprintf("module %s\n\ngo %s\n", inlineCopilotSDKDriverGoModuleName, constants.DefaultGoVersion),
		}
	case "java":
		return map[string]string{
			inlineCopilotSDKDriverJavaPomPath: inlineCopilotSDKDriverPomXML(string(constants.DefaultCopilotSDKVersion)),
		}
	default:
		return nil
	}
}

func inlineCopilotSDKDriverPomXML(version string) string {
	return fmt.Sprintf(`<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.github.gh-aw</groupId>
  <artifactId>inline-copilot-sdk-driver</artifactId>
  <version>1.0.0</version>
  <dependencies>
    <dependency>
      <groupId>com.github</groupId>
      <artifactId>copilot-sdk-java</artifactId>
      <version>%s</version>
    </dependency>
  </dependencies>
</project>
`, version)
}

func copilotSDKInlineDriverRuntimeID(workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.InlineDriver == nil {
		return ""
	}
	return workflowData.EngineConfig.InlineDriver.Runtime
}

func buildInlineCopilotSDKDriverWriteStep(workflowData *WorkflowData) GitHubActionStep {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.InlineDriver == nil {
		return GitHubActionStep{}
	}

	inlineDriver := workflowData.EngineConfig.InlineDriver
	sourcePath := inlineDriver.sourcePath()
	if sourcePath == "" {
		return GitHubActionStep{}
	}

	step := GitHubActionStep{
		"      - name: Write Inline Copilot SDK Driver",
		"        run: |",
		fmt.Sprintf("          mkdir -p \"${GITHUB_WORKSPACE}/%s\"", inlineCopilotSDKDriverDir),
	}

	appendHeredocWrite := func(path, content string, chmod bool) {
		delimiter := GenerateHeredocDelimiterFromContent("INLINE_COPILOT_SDK_DRIVER", content)
		step = append(step, fmt.Sprintf("          cat > \"${GITHUB_WORKSPACE}/%s\" << '%s'", path, delimiter))
		for line := range strings.SplitSeq(content, "\n") {
			step = append(step, "          "+line)
		}
		step = append(step, "          "+delimiter)
		if chmod {
			step = append(step, fmt.Sprintf("          chmod +x \"${GITHUB_WORKSPACE}/%s\"", path))
		}
	}

	appendHeredocWrite(sourcePath, inlineDriver.Source, false)
	for path, content := range inlineDriver.additionalFiles() {
		appendHeredocWrite(path, content, false)
	}
	appendHeredocWrite(inlineCopilotSDKDriverWrapperPath, inlineDriver.wrapperScript(), true)

	return step
}
