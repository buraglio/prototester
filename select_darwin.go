//go:build darwin

package main

import "syscall"

// selectWithTimeout performs a select call and returns whether the fd is ready
func selectWithTimeout(fd int, fdSet *syscall.FdSet, tv *syscall.Timeval) (bool, error) {
	err := syscall.Select(fd+1, fdSet, nil, nil, tv)
	if err != nil {
		return false, err
	}
	// Check if fd is set in fdSet
	return fdSet.Bits[fd/64]&(1<<(uint(fd)%64)) != 0, nil
}
