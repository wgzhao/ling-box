//go:build !unix && !windows

package cmd

import "syscall"

func detachedProcAttr() *syscall.SysProcAttr {
	return nil
}
