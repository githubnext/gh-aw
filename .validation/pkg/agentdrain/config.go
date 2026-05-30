package agentdrain

// DefaultConfig returns a Config pre-loaded with sensible production defaults.
func DefaultConfig() Config {
	return Config{
		Depth:                4,
		SimThreshold:         0.4,
		MaxChildren:          100,
		ParamToken:           "<*>",
		RareClusterThreshold: 2,
		ExcludeFields:        []string{"session_id", "trace_id", "span_id", "timestamp"},
		MaskRules: []MaskRule{
			{
				Name:        "uuid",
				Pattern:     `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
				Replacement: "<UUID>",
			},
			{
				Name:        "session_id",
				Pattern:     `session=[a-z0-9]+`,
				Replacement: "session=<*>",
			},
			{
				Name:        "number_value",
				Pattern:     `=\d+`,
				Replacement: "=<NUM>",
			},
			{
				Name:        "url",
				Pattern:     `https?://[^\s]+`,
				Replacement: "<URL>",
			},
			{
				Name:        "quoted_string",
				Pattern:     `"[^"]*"`,
				Replacement: `"<*>"`,
			},
			{
				Name:        "timestamp",
				Pattern:     `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?`,
				Replacement: "<TIMESTAMP>",
			},
		},
	}
}
