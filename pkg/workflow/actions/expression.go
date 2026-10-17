package actions

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Expression is shared by conditions, environment values, and action inputs.
// Environment values wrap everything except Literal in an Actions expression.
// Inputs also preserve Boolean and Number literals as native YAML scalars.
type Expression interface {
	actionsExpression()
}

type Literal string
type Boolean bool
type Number float64

// Context is the escape hatch for expressions not yet modeled by the AST.
// Its contents exclude the surrounding ${{ }} delimiters.
type Context string
type parsedExpression string
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

func (Literal) actionsExpression()          {}
func (Boolean) actionsExpression()          {}
func (Number) actionsExpression()           {}
func (Context) actionsExpression()          {}
func (parsedExpression) actionsExpression() {}
func (SecretRef) actionsExpression()        {}
func (VariableRef) actionsExpression()      {}
func (OutputRef) actionsExpression()        {}
func (Call) actionsExpression()             {}
func (Binary) actionsExpression()           {}
func (Not) actionsExpression()              {}

func Secret(name string) Expression {
	return SecretRef(name)
}

func Variable(name string) Expression {
	return VariableRef(name)
}

// Or selects the fallback for falsy values using the Actions || operator.
func Or(value, fallback Expression) Expression {
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
	case Number:
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return "", errors.New("expression numbers must be finite")
		}
		return strconv.FormatFloat(float64(value), 'g', -1, 64), nil
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
			return "", errors.New("raw workflow expression must be nonempty, single-line, and exclude expression delimiters")
		}
		return text, nil
	case parsedExpression:
		text := string(value)
		if strings.TrimSpace(text) == "" || strings.Contains(text, "${{") || strings.Contains(text, "}}") {
			return "", errors.New("parsed Actions expression must be nonempty and exclude expression delimiters")
		}
		return text, nil
	case OutputRef:
		return ctx.resolve(value)
	case Call:
		return ctx.renderCall(value)
	case Binary:
		return ctx.renderBinary(value)
	case Not:
		inner, err := ctx.render(value.Value)
		if err != nil {
			return "", err
		}
		return "!(" + inner + ")", nil
	default:
		return "", errors.New("workflow expression cannot be nil or an unsupported type")
	}
}

func (ctx expressionContext) renderBinary(value Binary) (string, error) {
	switch value.Operator {
	case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
	default:
		return "", fmt.Errorf("unsupported workflow expression operator %q", value.Operator)
	}
	left, err := ctx.render(value.Left)
	if err != nil {
		return "", err
	}
	right, err := ctx.render(value.Right)
	if err != nil {
		return "", err
	}
	if _, ok := value.Left.(Context); ok {
		left = "(" + left + ")"
	}
	if _, ok := value.Right.(Context); ok {
		right = "(" + right + ")"
	}
	return "(" + left + " " + value.Operator + " " + right + ")", nil
}

func (ctx expressionContext) renderCall(value Call) (string, error) {
	switch strings.ToLower(value.Name) {
	case "contains", "startswith", "endswith", "format", "join", "tojson", "fromjson",
		"hashfiles", "success", "always", "cancelled", "failure":
	default:
		return "", fmt.Errorf("unsupported GitHub Actions expression function %q", value.Name)
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
