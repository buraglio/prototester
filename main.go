package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"gopkg.in/yaml.v3"
)

type PingResult struct {
	Success   bool          `json:"success"`
	Latency   time.Duration `json:"latency_ms"`
	Error     error         `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

type JSONOutput struct {
	Mode        string            `json:"mode"`
	Protocol    string            `json:"protocol"`
	Targets     map[string]string `json:"targets"`
	IPv4Results Statistics        `json:"ipv4_results,omitempty"`
	IPv6Results Statistics        `json:"ipv6_results,omitempty"`
	Comparison  *ComparisonResult `json:"comparison,omitempty"`
	TestConfig  TestConfig        `json:"test_config"`
	Timestamp   time.Time         `json:"timestamp"`
}

type TestConfig struct {
	Count       int           `json:"count"`
	Interval    time.Duration `json:"interval_ms"`
	Timeout     time.Duration `json:"timeout_ms"`
	Port        int           `json:"port"`
	Size        int           `json:"size,omitempty"`
	DNSQuery    string        `json:"dns_query,omitempty"`
	DNSProtocol string        `json:"dns_protocol,omitempty"`
	Verbose     bool          `json:"verbose"`
}

type Statistics struct {
	Sent        int             `json:"sent"`
	Received    int             `json:"received"`
	Lost        int             `json:"lost"`
	Min         time.Duration   `json:"min_ms"`
	Max         time.Duration   `json:"max_ms"`
	Avg         time.Duration   `json:"avg_ms"`
	StdDev      time.Duration   `json:"stddev_ms"`
	Jitter      time.Duration   `json:"jitter_ms"`
	Latencies   []time.Duration `json:"-"`
	SuccessRate float64         `json:"success_rate"`
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
	dnsMode     bool
	dnsProtocol string // "udp", "tcp", "dot", "doh"
	dnsQuery    string // domain to query
	compareMode bool
	jsonOutput  bool
	results4    []PingResult
	results6    []PingResult
	mu          sync.Mutex
}

type ComparisonResult struct {
	TCPv4Stats   Statistics `json:"tcp_v4_stats,omitempty"`
	TCPv6Stats   Statistics `json:"tcp_v6_stats,omitempty"`
	UDPv4Stats   Statistics `json:"udp_v4_stats,omitempty"`
	UDPv6Stats   Statistics `json:"udp_v6_stats,omitempty"`
	DNSv4Stats   Statistics `json:"dns_v4_stats,omitempty"`
	DNSv6Stats   Statistics `json:"dns_v6_stats,omitempty"`
	HTTPv4Stats  Statistics `json:"http_v4_stats,omitempty"`
	HTTPv6Stats  Statistics `json:"http_v6_stats,omitempty"`
	ICMPv4Stats  Statistics `json:"icmp_v4_stats,omitempty"`
	ICMPv6Stats  Statistics `json:"icmp_v6_stats,omitempty"`
	IPv4Score    float64    `json:"ipv4_score"`
	IPv6Score    float64    `json:"ipv6_score"`
	Winner       string     `json:"winner"`
	ResolvedIPv4 string     `json:"resolved_ipv4"`
	ResolvedIPv6 string     `json:"resolved_ipv6"`
	Protocol     string     `json:"protocol"`
	Hostname     string     `json:"hostname"`
	Port         int        `json:"port"`
	DNSQuery     string     `json:"dns_query,omitempty"`
	Timestamp    time.Time  `json:"timestamp"`
}

// DNS query structures
type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

type DoHRequest struct {
	Questions []DoHQuestion `json:"question"`
}

type DoHQuestion struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

// Configuration file structures
type Config struct {
	Global GlobalConfig `yaml:"global" json:"global"`
	Tests  []TestSpec   `yaml:"tests" json:"tests"`
	Daemon DaemonConfig `yaml:"daemon" json:"daemon"`
}

type GlobalConfig struct {
	OutputFile   string         `yaml:"output_file" json:"output_file"`
	LogLevel     string         `yaml:"log_level" json:"log_level"`
	DefaultCount int            `yaml:"default_count" json:"default_count"`
	Timeout      time.Duration  `yaml:"timeout" json:"timeout"`
	Interval     time.Duration  `yaml:"interval" json:"interval"`
	JSONOutput   bool           `yaml:"json_output" json:"json_output"`
	InfluxDB     InfluxDBConfig `yaml:"influxdb" json:"influxdb"`
}

type InfluxDBConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	URL           string        `yaml:"url" json:"url"`
	Token         string        `yaml:"token" json:"token"`
	Organization  string        `yaml:"organization" json:"organization"`
	Bucket        string        `yaml:"bucket" json:"bucket"`
	Measurement   string        `yaml:"measurement" json:"measurement"`
	BatchSize     int           `yaml:"batch_size" json:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval"`
}

type TestSpec struct {
	Name        string        `yaml:"name" json:"name"`
	Type        string        `yaml:"type" json:"type"` // tcp, udp, icmp, http, dns, compare
	Target4     string        `yaml:"target_ipv4" json:"target_ipv4"`
	Target6     string        `yaml:"target_ipv6" json:"target_ipv6"`
	Hostname    string        `yaml:"hostname" json:"hostname"` // for compare mode
	Port        int           `yaml:"port" json:"port"`
	Count       int           `yaml:"count" json:"count"`
	Interval    time.Duration `yaml:"interval" json:"interval"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	Size        int           `yaml:"size" json:"size"` // ICMP packet size
	DNSProtocol string        `yaml:"dns_protocol" json:"dns_protocol"`
	DNSQuery    string        `yaml:"dns_query" json:"dns_query"`
	IPv4Only    bool          `yaml:"ipv4_only" json:"ipv4_only"`
	IPv6Only    bool          `yaml:"ipv6_only" json:"ipv6_only"`
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	Schedule    string        `yaml:"schedule" json:"schedule"` // cron-like schedule
}

type DaemonConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	RunInterval   time.Duration `yaml:"run_interval" json:"run_interval"`
	OutputFile    string        `yaml:"output_file" json:"output_file"`
	LogFile       string        `yaml:"log_file" json:"log_file"`
	PidFile       string        `yaml:"pid_file" json:"pid_file"`
	MaxLogSize    int64         `yaml:"max_log_size" json:"max_log_size"`
	RotateLogs    bool          `yaml:"rotate_logs" json:"rotate_logs"`
	StopOnFailure bool          `yaml:"stop_on_failure" json:"stop_on_failure"`
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`
}

type DaemonResult struct {
	TestName  string      `json:"test_name"`
	Timestamp time.Time   `json:"timestamp"`
	TestType  string      `json:"test_type"`
	Target    string      `json:"target"`
	Success   bool        `json:"success"`
	Results   interface{} `json:"results"`
	Error     string      `json:"error,omitempty"`
	Duration  float64     `json:"duration_seconds"`
}

// Global InfluxDB client
var influxClient influxdb2.Client

func initInfluxDB(config InfluxDBConfig) error {
	if !config.Enabled {
		return nil
	}

	influxClient = influxdb2.NewClient(config.URL, config.Token)

	// Test connection
	health, err := influxClient.Health(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return fmt.Errorf("InfluxDB health check failed: %s", msg)
	}

	log.Printf("InfluxDB connection established to %s", config.URL)
	return nil
}

func writeToInfluxDB(config InfluxDBConfig, testName, testType, target string, stats Statistics, tags map[string]string) error {
	if !config.Enabled || influxClient == nil {
		return nil
	}

	// Create write API
	writeAPI := influxClient.WriteAPIBlocking(config.Organization, config.Bucket)

	// Set default measurement name if not specified
	measurement := config.Measurement
	if measurement == "" {
		measurement = "network_latency"
	}

	// Prepare tags
	allTags := map[string]string{
		"test_name": testName,
		"test_type": testType,
		"target":    target,
	}

	// Add custom tags
	for k, v := range tags {
		allTags[k] = v
	}

	// Prepare fields
	fields := map[string]interface{}{
		"sent":         stats.Sent,
		"received":     stats.Received,
		"lost":         stats.Lost,
		"min_ms":       float64(stats.Min.Nanoseconds()) / 1e6,
		"max_ms":       float64(stats.Max.Nanoseconds()) / 1e6,
		"avg_ms":       float64(stats.Avg.Nanoseconds()) / 1e6,
		"stddev_ms":    float64(stats.StdDev.Nanoseconds()) / 1e6,
		"jitter_ms":    float64(stats.Jitter.Nanoseconds()) / 1e6,
		"success_rate": stats.SuccessRate,
	}

	// Create point
	point := influxdb2.NewPoint(measurement, allTags, fields, time.Now())

	// Write point
	err := writeAPI.WritePoint(context.Background(), point)
	if err != nil {
		return fmt.Errorf("failed to write to InfluxDB: %w", err)
	}

	return nil
}

func writeResultToInfluxDB(config InfluxDBConfig, result DaemonResult) {
	if !config.Enabled || influxClient == nil {
		return
	}

	// Extract statistics from the results interface{}
	var stats4, stats6 *Statistics
	if result.Results != nil {
		if jsonData, ok := result.Results.(map[string]interface{}); ok {
			// Handle IPv4 results
			if ipv4Data, exists := jsonData["ipv4_results"]; exists {
				if ipv4Map, ok := ipv4Data.(map[string]interface{}); ok {
					stats4 = extractStatsFromMap(ipv4Map)
				}
			}
			// Handle IPv6 results
			if ipv6Data, exists := jsonData["ipv6_results"]; exists {
				if ipv6Map, ok := ipv6Data.(map[string]interface{}); ok {
					stats6 = extractStatsFromMap(ipv6Map)
				}
			}
		}
	}

	// Write IPv4 results if available
	if stats4 != nil {
		tags := map[string]string{
			"ip_version": "4",
		}
		if err := writeToInfluxDB(config, result.TestName, result.TestType, result.Target, *stats4, tags); err != nil {
			log.Printf("Error writing IPv4 results to InfluxDB: %v", err)
		}
	}

	// Write IPv6 results if available
	if stats6 != nil {
		tags := map[string]string{
			"ip_version": "6",
		}
		if err := writeToInfluxDB(config, result.TestName, result.TestType, result.Target, *stats6, tags); err != nil {
			log.Printf("Error writing IPv6 results to InfluxDB: %v", err)
		}
	}
}

func extractStatsFromMap(data map[string]interface{}) *Statistics {
	getFloat := func(key string) float64 {
		if val, ok := data[key]; ok {
			switch v := val.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			}
		}
		return 0
	}

	getDuration := func(key string) time.Duration {
		ms := getFloat(key)
		return time.Duration(ms * 1e6) // Convert ms to nanoseconds
	}

	return &Statistics{
		Sent:        int(getFloat("sent")),
		Received:    int(getFloat("received")),
		Lost:        int(getFloat("lost")),
		Min:         getDuration("min_ms"),
		Max:         getDuration("max_ms"),
		Avg:         getDuration("avg_ms"),
		StdDev:      getDuration("stddev_ms"),
		Jitter:      getDuration("jitter_ms"),
		SuccessRate: getFloat("success_rate"),
	}
}

func closeInfluxDB() {
	if influxClient != nil {
		influxClient.Close()
	}
}

func main() {
	var (
		target4     = flag.String("4", "8.8.8.8", "IPv4 target address (auto-enables IPv4-only if custom)")
		target6     = flag.String("6", "2001:4860:4860::8888", "IPv6 target address (auto-enables IPv6-only if custom)")
		hostname    = flag.String("compare", "", "Compare mode: resolve hostname and test protocols on both IPv4/IPv6 (TCP/UDP by default, or use -icmp, -http, -dns for specific protocol)")
		port        = flag.Int("p", 53, "Port to test (for TCP/UDP/HTTP/DNS modes)")
		count       = flag.Int("c", 10, "Number of tests to perform")
		interval    = flag.Duration("i", time.Second, "Interval between tests")
		timeout     = flag.Duration("timeout", 3*time.Second, "Timeout for each test")
		size        = flag.Int("s", 64, "Packet size in bytes (ICMP only)")
		ipv4Only    = flag.Bool("4only", false, "Test IPv4 only")
		ipv6Only    = flag.Bool("6only", false, "Test IPv6 only")
		verbose     = flag.Bool("v", false, "Verbose output")
		tcpMode     = flag.Bool("t", false, "Use TCP connect test (default mode)")
		udpMode     = flag.Bool("u", false, "Use UDP test")
		icmpMode    = flag.Bool("icmp", false, "Use ICMP ping test (auto-fallback to TCP if no root permissions)")
		httpMode    = flag.Bool("http", false, "Use HTTP/HTTPS HEAD request timing test (HTTPS on ports 443/8443)")
		dnsMode     = flag.Bool("dns", false, "Use DNS query testing (supports UDP, TCP, DoT, DoH protocols)")
		dnsProtocol = flag.String("dns-protocol", "udp", "DNS protocol: udp, tcp, dot, doh")
		dnsQuery    = flag.String("dns-query", "dns-query.qosbox.com", "Domain name to query for DNS testing")
		jsonOutput  = flag.Bool("json", false, "Output results in JSON format instead of human-readable text")
		configFile  = flag.String("config", "", "Configuration file (YAML or JSON format)")
		daemon      = flag.Bool("daemon", false, "Run in daemon mode using configuration file")
		outputFile  = flag.String("output", "", "Output file for results (stdout if not specified)")
	)
	flag.Parse()

	// Handle configuration file and daemon mode
	if *configFile != "" || *daemon {
		if *configFile == "" {
			log.Fatal("Configuration file required for daemon mode. Use -config flag.")
		}
		runWithConfig(*configFile, *daemon, *outputFile)
		return
	}

	// Validate DNS protocol
	validDNSProtocols := map[string]bool{
		"udp": true,
		"tcp": true,
		"dot": true,
		"doh": true,
	}
	if !validDNSProtocols[*dnsProtocol] {
		log.Fatal("Invalid DNS protocol. Must be one of: udp, tcp, dot, doh")
	}

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
	if *dnsMode {
		modeCount++
	}

	if modeCount > 1 {
		log.Fatal("Cannot specify multiple protocol flags (-t, -u, -icmp, -http, -dns) simultaneously")
	}

	compareMode := *hostname != ""

	// If no explicit mode is set, default to TCP (unless in compare mode which handles its own defaults)
	if modeCount == 0 && !compareMode {
		*tcpMode = true
		modeCount = 1
	}

	if compareMode && (*tcpMode || *udpMode) {
		log.Fatal("Compare mode cannot be used with -t or -u flags (compare mode tests TCP/UDP by default, or use -icmp, -http, or -dns for specific protocol comparison)")
	}

	// Special handling for DNS compare mode
	if compareMode && *dnsMode {
		// DNS compare mode: test the specified DNS protocol across IPv4/IPv6
		// This is allowed and will test the same DNS protocol on both IP versions
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
		dnsMode:     *dnsMode,
		dnsProtocol: *dnsProtocol,
		dnsQuery:    *dnsQuery,
		compareMode: compareMode,
		jsonOutput:  *jsonOutput,
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
		} else if *dnsMode {
			protocol = fmt.Sprintf("DNS (%s)", strings.ToUpper(*dnsProtocol))
		}

		fmt.Printf("High-Fidelity IPv4/IPv6 Latency Tester (%s)\n", protocol)
		fmt.Printf("===============================================\n\n")

		if !*ipv4Only {
			if *tcpMode || *udpMode || *httpMode || *dnsMode {
				if *dnsMode {
					fmt.Printf("Testing IPv6 DNS to [%s]:%d (query: %s)...\n", *target6, *port, *dnsQuery)
				} else {
					fmt.Printf("Testing IPv6 connectivity to [%s]:%d...\n", *target6, *port)
				}
			} else {
				fmt.Printf("Testing IPv6 connectivity to %s...\n", *target6)
			}
			tester.testIPv6()
		}

		if !*ipv6Only {
			if *tcpMode || *udpMode || *httpMode || *dnsMode {
				if *dnsMode {
					fmt.Printf("Testing IPv4 DNS to %s:%d (query: %s)...\n", *target4, *port, *dnsQuery)
				} else {
					fmt.Printf("Testing IPv4 connectivity to %s:%d...\n", *target4, *port)
				}
			} else {
				fmt.Printf("Testing IPv4 connectivity to %s...\n", *target4)
			}
			tester.testIPv4()
		}

		if tester.jsonOutput {
			tester.printJSONResults()
		} else {
			tester.printResults()
		}
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
		} else if lt.dnsMode {
			result = lt.testDNS("4", lt.target4, i+1)
		} else if lt.icmpMode {
			result = lt.testICMPv4(i + 1)
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
		} else if lt.dnsMode {
			result = lt.testDNS("6", lt.target6, i+1)
		} else if lt.icmpMode {
			result = lt.testICMPv6(i + 1)
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
	// TODO: Unprivileged ICMP on Linux requires more investigation
	// Skipping for now and using raw sockets directly

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

	// Connect the socket to the destination
	addr := &syscall.SockaddrInet4{}
	copy(addr.Addr[:], dst.IP.To4())
	err = syscall.Connect(fd, addr)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error connecting socket: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv4Unprivileged(fd, dst, seq)
}

func (lt *LatencyTester) sendICMPv4Unprivileged(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	// Create ICMP Echo Request packet
	packet := make([]byte, 8+lt.size)                    // 8 bytes ICMP header + data
	packet[0] = 8                                        // ICMP Echo Request
	packet[1] = 0                                        // Code
	packet[2] = 0                                        // Checksum (kernel will calculate for SOCK_DGRAM)
	packet[3] = 0                                        // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid)) // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq)) // Sequence

	// Fill data with timestamp for verification
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	// Send packet (socket is already connected)
	_, err := syscall.Write(fd, packet)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Use select to wait for response with timeout
	tv := syscall.Timeval{
		Sec:  int64(lt.timeout.Seconds()),
		Usec: int64(lt.timeout.Nanoseconds()/1000) % 1000000,
	}

	// Read response
	reply := make([]byte, 1500)
	for {
		// Wait for socket to be readable
		fdSet := &syscall.FdSet{}
		fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)

		n, err := syscall.Select(fd+1, fdSet, nil, nil, &tv)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}
		if n == 0 {
			return PingResult{Success: false, Error: fmt.Errorf("timeout"), Timestamp: start}
		}

		n, _, err = syscall.Recvfrom(fd, reply, 0)
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
	packet := make([]byte, 8+lt.size)                    // 8 bytes ICMP header + data
	packet[0] = 8                                        // ICMP Echo Request
	packet[1] = 0                                        // Code
	packet[2] = 0                                        // Checksum (will be calculated)
	packet[3] = 0                                        // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid)) // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq)) // Sequence

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
		Usec: int64(lt.timeout.Nanoseconds()/1000) % 1000000,
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
	// TODO: Unprivileged ICMP on Linux requires more investigation
	// Skipping for now and using raw sockets directly

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

	// Connect the socket to the destination
	addr := &syscall.SockaddrInet6{}
	copy(addr.Addr[:], dst.IP.To16())
	err = syscall.Connect(fd, addr)
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("error connecting socket: %v", err), Timestamp: time.Now()}
	}

	return lt.sendICMPv6Unprivileged(fd, dst, seq)
}

func (lt *LatencyTester) sendICMPv6Unprivileged(fd int, dst *net.IPAddr, seq int) PingResult {
	start := time.Now()
	pid := os.Getpid() & 0xffff

	// Create ICMPv6 Echo Request packet
	packet := make([]byte, 8+lt.size)                    // 8 bytes ICMPv6 header + data
	packet[0] = 128                                      // ICMPv6 Echo Request
	packet[1] = 0                                        // Code
	packet[2] = 0                                        // Checksum (kernel will calculate for SOCK_DGRAM)
	packet[3] = 0                                        // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid)) // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq)) // Sequence

	// Fill data with timestamp for verification
	binary.BigEndian.PutUint64(packet[8:16], uint64(start.UnixNano()))

	// Send packet (socket is already connected)
	_, err := syscall.Write(fd, packet)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Use select to wait for response with timeout
	tv := syscall.Timeval{
		Sec:  int64(lt.timeout.Seconds()),
		Usec: int64(lt.timeout.Nanoseconds()/1000) % 1000000,
	}

	// Read response
	reply := make([]byte, 1500)
	for {
		// Wait for socket to be readable
		fdSet := &syscall.FdSet{}
		fdSet.Bits[fd/64] |= 1 << (uint(fd) % 64)

		n, err := syscall.Select(fd+1, fdSet, nil, nil, &tv)
		if err != nil {
			return PingResult{Success: false, Error: err, Timestamp: start}
		}
		if n == 0 {
			return PingResult{Success: false, Error: fmt.Errorf("timeout"), Timestamp: start}
		}

		n, _, err = syscall.Recvfrom(fd, reply, 0)
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
	packet := make([]byte, 8+lt.size)                    // 8 bytes ICMPv6 header + data
	packet[0] = 128                                      // ICMPv6 Echo Request
	packet[1] = 0                                        // Code
	packet[2] = 0                                        // Checksum (will be calculated by kernel for IPv6)
	packet[3] = 0                                        // Checksum
	binary.BigEndian.PutUint16(packet[4:6], uint16(pid)) // ID
	binary.BigEndian.PutUint16(packet[6:8], uint16(seq)) // Sequence

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
		Usec: int64(lt.timeout.Nanoseconds()/1000) % 1000000,
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
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, // Skip cert verification for testing
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

func (lt *LatencyTester) testDNSUDP(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	// Build DNS query packet
	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	// Create UDP connection
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

	// Send DNS query
	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(queryPacket)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Read DNS response
	conn.SetReadDeadline(time.Now().Add(lt.timeout))
	response := make([]byte, 512) // Standard DNS UDP response size
	n, err := conn.Read(response)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Validate DNS response
	if n < 12 { // Minimum DNS header size
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", n), Timestamp: start}
	}

	// Check if response ID matches query ID
	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

func (lt *LatencyTester) testDNSTCP(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	// Build DNS query packet
	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	// Create TCP connection
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

	// TCP DNS requires length prefix (2 bytes)
	lengthPrefix := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthPrefix, uint16(len(queryPacket)))
	tcpQuery := append(lengthPrefix, queryPacket...)

	// Send DNS query
	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(tcpQuery)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Read response length
	conn.SetReadDeadline(time.Now().Add(lt.timeout))
	lengthBytes := make([]byte, 2)
	_, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	responseLength := binary.BigEndian.Uint16(lengthBytes)
	if responseLength > 4096 { // Sanity check
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too large: %d bytes", responseLength), Timestamp: start}
	}

	// Read DNS response
	response := make([]byte, responseLength)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Validate DNS response
	if len(response) < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", len(response)), Timestamp: start}
	}

	// Check if response ID matches query ID
	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

func (lt *LatencyTester) testDNSDoT(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	// Build DNS query packet
	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	// Create TLS connection
	var address string
	if ipVersion == "6" {
		address = fmt.Sprintf("[%s]:%d", target, lt.port)
	} else {
		address = fmt.Sprintf("%s:%d", target, lt.port)
	}

	config := &tls.Config{
		InsecureSkipVerify: true, // For testing purposes
		ServerName:         target,
	}

	dialer := &net.Dialer{Timeout: lt.timeout}
	network := "tcp" + ipVersion
	conn, err := tls.DialWithDialer(dialer, network, address, config)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer conn.Close()

	// TCP DNS requires length prefix (2 bytes)
	lengthPrefix := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthPrefix, uint16(len(queryPacket)))
	tcpQuery := append(lengthPrefix, queryPacket...)

	// Send DNS query
	conn.SetWriteDeadline(time.Now().Add(lt.timeout))
	_, err = conn.Write(tcpQuery)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Read response length
	conn.SetReadDeadline(time.Now().Add(lt.timeout))
	lengthBytes := make([]byte, 2)
	_, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	responseLength := binary.BigEndian.Uint16(lengthBytes)
	if responseLength > 4096 { // Sanity check
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too large: %d bytes", responseLength), Timestamp: start}
	}

	// Read DNS response
	response := make([]byte, responseLength)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Validate DNS response
	if len(response) < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", len(response)), Timestamp: start}
	}

	// Check if response ID matches query ID
	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

func (lt *LatencyTester) testDNSDoH(ipVersion, target string, seq int) PingResult {
	start := time.Now()

	// Build DNS query packet
	queryPacket, err := lt.buildDNSQuery()
	if err != nil {
		return PingResult{Success: false, Error: fmt.Errorf("failed to build DNS query: %v", err), Timestamp: start}
	}

	// Create HTTPS URL
	var baseURL string
	var port int
	if lt.port == 443 {
		port = 443
	} else {
		port = lt.port
	}

	if ipVersion == "6" {
		baseURL = fmt.Sprintf("https://[%s]:%d/dns-query", target, port)
	} else {
		baseURL = fmt.Sprintf("https://%s:%d/dns-query", target, port)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", baseURL, bytes.NewReader(queryPacket))
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // For testing purposes
		},
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

	// Make HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return PingResult{Success: false, Error: fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status), Timestamp: start}
	}

	// Read DNS response
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return PingResult{Success: false, Error: err, Timestamp: start}
	}

	// Validate DNS response
	if len(response) < 12 {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response too short: %d bytes", len(response)), Timestamp: start}
	}

	// Check if response ID matches query ID
	responseID := binary.BigEndian.Uint16(response[0:2])
	queryID := binary.BigEndian.Uint16(queryPacket[0:2])
	if responseID != queryID {
		return PingResult{Success: false, Error: fmt.Errorf("DNS response ID mismatch: got %d, expected %d", responseID, queryID), Timestamp: start}
	}

	latency := time.Since(start)
	return PingResult{Success: true, Latency: latency, Timestamp: start}
}

func (lt *LatencyTester) buildDNSQuery() ([]byte, error) {
	// Generate random query ID
	queryID := make([]byte, 2)
	_, err := rand.Read(queryID)
	if err != nil {
		return nil, err
	}

	// Build DNS header
	header := DNSHeader{
		ID:      binary.BigEndian.Uint16(queryID),
		Flags:   0x0100, // Standard query, recursion desired
		QDCount: 1,      // One question
		ANCount: 0,
		NSCount: 0,
		ARCount: 0,
	}

	// Build DNS question
	question := DNSQuestion{
		Name:  lt.dnsQuery,
		Type:  1, // A record
		Class: 1, // IN class
	}

	// Serialize DNS packet
	packet := make([]byte, 0, 512)

	// Add header
	headerBytes := make([]byte, 12)
	binary.BigEndian.PutUint16(headerBytes[0:2], header.ID)
	binary.BigEndian.PutUint16(headerBytes[2:4], header.Flags)
	binary.BigEndian.PutUint16(headerBytes[4:6], header.QDCount)
	binary.BigEndian.PutUint16(headerBytes[6:8], header.ANCount)
	binary.BigEndian.PutUint16(headerBytes[8:10], header.NSCount)
	binary.BigEndian.PutUint16(headerBytes[10:12], header.ARCount)
	packet = append(packet, headerBytes...)

	// Add question
	// Encode domain name
	domainParts := strings.Split(question.Name, ".")
	for _, part := range domainParts {
		if len(part) > 63 {
			return nil, fmt.Errorf("domain label too long: %s", part)
		}
		packet = append(packet, byte(len(part)))
		packet = append(packet, []byte(part)...)
	}
	packet = append(packet, 0) // Null terminator

	// Add type and class
	typeClassBytes := make([]byte, 4)
	binary.BigEndian.PutUint16(typeClassBytes[0:2], question.Type)
	binary.BigEndian.PutUint16(typeClassBytes[2:4], question.Class)
	packet = append(packet, typeClassBytes...)

	return packet, nil
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
	if lt.dnsMode {
		lt.runDNSCompareMode()
		return
	}
	if lt.icmpMode {
		lt.runICMPCompareMode()
		return
	}
	if lt.httpMode {
		lt.runHTTPCompareMode()
		return
	}

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
	lt.dnsMode = false
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
	result.Protocol = "TCP/UDP"
	result.Hostname = lt.hostname
	result.Port = lt.port
	result.Timestamp = time.Now()

	if lt.jsonOutput {
		lt.printJSONComparisonResults(result)
	} else {
		lt.printComparisonResults(result)
	}
}

func (lt *LatencyTester) runDNSCompareMode() {
	fmt.Printf("High-Fidelity IPv4/IPv6 DNS Comparison Mode (%s)\n", strings.ToUpper(lt.dnsProtocol))
	fmt.Printf("================================================\n\n")

	fmt.Printf("Resolving %s...\n", lt.hostname)
	ipv4, ipv6, err := lt.resolveHostname(lt.hostname)
	if err != nil {
		log.Fatalf("Error resolving hostname: %v", err)
	}

	fmt.Printf("Resolved DNS servers:\n")
	if ipv4 != "" {
		fmt.Printf("  IPv4 (A): %s\n", ipv4)
	}
	if ipv6 != "" {
		fmt.Printf("  IPv6 (AAAA): %s\n", ipv6)
	}
	fmt.Printf("\n")

	if ipv4 == "" {
		log.Fatal("No IPv4 address found - cannot perform DNS comparison")
	}
	if ipv6 == "" {
		log.Fatal("No IPv6 address found - cannot perform DNS comparison")
	}

	// Override count to 10 for comparison mode
	originalCount := lt.count
	lt.count = 10

	// Store original mode states
	originalTcpMode := lt.tcpMode
	originalUdpMode := lt.udpMode

	// Set DNS mode for both tests
	lt.tcpMode = false
	lt.udpMode = false

	// Test DNS IPv6
	fmt.Printf("Testing DNS %s IPv6 ([%s]:%d) querying %s...\n", strings.ToUpper(lt.dnsProtocol), ipv6, lt.port, lt.dnsQuery)
	lt.target6 = ipv6
	lt.testIPv6()
	dnsv6Stats := lt.calculateStats(lt.results6)

	// Reset results and test DNS IPv4
	lt.results6 = nil

	// Test DNS IPv4
	fmt.Printf("Testing DNS %s IPv4 (%s:%d) querying %s...\n", strings.ToUpper(lt.dnsProtocol), ipv4, lt.port, lt.dnsQuery)
	lt.target4 = ipv4
	lt.testIPv4()
	dnsv4Stats := lt.calculateStats(lt.results4)

	// Restore original settings
	lt.count = originalCount
	lt.tcpMode = originalTcpMode
	lt.udpMode = originalUdpMode

	// Create comparison result for JSON output
	result := &ComparisonResult{
		DNSv4Stats:   dnsv4Stats,
		DNSv6Stats:   dnsv6Stats,
		ResolvedIPv4: ipv4,
		ResolvedIPv6: ipv6,
		Protocol:     fmt.Sprintf("DNS-%s", strings.ToUpper(lt.dnsProtocol)),
		Hostname:     lt.hostname,
		Port:         lt.port,
		DNSQuery:     lt.dnsQuery,
		Timestamp:    time.Now(),
	}

	// Calculate DNS comparison scores
	lt.calculateDNSComparisonScores(result)

	// Print DNS comparison results
	if lt.jsonOutput {
		lt.printJSONComparisonResults(result)
	} else {
		lt.printDNSComparisonResults(dnsv4Stats, dnsv6Stats, ipv4, ipv6)
	}
}

func (lt *LatencyTester) printDNSComparisonResults(ipv4Stats, ipv6Stats Statistics, ipv4Addr, ipv6Addr string) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("DNS %s COMPARISON RESULTS\n", strings.ToUpper(lt.dnsProtocol))
	fmt.Printf(strings.Repeat("=", 60) + "\n\n")

	// IPv6 Results
	fmt.Printf("IPv6 DNS Results ([%s]:%d)\n", ipv6Addr, lt.port)
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	if ipv6Stats.Received > 0 {
		successRate := float64(ipv6Stats.Received) / float64(ipv6Stats.Sent) * 100
		fmt.Printf("Success: %.1f%% (%d/%d)\n", successRate, ipv6Stats.Received, ipv6Stats.Sent)
		fmt.Printf("Latency: avg=%.3fms min=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(ipv6Stats.Avg.Nanoseconds())/1e6,
			float64(ipv6Stats.Min.Nanoseconds())/1e6,
			float64(ipv6Stats.Max.Nanoseconds())/1e6,
			float64(ipv6Stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n", float64(ipv6Stats.Jitter.Nanoseconds())/1e6)
	} else {
		fmt.Printf("Failed: No successful DNS queries\n")
	}
	fmt.Printf("\n")

	// IPv4 Results
	fmt.Printf("IPv4 DNS Results (%s:%d)\n", ipv4Addr, lt.port)
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	if ipv4Stats.Received > 0 {
		successRate := float64(ipv4Stats.Received) / float64(ipv4Stats.Sent) * 100
		fmt.Printf("Success: %.1f%% (%d/%d)\n", successRate, ipv4Stats.Received, ipv4Stats.Sent)
		fmt.Printf("Latency: avg=%.3fms min=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(ipv4Stats.Avg.Nanoseconds())/1e6,
			float64(ipv4Stats.Min.Nanoseconds())/1e6,
			float64(ipv4Stats.Max.Nanoseconds())/1e6,
			float64(ipv4Stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n", float64(ipv4Stats.Jitter.Nanoseconds())/1e6)
	} else {
		fmt.Printf("Failed: No successful DNS queries\n")
	}
	fmt.Printf("\n")

	// Comparison
	fmt.Printf("DNS Performance Comparison\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")

	if ipv4Stats.Received > 0 && ipv6Stats.Received > 0 {
		diff := float64(ipv4Stats.Avg.Nanoseconds()-ipv6Stats.Avg.Nanoseconds()) / 1e6
		faster := "IPv6"
		if diff < 0 {
			faster = "IPv4"
			diff = -diff
		}
		fmt.Printf("Average latency difference: %.3fms (%s is faster)\n", diff, faster)

		success6 := float64(ipv6Stats.Received) / float64(ipv6Stats.Sent) * 100
		success4 := float64(ipv4Stats.Received) / float64(ipv4Stats.Sent) * 100
		fmt.Printf("Success rate: IPv6=%.1f%% IPv4=%.1f%%\n", success6, success4)

		// Simple scoring for DNS
		ipv6Score := success6 * (1000 / (float64(ipv6Stats.Avg.Nanoseconds()) / 1e6))
		ipv4Score := success4 * (1000 / (float64(ipv4Stats.Avg.Nanoseconds()) / 1e6))

		fmt.Printf("\nPerformance Scores:\n")
		fmt.Printf("IPv6: %.2f\n", ipv6Score)
		fmt.Printf("IPv4: %.2f\n", ipv4Score)

		if ipv6Score > ipv4Score {
			percent := ((ipv6Score - ipv4Score) / ipv4Score) * 100
			fmt.Printf("\n🏆 Winner: IPv6 (%.1f%% better)\n", percent)
		} else if ipv4Score > ipv6Score {
			percent := ((ipv4Score - ipv6Score) / ipv6Score) * 100
			fmt.Printf("\n🏆 Winner: IPv4 (%.1f%% better)\n", percent)
		} else {
			fmt.Printf("\n🏆 Winner: Tie\n")
		}
	} else {
		fmt.Printf("Cannot compare: One or both protocols failed completely\n")
	}

	fmt.Printf("\nQuery: %s\n", lt.dnsQuery)
	fmt.Printf("Protocol: %s\n", strings.ToUpper(lt.dnsProtocol))
	fmt.Printf("Scoring: Based on success rate and latency (higher success + lower latency = higher score)\n\n")
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
	fmt.Printf("\n Winner: %s", result.Winner)

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
	} else if lt.dnsMode {
		testType = fmt.Sprintf("DNS Queries (%s)", strings.ToUpper(lt.dnsProtocol))
	}

	lossType := "loss"
	if lt.tcpMode {
		lossType = "failed"
	} else if lt.udpMode {
		lossType = "failed"
	} else if lt.httpMode {
		lossType = "failed"
	} else if lt.dnsMode {
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

		if lt.tcpMode || lt.udpMode || lt.httpMode || lt.dnsMode {
			fmt.Printf("Success rate: IPv6=%.1f%% IPv4=%.1f%%\n", success6, success4)
		} else {
			loss6 := float64(stats6.Lost) / float64(stats6.Sent) * 100
			loss4 := float64(stats4.Lost) / float64(stats4.Sent) * 100
			fmt.Printf("Packet loss: IPv6=%.1f%% IPv4=%.1f%%\n", loss6, loss4)
		}
	}
	fmt.Printf("\n")
}

func (lt *LatencyTester) calculateDNSComparisonScores(result *ComparisonResult) {
	// Simple scoring for DNS based on success rate and latency
	ipv4Score := 0.0
	ipv6Score := 0.0

	if result.DNSv4Stats.Received > 0 {
		success4 := float64(result.DNSv4Stats.Received) / float64(result.DNSv4Stats.Sent) * 100
		ipv4Score = success4 * (1000 / (float64(result.DNSv4Stats.Avg.Nanoseconds()) / 1e6))
	}

	if result.DNSv6Stats.Received > 0 {
		success6 := float64(result.DNSv6Stats.Received) / float64(result.DNSv6Stats.Sent) * 100
		ipv6Score = success6 * (1000 / (float64(result.DNSv6Stats.Avg.Nanoseconds()) / 1e6))
	}

	result.IPv4Score = ipv4Score
	result.IPv6Score = ipv6Score

	if ipv4Score > ipv6Score {
		result.Winner = "IPv4"
	} else if ipv6Score > ipv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}

func (lt *LatencyTester) printJSONResults() {
	protocol := "TCP"
	if lt.udpMode {
		protocol = "UDP"
	} else if lt.icmpMode {
		protocol = "ICMP"
	} else if lt.httpMode {
		protocol = "HTTP/HTTPS"
	} else if lt.dnsMode {
		protocol = fmt.Sprintf("DNS-%s", strings.ToUpper(lt.dnsProtocol))
	}

	output := JSONOutput{
		Mode:     "single",
		Protocol: protocol,
		Targets: map[string]string{
			"ipv4": lt.target4,
			"ipv6": lt.target6,
		},
		TestConfig: TestConfig{
			Count:       lt.count,
			Interval:    lt.interval,
			Timeout:     lt.timeout,
			Port:        lt.port,
			Size:        lt.size,
			DNSQuery:    lt.dnsQuery,
			DNSProtocol: lt.dnsProtocol,
			Verbose:     lt.verbose,
		},
		Timestamp: time.Now(),
	}

	if !lt.ipv6Only && len(lt.results4) > 0 {
		stats4 := lt.calculateStats(lt.results4)
		stats4.SuccessRate = float64(stats4.Received) / float64(stats4.Sent) * 100
		output.IPv4Results = stats4
	}

	if !lt.ipv4Only && len(lt.results6) > 0 {
		stats6 := lt.calculateStats(lt.results6)
		stats6.SuccessRate = float64(stats6.Received) / float64(stats6.Sent) * 100
		output.IPv6Results = stats6
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

func (lt *LatencyTester) printJSONComparisonResults(result *ComparisonResult) {
	protocol := result.Protocol
	if result.DNSQuery != "" {
		protocol = fmt.Sprintf("DNS-%s", strings.ToUpper(lt.dnsProtocol))
	}

	output := JSONOutput{
		Mode:     "compare",
		Protocol: protocol,
		Targets: map[string]string{
			"hostname": lt.hostname,
			"ipv4":     result.ResolvedIPv4,
			"ipv6":     result.ResolvedIPv6,
		},
		Comparison: result,
		TestConfig: TestConfig{
			Count:       lt.count,
			Interval:    lt.interval,
			Timeout:     lt.timeout,
			Port:        lt.port,
			Size:        lt.size,
			DNSQuery:    lt.dnsQuery,
			DNSProtocol: lt.dnsProtocol,
			Verbose:     lt.verbose,
		},
		Timestamp: time.Now(),
	}

	// Calculate success rates for comparison results
	if result.TCPv4Stats.Sent > 0 {
		result.TCPv4Stats.SuccessRate = float64(result.TCPv4Stats.Received) / float64(result.TCPv4Stats.Sent) * 100
	}
	if result.TCPv6Stats.Sent > 0 {
		result.TCPv6Stats.SuccessRate = float64(result.TCPv6Stats.Received) / float64(result.TCPv6Stats.Sent) * 100
	}
	if result.UDPv4Stats.Sent > 0 {
		result.UDPv4Stats.SuccessRate = float64(result.UDPv4Stats.Received) / float64(result.UDPv4Stats.Sent) * 100
	}
	if result.UDPv6Stats.Sent > 0 {
		result.UDPv6Stats.SuccessRate = float64(result.UDPv6Stats.Received) / float64(result.UDPv6Stats.Sent) * 100
	}
	if result.DNSv4Stats.Sent > 0 {
		result.DNSv4Stats.SuccessRate = float64(result.DNSv4Stats.Received) / float64(result.DNSv4Stats.Sent) * 100
	}
	if result.DNSv6Stats.Sent > 0 {
		result.DNSv6Stats.SuccessRate = float64(result.DNSv6Stats.Received) / float64(result.DNSv6Stats.Sent) * 100
	}
	if result.HTTPv4Stats.Sent > 0 {
		result.HTTPv4Stats.SuccessRate = float64(result.HTTPv4Stats.Received) / float64(result.HTTPv4Stats.Sent) * 100
	}
	if result.HTTPv6Stats.Sent > 0 {
		result.HTTPv6Stats.SuccessRate = float64(result.HTTPv6Stats.Received) / float64(result.HTTPv6Stats.Sent) * 100
	}
	if result.ICMPv4Stats.Sent > 0 {
		result.ICMPv4Stats.SuccessRate = float64(result.ICMPv4Stats.Received) / float64(result.ICMPv4Stats.Sent) * 100
	}
	if result.ICMPv6Stats.Sent > 0 {
		result.ICMPv6Stats.SuccessRate = float64(result.ICMPv6Stats.Received) / float64(result.ICMPv6Stats.Sent) * 100
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}

func (lt *LatencyTester) runICMPCompareMode() {
	fmt.Printf("High-Fidelity IPv4/IPv6 ICMP Comparison Mode\n")
	fmt.Printf("==========================================\n\n")

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
		Protocol:     "ICMP",
		Hostname:     lt.hostname,
		Port:         0, // ICMP doesn't use ports
		Timestamp:    time.Now(),
	}

	// Store original mode states
	originalTcpMode := lt.tcpMode
	originalUdpMode := lt.udpMode
	originalDnsMode := lt.dnsMode

	// Set ICMP mode for both tests
	lt.tcpMode = false
	lt.udpMode = false
	lt.dnsMode = false

	// Test ICMP IPv6
	fmt.Printf("Testing ICMP IPv6 (%s)...\n", ipv6)
	lt.target6 = ipv6
	lt.testIPv6()
	result.ICMPv6Stats = lt.calculateStats(lt.results6)

	// Reset results and test ICMP IPv4
	lt.results6 = nil

	// Test ICMP IPv4
	fmt.Printf("Testing ICMP IPv4 (%s)...\n", ipv4)
	lt.target4 = ipv4
	lt.testIPv4()
	result.ICMPv4Stats = lt.calculateStats(lt.results4)

	// Restore original settings
	lt.count = originalCount
	lt.tcpMode = originalTcpMode
	lt.udpMode = originalUdpMode
	lt.dnsMode = originalDnsMode

	// Calculate comparison scores
	lt.calculateICMPComparisonScores(result)

	// Print results
	if lt.jsonOutput {
		lt.printJSONComparisonResults(result)
	} else {
		lt.printICMPComparisonResults(result)
	}
}

func (lt *LatencyTester) runHTTPCompareMode() {
	fmt.Printf("High-Fidelity IPv4/IPv6 HTTP Comparison Mode\n")
	fmt.Printf("==========================================\n\n")

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
		Protocol:     "HTTP/HTTPS",
		Hostname:     lt.hostname,
		Port:         lt.port,
		Timestamp:    time.Now(),
	}

	// Store original mode states
	originalTcpMode := lt.tcpMode
	originalUdpMode := lt.udpMode
	originalIcmpMode := lt.icmpMode
	originalDnsMode := lt.dnsMode

	// Set HTTP mode for both tests
	lt.tcpMode = false
	lt.udpMode = false
	lt.icmpMode = false
	lt.dnsMode = false

	// Test HTTP IPv6
	fmt.Printf("Testing HTTP IPv6 ([%s]:%d)...\n", ipv6, lt.port)
	lt.target6 = ipv6
	lt.testIPv6()
	result.HTTPv6Stats = lt.calculateStats(lt.results6)

	// Reset results and test HTTP IPv4
	lt.results6 = nil

	// Test HTTP IPv4
	fmt.Printf("Testing HTTP IPv4 (%s:%d)...\n", ipv4, lt.port)
	lt.target4 = ipv4
	lt.testIPv4()
	result.HTTPv4Stats = lt.calculateStats(lt.results4)

	// Restore original settings
	lt.count = originalCount
	lt.tcpMode = originalTcpMode
	lt.udpMode = originalUdpMode
	lt.icmpMode = originalIcmpMode
	lt.dnsMode = originalDnsMode

	// Calculate comparison scores
	lt.calculateHTTPComparisonScores(result)

	// Print results
	if lt.jsonOutput {
		lt.printJSONComparisonResults(result)
	} else {
		lt.printHTTPComparisonResults(result)
	}
}

func (lt *LatencyTester) calculateICMPComparisonScores(result *ComparisonResult) {
	// Score calculation for ICMP: lower latency and higher success rate are better
	ipv4Score := 0.0
	ipv6Score := 0.0

	if result.ICMPv4Stats.Received > 0 {
		successRate := float64(result.ICMPv4Stats.Received) / float64(result.ICMPv4Stats.Sent)
		avgLatencyMs := float64(result.ICMPv4Stats.Avg.Nanoseconds()) / 1e6
		ipv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.ICMPv6Stats.Received > 0 {
		successRate := float64(result.ICMPv6Stats.Received) / float64(result.ICMPv6Stats.Sent)
		avgLatencyMs := float64(result.ICMPv6Stats.Avg.Nanoseconds()) / 1e6
		ipv6Score = successRate * (1000 / avgLatencyMs)
	}

	result.IPv4Score = ipv4Score
	result.IPv6Score = ipv6Score

	if ipv4Score > ipv6Score {
		result.Winner = "IPv4"
	} else if ipv6Score > ipv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}

func (lt *LatencyTester) calculateHTTPComparisonScores(result *ComparisonResult) {
	// Score calculation for HTTP: lower latency and higher success rate are better
	ipv4Score := 0.0
	ipv6Score := 0.0

	if result.HTTPv4Stats.Received > 0 {
		successRate := float64(result.HTTPv4Stats.Received) / float64(result.HTTPv4Stats.Sent)
		avgLatencyMs := float64(result.HTTPv4Stats.Avg.Nanoseconds()) / 1e6
		ipv4Score = successRate * (1000 / avgLatencyMs)
	}

	if result.HTTPv6Stats.Received > 0 {
		successRate := float64(result.HTTPv6Stats.Received) / float64(result.HTTPv6Stats.Sent)
		avgLatencyMs := float64(result.HTTPv6Stats.Avg.Nanoseconds()) / 1e6
		ipv6Score = successRate * (1000 / avgLatencyMs)
	}

	result.IPv4Score = ipv4Score
	result.IPv6Score = ipv6Score

	if ipv4Score > ipv6Score {
		result.Winner = "IPv4"
	} else if ipv6Score > ipv4Score {
		result.Winner = "IPv6"
	} else {
		result.Winner = "Tie"
	}
}

func (lt *LatencyTester) printICMPComparisonResults(result *ComparisonResult) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("ICMP COMPARISON RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n\n")

	// IPv6 Results
	fmt.Printf("IPv6 ICMP Results (%s)\n", result.ResolvedIPv6)
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	if result.ICMPv6Stats.Received > 0 {
		successRate := float64(result.ICMPv6Stats.Received) / float64(result.ICMPv6Stats.Sent) * 100
		fmt.Printf("Success: %.1f%% (%d/%d)\n", successRate, result.ICMPv6Stats.Received, result.ICMPv6Stats.Sent)
		fmt.Printf("Latency: avg=%.3fms min=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(result.ICMPv6Stats.Avg.Nanoseconds())/1e6,
			float64(result.ICMPv6Stats.Min.Nanoseconds())/1e6,
			float64(result.ICMPv6Stats.Max.Nanoseconds())/1e6,
			float64(result.ICMPv6Stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n", float64(result.ICMPv6Stats.Jitter.Nanoseconds())/1e6)
	} else {
		fmt.Printf("Failed: No successful ICMP packets\n")
	}
	fmt.Printf("\n")

	// IPv4 Results
	fmt.Printf("IPv4 ICMP Results (%s)\n", result.ResolvedIPv4)
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	if result.ICMPv4Stats.Received > 0 {
		successRate := float64(result.ICMPv4Stats.Received) / float64(result.ICMPv4Stats.Sent) * 100
		fmt.Printf("Success: %.1f%% (%d/%d)\n", successRate, result.ICMPv4Stats.Received, result.ICMPv4Stats.Sent)
		fmt.Printf("Latency: avg=%.3fms min=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(result.ICMPv4Stats.Avg.Nanoseconds())/1e6,
			float64(result.ICMPv4Stats.Min.Nanoseconds())/1e6,
			float64(result.ICMPv4Stats.Max.Nanoseconds())/1e6,
			float64(result.ICMPv4Stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n", float64(result.ICMPv4Stats.Jitter.Nanoseconds())/1e6)
	} else {
		fmt.Printf("Failed: No successful ICMP packets\n")
	}
	fmt.Printf("\n")

	// Comparison
	fmt.Printf("ICMP Performance Comparison\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")

	if result.ICMPv4Stats.Received > 0 && result.ICMPv6Stats.Received > 0 {
		diff := float64(result.ICMPv4Stats.Avg.Nanoseconds()-result.ICMPv6Stats.Avg.Nanoseconds()) / 1e6
		faster := "IPv6"
		if diff < 0 {
			faster = "IPv4"
			diff = -diff
		}
		fmt.Printf("Average latency difference: %.3fms (%s is faster)\n", diff, faster)

		success6 := float64(result.ICMPv6Stats.Received) / float64(result.ICMPv6Stats.Sent) * 100
		success4 := float64(result.ICMPv4Stats.Received) / float64(result.ICMPv4Stats.Sent) * 100
		fmt.Printf("Success rate: IPv6=%.1f%% IPv4=%.1f%%\n", success6, success4)

		fmt.Printf("\nPerformance Scores:\n")
		fmt.Printf("IPv6: %.2f\n", result.IPv6Score)
		fmt.Printf("IPv4: %.2f\n", result.IPv4Score)

		if result.IPv6Score > result.IPv4Score {
			percent := ((result.IPv6Score - result.IPv4Score) / result.IPv4Score) * 100
			fmt.Printf("\n🏆 Winner: IPv6 (%.1f%% better)\n", percent)
		} else if result.IPv4Score > result.IPv6Score {
			percent := ((result.IPv4Score - result.IPv6Score) / result.IPv6Score) * 100
			fmt.Printf("\n🏆 Winner: IPv4 (%.1f%% better)\n", percent)
		} else {
			fmt.Printf("\n🏆 Winner: Tie\n")
		}
	} else {
		fmt.Printf("Cannot compare: One or both protocols failed completely\n")
	}

	fmt.Printf("\nScoring: Based on success rate and latency (higher success + lower latency = higher score)\n\n")
}

func (lt *LatencyTester) printHTTPComparisonResults(result *ComparisonResult) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("HTTP/HTTPS COMPARISON RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n\n")

	scheme := "HTTP"
	if lt.port == 443 || lt.port == 8443 {
		scheme = "HTTPS"
	}

	// IPv6 Results
	fmt.Printf("IPv6 %s Results ([%s]:%d)\n", scheme, result.ResolvedIPv6, lt.port)
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	if result.HTTPv6Stats.Received > 0 {
		successRate := float64(result.HTTPv6Stats.Received) / float64(result.HTTPv6Stats.Sent) * 100
		fmt.Printf("Success: %.1f%% (%d/%d)\n", successRate, result.HTTPv6Stats.Received, result.HTTPv6Stats.Sent)
		fmt.Printf("Latency: avg=%.3fms min=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(result.HTTPv6Stats.Avg.Nanoseconds())/1e6,
			float64(result.HTTPv6Stats.Min.Nanoseconds())/1e6,
			float64(result.HTTPv6Stats.Max.Nanoseconds())/1e6,
			float64(result.HTTPv6Stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n", float64(result.HTTPv6Stats.Jitter.Nanoseconds())/1e6)
	} else {
		fmt.Printf("Failed: No successful HTTP requests\n")
	}
	fmt.Printf("\n")

	// IPv4 Results
	fmt.Printf("IPv4 %s Results (%s:%d)\n", scheme, result.ResolvedIPv4, lt.port)
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	if result.HTTPv4Stats.Received > 0 {
		successRate := float64(result.HTTPv4Stats.Received) / float64(result.HTTPv4Stats.Sent) * 100
		fmt.Printf("Success: %.1f%% (%d/%d)\n", successRate, result.HTTPv4Stats.Received, result.HTTPv4Stats.Sent)
		fmt.Printf("Latency: avg=%.3fms min=%.3fms max=%.3fms stddev=%.3fms\n",
			float64(result.HTTPv4Stats.Avg.Nanoseconds())/1e6,
			float64(result.HTTPv4Stats.Min.Nanoseconds())/1e6,
			float64(result.HTTPv4Stats.Max.Nanoseconds())/1e6,
			float64(result.HTTPv4Stats.StdDev.Nanoseconds())/1e6)
		fmt.Printf("Jitter: %.3fms\n", float64(result.HTTPv4Stats.Jitter.Nanoseconds())/1e6)
	} else {
		fmt.Printf("Failed: No successful HTTP requests\n")
	}
	fmt.Printf("\n")

	// Comparison
	fmt.Printf("%s Performance Comparison\n", scheme)
	fmt.Printf(strings.Repeat("-", 40) + "\n")

	if result.HTTPv4Stats.Received > 0 && result.HTTPv6Stats.Received > 0 {
		diff := float64(result.HTTPv4Stats.Avg.Nanoseconds()-result.HTTPv6Stats.Avg.Nanoseconds()) / 1e6
		faster := "IPv6"
		if diff < 0 {
			faster = "IPv4"
			diff = -diff
		}
		fmt.Printf("Average latency difference: %.3fms (%s is faster)\n", diff, faster)

		success6 := float64(result.HTTPv6Stats.Received) / float64(result.HTTPv6Stats.Sent) * 100
		success4 := float64(result.HTTPv4Stats.Received) / float64(result.HTTPv4Stats.Sent) * 100
		fmt.Printf("Success rate: IPv6=%.1f%% IPv4=%.1f%%\n", success6, success4)

		fmt.Printf("\nPerformance Scores:\n")
		fmt.Printf("IPv6: %.2f\n", result.IPv6Score)
		fmt.Printf("IPv4: %.2f\n", result.IPv4Score)

		if result.IPv6Score > result.IPv4Score {
			percent := ((result.IPv6Score - result.IPv4Score) / result.IPv4Score) * 100
			fmt.Printf("\n🏆 Winner: IPv6 (%.1f%% better)\n", percent)
		} else if result.IPv4Score > result.IPv6Score {
			percent := ((result.IPv4Score - result.IPv6Score) / result.IPv6Score) * 100
			fmt.Printf("\n🏆 Winner: IPv4 (%.1f%% better)\n", percent)
		} else {
			fmt.Printf("\n🏆 Winner: Tie\n")
		}
	} else {
		fmt.Printf("Cannot compare: One or both protocols failed completely\n")
	}

	fmt.Printf("\nScoring: Based on success rate and latency (higher success + lower latency = higher score)\n\n")
}

// Configuration file and daemon mode functions
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config

	// Determine file format by extension
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %v", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %v", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if err2 := json.Unmarshal(data, &config); err2 != nil {
				return nil, fmt.Errorf("failed to parse config as YAML (%v) or JSON (%v)", err, err2)
			}
		}
	}

	// Set defaults for missing values
	setConfigDefaults(&config)

	return &config, nil
}

func setConfigDefaults(config *Config) {
	// Global defaults
	if config.Global.DefaultCount == 0 {
		config.Global.DefaultCount = 10
	}
	if config.Global.Timeout == 0 {
		config.Global.Timeout = 3 * time.Second
	}
	if config.Global.Interval == 0 {
		config.Global.Interval = 1 * time.Second
	}
	if config.Global.LogLevel == "" {
		config.Global.LogLevel = "info"
	}

	// Daemon defaults
	if config.Daemon.RunInterval == 0 {
		config.Daemon.RunInterval = 5 * time.Minute
	}
	if config.Daemon.MaxLogSize == 0 {
		config.Daemon.MaxLogSize = 100 * 1024 * 1024 // 100MB
	}
	if config.Daemon.MaxRetries == 0 {
		config.Daemon.MaxRetries = 3
	}
	if config.Daemon.RetryInterval == 0 {
		config.Daemon.RetryInterval = 30 * time.Second
	}

	// Test defaults
	for i := range config.Tests {
		test := &config.Tests[i]
		if test.Count == 0 {
			test.Count = config.Global.DefaultCount
		}
		if test.Timeout == 0 {
			test.Timeout = config.Global.Timeout
		}
		if test.Interval == 0 {
			test.Interval = config.Global.Interval
		}
		if test.Port == 0 {
			switch test.Type {
			case "http":
				test.Port = 80
			case "https":
				test.Port = 443
			case "dns":
				test.Port = 53
			case "dot":
				test.Port = 853
			case "doh":
				test.Port = 443
			default:
				test.Port = 53
			}
		}
		if test.Size == 0 {
			test.Size = 64
		}
		if test.DNSProtocol == "" {
			test.DNSProtocol = "udp"
		}
		if test.DNSQuery == "" {
			test.DNSQuery = "dns-query.qosbox.com"
		}
		if test.Target4 == "" {
			test.Target4 = "8.8.8.8"
		}
		if test.Target6 == "" {
			test.Target6 = "2001:4860:4860::8888"
		}
	}
}

func runWithConfig(configFile string, daemonMode bool, outputFile string) {
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Override output file if specified on command line
	if outputFile != "" {
		config.Global.OutputFile = outputFile
		config.Daemon.OutputFile = outputFile
	}

	// Initialize InfluxDB if enabled
	if err := initInfluxDB(config.Global.InfluxDB); err != nil {
		log.Fatalf("Error initializing InfluxDB: %v", err)
	}
	defer closeInfluxDB()

	if daemonMode || config.Daemon.Enabled {
		runDaemon(config)
	} else {
		runConfigTests(config)
	}
}

func runConfigTests(config *Config) {
	var outputWriter io.Writer = os.Stdout

	// Setup output file if specified
	if config.Global.OutputFile != "" {
		file, err := os.OpenFile(config.Global.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open output file: %v", err)
		}
		defer file.Close()
		outputWriter = file
	}

	results := make([]DaemonResult, 0)

	for _, testConfig := range config.Tests {
		if !testConfig.Enabled {
			continue
		}

		result := runSingleTest(testConfig)
		results = append(results, result)

		// Write result immediately
		writeResult(outputWriter, result, config.Global.JSONOutput)

		// Write to InfluxDB if enabled and test was successful
		if result.Success {
			writeResultToInfluxDB(config.Global.InfluxDB, result)
		}
	}

	// Write summary if not in JSON mode
	if !config.Global.JSONOutput {
		writeSummary(outputWriter, results)
	}
}

func runSingleTest(testConfig TestSpec) DaemonResult {
	start := time.Now()

	result := DaemonResult{
		TestName:  testConfig.Name,
		Timestamp: start,
		TestType:  testConfig.Type,
		Success:   false,
	}

	// Create a LatencyTester for this test
	tester := &LatencyTester{
		target4:     testConfig.Target4,
		target6:     testConfig.Target6,
		hostname:    testConfig.Hostname,
		port:        testConfig.Port,
		count:       testConfig.Count,
		interval:    testConfig.Interval,
		timeout:     testConfig.Timeout,
		size:        testConfig.Size,
		ipv4Only:    testConfig.IPv4Only,
		ipv6Only:    testConfig.IPv6Only,
		verbose:     false, // Disable verbose in config mode
		dnsProtocol: testConfig.DNSProtocol,
		dnsQuery:    testConfig.DNSQuery,
		jsonOutput:  true, // Always use JSON for structured results
	}

	// Set protocol modes based on test type
	switch testConfig.Type {
	case "tcp":
		tester.tcpMode = true
	case "udp":
		tester.udpMode = true
	case "icmp":
		tester.icmpMode = true
	case "http", "https":
		tester.httpMode = true
	case "dns", "dot", "doh":
		tester.dnsMode = true
		if testConfig.Type == "dot" {
			tester.dnsProtocol = "dot"
		} else if testConfig.Type == "doh" {
			tester.dnsProtocol = "doh"
		}
	case "compare":
		tester.compareMode = true
		if testConfig.Hostname == "" {
			result.Error = "Compare mode requires hostname"
			result.Duration = time.Since(start).Seconds()
			return result
		}
	default:
		tester.tcpMode = true // Default to TCP
	}

	// Set target information
	if testConfig.Type == "compare" {
		result.Target = testConfig.Hostname
	} else if testConfig.IPv4Only {
		result.Target = testConfig.Target4
	} else if testConfig.IPv6Only {
		result.Target = testConfig.Target6
	} else {
		result.Target = fmt.Sprintf("IPv4:%s IPv6:%s", testConfig.Target4, testConfig.Target6)
	}

	// Run the test
	defer func() {
		if r := recover(); r != nil {
			result.Error = fmt.Sprintf("Test panicked: %v", r)
		}
		result.Duration = time.Since(start).Seconds()
	}()

	// Execute the test based on mode
	if tester.compareMode {
		// For compare mode, we need to capture the output differently
		// We'll run a simplified version and capture statistics
		if tester.dnsMode {
			tester.runDNSCompareMode()
		} else if tester.icmpMode {
			tester.runICMPCompareMode()
		} else if tester.httpMode {
			tester.runHTTPCompareMode()
		} else {
			tester.runCompareMode()
		}
		result.Success = true
		result.Results = "Compare mode completed"
	} else {
		// Run single protocol tests
		if !tester.ipv4Only {
			tester.testIPv6()
		}
		if !tester.ipv6Only {
			tester.testIPv4()
		}

		// Calculate statistics
		var stats4, stats6 Statistics
		if len(tester.results4) > 0 {
			stats4 = tester.calculateStats(tester.results4)
			stats4.SuccessRate = float64(stats4.Received) / float64(stats4.Sent) * 100
		}
		if len(tester.results6) > 0 {
			stats6 = tester.calculateStats(tester.results6)
			stats6.SuccessRate = float64(stats6.Received) / float64(stats6.Sent) * 100
		}

		// Create result structure
		testResult := struct {
			IPv4Results Statistics `json:"ipv4_results,omitempty"`
			IPv6Results Statistics `json:"ipv6_results,omitempty"`
		}{
			IPv4Results: stats4,
			IPv6Results: stats6,
		}

		result.Results = testResult
		result.Success = (stats4.Received > 0 || stats6.Received > 0)
	}

	return result
}

func writeResult(writer io.Writer, result DaemonResult, jsonOutput bool) {
	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err == nil {
			fmt.Fprintln(writer, string(data))
		}
	} else {
		fmt.Fprintf(writer, "[%s] %s (%s): ",
			result.Timestamp.Format("2006-01-02 15:04:05"),
			result.TestName,
			result.TestType)

		if result.Success {
			fmt.Fprintf(writer, "SUCCESS - Duration: %.2fs\n", result.Duration)
		} else {
			fmt.Fprintf(writer, "FAILED - %s - Duration: %.2fs\n", result.Error, result.Duration)
		}
	}
}

func writeSummary(writer io.Writer, results []DaemonResult) {
	successful := 0
	failed := 0
	totalDuration := 0.0

	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}

	fmt.Fprintf(writer, "\n=== Test Summary ===\n")
	fmt.Fprintf(writer, "Total tests: %d\n", len(results))
	fmt.Fprintf(writer, "Successful: %d\n", successful)
	fmt.Fprintf(writer, "Failed: %d\n", failed)
	fmt.Fprintf(writer, "Total duration: %.2fs\n", totalDuration)
	fmt.Fprintf(writer, "Success rate: %.1f%%\n", float64(successful)/float64(len(results))*100)
}

func runDaemon(config *Config) {
	log.Printf("Starting ProtoTester daemon with %d tests", len(config.Tests))

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup output file
	var outputWriter io.Writer = os.Stdout
	if config.Daemon.OutputFile != "" {
		file, err := os.OpenFile(config.Daemon.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open daemon output file: %v", err)
		}
		defer file.Close()
		outputWriter = file
	}

	// Write PID file if specified
	if config.Daemon.PidFile != "" {
		pidFile, err := os.Create(config.Daemon.PidFile)
		if err != nil {
			log.Fatalf("Failed to create PID file: %v", err)
		}
		fmt.Fprintf(pidFile, "%d", os.Getpid())
		pidFile.Close()
		defer os.Remove(config.Daemon.PidFile)
	}

	// Main daemon loop
	ticker := time.NewTicker(config.Daemon.RunInterval)
	defer ticker.Stop()

	// Run tests immediately on startup
	log.Println("Running initial test cycle...")
	runTestCycle(config, outputWriter)

	for {
		select {
		case <-ticker.C:
			log.Println("Running scheduled test cycle...")
			runTestCycle(config, outputWriter)
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down daemon...", sig)
			return
		}
	}
}

func runTestCycle(config *Config, outputWriter io.Writer) {
	results := make([]DaemonResult, 0)

	for _, testConfig := range config.Tests {
		if !testConfig.Enabled {
			continue
		}

		retries := 0
		var result DaemonResult

		for retries <= config.Daemon.MaxRetries {
			result = runSingleTest(testConfig)

			if result.Success || retries == config.Daemon.MaxRetries {
				break
			}

			retries++
			log.Printf("Test %s failed (attempt %d/%d): %s",
				testConfig.Name, retries, config.Daemon.MaxRetries+1, result.Error)

			if retries <= config.Daemon.MaxRetries {
				time.Sleep(config.Daemon.RetryInterval)
			}
		}

		results = append(results, result)
		writeResult(outputWriter, result, config.Global.JSONOutput)

		// Write to InfluxDB if enabled and test was successful
		if result.Success {
			writeResultToInfluxDB(config.Global.InfluxDB, result)
		}

		// Stop on failure if configured
		if !result.Success && config.Daemon.StopOnFailure {
			log.Printf("Stopping daemon due to test failure: %s", result.Error)
			return
		}
	}

	// Write cycle summary if not in JSON mode
	if !config.Global.JSONOutput {
		writeSummary(outputWriter, results)
	}
}
