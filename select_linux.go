//go:build linux

package main

import "syscall"

// selectWithTimeout performs a select call and returns whether the fd is ready
func selectWithTimeout(fd socketFd, fdSet *FdSet, tv *syscall.Timeval) (bool, error) {
	intFd := int(fd)
	sysSet := fdSet.toSyscallFdSet()
	n, err := syscall.Select(intFd+1, sysSet, nil, nil, tv)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
