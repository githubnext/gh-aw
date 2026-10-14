//go:build !integration

package workflow

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCommentsMaxLinks(t *testing.T) {
	for _, maxLinks := range []int{1, 50, 75} {
		config := NewCompiler().parseCommentsConfig(map[string]any{
			"add-comment": map[string]any{"max-links": maxLinks},
		})
		require.NotNil(t, config)
		assert.Equal(t, maxLinks, config.MaxLinks)
		handler := commentHandlerRegistry["add_comment"](&SafeOutputsConfig{AddComments: config})
		assert.Equal(t, maxLinks, handler["max_links"])
		assert.Contains(t, addCommentConstraints(config), "Maximum "+strconv.Itoa(maxLinks)+" link(s) in the comment body, excluding generated metadata.")
	}
	config := NewCompiler().parseCommentsConfig(map[string]any{"add-comment": map[string]any{}})
	require.NotNil(t, config)
	handler := commentHandlerRegistry["add_comment"](&SafeOutputsConfig{AddComments: config})
	assert.NotContains(t, handler, "max_links", "omitted limits use the runtime default of 50")
}
