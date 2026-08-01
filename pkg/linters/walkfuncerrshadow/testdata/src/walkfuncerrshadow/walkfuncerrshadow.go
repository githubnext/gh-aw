package walkfuncerrshadow

import (
	"io/fs"
	"os"
	"path/filepath"
)

func BadWalk(root string) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error { // want `callback parameter err shadows outer err assigned from filepath\.Walk; rename the callback parameter \(for example walkErr\)`
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func BadWalkDir(root string) error {
	var err error
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { // want `callback parameter err shadows outer err assigned from filepath\.WalkDir; rename the callback parameter \(for example walkErr\)`
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func BadWalkDirShort(root string) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { // want `callback parameter err shadows outer err assigned from filepath\.WalkDir; rename the callback parameter \(for example walkErr\)`
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GoodDistinctParam(root string) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		return nil
	})
	return err
}

func GoodDistinctOuter(root string) error {
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return nil
	})
	return walkErr
}

func GoodOtherWalker(root string) error {
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GoodNolint(root string) error {
	//nolint:walkfuncerrshadow
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func GoodNolintMultiLine(root string) error {
	//nolint:walkfuncerrshadow
	err := filepath.Walk(root, func(
		path string,
		info os.FileInfo,
		err error,
	) error {
		if err != nil {
			return err
		}
		return nil
	})
	return err
}
