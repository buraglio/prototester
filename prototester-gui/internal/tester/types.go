package tester

import (
	"sync"
	"time"
)

// PingResult represents the result of a single test
type PingResult struct {
	Success   bool          `json:"success"`
	Latency   time.Duration `json:"latency_ms"`
	Error     error         `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// Statistics holds aggregated test results
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

// TestConfig holds test configuration
type TestConfig struct {
	Target4     string        `json:"target_ipv4,omitempty"`
	Target6     string        `json:"target_ipv6,omitempty"`
	Hostname    string        `json:"hostname,omitempty"`
	Port        int           `json:"port"`
	Count       int           `json:"count"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Size        int           `json:"size,omitempty"`
	DNSProtocol string        `json:"dns_protocol,omitempty"`
	DNSQuery    string        `json:"dns_query,omitempty"`
	IPv4Only    bool          `json:"ipv4_only"`
	IPv6Only    bool          `json:"ipv6_only"`
	Verbose     bool          `json:"verbose"`
}

// TestResult represents the complete test output
type TestResult struct {
	Mode         string            `json:"mode"`
	Protocol     string            `json:"protocol"`
	Targets      map[string]string `json:"targets"`
	IPv4Results  *Statistics       `json:"ipv4_results,omitempty"`
	IPv6Results  *Statistics       `json:"ipv6_results,omitempty"`
	Comparison   *ComparisonResult `json:"comparison,omitempty"`
	TestConfig   TestConfig        `json:"test_config"`
	Timestamp    time.Time         `json:"timestamp"`
	ErrorMessage string            `json:"error,omitempty"`
}

// ComparisonResult holds comparison test results
type ComparisonResult struct {
	TCPv4Stats   *Statistics `json:"tcp_v4_stats,omitempty"`
	TCPv6Stats   *Statistics `json:"tcp_v6_stats,omitempty"`
	UDPv4Stats   *Statistics `json:"udp_v4_stats,omitempty"`
	UDPv6Stats   *Statistics `json:"udp_v6_stats,omitempty"`
	DNSv4Stats   *Statistics `json:"dns_v4_stats,omitempty"`
	DNSv6Stats   *Statistics `json:"dns_v6_stats,omitempty"`
	HTTPv4Stats  *Statistics `json:"http_v4_stats,omitempty"`
	HTTPv6Stats  *Statistics `json:"http_v6_stats,omitempty"`
	ICMPv4Stats  *Statistics `json:"icmp_v4_stats,omitempty"`
	ICMPv6Stats  *Statistics `json:"icmp_v6_stats,omitempty"`
	IPv4Score    float64     `json:"ipv4_score"`
	IPv6Score    float64     `json:"ipv6_score"`
	Winner       string      `json:"winner"`
	ResolvedIPv4 string      `json:"resolved_ipv4"`
	ResolvedIPv6 string      `json:"resolved_ipv6"`
	Protocol     string      `json:"protocol"`
	Hostname     string      `json:"hostname"`
	Port         int         `json:"port"`
	DNSQuery     string      `json:"dns_query,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
}

// LatencyTester holds the tester state
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
	dnsProtocol string
	dnsQuery    string
	compareMode bool
	results4    []PingResult
	results6    []PingResult
	mu          sync.Mutex
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
