package cli

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/github/gh-aw/pkg/fileutil"
)

func validateContainerMountPath(containerPath string) (string, error) {
	if containerPath == "" {
		return "", errors.New("container path cannot be empty")
	}
	if strings.ContainsAny(containerPath, "\x00\r\n:") {
		return "", errors.New("container path contains invalid control characters or reserved characters")
	}
	if !path.IsAbs(containerPath) {
		return "", fmt.Errorf("container path must be absolute: %s", containerPath)
	}
	cleanPath := path.Clean(containerPath)
	if cleanPath != containerPath {
		return "", fmt.Errorf("container path must be normalized: %s", containerPath)
	}
	return cleanPath, nil
}

func validateHostMountPath(hostPath string) (string, error) {
	cleanHostPath, err := fileutil.ValidateAbsolutePath(hostPath)
	if err != nil {
		return "", fmt.Errorf("invalid host path %q: %w", hostPath, err)
	}
	if strings.Contains(cleanHostPath[2:], ":") || (!isWindowsDrivePath(cleanHostPath) && strings.Contains(cleanHostPath, ":")) {
		return "", fmt.Errorf("host path contains unsupported ':' for docker -v mount syntax: %s", cleanHostPath)
	}
	return cleanHostPath, nil
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
