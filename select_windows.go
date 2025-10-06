//go:build windows

package main

import (
	"syscall"
	"time"
)

// selectWithTimeout performs a select-like call on Windows
// Since Windows select() works differently and requires winsock initialization,
// we use a simpler approach with socket recv timeout
func selectWithTimeout(fd socketFd, fdSet *FdSet, tv *syscall.Timeval) (bool, error) {
	// For Windows, we set socket timeout and rely on the recv call to timeout
	// This is simpler and more reliable than trying to use Windows select()
	timeout := time.Duration(tv.Sec)*time.Second + time.Duration(tv.Usec)*time.Microsecond

	// Set socket recv timeout
	timeoutMs := uint32(timeout.Milliseconds())
	err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, SO_RCVTIMEO, int(timeoutMs))
	if err != nil {
		return false, err
	}

	// On Windows, we'll just return true and let the recv call handle the timeout
	// This is because properly implementing select on Windows is complex and requires
	// handling the fd_set structure differently
	return true, nil
}
