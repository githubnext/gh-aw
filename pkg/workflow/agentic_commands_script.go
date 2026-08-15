package workflow

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	setupjs "github.com/github/gh-aw/actions/setup/js"
)

var (
	agenticCommandsLocalRequirePattern = regexp.MustCompile(`require\(\s*["'](\.{1,2}/[^"']+)["']\s*\)`)
	agenticCommandsScriptOnce          sync.Once
	agenticCommandsScript              string
	agenticCommandsScriptErr           error
)

func getAgenticCommandsScript() (string, error) {
	agenticCommandsScriptOnce.Do(func() {
		agenticCommandsScript, agenticCommandsScriptErr = bundleAgenticCommandsScript(
			setupjs.AgenticCommandsScripts,
			"route_slash_command.cjs",
		)
	})
	return agenticCommandsScript, agenticCommandsScriptErr
}

func bundleAgenticCommandsScript(scripts fs.FS, entrypoint string) (string, error) {
	modules := make(map[string]string)
	if err := collectAgenticCommandsModules(scripts, entrypoint, modules); err != nil {
		return "", err
	}

	moduleNames := make([]string, 0, len(modules))
	for name := range modules {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)

	var script strings.Builder
	script.WriteString("const __ghAwNativeRequire = require;\n")
	script.WriteString("const __ghAwPath = __ghAwNativeRequire('node:path').posix;\n")
	script.WriteString("const __ghAwModules = {\n")
	for _, name := range moduleNames {
		fmt.Fprintf(&script, "  %s: (module, exports, require) => {\n", strconv.Quote(name))
		for line := range strings.SplitSeq(modules[name], "\n") {
			script.WriteString("    ")
			script.WriteString(strings.TrimRight(line, " \t"))
			script.WriteByte('\n')
		}
		script.WriteString("  },\n")
	}
	script.WriteString(`};
const __ghAwModuleCache = Object.create(null);
function __ghAwRequire(request, parentDirectory = "") {
  if (!request.startsWith(".")) {
    return __ghAwNativeRequire(request);
  }
  let moduleName = __ghAwPath.normalize(__ghAwPath.join(parentDirectory, request));
  if (!moduleName.endsWith(".cjs") && !moduleName.endsWith(".js")) {
    moduleName += ".cjs";
  }
  if (!Object.hasOwn(__ghAwModules, moduleName)) {
    throw new Error(` + "`Agentic commands module not found: ${moduleName}`" + `);
  }
  if (Object.hasOwn(__ghAwModuleCache, moduleName)) {
    return __ghAwModuleCache[moduleName].exports;
  }
  const module = { exports: {} };
  __ghAwModuleCache[moduleName] = module;
  const localRequire = dependency => __ghAwRequire(dependency, __ghAwPath.dirname(moduleName));
  __ghAwModules[moduleName](module, module.exports, localRequire);
  return module.exports;
}
`)
	fmt.Fprintf(
		&script,
		"const { main: __ghAwMain } = __ghAwRequire(%s);\n",
		strconv.Quote("./"+path.Clean(strings.TrimPrefix(entrypoint, "./"))),
	)
	script.WriteString(`
await __ghAwMain();
`)
	return trimAgenticCommandsScript(removeJavaScriptComments(script.String())), nil
}

func trimAgenticCommandsScript(script string) string {
	var trimmed strings.Builder
	for line := range strings.SplitSeq(script, "\n") {
		trimmed.WriteString(strings.TrimRight(line, " \t"))
		trimmed.WriteByte('\n')
	}
	return trimmed.String()
}

func collectAgenticCommandsModules(scripts fs.FS, moduleName string, modules map[string]string) error {
	moduleName = path.Clean(strings.TrimPrefix(moduleName, "./"))
	if moduleName == "." || moduleName == ".." || strings.HasPrefix(moduleName, "../") {
		return fmt.Errorf("invalid agentic commands module path: %q", moduleName)
	}
	if _, exists := modules[moduleName]; exists {
		return nil
	}

	content, err := fs.ReadFile(scripts, moduleName)
	if err != nil {
		return fmt.Errorf("failed to read agentic commands module %q: %w", moduleName, err)
	}
	modules[moduleName] = string(content)

	for _, match := range agenticCommandsLocalRequirePattern.FindAllStringSubmatch(string(content), -1) {
		dependency := path.Clean(path.Join(path.Dir(moduleName), match[1]))
		if path.Ext(dependency) == "" {
			dependency += ".cjs"
		}
		if err := collectAgenticCommandsModules(scripts, dependency, modules); err != nil {
			return err
		}
	}
	return nil
}
