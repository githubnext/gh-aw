package taint

func jsLocalNames(fn *JavaScriptNode) map[string]struct{} {
	result := make(map[string]struct{})
	var binding func(*JavaScriptNode)
	binding = func(n *JavaScriptNode) {
		if n == nil {
			return
		}
		if n.Kind == "Identifier" {
			result[n.Text] = struct{}{}
			return
		}
		for _, element := range n.Elements {
			binding(element.Name)
		}
	}
	for _, parameter := range fn.Parameters {
		binding(parameter.Name)
	}
	var walk func(*JavaScriptNode)
	walk = func(n *JavaScriptNode) {
		if n == nil {
			return
		}
		switch n.Kind {
		case "FunctionDeclaration":
			binding(n.Name)
			return
		case "FunctionExpression", "ArrowFunction", "MethodDeclaration":
			return
		case "VariableDeclaration":
			binding(n.Name)
		}
		for _, child := range []*JavaScriptNode{n.Body, n.Block, n.DeclarationList, n.Initializer, n.Expression, n.ThenStatement, n.ElseStatement, n.TryBlock, n.CatchClause, n.FinallyBlock} {
			walk(child)
		}
		for _, list := range [][]*JavaScriptNode{n.Statements, n.Declarations, n.Clauses, n.Other} {
			for _, child := range list {
				walk(child)
			}
		}
	}
	walk(fn.Body)
	return result
}
