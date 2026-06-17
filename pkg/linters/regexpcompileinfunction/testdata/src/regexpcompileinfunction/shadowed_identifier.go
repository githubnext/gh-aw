package regexpcompileinfunction

// customRegexp is an unrelated type that happens to have Compile/MustCompile methods.
type customRegexp struct{}

func (customRegexp) Compile(_ string) (*customRegexp, error) { return &customRegexp{}, nil }
func (customRegexp) MustCompile(_ string) *customRegexp      { return &customRegexp{} }

// not flagged: local variable named "regexp" is not the stdlib regexp package.
func GoodShadowedRegexpIdentifier() bool {
	regexp := customRegexp{}
	_ = regexp.MustCompile(`^[a-z]+$`)
	return true
}
