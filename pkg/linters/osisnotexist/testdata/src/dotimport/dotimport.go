package dotimport

import . "os"

func predicates(err error) {
	_ = IsNotExist(err)   // want `os.IsNotExist.*errors.Is`
	_ = (IsExist)(err)    // want `os.IsExist.*errors.Is`
	_ = IsPermission(err) // want `os.IsPermission.*errors.Is`
	_ = IsTimeout(err)    // want `os.IsTimeout.*errors.As`
}
