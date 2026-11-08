//go:build unix

package cmd

import "syscall"

// detachedProcAttr puts the worker in its own session so it survives the
// Script Filter process and is not killed as part of Alfred's process group.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
