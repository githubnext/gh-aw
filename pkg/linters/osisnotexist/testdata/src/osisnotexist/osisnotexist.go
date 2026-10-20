// Package osisnotexist is the test fixture for the osisnotexist analyzer.
package osisnotexist

import (
	"errors"
	"io/fs"
	"os"
)

func bad(err error) bool {
	return os.IsNotExist(err) // want `os\.IsNotExist does not unwrap wrapped errors; use errors\.Is\(err, fs\.ErrNotExist\) instead`
}

func badExist(err error) bool {
	return os.IsExist(err) // want `os\.IsExist does not unwrap wrapped errors; use errors\.Is\(err, fs\.ErrExist\) instead`
}

func badPermission(err error) bool {
	return os.IsPermission(err) // want `os\.IsPermission does not unwrap wrapped errors; use errors\.Is\(err, fs\.ErrPermission\) instead`
}

func badTimeout(err error) bool {
	return os.IsTimeout(err) // want `os\.IsTimeout does not unwrap wrapped errors; use errors\.Is\(err, os\.ErrDeadlineExceeded\) instead`
}

func good(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func suppressed(err error) bool {
	return os.IsNotExist(err) //nolint:osisnotexist
}
