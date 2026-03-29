//go:build darwin || linux

package loader

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// openNativeLib opens a native shared library using purego.Dlopen.
func openNativeLib(path string) (uintptr, error) {
	var flags int
	switch runtime.GOOS {
	case "darwin":
		flags = purego.RTLD_NOW | purego.RTLD_GLOBAL
	case "linux":
		flags = purego.RTLD_NOW | purego.RTLD_GLOBAL
	}
	return purego.Dlopen(path, flags)
}

// closeNativeLib closes a native library handle.
func closeNativeLib(handle uintptr) error {
	// purego does not expose Dlclose publicly.
	// The library will remain in memory until process exit.
	return nil
}

// loadNativeFunc loads a single function from the library using purego.RegisterLibFunc.
func loadNativeFunc(libHandle uintptr, name string, fptr interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			// Symbol not found - this is ok for optional functions
		}
	}()
	purego.RegisterLibFunc(fptr, libHandle, name)
	return nil
}
