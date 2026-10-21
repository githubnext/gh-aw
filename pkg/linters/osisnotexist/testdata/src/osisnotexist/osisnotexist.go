package osisnotexist

import (
	"errors"
	"fmt"
	"io/fs"
	operating "os"
)

func predicates(err error) {
	_ = operating.IsNotExist(err)   // want `os.IsNotExist.*errors.Is\(err, fs.ErrNotExist\)`
	_ = operating.IsExist(err)      // want `os.IsExist.*errors.Is\(err, fs.ErrExist\)`
	_ = operating.IsPermission(err) // want `os.IsPermission.*errors.Is\(err, fs.ErrPermission\)`
	_ = operating.IsTimeout(err)    // want `os.IsTimeout.*errors.As.*Timeout\(\).*only covers deadline errors`
	_ = (operating.IsNotExist)(err) // want `os.IsNotExist.*errors.Is`
	operating.IsExist(err)          // want `os.IsExist.*errors.Is`
	predicate := operating.IsNotExist
	_ = predicate(err)                                       // want `os.IsNotExist.*errors.Is`
	_ = operating.IsNotExist(fmt.Errorf("wrapped: %w", err)) // want `os.IsNotExist.*errors.Is`
}

func good(err error) {
	_ = errors.Is(err, fs.ErrNotExist)
	_ = errors.Is(err, fs.ErrExist)
	_ = errors.Is(err, fs.ErrPermission)
	_ = errors.Is(err, operating.ErrDeadlineExceeded)
	var timeout interface{ Timeout() bool }
	_ = errors.As(err, &timeout) && timeout.Timeout()
	_ = operating.IsPathSeparator('/')
}

func suppressed(err error) {
	_ = operating.IsNotExist(err) //nolint:osisnotexist
	//nolint:osisnotexist
	_ = operating.IsTimeout(err)
}

type fakeOS struct{}

func (fakeOS) IsNotExist(error) bool { return false }

func shadowed(err error) {
	os := fakeOS{}
	_ = os.IsNotExist(err)
	predicate := operating.IsExist
	predicate = func(error) bool { return false }
	_ = predicate(err)
}
