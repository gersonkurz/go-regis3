package regis3

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// Is64BitProcess checks if the current process is 64-bit.
func Is64BitProcess() bool {
	return unsafe.Sizeof(uintptr(0)) == 8
}

// Cached result for IsWow64Process call (expensive syscall, result never changes)
var (
	isWow64Once   sync.Once
	isWow64Result bool
)

// Is64BitOperatingSystem checks if the OS is 64-bit.
func Is64BitOperatingSystem() bool {
	if runtime.GOOS != "windows" {
		// On non-Windows, we just return based on architecture
		return runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
	}

	// On Windows with a 64-bit Go binary, we know it's a 64-bit OS
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		return true
	}

	// On Windows with a 32-bit Go binary (GOARCH=386), we could be running
	// on either a 32-bit or 64-bit OS. Use IsWow64Process to find out.
	isWow64Once.Do(func() {
		isWow64Result = checkIsWow64Process()
	})

	return isWow64Result
}

// checkIsWow64Process calls the Windows API to determine if we're a 32-bit
// process running on 64-bit Windows (WOW64).
func checkIsWow64Process() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	isWow64Process := kernel32.NewProc("IsWow64Process")

	// Check if the function exists (it might not on very old Windows versions)
	if err := isWow64Process.Find(); err != nil {
		// Function doesn't exist, so we're on 32-bit Windows
		return false
	}

	var isWow64 bool
	// GetCurrentProcess() returns -1 as a pseudo-handle
	currentProcess := uintptr(^uintptr(0)) // equivalent to (HANDLE)-1

	ret, _, _ := isWow64Process.Call(
		currentProcess,
		uintptr(unsafe.Pointer(&isWow64)),
	)

	if ret == 0 {
		// Call failed, assume 32-bit OS
		return false
	}

	return isWow64
}
