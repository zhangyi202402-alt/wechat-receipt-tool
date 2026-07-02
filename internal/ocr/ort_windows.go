//go:build cgo && windows

package ocr

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procSetDllDir = kernel32.NewProc("SetDllDirectoryW")
)

func resolveORTLibrary(exeDir, configured string) (string, error) {
	candidates := []string{
		resolvePath(exeDir, configured),
		filepath.Join(exeDir, "lib", "onnxruntime.dll"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("onnxruntime.dll not found (checked %v); ensure lib/*.dll are present and install VC++ Redistributable x64", candidates)
}

func resolvePath(exeDir, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(exeDir, p))
}

func prepareORTEnvironment(exeDir, configured string) error {
	resolved, err := resolveORTLibrary(exeDir, configured)
	if err != nil {
		return err
	}
	libDir, err := filepath.Abs(filepath.Dir(resolved))
	if err != nil {
		return err
	}
	exeAbs, err := filepath.Abs(exeDir)
	if err != nil {
		return err
	}

	if dirUTF16, e := syscall.UTF16PtrFromString(libDir); e == nil {
		procSetDllDir.Call(uintptr(unsafe.Pointer(dirUTF16)))
	}
	if cur, ok := syscall.Getenv("PATH"); ok {
		_ = syscall.Setenv("PATH", libDir+";"+exeAbs+";"+cur)
	} else {
		_ = syscall.Setenv("PATH", libDir+";"+exeAbs)
	}
	return nil
}
