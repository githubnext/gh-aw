package pipeline

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Expression is shared by conditions, environment values, and action inputs.
// Only Literal is emitted without an Actions expression wrapper in value fields.
type Expression interface {
	pipelineExpression()
}

type Literal string
type Boolean bool

// Context is the escape hatch for expressions not yet modeled by the AST.
// Its contents exclude the surrounding ${{ }} delimiters.
type Context string
type SecretRef string
type VariableRef string

// OutputRef identifies either a declared step output or, when Step is empty,
// an explicitly exposed job output.
type OutputRef struct {
	Job  JobID
	Step StepID
	Name OutputName
}

type Call struct {
	Name string
	Args []Expression
}

type Binary struct {
	Left     Expression
	Operator string
	Right    Expression
}

type Not struct{ Value Expression }

func (Literal) pipelineExpression()     {}
func (Boolean) pipelineExpression()     {}
func (Context) pipelineExpression()     {}
func (SecretRef) pipelineExpression()   {}
func (VariableRef) pipelineExpression() {}
func (OutputRef) pipelineExpression()   {}
func (Call) pipelineExpression()        {}
func (Binary) pipelineExpression()      {}
func (Not) pipelineExpression()         {}

func Secret(name string) Expression {
	return SecretRef(name)
}

func Variable(name string) Expression {
	return VariableRef(name)
}

// Coalesce uses Actions' truthiness semantics (empty, false, zero, and null
// select the fallback), rather than Azure's null/empty-only coalesce semantics.
func Coalesce(value, fallback Expression) Expression {
	return Binary{Left: value, Operator: "||", Right: fallback}
}

func Concat(values ...Expression) Expression {
	var format strings.Builder
	for i := range values {
		format.WriteString("{" + strconv.Itoa(i) + "}")
	}
	return Call{Name: "format", Args: append([]Expression{Literal(format.String())}, values...)}
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

type expressionContext struct {
	resolve func(OutputRef) (string, error)
}

func (ctx expressionContext) render(expr Expression) (string, error) {
	switch value := expr.(type) {
	case Literal:
		return quote(string(value)), nil
	case Boolean:
		return strconv.FormatBool(bool(value)), nil
	case SecretRef:
		if err := validateIdentifier("secret name", string(value)); err != nil {
			return "", err
		}
		return "secrets." + string(value), nil
	case VariableRef:
		if err := validateIdentifier("variable name", string(value)); err != nil {
			return "", err
		}
		return "vars." + string(value), nil
	case Context:
		text := string(value)
		if strings.TrimSpace(text) == "" || strings.ContainsAny(text, "\r\n") || strings.Contains(text, "${{") || strings.Contains(text, "}}") {
			return "", errors.New("raw pipeline expression must be nonempty, single-line, and exclude expression delimiters")
		}
		return text, nil
	case OutputRef:
		return ctx.resolve(value)
	case Call:
		if err := validateIdentifier("expression function", value.Name); err != nil {
			return "", err
		}
		args := make([]string, 0, len(value.Args))
		for _, arg := range value.Args {
			rendered, err := ctx.render(arg)
			if err != nil {
				return "", err
			}
			args = append(args, rendered)
		}
		return value.Name + "(" + strings.Join(args, ", ") + ")", nil
	case Binary:
		switch value.Operator {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
		default:
			return "", fmt.Errorf("unsupported pipeline expression operator %q", value.Operator)
		}
		left, err := ctx.render(value.Left)
		if err != nil {
			return "", err
		}
		right, err := ctx.render(value.Right)
		if err != nil {
			return "", err
		}
		return "(" + left + " " + value.Operator + " " + right + ")", nil
	case Not:
		inner, err := ctx.render(value.Value)
		if err != nil {
			return "", err
		}
		return "!(" + inner + ")", nil
	default:
		return "", errors.New("pipeline expression cannot be nil or an unsupported type")
	}
}

func (ctx expressionContext) value(expr Expression) (string, error) {
	if value, ok := expr.(Literal); ok {
		return string(value), nil
	}
	rendered, err := ctx.render(expr)
	if err != nil {
		return "", err
	}
	return "${{ " + rendered + " }}", nil
}
