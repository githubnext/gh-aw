package taint

import (
	"errors"
	"io"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
)

func javascriptReadSource(files fs.FS, name string) ([]byte, error) {
	file, err := files.Open(name)
	if err != nil {
		return nil, err
	}
	source, readErr := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	closeErr := file.Close()
	if len(source) > 1024*1024 {
		return nil, errors.New("JavaScript source exceeds analysis size limit")
	}
	return source, errors.Join(readErr, closeErr)
}

func javascriptRuntimePath(mappings map[string]string, name string) (string, bool) {
	prefixes := slices.Sorted(maps.Keys(mappings))
	slices.SortFunc(prefixes, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	for _, prefix := range prefixes {
		if suffix, ok := strings.CutPrefix(name, strings.TrimSuffix(prefix, "/")+"/"); ok {
			if !fs.ValidPath(suffix) {
				return "", true
			}
			target := path.Join(mappings[prefix], suffix)
			return target, fs.ValidPath(target)
		}
	}
	return "", false
}
