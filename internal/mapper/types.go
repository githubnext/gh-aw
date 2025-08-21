package mapper

// Span describes a location in the source YAML file.
type Span struct {
	StartLine  int // 1-based
	StartCol   int // 1-based
	EndLine    int
	EndCol     int
	Confidence float64 // 0.0 - 1.0
	Reason     string  // short reason why this span was chosen
}

// ErrorMeta contains validator-provided metadata about the error.
type ErrorMeta struct {
	Kind          string // "type", "required", "additionalProperties", "oneOf", ...
	Property      string // property name when present (e.g., for additionalProperties or required)
	SchemaSnippet string // optional schema fragment or description to help keyword search
}
