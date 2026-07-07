package fileclosenotdeferred

import xos "os"

// flagged: aliased os import still resolves to os package
func ReadFileManualCloseAliased() error {
	file, err := xos.Open("test.txt") // want `file Close\(\) should be deferred immediately after successful open to prevent resource leaks`
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// not flagged: defer used correctly with aliased import
func ReadFileDeferCloseAliased() error {
	file, err := xos.Open("test.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}
