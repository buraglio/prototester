package tester

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
)

// NewTester creates a new LatencyTester instance with the provided configuration
func NewTester(config TestConfig) *LatencyTester {
	return &LatencyTester{
		target4:     config.Target4,
		target6:     config.Target6,
		hostname:    config.Hostname,
		port:        config.Port,
		count:       config.Count,
		interval:    config.Interval,
		timeout:     config.Timeout,
		size:        config.Size,
		ipv4Only:    config.IPv4Only,
		ipv6Only:    config.IPv6Only,
		verbose:     config.Verbose,
		dnsProtocol: config.DNSProtocol,
		dnsQuery:    config.DNSQuery,
	}
}

// RunTCPTest executes TCP connectivity tests
func (lt *LatencyTester) RunTCPTest() *TestResult {
	result := &TestResult{
		Mode:       "tcp",
		Protocol:   "TCP",
		Targets:    make(map[string]string),
		TestConfig: lt.toTestConfig(),
		Timestamp:  time.Now(),
	}

	lt.tcpMode = true
	lt.udpMode = false
	lt.icmpMode = false
	lt.httpMode = false
	lt.dnsMode = false

	if !lt.ipv6Only && lt.target4 != "" {
		result.Targets["ipv4"] = lt.target4
		lt.testIPv4()
		stats := lt.calculateStats(lt.results4)
		result.IPv4Results = &stats
	}

	if !lt.ipv4Only && lt.target6 != "" {
		result.Targets["ipv6"] = lt.target6
		lt.testIPv6()
		stats := lt.calculateStats(lt.results6)
		result.IPv6Results = &stats
	}

	return result
}

// RunUDPTest executes UDP connectivity tests
func (lt *LatencyTester) RunUDPTest() *TestResult {
	result := &TestResult{
		Mode:       "udp",
		Protocol:   "UDP",
		Targets:    make(map[string]string),
		TestConfig: lt.toTestConfig(),
		Timestamp:  time.Now(),
	}

	lt.tcpMode = false
	lt.udpMode = true
	lt.icmpMode = false
	lt.httpMode = false
	lt.dnsMode = false

	if !lt.ipv6Only && lt.target4 != "" {
		result.Targets["ipv4"] = lt.target4
		lt.testIPv4()
		stats := lt.calculateStats(lt.results4)
		result.IPv4Results = &stats
	}

	if !lt.ipv4Only && lt.target6 != "" {
		result.Targets["ipv6"] = lt.target6
		lt.testIPv6()
		stats := lt.calculateStats(lt.results6)
		result.IPv6Results = &stats
	}

	return result
}

// RunICMPTest executes ICMP ping tests
func (lt *LatencyTester) RunICMPTest() *TestResult {
	result := &TestResult{
		Mode:       "icmp",
		Protocol:   "ICMP",
		Targets:    make(map[string]string),
		TestConfig: lt.toTestConfig(),
		Timestamp:  time.Now(),
	}

	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = true
	lt.httpMode = false
	lt.dnsMode = false

	if !lt.ipv6Only && lt.target4 != "" {
		result.Targets["ipv4"] = lt.target4
		lt.testIPv4()
		stats := lt.calculateStats(lt.results4)
		result.IPv4Results = &stats
	}

	if !lt.ipv4Only && lt.target6 != "" {
		result.Targets["ipv6"] = lt.target6
		lt.testIPv6()
		stats := lt.calculateStats(lt.results6)
		result.IPv6Results = &stats
	}

	return result
}

// RunHTTPTest executes HTTP/HTTPS request timing tests
func (lt *LatencyTester) RunHTTPTest() *TestResult {
	result := &TestResult{
		Mode:       "http",
		Protocol:   "HTTP",
		Targets:    make(map[string]string),
		TestConfig: lt.toTestConfig(),
		Timestamp:  time.Now(),
	}

	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = false
	lt.httpMode = true
	lt.dnsMode = false

	if !lt.ipv6Only && lt.target4 != "" {
		result.Targets["ipv4"] = lt.target4
		lt.testIPv4()
		stats := lt.calculateStats(lt.results4)
		result.IPv4Results = &stats
	}

	if !lt.ipv4Only && lt.target6 != "" {
		result.Targets["ipv6"] = lt.target6
		lt.testIPv6()
		stats := lt.calculateStats(lt.results6)
		result.IPv6Results = &stats
	}

	return result
}

// RunDNSTest executes DNS query tests
func (lt *LatencyTester) RunDNSTest() *TestResult {
	result := &TestResult{
		Mode:       "dns",
		Protocol:   fmt.Sprintf("DNS-%s", strings.ToUpper(lt.dnsProtocol)),
		Targets:    make(map[string]string),
		TestConfig: lt.toTestConfig(),
		Timestamp:  time.Now(),
	}

	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = false
	lt.httpMode = false
	lt.dnsMode = true

	if !lt.ipv6Only && lt.target4 != "" {
		result.Targets["ipv4"] = lt.target4
		lt.testIPv4()
		stats := lt.calculateStats(lt.results4)
		result.IPv4Results = &stats
	}

	if !lt.ipv4Only && lt.target6 != "" {
		result.Targets["ipv6"] = lt.target6
		lt.testIPv6()
		stats := lt.calculateStats(lt.results6)
		result.IPv6Results = &stats
	}

	return result
}

// RunCompareTest executes comprehensive comparison tests between IPv4 and IPv6
func (lt *LatencyTester) RunCompareTest(protocol string) *TestResult {
	result := &TestResult{
		Mode:       "compare",
		Protocol:   protocol,
		Targets:    make(map[string]string),
		TestConfig: lt.toTestConfig(),
		Timestamp:  time.Now(),
	}

	if lt.hostname == "" {
		result.ErrorMessage = "hostname is required for comparison tests"
		return result
	}

	// Resolve hostname
	ipv4, ipv6, err := lt.resolveHostname(lt.hostname)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to resolve hostname: %v", err)
		return result
	}

	if ipv4 == "" || ipv6 == "" {
		result.ErrorMessage = "both IPv4 and IPv6 addresses are required for comparison"
		return result
	}

	result.Targets["hostname"] = lt.hostname
	result.Targets["ipv4"] = ipv4
	result.Targets["ipv6"] = ipv6

	comparison := &ComparisonResult{
		ResolvedIPv4: ipv4,
		ResolvedIPv6: ipv6,
		Protocol:     protocol,
		Hostname:     lt.hostname,
		Port:         lt.port,
		Timestamp:    time.Now(),
	}

	lt.target4 = ipv4
	lt.target6 = ipv6

	switch protocol {
	case "TCP/UDP":
		lt.runTCPUDPComparison(comparison)
	case "ICMP":
		lt.runICMPComparison(comparison)
	case "HTTP":
		lt.runHTTPComparison(comparison)
	case "DNS":
		comparison.DNSQuery = lt.dnsQuery
		lt.runDNSComparison(comparison)
	default:
		result.ErrorMessage = fmt.Sprintf("unsupported comparison protocol: %s", protocol)
		return result
	}

	result.Comparison = comparison
	return result
}

// Helper function to convert internal config to TestConfig
func (lt *LatencyTester) toTestConfig() TestConfig {
	return TestConfig{
		Target4:     lt.target4,
		Target6:     lt.target6,
		Hostname:    lt.hostname,
		Port:        lt.port,
		Count:       lt.count,
		Interval:    lt.interval,
		Timeout:     lt.timeout,
		Size:        lt.size,
		DNSProtocol: lt.dnsProtocol,
		DNSQuery:    lt.dnsQuery,
		IPv4Only:    lt.ipv4Only,
		IPv6Only:    lt.ipv6Only,
		Verbose:     lt.verbose,
	}
}

// runTCPUDPComparison runs TCP and UDP comparison tests
func (lt *LatencyTester) runTCPUDPComparison(result *ComparisonResult) {
	// Test TCP IPv4
	lt.tcpMode = true
	lt.udpMode = false
	lt.icmpMode = false
	lt.httpMode = false
	lt.dnsMode = false
	lt.results4 = nil
	lt.testIPv4()
	stats := lt.calculateStats(lt.results4)
	result.TCPv4Stats = &stats

	// Test TCP IPv6
	lt.results6 = nil
	lt.testIPv6()
	stats = lt.calculateStats(lt.results6)
	result.TCPv6Stats = &stats

	// Test UDP IPv4
	lt.tcpMode = false
	lt.udpMode = true
	lt.results4 = nil
	lt.testIPv4()
	stats = lt.calculateStats(lt.results4)
	result.UDPv4Stats = &stats

	// Test UDP IPv6
	lt.results6 = nil
	lt.testIPv6()
	stats = lt.calculateStats(lt.results6)
	result.UDPv6Stats = &stats

	lt.calculateComparisonScores(result)
}

// runICMPComparison runs ICMP comparison tests
func (lt *LatencyTester) runICMPComparison(result *ComparisonResult) {
	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = true
	lt.httpMode = false
	lt.dnsMode = false

	// Test ICMP IPv4
	lt.results4 = nil
	lt.testIPv4()
	stats := lt.calculateStats(lt.results4)
	result.ICMPv4Stats = &stats

	// Test ICMP IPv6
	lt.results6 = nil
	lt.testIPv6()
	stats = lt.calculateStats(lt.results6)
	result.ICMPv6Stats = &stats

	lt.calculateICMPComparisonScores(result)
}

// runHTTPComparison runs HTTP comparison tests
func (lt *LatencyTester) runHTTPComparison(result *ComparisonResult) {
	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = false
	lt.httpMode = true
	lt.dnsMode = false

	// Test HTTP IPv4
	lt.results4 = nil
	lt.testIPv4()
	stats := lt.calculateStats(lt.results4)
	result.HTTPv4Stats = &stats

	// Test HTTP IPv6
	lt.results6 = nil
	lt.testIPv6()
	stats = lt.calculateStats(lt.results6)
	result.HTTPv6Stats = &stats

	lt.calculateHTTPComparisonScores(result)
}

// runDNSComparison runs DNS comparison tests
func (lt *LatencyTester) runDNSComparison(result *ComparisonResult) {
	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = false
	lt.httpMode = false
	lt.dnsMode = true

	// Test DNS IPv4
	lt.results4 = nil
	lt.testIPv4()
	stats := lt.calculateStats(lt.results4)
	result.DNSv4Stats = &stats

	// Test DNS IPv6
	lt.results6 = nil
	lt.testIPv6()
	stats = lt.calculateStats(lt.results6)
	result.DNSv6Stats = &stats

	lt.calculateDNSComparisonScores(result)
}

// testIPv4 runs tests against IPv4 target
func (lt *LatencyTester) testIPv4() {
	lt.results4 = make([]PingResult, 0, lt.count)

	for i := 0; i < lt.count; i++ {
		var result PingResult
		if lt.tcpMode {
			result = lt.testTCPConnect("tcp4", lt.target4, i+1)
		} else if lt.udpMode {
			result = lt.testUDPConnect("udp4", lt.target4, i+1)
		} else if lt.httpMode {
			result = lt.testHTTP("4", lt.target4, i+1)
		} else if lt.dnsMode {
			result = lt.testDNS("4", lt.target4, i+1)
		} else if lt.icmpMode {
			result = lt.testICMPv4(i + 1)
		} else {
			result = lt.testTCPConnect("tcp4", lt.target4, i+1)
		}

		lt.mu.Lock()
		lt.results4 = append(lt.results4, result)
		lt.mu.Unlock()

		if i < lt.count-1 {
			time.Sleep(lt.interval)
		}
	}
}

// testIPv6 runs tests against IPv6 target
func (lt *LatencyTester) testIPv6() {
	lt.results6 = make([]PingResult, 0, lt.count)

	for i := 0; i < lt.count; i++ {
		var result PingResult
		if lt.tcpMode {
			result = lt.testTCPConnect("tcp6", lt.target6, i+1)
		} else if lt.udpMode {
			result = lt.testUDPConnect("udp6", lt.target6, i+1)
		} else if lt.httpMode {
			result = lt.testHTTP("6", lt.target6, i+1)
		} else if lt.dnsMode {
			result = lt.testDNS("6", lt.target6, i+1)
		} else if lt.icmpMode {
			result = lt.testICMPv6(i + 1)
		} else {
			result = lt.testTCPConnect("tcp6", lt.target6, i+1)
		}

		lt.mu.Lock()
		lt.results6 = append(lt.results6, result)
		lt.mu.Unlock()

		if i < lt.count-1 {
			time.Sleep(lt.interval)
		}
	}
}

// testTCPConnect performs a TCP connection test
func (lt *LatencyTester) testTCPConnect(network, target string, seq int) PingResult {
	start := time.Now()

	dialer := &net.Dialer{
		Timeout: lt.timeout,
	}

	var address string
	if network == "tcp6" {
		address = fmt.Sprintf("[%s]:%d", target, lt.port)
	} else {
		address = fmt.Sprintf("%s:%d", target, lt.port)
	}

	conn, err := dialer.Dial(network, address)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer conn.Close()

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// testUDPConnect performs a UDP connection test
func (lt *LatencyTester) testUDPConnect(network, target string, seq int) PingResult {
	start := time.Now()

	var address string
	if network == "udp6" {
		address = fmt.Sprintf("[%s]:%d", target, lt.port)
	} else {
		address = fmt.Sprintf("%s:%d", target, lt.port)
	}

	conn, err := net.DialTimeout(network, address, lt.timeout)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer conn.Close()

	testData := []byte("test")
	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(testData)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	buffer := make([]byte, 1024)
	_, _ = conn.Read(buffer)

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// testICMPv4 performs an ICMP ping test for IPv4
func (lt *LatencyTester) testICMPv4(seq int) PingResult {
	result := lt.tryUnprivilegedICMPv4(seq)
	if result.Success {
		return result
	}

	if strings.Contains(result.Error.Error(), "operation not permitted") ||
		strings.Contains(result.Error.Error(), "permission denied") {
		result = lt.tryRawICMPv4(seq)
		if result.Success {
			return result
		}
	}

	if strings.Contains(result.Error.Error(), "operation not permitted") ||
		strings.Contains(result.Error.Error(), "permission denied") {
		return lt.testTCPConnect("tcp4", lt.target4, seq)
	}

	return result
}

// tryUnprivilegedICMPv4 attempts unprivileged ICMP for IPv4
func (lt *LatencyTester) tryUnprivilegedICMPv4(seq int) PingResult {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_ICMP)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error creating IPv4 unprivileged ICMP socket: %v", err), Timestamp: time.Now()}
	}
	defer syscall.Close(fd)

	dst, err := net.ResolveIPAddr("ip4", lt.target4)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error resolving IPv4 address: %v", err), Timestamp: time.Now()}
	}

	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], dst.IP.To4())
	err = syscall.Connect(fd, addr)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error connecting socket: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv4Unprivileged(fd, dst, seq)
}

// tryRawICMPv4 attempts raw ICMP for IPv4
func (lt *LatencyTester) tryRawICMPv4(seq int) PingResult {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error creating IPv4 raw socket: %v (try running with sudo)", err), Timestamp: time.Now()}
	}
	defer syscall.Close(fd)

	dst, err := net.ResolveIPAddr("ip4", lt.target4)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error resolving IPv4 address: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv4Raw(fd, dst, seq)
}

// sendICMPv4Unprivileged sends an unprivileged ICMP echo request for IPv4
func (lt *LatencyTester) sendICMPv4Unprivileged(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	packet := make([]byte, 8+lt.size)
	packet[0] = 8
	packet[1] = 0
	packet[2] = 0
	packet[3] = 0
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	_, err := syscall.Write(fd, packet)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	reply := make([]byte, 1500)
	deadline := start.Add(lt.timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return PingResult{Success: false, Error: fmt.Errorf("timeout"), Timestamp: start}
		}

		fdSet := &syscall.FdSet{}
		fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)
		tv := syscall.NsecToTimeval(remaining.Nanoseconds())

		ready, err := selectWithTimeout(fd, fdSet, &tv)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return PingResult{Success: false, Error: err, Timestamp: start}
		}
		if !ready {
			return PingResult{Success: false, Error: fmt.Errorf("timeout"), Timestamp: start}
		}

		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		if n < 8 {
			continue
		}

		if reply[0] == 0 {
			replySeq := binary.BigEndian.Uint16(reply[6:8])
			if int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

// sendICMPv4Raw sends a raw ICMP echo request for IPv4
func (lt *LatencyTester) sendICMPv4Raw(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	packet := make([]byte, 8+lt.size)
	packet[0] = 8
	packet[1] = 0
	packet[2] = 0
	packet[3] = 0
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	checksum := calculateChecksum(packet)
	binary.BigEndian.PutUint16(packet[2:4], checksum)

	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], dst.IP.To4())

	err := syscall.Sendto(fd, packet, 0, addr)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	tv := syscall.NsecToTimeval(lt.timeout.Nanoseconds())
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	reply := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		if n < 28 {
			continue
		}

		ipHeaderLen := int(reply[0]&0x0f) * 4
		if n < ipHeaderLen+8 {
			continue
		}

		icmpPacket := reply[ipHeaderLen:]
		if icmpPacket[0] == 0 {
			replyID := binary.BigEndian.Uint16(icmpPacket[4:6])
			replySeq := binary.BigEndian.Uint16(icmpPacket[6:8])

			if int(replyID) == pid && int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

// testICMPv6 performs an ICMP ping test for IPv6
func (lt *LatencyTester) testICMPv6(seq int) PingResult {
	result := lt.tryUnprivilegedICMPv6(seq)
	if result.Success {
		return result
	}

	if strings.Contains(result.Error.Error(), "operation not permitted") ||
		strings.Contains(result.Error.Error(), "permission denied") {
		result = lt.tryRawICMPv6(seq)
		if result.Success {
			return result
		}
	}

	if strings.Contains(result.Error.Error(), "operation not permitted") ||
		strings.Contains(result.Error.Error(), "permission denied") {
		return lt.testTCPConnect("tcp6", lt.target6, seq)
	}

	return result
}

// tryUnprivilegedICMPv6 attempts unprivileged ICMP for IPv6
func (lt *LatencyTester) tryUnprivilegedICMPv6(seq int) PingResult {
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_DGRAM, syscall.IPPROTO_ICMPV6)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error creating IPv6 unprivileged ICMP socket: %v", err), Timestamp: time.Now()}
	}
	defer syscall.Close(fd)

	dst, err := net.ResolveIPAddr("ip6", lt.target6)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error resolving IPv6 address: %v", err), Timestamp: time.Now()}
	}

	addr := &syscall.SockaddrInet6{}
	copy(addr.Addr[:], dst.IP.To16())
	err = syscall.Connect(fd, addr)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error connecting socket: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv6Unprivileged(fd, dst, seq)
}

// tryRawICMPv6 attempts raw ICMP for IPv6
func (lt *LatencyTester) tryRawICMPv6(seq int) PingResult {
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_ICMPV6)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error creating IPv6 raw socket: %v (try running with sudo)", err), Timestamp: time.Now()}
	}
	defer syscall.Close(fd)

	dst, err := net.ResolveIPAddr("ip6", lt.target6)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error resolving IPv6 address: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv6Raw(fd, dst, seq)
}

// sendICMPv6Unprivileged sends an unprivileged ICMP echo request for IPv6
func (lt *LatencyTester) sendICMPv6Unprivileged(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	packet := make([]byte, 8+lt.size)
	packet[0] = 128
	packet[1] = 0
	packet[2] = 0
	packet[3] = 0
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	_, err := syscall.Write(fd, packet)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	reply := make([]byte, 1500)
	deadline := start.Add(lt.timeout)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return PingResult{Success: false, Error: fmt.Errorf("timeout"), Timestamp: start}
		}

		fdSet := &syscall.FdSet{}
		fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)
		tv := syscall.NsecToTimeval(remaining.Nanoseconds())

		ready, err := selectWithTimeout(fd, fdSet, &tv)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return PingResult{Success: false, Error: err, Timestamp: start}
		}
		if !ready {
			return PingResult{Success: false, Error: fmt.Errorf("timeout"), Timestamp: start}
		}

		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		if n < 8 {
			continue
		}

		if reply[0] == 129 {
			replySeq := binary.BigEndian.Uint16(reply[6:8])
			if int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

// sendICMPv6Raw sends a raw ICMP echo request for IPv6
func (lt *LatencyTester) sendICMPv6Raw(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	packet := make([]byte, 8+lt.size)
	packet[0] = 128
	packet[1] = 0
	packet[2] = 0
	packet[3] = 0
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	addr := &syscall.SockaddrInet6{}
	copy(addr.Addr[:], dst.IP.To16())

	err := syscall.Sendto(fd, packet, 0, addr)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	tv := syscall.NsecToTimeval(lt.timeout.Nanoseconds())
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	reply := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		if n < 8 {
			continue
		}

		if reply[0] == 129 {
			replyID := binary.BigEndian.Uint16(reply[4:6])
			replySeq := binary.BigEndian.Uint16(reply[6:8])

			if int(replyID) == pid && int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

// testHTTP performs an HTTP/HTTPS request timing test
func (lt *LatencyTester) testHTTP(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	var scheme string
	if lt.port == 443 || lt.port == 8443 {
		scheme = "https"
	} else {
		scheme = "http"
	}

	var url string
	if ipVersion == "6" {
		url = fmt.Sprintf("%s://[%s]:%d/", scheme, target, lt.port)
	} else {
		url = fmt.Sprintf("%s://%s:%d/", scheme, target, lt.port)
	}

	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	if ipVersion == "4" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: lt.timeout}
			return dialer.DialContext(ctx, "tcp4", addr)
		}
	} else {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: lt.timeout}
			return dialer.DialContext(ctx, "tcp6", addr)
		}
	}

	client := &http.Client{
		Timeout:   lt.timeout,
		Transport: transport,
	}

	resp, err := client.Head(url)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// testDNS performs a DNS query test
func (lt *LatencyTester) testDNS(ipVersion, target string, seq int) PingResult {
	switch lt.dnsProtocol {
	case "udp":
		return lt.testDNSUDP(ipVersion, target, seq)
	case "tcp":
		return lt.testDNSTCP(ipVersion, target, seq)
	case "dot":
		return lt.testDNSDoT(ipVersion, target, seq)
	case "doh":
		return lt.testDNSDoH(ipVersion, target, seq)
	default:
		return PingResult{Success: false, Error: fmt.Errorf("unsupported DNS protocol: %s", lt.dnsProtocol), Timestamp: time.Now()}
	}
}

// testDNSUDP performs a DNS query over UDP
func (lt *LatencyTester) testDNSUDP(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	var address string
	if ipVersion == "6" {
		address = fmt.Sprintf("[%s]:%d", target, lt.port)
	} else {
		address = fmt.Sprintf("%s:%d", target, lt.port)
	}

	network := "udp" + ipVersion
	conn, err := net.DialTimeout(network, address, lt.timeout)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(queryPacket)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	conn.SetReadDeadline(time.Now().Add(lt.timeout))
	response := make([]byte, 512)
	n, err := conn.Read(response)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	if n < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", n), Timestamp: start}
	}

	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// testDNSTCP performs a DNS query over TCP
func (lt *LatencyTester) testDNSTCP(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	var address string
	if ipVersion == "6" {
		address = fmt.Sprintf("[%s]:%d", target, lt.port)
	} else {
		address = fmt.Sprintf("%s:%d", target, lt.port)
	}

	network := "tcp" + ipVersion
	conn, err := net.DialTimeout(network, address, lt.timeout)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer conn.Close()

	lengthPrefix := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthPrefix, uint16(len(queryPacket)))
	tcpQuery := append(lengthPrefix, queryPacket...)

	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(tcpQuery)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	conn.SetReadDeadline(time.Now().Add(lt.timeout))
	lengthBytes := make([]byte, 2)
	_, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	responseLength := binary.BigEndian.Uint16(lengthBytes)
	if responseLength > 4096 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too large: %d bytes", responseLength), Timestamp: start}
	}

	response := make([]byte, responseLength)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	if len(response) < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", len(response)), Timestamp: start}
	}

	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// testDNSDoT performs a DNS query over TLS (DoT)
func (lt *LatencyTester) testDNSDoT(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	var address string
	if ipVersion == "6" {
		address = fmt.Sprintf("[%s]:%d", target, lt.port)
	} else {
		address = fmt.Sprintf("%s:%d", target, lt.port)
	}

	config := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         target,
	}

	dialer := &net.Dialer{Timeout: lt.timeout}
	network := "tcp" + ipVersion
	conn, err := tls.DialWithDialer(dialer, network, address, config)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer conn.Close()

	lengthPrefix := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthPrefix, uint16(len(queryPacket)))
	tcpQuery := append(lengthPrefix, queryPacket...)

	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(tcpQuery)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	conn.SetReadDeadline(time.Now().Add(lt.timeout))
	lengthBytes := make([]byte, 2)
	_, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	responseLength := binary.BigEndian.Uint16(lengthBytes)
	if responseLength > 4096 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too large: %d bytes", responseLength), Timestamp: start}
	}

	response := make([]byte, responseLength)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	if len(response) < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", len(response)), Timestamp: start}
	}

	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// testDNSDoH performs a DNS query over HTTPS (DoH)
func (lt *LatencyTester) testDNSDoH(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	var port int
	if lt.port == 443 {
		port = 443
	} else {
		port = lt.port
	}

	var baseURL string
	if ipVersion == "6" {
		baseURL = fmt.Sprintf("https://[%s]:%d/dns-query", target, port)
	} else {
		baseURL = fmt.Sprintf("https://%s:%d/dns-query", target, port)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewReader(queryPacket))
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}

	if ipVersion == "4" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: lt.timeout}
			return dialer.DialContext(ctx, "tcp4", addr)
		}
	} else {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: lt.timeout}
			return dialer.DialContext(ctx, "tcp6", addr)
		}
	}

	client := &http.Client{
		Timeout:   lt.timeout,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return PingResult{Success: false, Error: fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status), Timestamp: start}
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	if len(response) < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", len(response)), Timestamp: start}
	}

	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// buildDNSQuery builds a DNS query packet
func (lt *LatencyTester) buildDNSQuery() ([]byte, error) {
	queryID := make([]byte, 2)
	_, err := rand.Read(queryID)
	if err != nil {
		return nil, err
	}

	header := DNSHeader{
		ID:      binary.BigEndian.Uint16(queryID),
		Flags:   0x0100,
		QDCount: 1,
		ANCount: 0,
		NSCount: 0,
		ARCount: 0,
	}

	question := DNSQuestion{
		Name:  lt.dnsQuery,
		Type:  1,
		Class: 1,
	}

	packet := make([]byte, 0, 512)

	headerBytes := make([]byte, 12)
	binary.BigEndian.PutUint16(headerBytes[0:2], header.ID)
	binary.BigEndian.PutUint16(headerBytes[2:4], header.Flags)
	binary.BigEndian.PutUint16(headerBytes[4:6], header.QDCount)
	binary.BigEndian.PutUint16(headerBytes[6:8], header.ANCount)
	binary.BigEndian.PutUint16(headerBytes[8:10], header.NSCount)
	binary.BigEndian.PutUint16(headerBytes[10:12], header.ARCount)
	packet = append(packet, headerBytes...)

	domainParts := strings.Split(question.Name, ".")
	for _, part := range domainParts {
		if len(part) > 63 {
			return nil, fmt.Errorf("domain label too long: %s", part)
		}
		packet = append(packet, byte(len(part)))
		packet = append(packet, []byte(part)...)
	}
	packet = append(packet, 0)

	typeClassBytes := make([]byte, 4)
	binary.BigEndian.PutUint16(typeClassBytes[0:2], question.Type)
	binary.BigEndian.PutUint16(typeClassBytes[2:4], question.Class)
	packet = append(packet, typeClassBytes...)

	return packet, nil
}

// resolveHostname resolves a hostname to IPv4 and IPv6 addresses
func (lt *LatencyTester) resolveHostname(hostname string) (ipv4, ipv6 string, err error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return "", "", err
	}

	for _, ip := range ips {
		if ipv4 == "" && ip.To4() != nil {
			ipv4 = ip.String()
		}
		if ipv6 == "" && ip.To4() == nil && ip.To16() != nil {
			ipv6 = ip.String()
		}
		if ipv4 != "" && ipv6 != "" {
			break
		}
	}

	if ipv4 == "" && ipv6 == "" {
		return "", "", fmt.Errorf("no A or AAAA records found for %s", hostname)
	}

	return ipv4, ipv6, nil
}

// calculateStats calculates statistics from test results
func (lt *LatencyTester) calculateStats(results []PingResult) Statistics {
	stats := Statistics{}
	var latencies []time.Duration

	for _, result := range results {
		stats.Sent++
		if result.Success {
			stats.Received++
			latencies = append(latencies, result.Latency)
		}
	}

	stats.Lost = stats.Sent - stats.Received
	stats.Latencies = latencies

	if len(latencies) == 0 {
		return stats
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	stats.Min = latencies[0]
	stats.Max = latencies[len(latencies)-1]

	var sum time.Duration
	for _, lat := range latencies {
		sum += lat
	}
	stats.Avg = sum / time.Duration(len(latencies))

	var variance float64
	avgNs := float64(stats.Avg.Nanoseconds())
	for _, lat := range latencies {
		diff := float64(lat.Nanoseconds()) - avgNs
		variance += diff * diff
	}
	variance /= float64(len(latencies))
	stats.StdDev = time.Duration(math.Sqrt(variance))

	if len(latencies) > 1 {
		var jitterSum float64
		for i := 1; i < len(latencies); i++ {
			diff := float64(latencies[i].Nanoseconds() - latencies[i-1].Nanoseconds())
			jitterSum += math.Abs(diff)
		}
		stats.Jitter = time.Duration(jitterSum / float64(len(latencies)-1))
	}

	stats.SuccessRate = float64(stats.Received) / float64(stats.Sent) * 100

	return stats
}

// calculateChecksum calculates the ICMP checksum
func calculateChecksum(data []byte) uint16 {
	data[2] = 0
	data[3] = 0

	var sum uint32

	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 + uint32(data[i+1])
	}

	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	for (sum >> 16) > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(^sum)
}

// calculateComparisonScores calculates comparison scores for TCP/UDP tests
func (lt *LatencyTester) calculateComparisonScores(result *ComparisonResult) {
	tcpv4Score := 0.0
	tcpv6Score := 0.0
	udpv4Score := 0.0
	udpv6Score := 0.0

	if result.TCPv4Stats != nil && result.TCPv4Stats.Received > 0 {
		successRate := float64(result.TCPv4Stats.Received) / float64(result.TCPv4Stats.Sent)
		avgLatencyMs := float64(result.TCPv4Stats.Avg.Nanoseconds()) / 1e6
		tcpv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.TCPv6Stats != nil && result.TCPv6Stats.Received > 0 {
		successRate := float64(result.TCPv6Stats.Received) / float64(result.TCPv6Stats.Sent)
		avgLatencyMs := float64(result.TCPv6Stats.Avg.Nanoseconds()) / 1e6
		tcpv6Score = successRate * (1000 / avgLatencyMs)
	}

	if result.UDPv4Stats != nil && result.UDPv4Stats.Received > 0 {
		successRate := float64(result.UDPv4Stats.Received) / float64(result.UDPv4Stats.Sent)
		avgLatencyMs := float64(result.UDPv4Stats.Avg.Nanoseconds()) / 1e6
		udpv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.UDPv6Stats != nil && result.UDPv6Stats.Received > 0 {
		successRate := float64(result.UDPv6Stats.Received) / float64(result.UDPv6Stats.Sent)
		avgLatencyMs := float64(result.UDPv6Stats.Avg.Nanoseconds()) / 1e6
		udpv6Score = successRate * (1000 / avgLatencyMs)
	}

	result.IPv4Score = (tcpv4Score * 0.6) + (udpv4Score * 0.4)
	result.IPv6Score = (tcpv6Score * 0.6) + (udpv6Score * 0.4)

	if result.IPv4Score > result.IPv6Score {
		result.Winner = "IPv4"
	} else if result.IPv6Score > result.IPv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}

// calculateICMPComparisonScores calculates comparison scores for ICMP tests
func (lt *LatencyTester) calculateICMPComparisonScores(result *ComparisonResult) {
	ipv4Score := 0.0
	ipv6Score := 0.0

	if result.ICMPv4Stats != nil && result.ICMPv4Stats.Received > 0 {
		successRate := float64(result.ICMPv4Stats.Received) / float64(result.ICMPv4Stats.Sent)
		avgLatencyMs := float64(result.ICMPv4Stats.Avg.Nanoseconds()) / 1e6
		ipv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.ICMPv6Stats != nil && result.ICMPv6Stats.Received > 0 {
		successRate := float64(result.ICMPv6Stats.Received) / float64(result.ICMPv6Stats.Sent)
		avgLatencyMs := float64(result.ICMPv6Stats.Avg.Nanoseconds()) / 1e6
		ipv6Score = successRate * (1000 / avgLatencyMs)
	}

	result.IPv4Score = ipv4Score
	result.IPv6Score = ipv6Score

	if result.IPv4Score > result.IPv6Score {
		result.Winner = "IPv4"
	} else if result.IPv6Score > result.IPv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}

// calculateHTTPComparisonScores calculates comparison scores for HTTP tests
func (lt *LatencyTester) calculateHTTPComparisonScores(result *ComparisonResult) {
	ipv4Score := 0.0
	ipv6Score := 0.0

	if result.HTTPv4Stats != nil && result.HTTPv4Stats.Received > 0 {
		successRate := float64(result.HTTPv4Stats.Received) / float64(result.HTTPv4Stats.Sent)
		avgLatencyMs := float64(result.HTTPv4Stats.Avg.Nanoseconds()) / 1e6
		ipv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.HTTPv6Stats != nil && result.HTTPv6Stats.Received > 0 {
		successRate := float64(result.HTTPv6Stats.Received) / float64(result.HTTPv6Stats.Sent)
		avgLatencyMs := float64(result.HTTPv6Stats.Avg.Nanoseconds()) / 1e6
		ipv6Score = successRate * (1000 / avgLatencyMs)
	}

	result.IPv4Score = ipv4Score
	result.IPv6Score = ipv6Score

	if result.IPv4Score > result.IPv6Score {
		result.Winner = "IPv4"
	} else if result.IPv6Score > result.IPv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}

// calculateDNSComparisonScores calculates comparison scores for DNS tests
func (lt *LatencyTester) calculateDNSComparisonScores(result *ComparisonResult) {
	ipv4Score := 0.0
	ipv6Score := 0.0

	if result.DNSv4Stats != nil && result.DNSv4Stats.Received > 0 {
		successRate := float64(result.DNSv4Stats.Received) / float64(result.DNSv4Stats.Sent)
		avgLatencyMs := float64(result.DNSv4Stats.Avg.Nanoseconds()) / 1e6
		ipv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.DNSv6Stats != nil && result.DNSv6Stats.Received > 0 {
		successRate := float64(result.DNSv6Stats.Received) / float64(result.DNSv6Stats.Sent)
		avgLatencyMs := float64(result.DNSv6Stats.Avg.Nanoseconds()) / 1e6
		ipv6Score = successRate * (1000 / avgLatencyMs)
	}

	result.IPv4Score = ipv4Score
	result.IPv6Score = ipv6Score

	if result.IPv4Score > result.IPv6Score {
		result.Winner = "IPv4"
	} else if result.IPv6Score > result.IPv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}
