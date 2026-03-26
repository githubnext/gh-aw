package fileutil

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var tarLog = logger.New("fileutil:tar")

// ExtractFileFromTar extracts a single file from a tar archive.
// Uses Go's standard archive/tar for cross-platform compatibility instead of
// spawning an external tar process which may not be available on all platforms.
//
// path must not be absolute and must not contain ".." components; an error is
// returned for any entry whose name matches these criteria to guard against
// path-traversal payloads embedded in a tar archive.
func ExtractFileFromTar(data []byte, path string) ([]byte, error) {
	// Reject obviously unsafe search targets before opening the archive.
	if filepath.IsAbs(path) || strings.Contains(path, "..") {
		return nil, fmt.Errorf("unsafe path requested from tar archive: %q", path)
	}

	tarLog.Printf("Extracting file from tar archive: target=%s, archive_size=%d bytes", path, len(data))
	tr := tar.NewReader(bytes.NewReader(data))
	entriesScanned := 0
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			tarLog.Printf("File not found in tar archive after scanning %d entries: %s", entriesScanned, path)
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar archive: %w", err)
		}
		entriesScanned++
		// Reject tar entries that could escape a destination directory.
		if filepath.IsAbs(header.Name) || strings.Contains(header.Name, "..") {
			tarLog.Printf("Skipping unsafe tar entry: %s", header.Name)
			continue
		}
		if header.Name == path {
			tarLog.Printf("Found file in tar archive after scanning %d entries: %s", entriesScanned, path)
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("file %q not found in archive", path)
}
