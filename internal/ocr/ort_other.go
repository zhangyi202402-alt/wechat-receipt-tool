//go:build cgo && !windows

package ocr

func prepareORTEnvironment(_, _ string) error { return nil }

func resolveORTLibrary(exeDir, configured string) (string, error) {
	return configured, nil
}
