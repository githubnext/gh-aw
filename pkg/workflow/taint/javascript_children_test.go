package taint

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJavaScriptNodeChildren(t *testing.T) {
	node := &JavaScriptNode{}
	value := reflect.ValueOf(node).Elem()
	var expected []*JavaScriptNode
	for i := range value.NumField() {
		field := value.Field(i)
		child := &JavaScriptNode{Kind: value.Type().Field(i).Name}
		switch field.Type() {
		case reflect.TypeFor[*JavaScriptNode]():
			field.Set(reflect.ValueOf(child))
		case reflect.TypeFor[[]*JavaScriptNode]():
			field.Set(reflect.ValueOf([]*JavaScriptNode{child}))
		default:
			continue
		}
		expected = append(expected, child)
	}
	assert.ElementsMatch(t, expected, node.Children(), "every native projection field must participate in policy traversal")
	assert.Empty(t, (*JavaScriptNode)(nil).Children())
	assert.Equal(t, expected, node.Children(), "child order must remain deterministic")
}
