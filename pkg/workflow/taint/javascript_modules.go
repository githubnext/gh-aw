package taint

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"
)

func (j *javascript) loadModule(n *JavaScriptNode, name string) jsValue { //nolint:largefunc // Keep source resolution and module initialization at one boundary.
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
	target, mapped := javascriptRuntimePath(j.optionsRuntimePaths(), name)
	if !mapped && !strings.HasPrefix(name, "./") && !strings.HasPrefix(name, "../") {
		j.gap(n, "external JavaScript module data flow is not modeled")
		return jsValue{}
	}
	if !mapped {
		target = path.Clean(path.Join(path.Dir(j.file), name))
	}
	if v, ok := j.modules[target]; ok {
		return v
	}
	if !fs.ValidPath(target) || j.options.Files == nil {
		j.gap(n, "JavaScript repository module source is unavailable")
		return jsValue{}
	}
	if len(j.modules) >= 64 {
		j.gap(n, "JavaScript module traversal limit reached")
		return jsValue{}
	}
	source, err := javascriptReadSource(j.options.Files, target)
	if err != nil && path.Ext(target) == "" {
		for _, candidate := range []string{target + ".cjs", target + ".js", path.Join(target, "index.js")} {
			source, err = javascriptReadSource(j.options.Files, candidate)
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
	if _, active := j.loading[target]; active {
		j.gap(n, "cyclic JavaScript module initialization is not modeled")
		return jsValue{}
	}
	if v, ok := j.modules[target]; ok {
		return v
	}
	if j.options.Mode == JavaScriptFast && mapped {
		return j.summaryModule(n, target, source)
	}
	tree, err := j.parse(string(source))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			j.failure = err
			return jsValue{}
		}
		j.gap(n, "JavaScript repository module AST is unavailable")
		return jsValue{}
	}
	env, file := j.env, j.file
	j.file = target
	j.env = j.globals()
	j.loading[target] = struct{}{}
	j.eval(tree)
	delete(j.loading, target)
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
