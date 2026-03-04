package fileutil

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"

	"github.com/github/gh-aw/pkg/logger"
)

var tarLog = logger.New("fileutil:tar")

// ExtractFileFromTar extracts a single file from a tar archive.
// Uses Go's standard archive/tar for cross-platform compatibility instead of
// spawning an external tar process which may not be available on all platforms.
func ExtractFileFromTar(data []byte, path string) ([]byte, error) {
	tarLog.Printf("Extracting file from tar archive: target=%s, archive_size=%d bytes", path, len(data))
	tr := tar.NewReader(bytes.NewReader(data))
	entriesScanned := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			tarLog.Printf("File not found in tar archive after scanning %d entries: %s", entriesScanned, path)
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar archive: %w", err)
		}
		entriesScanned++
		if header.Name == path {
			tarLog.Printf("Found file in tar archive after scanning %d entries: %s", entriesScanned, path)
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("file %q not found in archive", path)
}
