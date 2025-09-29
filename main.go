package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type PingResult struct {
	Success   bool
	Latency   time.Duration
	Error     error
	Timestamp time.Time
}

type Statistics struct {
	Sent     int
	Received int
	Lost     int
	Min      time.Duration
	Max      time.Duration
	Avg      time.Duration
	StdDev   time.Duration
	Jitter   time.Duration
	Latencies []time.Duration
}

type LatencyTester struct {
	target4     string
	target6     string
	hostname    string
	port        int
	count       int
	interval    time.Duration
	timeout     time.Duration
	size        int
	ipv4Only    bool
	ipv6Only    bool
	verbose     bool
	tcpMode     bool
	udpMode     bool
	icmpMode    bool
	httpMode    bool
	compareMode bool
	results4    []PingResult
	results6    []PingResult
	mu          sync.Mutex
}

type ComparisonResult struct {
	TCPv4Stats     Statistics
	TCPv6Stats     Statistics
	UDPv4Stats     Statistics
	UDPv6Stats     Statistics
	IPv4Score      float64
	IPv6Score      float64
	Winner         string
	ResolvedIPv4   string
	ResolvedIPv6   string
}

func main() {
	var (
		target4     = flag.String("4", "8.8.8.8", "IPv4 target address (auto-enables IPv4-only if custom)")
		target6     = flag.String("6", "2001:4860:4860::8888", "IPv6 target address (auto-enables IPv6-only if custom)")
		hostname    = flag.String("compare", "", "Compare mode: resolve hostname and test both TCP/UDP on IPv4/IPv6")
		port        = flag.Int("p", 53, "Port to test (for TCP/UDP/HTTP modes)")
		count       = flag.Int("c", 10, "Number of tests to perform")
		interval    = flag.Duration("i", time.Second, "Interval between tests")
		timeout     = flag.Duration("timeout", 3*time.Second, "Timeout for each test")
		size        = flag.Int("s", 64, "Packet size in bytes (ICMP only)")
		ipv4Only    = flag.Bool("4only", false, "Test IPv4 only")
		ipv6Only    = flag.Bool("6only", false, "Test IPv6 only")
		verbose     = flag.Bool("v", false, "Verbose output")
		tcpMode     = flag.Bool("t", false, "Use TCP connect test")
		udpMode     = flag.Bool("u", false, "Use UDP test")
		icmpMode    = flag.Bool("icmp", false, "Use ICMP ping test (may require root on some systems)")
		httpMode    = flag.Bool("http", false, "Use HTTP/HTTPS timing test")
	)
	flag.Parse()

	// Validate flags - only one protocol mode can be active
	modeCount := 0
	if *tcpMode {
		modeCount++
	}
	if *udpMode {
		modeCount++
	}
	if *icmpMode {
		modeCount++
	}
	if *httpMode {
		modeCount++
	}

	if modeCount > 1 {
		log.Fatal("Cannot specify multiple protocol flags (-t, -u, -icmp, -http) simultaneously")
	}

	// If no explicit mode is set, default to TCP
	if modeCount == 0 {
		*tcpMode = true
		modeCount = 1
	}

	compareMode := *hostname != ""
	if compareMode && (*tcpMode || *udpMode || *icmpMode || *httpMode) {
		log.Fatal("Compare mode cannot be used with protocol flags (compare mode tests both TCP and UDP)")
	}

	// Auto-enable single protocol mode when custom targets are specified
	defaultIPv4 := "8.8.8.8"
	defaultIPv6 := "2001:4860:4860::8888"

	// If user specified a custom IPv4 address but default IPv6, test IPv4 only
	if *target4 != defaultIPv4 && *target6 == defaultIPv6 && !*ipv6Only {
		*ipv4Only = true
	}

	// If user specified a custom IPv6 address but default IPv4, test IPv6 only
	if *target6 != defaultIPv6 && *target4 == defaultIPv4 && !*ipv4Only {
		*ipv6Only = true
	}

	tester := &LatencyTester{
		target4:     *target4,
		target6:     *target6,
		hostname:    *hostname,
		port:        *port,
		count:       *count,
		interval:    *interval,
		timeout:     *timeout,
		size:        *size,
		ipv4Only:    *ipv4Only,
		ipv6Only:    *ipv6Only,
		verbose:     *verbose,
		tcpMode:     *tcpMode,
		udpMode:     *udpMode,
		icmpMode:    *icmpMode,
		httpMode:    *httpMode,
		compareMode: compareMode,
	}

	if compareMode {
		tester.runCompareMode()
	} else {
		protocol := "TCP"
		if *udpMode {
			protocol = "UDP"
		} else if *icmpMode {
			protocol = "ICMP"
		} else if *httpMode {
			protocol = "HTTP/HTTPS"
		}

		fmt.Printf("High-Fidelity IPv4/IPv6 Latency Tester (%s)\n", protocol)
		fmt.Printf("===============================================\n\n")

		if !*ipv4Only {
			if *tcpMode || *udpMode || *httpMode {
				fmt.Printf("Testing IPv6 connectivity to [%s]:%d...\n", *target6, *port)
			} else {
				fmt.Printf("Testing IPv6 connectivity to %s...\n", *target6)
			}
			tester.testIPv6()
		}

		if !*ipv6Only {
			if *tcpMode || *udpMode || *httpMode {
				fmt.Printf("Testing IPv4 connectivity to %s:%d...\n", *target4, *port)
			} else {
				fmt.Printf("Testing IPv4 connectivity to %s...\n", *target4)
			}
			tester.testIPv4()
		}

		tester.printResults()
	}
}

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
		} else if lt.icmpMode {
			result = lt.testICMPv4(i+1)
		} else {
			// Default TCP mode
			result = lt.testTCPConnect("tcp4", lt.target4, i+1)
		}

		lt.mu.Lock()
		lt.results4 = append(lt.results4, result)
		lt.mu.Unlock()

		if lt.verbose {
			if result.Success {
				fmt.Printf("IPv4 test %d: %v\n", i+1, result.Latency)
			} else {
				fmt.Printf("IPv4 test %d: %v\n", i+1, result.Error)
			}
		}

		if i < lt.count-1 {
			time.Sleep(lt.interval)
		}
	}
}

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
		} else if lt.icmpMode {
			result = lt.testICMPv6(i+1)
		} else {
			// Default TCP mode
			result = lt.testTCPConnect("tcp6", lt.target6, i+1)
		}

		lt.mu.Lock()
		lt.results6 = append(lt.results6, result)
		lt.mu.Unlock()

		if lt.verbose {
			if result.Success {
				fmt.Printf("IPv6 test %d: %v\n", i+1, result.Latency)
			} else {
				fmt.Printf("IPv6 test %d: %v\n", i+1, result.Error)
			}
		}

		if i < lt.count-1 {
			time.Sleep(lt.interval)
		}
	}
}

func (lt *LatencyTester) testICMPv4(seq int) PingResult {
	// Try unprivileged ICMP first on Linux
	if runtime.GOOS == "linux" {
		result := lt.tryUnprivilegedICMPv4(seq)
		if result.Success || !strings.Contains(result.Error.Error(), "operation not permitted") {
			return result
		}
	}

	// Try raw socket ICMP
	result := lt.tryRawICMPv4(seq)
	if result.Success {
		return result
	}

	// If ICMP fails due to permissions, fall back to TCP
	if strings.Contains(result.Error.Error(), "operation not permitted") {
		if lt.verbose {
			fmt.Printf("ICMP failed (no root), falling back to TCP connect test...\n")
		}
		return lt.testTCPConnect("tcp4", lt.target4, seq)
	}

	return result
}

func (lt *LatencyTester) tryRawICMPv4(seq int) PingResult {
	// Create raw socket for IPv4 ICMP
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

func (lt *LatencyTester) tryUnprivilegedICMPv4(seq int) PingResult {
	// Try unprivileged ICMP socket on Linux
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_ICMP)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error creating IPv4 unprivileged ICMP socket: %v", err), Timestamp: time.Now()}
	}
	defer syscall.Close(fd)

	dst, err := net.ResolveIPAddr("ip4", lt.target4)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error resolving IPv4 address: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv4Unprivileged(fd, dst, seq)
}

func (lt *LatencyTester) sendICMPv4Unprivileged(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	// Create ICMP Echo Request packet
	packet := make([]byte, 8+lt.size) // 8 bytes ICMP header + data
	packet[0] = 8  // ICMP Echo Request
	packet[1] = 0  // Code
	packet[2] = 0  // Checksum (kernel will calculate for SOCK_DGRAM)
	packet[3] = 0  // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))  // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))  // Sequence

	// Fill data with timestamp for verification
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	// Create destination address structure
	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], dst.IP.To4())

	// Send packet
	err := syscall.Sendto(fd, packet, 0, addr)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Set socket timeout
	tv := syscall.Timeval{
		Sec:  int64(lt.timeout.Seconds()),
		Usec: int32(lt.timeout.Nanoseconds()/1000) % 1000000,
	}
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	// Read response
	reply := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		// For unprivileged sockets, we get ICMP directly without IP header
		if n < 8 { // Not enough for ICMP header
			continue
		}

		// Check if it's an ICMP Echo Reply
		if reply[0] == 0 { // ICMP Echo Reply
			replyID := binary.BigEndian.Uint16(reply[4:6])
			replySeq := binary.BigEndian.Uint16(reply[6:8])

			if int(replyID) == pid && int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

func (lt *LatencyTester) sendICMPv4Raw(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	// Create ICMP Echo Request packet
	packet := make([]byte, 8+lt.size) // 8 bytes ICMP header + data
	packet[0] = 8  // ICMP Echo Request
	packet[1] = 0  // Code
	packet[2] = 0  // Checksum (will be calculated)
	packet[3] = 0  // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))  // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))  // Sequence

	// Fill data with timestamp for verification
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	// Calculate checksum
	checksum := calculateChecksum(packet)
	binary.BigEndian.PutUint16(packet[2:4], checksum)

	// Create destination address structure
	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], dst.IP.To4())

	// Send packet
	err := syscall.Sendto(fd, packet, 0, addr)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Set socket timeout
	tv := syscall.Timeval{
		Sec:  int64(lt.timeout.Seconds()),
		Usec: int32(lt.timeout.Nanoseconds()/1000) % 1000000,
	}
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	// Read response
	reply := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		// Skip IP header (typically 20 bytes)
		if n < 28 { // IP header + ICMP header minimum
			continue
		}

		ipHeaderLen := int(reply[0]&0x0f) * 4
		if n < ipHeaderLen+8 { // Not enough for ICMP header
			continue
		}

		icmpPacket := reply[ipHeaderLen:]

		// Check if it's an ICMP Echo Reply
		if icmpPacket[0] == 0 { // ICMP Echo Reply
			replyID := binary.BigEndian.Uint16(icmpPacket[4:6])
			replySeq := binary.BigEndian.Uint16(icmpPacket[6:8])

			if int(replyID) == pid && int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

func (lt *LatencyTester) testICMPv6(seq int) PingResult {
	// Try unprivileged ICMP first on Linux
	if runtime.GOOS == "linux" {
		result := lt.tryUnprivilegedICMPv6(seq)
		if result.Success || !strings.Contains(result.Error.Error(), "operation not permitted") {
			return result
		}
	}

	// Try raw socket ICMP
	result := lt.tryRawICMPv6(seq)
	if result.Success {
		return result
	}

	// If ICMP fails due to permissions, fall back to TCP
	if strings.Contains(result.Error.Error(), "operation not permitted") {
		if lt.verbose {
			fmt.Printf("ICMP failed (no root), falling back to TCP connect test...\n")
		}
		return lt.testTCPConnect("tcp6", lt.target6, seq)
	}

	return result
}

func (lt *LatencyTester) tryRawICMPv6(seq int) PingResult {
	// Create raw socket for IPv6 ICMPv6
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

func (lt *LatencyTester) tryUnprivilegedICMPv6(seq int) PingResult {
	// Try unprivileged ICMP socket on Linux
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_DGRAM, syscall.IPPROTO_ICMPV6)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error creating IPv6 unprivileged ICMP socket: %v", err), Timestamp: time.Now()}
	}
	defer syscall.Close(fd)

	dst, err := net.ResolveIPAddr("ip6", lt.target6)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error resolving IPv6 address: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv6Unprivileged(fd, dst, seq)
}

func (lt *LatencyTester) sendICMPv6Unprivileged(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	// Create ICMPv6 Echo Request packet
	packet := make([]byte, 8+lt.size) // 8 bytes ICMPv6 header + data
	packet[0] = 128 // ICMPv6 Echo Request
	packet[1] = 0   // Code
	packet[2] = 0   // Checksum (kernel will calculate for SOCK_DGRAM)
	packet[3] = 0   // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))  // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))  // Sequence

	// Fill data with timestamp for verification
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	// Create destination address structure
	addr := &syscall.SockaddrInet6{}
	copy(addr.Addr[:], dst.IP.To16())

	// Send packet
	err := syscall.Sendto(fd, packet, 0, addr)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Set socket timeout
	tv := syscall.Timeval{
		Sec:  int64(lt.timeout.Seconds()),
		Usec: int32(lt.timeout.Nanoseconds()/1000) % 1000000,
	}
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	// Read response
	reply := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		// For unprivileged sockets, we get ICMPv6 directly
		if n < 8 { // Not enough for ICMPv6 header
			continue
		}

		// Check if it's an ICMPv6 Echo Reply
		if reply[0] == 129 { // ICMPv6 Echo Reply
			replyID := binary.BigEndian.Uint16(reply[4:6])
			replySeq := binary.BigEndian.Uint16(reply[6:8])

			if int(replyID) == pid && int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

func (lt *LatencyTester) sendICMPv6Raw(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	// Create ICMPv6 Echo Request packet
	packet := make([]byte, 8+lt.size) // 8 bytes ICMPv6 header + data
	packet[0] = 128 // ICMPv6 Echo Request
	packet[1] = 0   // Code
	packet[2] = 0   // Checksum (will be calculated by kernel for IPv6)
	packet[3] = 0   // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid))  // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq))  // Sequence

	// Fill data with timestamp for verification
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	// Create destination address structure
	addr := &syscall.SockaddrInet6{}
	copy(addr.Addr[:], dst.IP.To16())

	// Send packet
	err := syscall.Sendto(fd, packet, 0, addr)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Set socket timeout
	tv := syscall.Timeval{
		Sec:  int64(lt.timeout.Seconds()),
		Usec: int32(lt.timeout.Nanoseconds()/1000) % 1000000,
	}
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	// Read response
	reply := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, reply, 0)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}

		// ICMPv6 packets don't have IP header like IPv4
		if n < 8 { // Not enough for ICMPv6 header
			continue
		}

		// Check if it's an ICMPv6 Echo Reply
		if reply[0] == 129 { // ICMPv6 Echo Reply
			replyID := binary.BigEndian.Uint16(reply[4:6])
			replySeq := binary.BigEndian.Uint16(reply[6:8])

			if int(replyID) == pid && int(replySeq) == seq {
				latency := time.Since(start)
				return PingResult{Success: true, Latency: latency, Timestamp: start}
			}
		}
	}
}

func (lt *LatencyTester) testHTTP(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	// Determine if we should use HTTP or HTTPS based on port
	var scheme string
	if lt.port == 443 || lt.port == 8443 {
		scheme = "https"
	} else {
		scheme = "http"
	}

	// Construct URL
	var url string
	if ipVersion == "6" {
		url = fmt.Sprintf("%s://[%s]:%d/", scheme, target, lt.port)
	} else {
		url = fmt.Sprintf("%s://%s:%d/", scheme, target, lt.port)
	}

	// Create HTTP client with timeout and custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip cert verification for testing
		DisableKeepAlives: true,
	}

	// Force IPv4 or IPv6
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

	// Make HEAD request to minimize data transfer
	resp, err := client.Head(url)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

// calculateChecksum calculates the ICMP checksum
func calculateChecksum(data []byte) uint16 {
	// Clear checksum field
	data[2] = 0
	data[3] = 0

	var sum uint32

	// Sum all 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 + uint32(data[i+1])
	}

	// Add left-over byte, if any
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for (sum >> 16) > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(^sum)
}

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

	// For UDP, we need to actually send some data to test connectivity
	// since UDP is connectionless and Dial doesn't actually connect
	testData := []byte("test")
	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(testData)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Set read deadline and try to read (this may timeout, which is expected for many services)
	conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
	buffer := make([]byte, 1024)
	_, _ = conn.Read(buffer)

	latency := time.Since(start)

	// For UDP, we consider it successful if we could write to it
	// Even if read times out, the write success indicates the destination is reachable
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

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

func (lt *LatencyTester) runCompareMode() {
	fmt.Printf("High-Fidelity IPv4/IPv6 Comparison Mode\n")
	fmt.Printf("=======================================\n\n")

	fmt.Printf("Resolving %s...\n", lt.hostname)
	ipv4, ipv6, err := lt.resolveHostname(lt.hostname)
	if err != nil {
		log.Fatalf("Error resolving hostname: %v", err)
	}

	fmt.Printf("Resolved addresses:\n")
	if ipv4 != "" {
		fmt.Printf("  IPv4 (A): %s\n", ipv4)
	}
	if ipv6 != "" {
		fmt.Printf("  IPv6 (AAAA): %s\n", ipv6)
	}
	fmt.Printf("\n")

	if ipv4 == "" {
		log.Fatal("No IPv4 address found - cannot perform comparison")
	}
	if ipv6 == "" {
		log.Fatal("No IPv6 address found - cannot perform comparison")
	}

	// Override count to 10 for comparison mode
	originalCount := lt.count
	lt.count = 10

	result := &ComparisonResult{
		ResolvedIPv4: ipv4,
		ResolvedIPv6: ipv6,
	}

	// Test TCP IPv6
	fmt.Printf("Testing TCP IPv6 ([%s]:%d)...\n", ipv6, lt.port)
	lt.target6 = ipv6
	lt.tcpMode = true
	lt.udpMode = false
	lt.testIPv6()
	result.TCPv6Stats = lt.calculateStats(lt.results6)

	// Test TCP IPv4
	fmt.Printf("Testing TCP IPv4 (%s:%d)...\n", ipv4, lt.port)
	lt.target4 = ipv4
	lt.testIPv4()
	result.TCPv4Stats = lt.calculateStats(lt.results4)

	// Reset results and test UDP
	lt.results4 = nil
	lt.results6 = nil

	// Test UDP IPv6
	fmt.Printf("Testing UDP IPv6 ([%s]:%d)...\n", ipv6, lt.port)
	lt.tcpMode = false
	lt.udpMode = true
	lt.testIPv6()
	result.UDPv6Stats = lt.calculateStats(lt.results6)

	// Test UDP IPv4
	fmt.Printf("Testing UDP IPv4 (%s:%d)...\n", ipv4, lt.port)
	lt.testIPv4()
	result.UDPv4Stats = lt.calculateStats(lt.results4)

	// Restore original count
	lt.count = originalCount

	// Calculate scores and determine winner
	lt.calculateComparisonScores(result)
	lt.printComparisonResults(result)
}

func (lt *LatencyTester) calculateComparisonScores(result *ComparisonResult) {
	// Score calculation: lower latency and higher success rate are better
	// Formula: (success_rate / 100) * (1000 / avg_latency_ms)
	// This gives higher scores to faster, more reliable connections

	tcpv4Score := 0.0
	tcpv6Score := 0.0
	udpv4Score := 0.0
	udpv6Score := 0.0

	if result.TCPv4Stats.Received > 0 {
		successRate := float64(result.TCPv4Stats.Received) / float64(result.TCPv4Stats.Sent)
		avgLatencyMs := float64(result.TCPv4Stats.Avg.Nanoseconds()) / 1e6
		tcpv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.TCPv6Stats.Received > 0 {
		successRate := float64(result.TCPv6Stats.Received) / float64(result.TCPv6Stats.Sent)
		avgLatencyMs := float64(result.TCPv6Stats.Avg.Nanoseconds()) / 1e6
		tcpv6Score = successRate * (1000 / avgLatencyMs)
	}

	if result.UDPv4Stats.Received > 0 {
		successRate := float64(result.UDPv4Stats.Received) / float64(result.UDPv4Stats.Sent)
		avgLatencyMs := float64(result.UDPv4Stats.Avg.Nanoseconds()) / 1e6
		udpv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.UDPv6Stats.Received > 0 {
		successRate := float64(result.UDPv6Stats.Received) / float64(result.UDPv6Stats.Sent)
		avgLatencyMs := float64(result.UDPv6Stats.Avg.Nanoseconds()) / 1e6
		udpv6Score = successRate * (1000 / avgLatencyMs)
	}

	// Combined scores (TCP weighted 60%, UDP weighted 40%)
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

func (lt *LatencyTester) printComparisonResults(result *ComparisonResult) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("COMPREHENSIVE COMPARISON RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n\n")

	// TCP Results
	fmt.Printf("TCP Results\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	lt.printProtocolComparisonStats("IPv6", fmt.Sprintf("[%s]:%d", result.ResolvedIPv6, lt.port), result.TCPv6Stats)
	lt.printProtocolComparisonStats("IPv4", fmt.Sprintf("%s:%d", result.ResolvedIPv4, lt.port), result.TCPv4Stats)

	// UDP Results
	fmt.Printf("UDP Results\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	lt.printProtocolComparisonStats("IPv6", fmt.Sprintf("[%s]:%d", result.ResolvedIPv6, lt.port), result.UDPv6Stats)
	lt.printProtocolComparisonStats("IPv4", fmt.Sprintf("%s:%d", result.ResolvedIPv4, lt.port), result.UDPv4Stats)

	// Overall Comparison
	fmt.Printf("Overall Performance Ranking\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	fmt.Printf("IPv6 Score: %.2f\n", result.IPv6Score)
	fmt.Printf("IPv4 Score: %.2f\n", result.IPv4Score)
	fmt.Printf("\n🏆 Winner: %s", result.Winner)

	if result.Winner != "Tie" {
		scorePercent := 0.0
		if result.Winner == "IPv4" {
			scorePercent = ((result.IPv4Score - result.IPv6Score) / result.IPv6Score) * 100
		} else {
			scorePercent = ((result.IPv6Score - result.IPv4Score) / result.IPv4Score) * 100
		}
		fmt.Printf(" (%.1f%% better)\n", scorePercent)
	} else {
		fmt.Printf("\n")
	}

	fmt.Printf("\nScoring: Based on success rate and latency (lower latency + higher success = higher score)\n")
	fmt.Printf("Weighting: TCP 60%%, UDP 40%%\n\n")
}

func (lt *LatencyTester) printProtocolComparisonStats(protocol, target string, stats Statistics) {
	fmt.Printf("%s (%s):\n", protocol, target)
	if stats.Received > 0 {
		successRate := float64(stats.Received) / float64(stats.Sent) * 100
		fmt.Printf("  Success: %.1f%% (%d/%d)\n", successRate, stats.Received, stats.Sent)
		fmt.Printf("  Latency: avg=%.3fms min=%.3fms max=%.3fms\n",
			float64(stats.Avg.Nanoseconds())/1e6,
			float64(stats.Min.Nanoseconds())/1e6,
			float64(stats.Max.Nanoseconds())/1e6)
	} else {
		fmt.Printf("  Failed: No successful connections\n")
	}
	fmt.Printf("\n")
}

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

	return stats
}

func (lt *LatencyTester) printResults() {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("LATENCY TEST RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n\n")

	if !lt.ipv4Only && len(lt.results6) > 0 {
		stats6 := lt.calculateStats(lt.results6)
		lt.printProtocolStats("IPv6", lt.target6, stats6)
	}

	if !lt.ipv6Only && len(lt.results4) > 0 {
		stats4 := lt.calculateStats(lt.results4)
		lt.printProtocolStats("IPv4", lt.target4, stats4)
	}

	if !lt.ipv4Only && !lt.ipv6Only && len(lt.results4) > 0 && len(lt.results6) > 0 {
		lt.printComparison()
	}
}

func (lt *LatencyTester) printProtocolStats(protocol, target string, stats Statistics) {
	fmt.Printf("%s Results (%s)\n", protocol, target)
	fmt.Printf(strings.Repeat("-", 40) + "\n")

	testType := "Packets"
	if lt.tcpMode {
		testType = "Connections"
	} else if lt.udpMode {
		testType = "UDP Tests"
	} else if lt.httpMode {
		testType = "HTTP Requests"
	}

	lossType := "loss"
	if lt.tcpMode {
		lossType = "failed"
	} else if lt.udpMode {
		lossType = "failed"
	} else if lt.httpMode {
		lossType = "failed"
	}

	fmt.Printf("%s: %d sent, %d successful, %d %s (%.1f%% success)\n",
		testType, stats.Sent, stats.Received, stats.Lost,
		lossType, float64(stats.Received)/float64(stats.Sent)*100)

	if stats.Received > 0 {
		fmt.Printf("Latency: min=%.3fms avg=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(stats.Min.Nanoseconds())/1e6,
			float64(stats.Avg.Nanoseconds())/1e6,
			float64(stats.Max.Nanoseconds())/1e6,
			float64(stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n",
			float64(stats.Jitter.Nanoseconds())/1e6)

		if len(stats.Latencies) > 0 {
			percentiles := []int{50, 95, 99}
			fmt.Printf("Percentiles: ")
			for i, p := range percentiles {
				idx := int(float64(p)/100.0*float64(len(stats.Latencies))) - 1
				if idx < 0 {
					idx = 0
				}
				if idx >= len(stats.Latencies) {
					idx = len(stats.Latencies) - 1
				}
				fmt.Printf("P%d=%.3fms", p, float64(stats.Latencies[idx].Nanoseconds())/1e6)
				if i < len(percentiles)-1 {
					fmt.Printf(" ")
				}
			}
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n")
}

func (lt *LatencyTester) printComparison() {
	stats4 := lt.calculateStats(lt.results4)
	stats6 := lt.calculateStats(lt.results6)

	fmt.Printf("IPv6 vs IPv4 Comparison\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")

	if stats4.Received > 0 && stats6.Received > 0 {
		diff := float64(stats4.Avg.Nanoseconds()-stats6.Avg.Nanoseconds()) / 1e6
		faster := "IPv6"
		if diff < 0 {
			faster = "IPv4"
			diff = -diff
		}
		fmt.Printf("Average latency difference: %.3fms (%s is faster)\n", diff, faster)

		success6 := float64(stats6.Received) / float64(stats6.Sent) * 100
		success4 := float64(stats4.Received) / float64(stats4.Sent) * 100

		if lt.tcpMode || lt.udpMode || lt.httpMode {
			fmt.Printf("Success rate: IPv6=%.1f%% IPv4=%.1f%%\n", success6, success4)
		} else {
			loss6 := float64(stats6.Lost) / float64(stats6.Sent) * 100
			loss4 := float64(stats4.Lost) / float64(stats4.Sent) * 100
			fmt.Printf("Packet loss: IPv6=%.1f%% IPv4=%.1f%%\n", loss6, loss4)
		}
	}
	fmt.Printf("\n")
}