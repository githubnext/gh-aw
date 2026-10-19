package lint

import (
	"maps"

	"github.com/github/gh-aw/pkg/workflow/taint"
)

func graderJavaScriptLocals(node *taint.JavaScriptNode, inherited map[string]struct{}) map[string]struct{} {
	var declarations []*taint.JavaScriptNode
	switch node.Kind {
	case "SourceFile", "Block":
		for _, statement := range node.Statements {
			if statement == nil {
				continue
			}
			switch statement.Kind {
			case "VariableStatement":
				if statement.DeclarationList != nil {
					declarations = append(declarations, statement.DeclarationList.Declarations...)
				}
			case "FunctionDeclaration", "ClassDeclaration":
				declarations = append(declarations, statement)
			}
		}
	case "FunctionDeclaration", "FunctionExpression":
		declarations = append(declarations, node)
		declarations = append(declarations, node.Parameters...)
	case "ArrowFunction", "MethodDeclaration", "GetAccessor", "SetAccessor", "Constructor":
		declarations = append(declarations, node.Parameters...)
	case "ClassDeclaration", "ClassExpression":
		declarations = append(declarations, node)
	case "CatchClause":
		declarations = append(declarations, node.VariableDeclaration)
	case "ForStatement", "ForInStatement", "ForOfStatement":
		if node.Initializer != nil && node.Initializer.Kind == "VariableDeclarationList" {
			declarations = append(declarations, node.Initializer.Declarations...)
		}
	}
	if len(declarations) == 0 {
		return inherited
	}
	locals := maps.Clone(inherited)
	if locals == nil {
		locals = make(map[string]struct{})
	}
	for _, declaration := range declarations {
		if declaration != nil {
			graderJavaScriptBinding(declaration.Name, locals)
		}
	}
	return locals
}

func graderJavaScriptBinding(node *taint.JavaScriptNode, locals map[string]struct{}) {
	if node == nil {
		return
	}
	if node.Kind == "Identifier" {
		locals[node.Text] = struct{}{}
		return
	}
	for _, element := range node.Elements {
		if element != nil {
			graderJavaScriptBinding(element.Name, locals)
		}
	}
}
