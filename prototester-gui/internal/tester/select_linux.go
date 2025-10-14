//go:build linux

package tester

import "syscall"

// selectWithTimeout performs a select call and returns whether the fd is ready
func selectWithTimeout(fd int, fdSet *syscall.FdSet, tv *syscall.Timeval) (bool, error) {
	n, err := syscall.Select(fd+1, fdSet, nil, nil, tv)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
