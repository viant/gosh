//go:build windows
// +build windows

package runner

func init() {
	// Use the default Windows command interpreter unless overridden.
	// Consumers can change via WithShell("powershell") if desired.
	defaultShell = "cmd.exe"
}
