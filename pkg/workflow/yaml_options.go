package workflow

import "github.com/goccy/go-yaml"

// DefaultMarshalOptions provides standard YAML formatting options
// used throughout gh-aw for workflow and frontmatter generation.
//
// These options ensure consistent YAML output that follows GitHub Actions
// conventions and best practices:
//   - yaml.Indent(2): Use 2-space indentation (GitHub Actions standard)
//   - yaml.UseLiteralStyleIfMultiline(true): Use literal block scalars (|)
//     for multiline strings to preserve formatting and readability
var DefaultMarshalOptions = []yaml.EncodeOption{
	yaml.Indent(2),                        // Use 2-space indentation
	yaml.UseLiteralStyleIfMultiline(true), // Use literal block scalars for multiline strings
}
