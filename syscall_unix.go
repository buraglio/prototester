//go:build unix || darwin

package main

import (
	"syscall"
)

// Platform-specific constants
const (
	IPPROTO_ICMP   = syscall.IPPROTO_ICMP
	IPPROTO_ICMPV6 = syscall.IPPROTO_ICMPV6
	SO_RCVTIMEO    = syscall.SO_RCVTIMEO
)

// FdSet is a type alias for syscall.FdSet on Unix systems
type FdSet syscall.FdSet

// newFdSet creates a new FdSet
func newFdSet() *FdSet {
	return &FdSet{}
}

// setFd sets the bit for the given fd in the FdSet
func (f *FdSet) setFd(fd socketFd) {
	intFd := int(fd)
	f.Bits[intFd/64] |= 1 << (uint(intFd) % 64)
}

// toSyscallFdSet converts FdSet to syscall.FdSet for use with select
func (f *FdSet) toSyscallFdSet() *syscall.FdSet {
	return (*syscall.FdSet)(f)
}

// socketFd represents a socket file descriptor
type socketFd int

// socketWrite wraps syscall.Write for Unix systems
func socketWrite(fd socketFd, p []byte) (int, error) {
	return syscall.Write(int(fd), p)
}

// socketRecvfrom wraps syscall.Recvfrom for Unix systems
func socketRecvfrom(fd socketFd, p []byte, flags int) (n int, from syscall.Sockaddr, err error) {
	return syscall.Recvfrom(int(fd), p, flags)
}

// socketSendto wraps syscall.Sendto for Unix systems
func socketSendto(fd socketFd, p []byte, flags int, to syscall.Sockaddr) error {
	return syscall.Sendto(int(fd), p, flags, to)
}

// socketSetsockoptTimeval wraps syscall.SetsockoptTimeval for Unix systems
func socketSetsockoptTimeval(fd socketFd, level, opt int, tv *syscall.Timeval) error {
	return syscall.SetsockoptTimeval(int(fd), level, opt, tv)
}

// socketClose wraps syscall.Close for Unix systems
func socketClose(fd socketFd) error {
	return syscall.Close(int(fd))
}

// socketCreate creates a socket using syscall.Socket
func socketCreate(domain, typ, proto int) (socketFd, error) {
	fd, err := syscall.Socket(domain, typ, proto)
	return socketFd(fd), err
}

// socketConnect connects a socket using syscall.Connect
func socketConnect(fd socketFd, sa syscall.Sockaddr) error {
	return syscall.Connect(int(fd), sa)
}
