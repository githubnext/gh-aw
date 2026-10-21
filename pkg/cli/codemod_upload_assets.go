package cli

import "github.com/github/gh-aw/pkg/workflow/codemod"

// getUploadAssetsCodemod migrates upload-assets to upload-asset.
func getUploadAssetsCodemod() Codemod {
	return Codemod{
		ID:           "upload-assets-to-upload-asset-migration",
		Name:         "Migrate upload-assets to upload-asset",
		Description:  "Replaces deprecated 'safe-outputs.upload-assets' field with 'safe-outputs.upload-asset' (plural to singular)",
		IntroducedIn: "0.3.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Rename([]string{"safe-outputs", "upload-assets"}, "upload-asset")
			})
		},
	}
}
