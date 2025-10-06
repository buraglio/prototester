//go:build windows

package main

import (
	"syscall"
)

// Platform-specific constants for Windows
const (
	IPPROTO_ICMP   = 1      // IPPROTO_ICMP value on Windows
	IPPROTO_ICMPV6 = 58     // IPPROTO_ICMPV6 value on Windows
	SO_RCVTIMEO    = 0x1006 // SO_RCVTIMEO value on Windows
)

// socketFd represents a socket file descriptor (Handle on Windows)
type socketFd syscall.Handle

// FdSet is a Windows implementation of fd_set structure
type FdSet struct {
	fd_count uint32
	fd_array [64]syscall.Handle
}

// newFdSet creates a new FdSet
func newFdSet() *FdSet {
	return &FdSet{}
}

// setFd adds the fd to the FdSet
func (f *FdSet) setFd(fd socketFd) {
	if f.fd_count < 64 {
		f.fd_array[f.fd_count] = syscall.Handle(fd)
		f.fd_count++
	}
}

// toSyscallFdSet converts FdSet to syscall.FdSet for use with select
// On Windows, we don't use syscall.FdSet, so this is just a placeholder
func (f *FdSet) toSyscallFdSet() *FdSet {
	return f
}

// socketWrite wraps syscall.Write for Windows
func socketWrite(fd socketFd, p []byte) (int, error) {
	var written uint32
	err := syscall.WriteFile(syscall.Handle(fd), p, &written, nil)
	return int(written), err
}

// socketRecvfrom wraps syscall.Recvfrom for Windows
func socketRecvfrom(fd socketFd, p []byte, flags int) (n int, from syscall.Sockaddr, err error) {
	return syscall.Recvfrom(syscall.Handle(fd), p, flags)
}

// socketSendto wraps syscall.Sendto for Windows
func socketSendto(fd socketFd, p []byte, flags int, to syscall.Sockaddr) error {
	return syscall.Sendto(syscall.Handle(fd), p, flags, to)
}

// socketSetsockoptTimeval wraps syscall.SetsockoptTimeval for Windows
func socketSetsockoptTimeval(fd socketFd, level, opt int, tv *syscall.Timeval) error {
	// Windows expects timeout in milliseconds as a DWORD
	timeout := uint32(tv.Sec*1000 + tv.Usec/1000)
	return syscall.SetsockoptInt(syscall.Handle(fd), level, opt, int(timeout))
}

// socketClose wraps syscall.Close for Windows
func socketClose(fd socketFd) error {
	return syscall.Closesocket(syscall.Handle(fd))
}

// socketCreate creates a socket using syscall.Socket
func socketCreate(domain, typ, proto int) (socketFd, error) {
	fd, err := syscall.Socket(domain, typ, proto)
	return socketFd(fd), err
}

// socketConnect connects a socket using syscall.Connect
func socketConnect(fd socketFd, sa syscall.Sockaddr) error {
	return syscall.Connect(syscall.Handle(fd), sa)
}

