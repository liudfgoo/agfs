//go:build windows

package loader

import (
	"fmt"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

// openNativeLib opens a native shared library (DLL) using Windows LoadLibrary.
func openNativeLib(path string) (uintptr, error) {
	handle, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, fmt.Errorf("LoadLibrary failed: %w", err)
	}
	return uintptr(handle), nil
}

// closeNativeLib closes a native library handle.
func closeNativeLib(handle uintptr) error {
	return windows.FreeLibrary(windows.Handle(handle))
}

// loadNativeFunc loads a single function from the DLL using GetProcAddress
// and registers it via purego.RegisterFunc.
func loadNativeFunc(libHandle uintptr, name string, fptr interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			// Symbol not found - this is ok for optional functions
		}
	}()
	sym, err := windows.GetProcAddress(windows.Handle(libHandle), name)
	if err != nil {
		// Symbol not found - ok for optional functions
		return nil
	}
	purego.RegisterFunc(fptr, sym)
	return nil
}
