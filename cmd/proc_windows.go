//go:build windows

package cmd

import "syscall"

// detachedProcAttr starts the worker without a console window and outside
// the parent's process group.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008, // DETACHED_PROCESS
	}
}
