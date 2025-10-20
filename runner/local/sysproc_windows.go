//go:build windows
// +build windows

package local

import "syscall"

// newSysProcAttr returns Windows-specific process attributes.
func newSysProcAttr() *syscall.SysProcAttr {
	// No special flags by default; adjust here if needed (e.g., HideWindow)
	return &syscall.SysProcAttr{}
}
