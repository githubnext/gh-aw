package fileutil

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
)

// ExtractFileFromTar extracts a single file from a tar archive.
// Uses Go's standard archive/tar for cross-platform compatibility instead of
// spawning an external tar process which may not be available on all platforms.
func ExtractFileFromTar(data []byte, path string) ([]byte, error) {
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar archive: %w", err)
		}
		if header.Name == path {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("file %q not found in archive", path)
}
