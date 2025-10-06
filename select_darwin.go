//go:build darwin

package main

import "syscall"

// selectWithTimeout performs a select call and returns whether the fd is ready
func selectWithTimeout(fd socketFd, fdSet *FdSet, tv *syscall.Timeval) (bool, error) {
	intFd := int(fd)
	sysSet := fdSet.toSyscallFdSet()
	err := syscall.Select(intFd+1, sysSet, nil, nil, tv)
	if err != nil {
		return false, err
	}
	// Check if fd is set in fdSet
	return sysSet.Bits[intFd/64]&(1<<(uint(intFd)%64)) != 0, nil
}
