//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildOrgXMLMarker(t *testing.T) {
	t.Run("with tag", func(t *testing.T) {
		marker := buildOrgXMLMarker(ghawUpgradeMarkerPrefix, "v1.2.3")
		assert.Equal(t, "<!-- gh-aw-upgrade: v1.2.3 -->", marker)
	})

	t.Run("without tag uses latest placeholder", func(t *testing.T) {
		marker := buildOrgXMLMarker(ghawUpgradeMarkerPrefix, "")
		assert.Equal(t, "<!-- gh-aw-upgrade: latest -->", marker)
	})

	t.Run("update prefix", func(t *testing.T) {
		marker := buildOrgXMLMarker(ghawUpdateMarkerPrefix, "v2.0.0")
		assert.Equal(t, "<!-- gh-aw-update: v2.0.0 -->", marker)
	})
}

func TestMarkerPrefixesAreDistinct(t *testing.T) {
	assert.NotEqual(t, ghawUpgradeMarkerPrefix, ghawUpdateMarkerPrefix)
}
