//go:build !windows

package regis3

import (
	"runtime"
	"unsafe"
)

// Is64BitProcess checks if the current process is 64-bit.
func Is64BitProcess() bool {
	return unsafe.Sizeof(uintptr(0)) == 8
}

// Is64BitOperatingSystem checks if the OS is 64-bit.
// On non-Windows platforms, we determine this based on the architecture.
func Is64BitOperatingSystem() bool {
	return runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
}
