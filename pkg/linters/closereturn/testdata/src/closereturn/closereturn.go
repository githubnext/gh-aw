package closereturn

import "os"

func bad(path string) error {
	f, err := os.Open(path) // want `resource Close\(\) should be deferred immediately after successful open before any early-return error guard`
	if err != nil {
		return err
	}
	return nil
}

func good(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func goodImmediateDeferAfterGuard(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, _ = f.Stat()
	return nil
}

func goodBlankErr(path string) error {
	f, _ := os.Open(path)
	defer f.Close()
	return nil
}

func suppressed(path string) error {
	//nolint:closereturn
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return nil
}
