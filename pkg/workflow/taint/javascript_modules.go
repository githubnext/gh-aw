package taint

import (
	"io/fs"
	"path"
	"strings"
)

func (j *javascript) loadModule(n *JavaScriptNode, name string) jsValue {
	module := jsRefModule(name)
	switch module {
	case "fs", "fs/promises", "path", "http", "https", "child_process", "os", "util", "crypto", "stream", "readline":
		if module == "fs/promises" {
			module = "fs.promises"
		}
		return jsValue{ref: module}
	case "@actions/core":
		return jsValue{ref: "core"}
	}
	if !strings.HasPrefix(name, "./") && !strings.HasPrefix(name, "../") {
		j.gap(n, "external JavaScript module data flow is not modeled")
		return jsValue{}
	}
	target := path.Clean(path.Join(path.Dir(j.file), name))
	if !fs.ValidPath(target) || j.options.Files == nil {
		j.gap(n, "JavaScript repository module source is unavailable")
		return jsValue{}
	}
	if v, ok := j.modules[target]; ok {
		return v
	}
	if len(j.modules) >= 64 {
		j.gap(n, "JavaScript module traversal limit reached")
		return jsValue{}
	}
	source, err := fs.ReadFile(j.options.Files, target)
	if err != nil && path.Ext(target) == "" {
		for _, candidate := range []string{target + ".cjs", target + ".js", path.Join(target, "index.js")} {
			source, err = fs.ReadFile(j.options.Files, candidate)
			if err == nil {
				target = candidate
				break
			}
		}
	}
	if err != nil {
		j.gap(n, "JavaScript repository module source is unavailable")
		return jsValue{}
	}
	tree, err := j.parse(string(source))
	if err != nil {
		j.gap(n, "JavaScript repository module AST is unavailable")
		return jsValue{}
	}
	env, file := j.env, j.file
	j.env, j.file = j.globals(), target
	// A cyclic module gets an explicit gap, not an assumed-safe export.
	j.modules[target] = jsValue{}
	j.eval(tree)
	result := j.env["module"].property("exports")
	j.modules[target] = result
	j.env, j.file = env, file
	return result
}

func (j *javascript) importDeclaration(n *JavaScriptNode) jsValue {
	if n.ModuleSpecifier == nil {
		j.gap(n, "JavaScript import source is unresolved")
		return jsValue{}
	}
	module := j.loadModule(n, n.ModuleSpecifier.Text)
	if n.ImportClause != nil {
		j.bind(n.ImportClause.Name, module)
		bindings := n.ImportClause.NamedBindings
		if bindings != nil {
			if bindings.Kind == "NamespaceImport" {
				j.bind(bindings.Name, module)
			} else {
				for _, binding := range bindings.Elements {
					name := jsName(binding.PropertyName)
					if name == "" {
						name = jsName(binding.Name)
					}
					j.bind(binding.Name, module.property(name))
				}
			}
		}
	}
	return module
}
