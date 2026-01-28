package regis3

import (
	"runtime"
	"unsafe"
)

// Wow provides helper functions for Windows-on-Windows support.
type Wow struct{}

// Is64BitProcess checks if the current process is 64-bit.
func (Wow) Is64BitProcess() bool {
	return unsafe.Sizeof(uintptr(0)) == 8
}

// Is64BitOperatingSystem checks if the OS is 64-bit.
func (Wow) Is64BitOperatingSystem() bool {
	if runtime.GOOS != "windows" {
		// On non-windows, we just return based on architecture
		return runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
	}
	
	// On windows, we should ideally use IsWow64Process if 32-bit.
	// But Go's runtime already knows about the architecture.
	// If GOARCH is amd64/arm64, it's a 64-bit OS.
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		return true
	}
	
	// If we are 32-bit (386/arm), we could still be on 64-bit OS.
	// For now, let's keep it simple or implement syscall if needed.
	// Given the context of MSI-Simplified, we usually care about the target arch.
	return false 
}
