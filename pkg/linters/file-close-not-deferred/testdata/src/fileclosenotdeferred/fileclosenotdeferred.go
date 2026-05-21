package fileclosenotdeferred

import "os"

// flagged: file.Close() not deferred
func ReadFileManualClose() error {
	file, err := os.Open("test.txt") // want `file Close\(\) should be deferred immediately after successful open to prevent resource leaks`
	if err != nil {
		return err
	}
	// ... code that might return early ...
	file.Close()
	return nil
}

// not flagged: defer used correctly
func ReadFileDeferClose() error {
	file, err := os.Open("test.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	// ... rest of code ...
	return nil
}

// flagged: os.Create with manual close
func CreateFileManualClose() error {
	f, err := os.Create("output.txt") // want `file Close\(\) should be deferred immediately after successful open to prevent resource leaks`
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

// flagged: os.Open with Close() assigned to error variable
func ReadFileCloseWithErrAssign() error {
	file, err := os.Open("test.txt") // want `file Close\(\) should be deferred immediately after successful open to prevent resource leaks`
	if err != nil {
		return err
	}
	// ... code ...
	closeErr := file.Close()
	if closeErr != nil {
		return closeErr
	}
	return nil
}

// not flagged: blank identifier
func IgnoredFile() error {
	_, err := os.Open("test.txt")
	return err
}
