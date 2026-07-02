package pathutil

import (
	"os"
	"path/filepath"
)

// ExeDir returns the directory containing the running executable.
// Falls back to the current working directory if unavailable.
func ExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", err
		}
		return wd, nil
	}
	return filepath.Dir(exe), nil
}

// Resolve joins baseDir with a relative path and cleans the result.
func Resolve(baseDir, relPath string) string {
	if filepath.IsAbs(relPath) {
		return filepath.Clean(relPath)
	}
	return filepath.Clean(filepath.Join(baseDir, relPath))
}

// ResolveConfig finds config.yaml: explicit path, then exeDir/config.yaml.
func ResolveConfig(exeDir, explicit string) string {
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return explicit
		}
		return Resolve(exeDir, explicit)
	}
	return Resolve(exeDir, "config.yaml")
}
