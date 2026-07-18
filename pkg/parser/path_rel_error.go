package parser

// pathRelError is returned when filepath.Rel fails during a security path check.
// Its Error() method omits the internal absolute paths from the filepath.Rel
// diagnostic so they do not leak into user-facing error messages, while Unwrap()
// preserves the upstream error for errors.Is / errors.As traversal.
type pathRelError struct {
	msg string
	err error
}

func (e *pathRelError) Error() string { return e.msg }
func (e *pathRelError) Unwrap() error { return e.err }
