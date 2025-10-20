//go:build !windows
// +build !windows

package local

import "syscall"

// newSysProcAttr returns Unix-specific process attributes.
func newSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
