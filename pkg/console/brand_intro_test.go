//go:build !integration

package console

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatBrandIntroFrame(t *testing.T) {
	t.Setenv("TERM", "dumb")

	lines := formatBrandIntroFrame(len(brandLogoFrames)-1, "Welcome", "Add a workflow")

	require.Len(t, lines, 8)
	assert.Contains(t, lines[0], "+-----+")
	assert.Contains(t, lines[1], "+++", "final frame should show the logo sparkle")
	assert.Contains(t, lines[1], "Welcome")
	assert.Contains(t, lines[3], "Add a workflow")
	assert.Contains(t, strings.Join(lines, "\n"), "+----+", "logo should show linked workflow nodes")
}

func TestBrandLogoAnimationFramesKeepStableHeight(t *testing.T) {
	for _, frame := range brandLogoFrames {
		assert.Len(t, frame, len(brandLogoFrames[0]))
	}
	assert.NotEqual(t, brandLogoFrames[0], brandLogoFrames[len(brandLogoFrames)-1])
}
