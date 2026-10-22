// Package data holds data files bundled with gh-aw that more than one package reads.
package data

import _ "embed"

// ModelsJSON is gh-aw's bundled model catalogue (models.json), keyed by provider.
//
//go:embed models.json
var ModelsJSON []byte
