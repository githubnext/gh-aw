package cli

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/github/gh-aw/pkg/fileutil"
)

func containsControlCharacters(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || unicode.In(r, unicode.Cf) || r == '\u2028' || r == '\u2029'
	}) >= 0
}

func validateContainerMountPath(containerPath string) (string, error) {
	if containerPath == "" {
		return "", errors.New("container path cannot be empty. Example: /workdir")
	}
	if containsControlCharacters(containerPath) || strings.Contains(containerPath, ":") {
		return "", errors.New("container path contains invalid control characters or reserved characters. Example: /workdir")
	}
	if !path.IsAbs(containerPath) {
		return "", fmt.Errorf("container path must be absolute. Example: /workdir. Got: %s", containerPath)
	}
	cleanPath := path.Clean(containerPath)
	if cleanPath != containerPath {
		return "", fmt.Errorf("container path must be normalized. Example: /workdir/config. Got: %s", containerPath)
	}
	return cleanPath, nil
}

func validateHostMountPath(hostPath string) (string, error) {
	cleanHostPath, err := fileutil.ValidateAbsolutePath(hostPath)
	if err != nil {
		return "", fmt.Errorf("invalid host path %q: %w", hostPath, err)
	}
	if strings.Contains(cleanHostPath[2:], ":") || (!isWindowsDrivePath(cleanHostPath) && strings.Contains(cleanHostPath, ":")) {
		return "", fmt.Errorf("host path contains unsupported ':' for docker -v mount syntax. Example: /tmp/repo or C:/repo. Got: %s", cleanHostPath)
	}
	return cleanHostPath, nil
}

func validateDockerImageRef(imageRef string) (string, error) {
	if imageRef == "" {
		return "", errors.New("grant image reference cannot be empty. Example: ghcr.io/example/image:tag")
	}
	// Image refs disallow all Unicode whitespace, while containsControlCharacters also rejects
	// non-whitespace spoofing characters such as bidi overrides and other format controls.
	if containsControlCharacters(imageRef) || strings.IndexFunc(imageRef, unicode.IsSpace) >= 0 {
		return "", fmt.Errorf("grant image reference contains invalid whitespace/control characters. Example: ghcr.io/example/image:tag. Got: %q", imageRef)
	}
	if strings.HasPrefix(imageRef, "-") {
		return "", fmt.Errorf("grant image reference cannot start with '-'. Example: ghcr.io/example/image:tag. Got: %q", imageRef)
	}
	return imageRef, nil
}

func isWindowsDrivePath(hostPath string) bool {
	if len(hostPath) < 3 {
		return false
	}
	driveLetter := hostPath[0]
	return ((driveLetter >= 'a' && driveLetter <= 'z') || (driveLetter >= 'A' && driveLetter <= 'Z')) &&
		hostPath[1] == ':' &&
		(hostPath[2] == '\\' || hostPath[2] == '/')
}

func buildDockerVolumeMount(hostPath, containerPath string) (string, error) {
	cleanHostPath, err := validateHostMountPath(hostPath)
	if err != nil {
		return "", err
	}
	cleanContainerPath, err := validateContainerMountPath(containerPath)
	if err != nil {
		return "", err
	}
	return cleanHostPath + ":" + cleanContainerPath, nil
}

func buildDockerReadonlyFileMount(hostFile, containerPath string) (string, error) {
	cleanHostFile, err := validateHostMountPath(hostFile)
	if err != nil {
		return "", fmt.Errorf("invalid host file %q: %w", hostFile, err)
	}
	info, err := os.Stat(cleanHostFile)
	if err != nil {
		return "", fmt.Errorf("failed to stat host file %q: %w", cleanHostFile, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("host file is not a regular file: %s", cleanHostFile)
	}
	cleanContainerPath, err := validateContainerMountPath(containerPath)
	if err != nil {
		return "", err
	}
	return cleanHostFile + ":" + cleanContainerPath + ":ro", nil
}
