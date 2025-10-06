# ProtoTester - High-Fidelity IPv4/IPv6 Latency Tester

A comprehensive Go program that tests IPv4 and IPv6 connectivity and performance with high precision timing and detailed comparative analysis. **Now works without root privileges by default!**

## Features

- **Multiple Protocol Support**: TCP, UDP, ICMP, HTTP/HTTPS, DNS (UDP/TCP/DoT/DoH) latency testing
- **No Root Required**: Defaults to TCP mode, works out-of-the-box for all users
- **Smart Fallbacks**: Automatically falls back from ICMP to TCP when permissions are insufficient
- **Linux Optimization**: Uses unprivileged ICMP sockets on Linux when available
- **Compare Mode**: Automatic hostname resolution and comprehensive IPv4 vs IPv6 performance comparison (supports all protocols)
- **High-Precision Timing**: Uses nanosecond-precision timing for accurate latency measurements
- **Comprehensive Statistics**: Provides min/max/avg latency, standard deviation, jitter, and percentiles
- **Cross-Platform**: Works on Linux, macOS, and other Unix-like systems
- **IPv4/IPv6 Dual Stack**: Tests both protocols simultaneously or individually
- **Intelligent Scoring**: Performance ranking system based on success rate and latency
- **JSON Output**: Machine-readable JSON output for programmatic analysis and automation
- **Configuration Files**: YAML/JSON configuration files for defining multiple test scenarios
- **Daemon Mode**: Run as a background service with scheduled test execution and logging
- **InfluxDB Integration**: Optional time-series database integration for long-term metrics storage and monitoring
- **Flexible Configuration**: Customizable targets, connection count, intervals, timeouts, and ports

## How Performance Comparison and Scoring Works

ProtoTester uses a reasonably sophisticated scoring algorithm to compare network performance across different protocols and IP versions. Understanding how the metrics are calculated and combined helps interpret the results effectively.

### Core Metrics

#### 1. Latency (Round-Trip Time)
- **Measurement**: Time taken for a packet/request to reach the destination and return
- **Precision**: Nanosecond-level timing for maximum accuracy. This obviously requires a decent clock. It *should* be very accurate, but has (to date) not been tested with a PTP 1588 clock source. Any volunteers?
- **Statistics Provided**:
  - **Minimum**: Fastest observed latency (best-case performance)
  - **Maximum**: Slowest observed latency (worst-case performance)
  - **Average**: Mean latency across all tests (typical performance)
  - **Standard Deviation**: Variability in latency measurements
  - **Percentiles**: P50 (median), P95, P99 for distribution analysis

#### 2. Jitter
- **Definition**: Variation in latency between consecutive packets
- **Calculation**: Average absolute difference between consecutive latencies
  ```
  jitter = Σ|latency[i] - latency[i-1]| / (n-1)
  ```
- **Impact**: High jitter indicates unstable network conditions
- **Importance**: Critical for real-time applications (VoIP, video conferencing, gaming)

#### 3. Availability (Success Rate)
- **Calculation**: `(successful_tests / total_tests) × 100`
- **Range**: 0% (complete failure) to 100% (perfect reliability)
- **Impact**: Directly affects the performance score

### Performance Scoring Algorithm

The scoring system combines **availability** and **latency** to produce a single performance metric.

#### Basic Score Formula

For each protocol and IP version:
```
score = success_rate × (1000 / avg_latency_ms)
```

**How it works**:
- **Success Rate Component**: Fraction of successful tests (0.0 to 1.0)
  - 100% success rate = 1.0 multiplier
  - 50% success rate = 0.5 multiplier (score is halved)
  - 0% success rate = 0.0 (zero score)

- **Latency Component**: `1000 / avg_latency_ms`
  - Lower latency = higher score
  - 10ms average latency: `1000/10 = 100` points
  - 100ms average latency: `1000/100 = 10` points
  - The constant 1000 provides score normalization

**Example Calculation**:
- Test with 100% success rate and 10ms average latency:
  ```
  score = 1.0 × (1000 / 10) = 100.0
  ```
- Test with 80% success rate and 20ms average latency:
  ```
  score = 0.8 × (1000 / 20) = 40.0
  ```

#### Multi-Protocol Compare Mode Scoring

When using `-compare` mode (default TCP/UDP comparison), the final score is a **weighted combination**:

```
IPv4 Score = (TCP_IPv4_score × 0.6) + (UDP_IPv4_score × 0.4)
IPv6 Score = (TCP_IPv6_score × 0.6) + (UDP_IPv6_score × 0.4)
```

**Weighting Rationale**:
- **TCP: 60%** - Most internet traffic uses TCP, making it more representative of real-world performance
- **UDP: 40%** - Important for real-time services (DNS, VoIP, streaming, gaming)

**Example**:
```
TCP IPv4: 100% success, 15ms avg → score = 1.0 × (1000/15) = 66.67
UDP IPv4: 100% success, 25ms avg → score = 1.0 × (1000/25) = 40.00
IPv4 Final Score = (66.67 × 0.6) + (40.00 × 0.4) = 40.00 + 16.00 = 56.00

TCP IPv6: 100% success, 12ms avg → score = 1.0 × (1000/12) = 83.33
UDP IPv6: 100% success, 20ms avg → score = 1.0 × (1000/20) = 50.00
IPv6 Final Score = (83.33 × 0.6) + (50.00 × 0.4) = 50.00 + 20.00 = 70.00

Winner: IPv6 (25% better)
Percentage Difference = ((70.00 - 56.00) / 56.00) × 100 = 25%
```

#### Protocol-Specific Compare Modes

When using `-compare` with specific protocols (`-icmp`, `-http`, `-dns`), only that protocol is tested:

```
score = success_rate × (1000 / avg_latency_ms)
```

No weighting is applied - the direct protocol comparison determines the winner.

### Interpreting Results

#### Score Comparison
- **Higher Score = Better Performance**: Combines speed and reliability
- **Winner Determination**: The IP version with the higher score wins
- **Percentage Difference**: Shows how much better the winner performed
  ```
  percentage = ((winner_score - loser_score) / loser_score) × 100
  ```

#### What the Metrics Tell You

**Latency**:
- **< 10ms**: Excellent (local/regional network)
- **10-50ms**: Good (typical internet performance)
- **50-100ms**: Moderate (acceptable for most applications)
- **> 100ms**: High (may affect real-time applications)

**Jitter**:
- **< 5ms**: Excellent (stable connection)
- **5-20ms**: Good (minor variations)
- **20-50ms**: Moderate (noticeable in real-time apps)
- **> 50ms**: Poor (unstable, affects quality)

**Success Rate**:
- **100%**: Perfect reliability
- **95-99%**: Excellent (minor packet loss)
- **90-95%**: Good (some packet loss)
- **< 90%**: Poor (significant reliability issues)

### Example Output Interpretation

```
Overall Performance Ranking
----------------------------------------
IPv6 Score: 31.17
IPv4 Score: 24.38

Winner: IPv6 (27.9% better)

Scoring: Based on success rate and latency (lower latency + higher success = higher score)
Weighting: TCP 60%, UDP 40%
```

**Analysis**:
- IPv6 scored 31.17 vs IPv4's 24.38
- IPv6 is 27.9% better overall
- Both success rate and latency contribute to this difference
- TCP performance (60% weight) has more influence than UDP (40% weight)

## Requirements

- Go 1.21 or higher
- Network connectivity to test targets
- **No special privileges required for default operation**
- Optional: Root/Administrator privileges for true ICMP testing
- **Cross-Platform**: Fully supports Linux, macOS, and Windows

## Installation

```bash
git clone https://github.com/buraglio/prototester.git
cd prototester
go mod tidy
go build -o prototester
```

### Platform-Specific Builds

**Important**: Use `go build -o prototester` (not `go build -o prototester main.go`) to ensure platform-specific files are included in the build.

```bash
# macOS
go build -o prototester

# Linux
GOOS=linux GOARCH=amd64 go build -o prototester-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o prototester.exe
```

## Quick Start

### Basic Usage (No Root Required)
```bash
# Test default targets with TCP (works immediately)
./prototester

# Test specific targets
./prototester -4 1.1.1.1 -6 2606:4700:4700::1111

# Verbose output with 5 tests
./prototester -v -c 5
```

## Usage Guide

### Protocol Selection

#### DNS Query Testing
```bash
# DNS UDP queries (default)
./prototester -dns

# DNS TCP queries
./prototester -dns -dns-protocol tcp

# DNS over TLS (DoT)
./prototester -dns -dns-protocol dot -p 853

# DNS over HTTPS (DoH)
./prototester -dns -dns-protocol doh -p 443

# Custom query domain
./prototester -dns -dns-query google.com

# Test specific DNS server
./prototester -dns -4 1.1.1.1 -dns-query dns-query.qosbox.com
```

#### TCP Connect Testing (Default - No Root Required)
```bash
# Default TCP mode
./prototester

# Explicit TCP mode
./prototester -t -p 80

# Test web servers
./prototester -t -p 443 -4 google.com
```

#### UDP Testing
```bash
# Test DNS servers
./prototester -u -p 53

# Test custom UDP service
./prototester -u -p 1234 -4 example.com
```

#### ICMP Testing (Smart Fallback)
```bash
# ICMP mode (automatically falls back to TCP if no root)
./prototester -icmp

# ICMP with custom packet size
./prototester -icmp -s 128

# True ICMP with root privileges
sudo ./prototester -icmp
```

#### HTTP/HTTPS Testing
```bash
# HTTP testing (port 80)
./prototester -http -p 80 -4 example.com

# HTTPS testing (port 443 - auto-detected)
./prototester -http -p 443 -4 google.com

# Custom HTTP service
./prototester -http -p 8080 -4 localhost
```

### Compare Mode (Comprehensive Analysis)
```bash
# Automatically resolve hostname and compare IPv4 vs IPv6 performance (TCP/UDP by default)
./prototester -compare google.com

# Compare using HTTPS port
./prototester -compare cloudflare.com -p 443

# Compare with verbose output
./prototester -compare github.com -p 22 -v

# Protocol-specific compare modes
./prototester -compare google.com -icmp           # ICMP comparison
./prototester -compare google.com -http -p 80     # HTTP comparison
./prototester -compare dns.google -dns            # DNS protocol comparison
./prototester -compare dns.google -dns -dns-protocol dot -p 853  # DoT comparison
```

### JSON Output
```bash
# Get results in JSON format for programmatic processing
./prototester -json

# JSON with compare mode
./prototester -compare google.com -json

# JSON with specific protocols
./prototester -dns -dns-protocol doh -json
```

### Configuration Files
```bash
# Run tests from a configuration file
./prototester -config config.yaml

# Run with config file and override output location
./prototester -config config.yaml -output custom-results.log

# Run in daemon mode using configuration file
./prototester -config daemon-config.yaml -daemon
```

## Command Line Options

### Basic Options
- `-4 <address>`: IPv4 target address (default: 8.8.8.8)
- `-6 <address>`: IPv6 target address (default: 2001:4860:4860::8888)
- `-c <count>`: Number of tests to perform (default: 10)
- `-i <duration>`: Interval between tests (default: 1s)
- `-timeout <duration>`: Timeout for each test (default: 3s)
- `-v`: Verbose output

### Protocol Selection (Mutually Exclusive)
- `-t`: Use TCP connect test (default)
- `-u`: Use UDP test
- `-icmp`: Use ICMP ping test (auto-fallback to TCP if no root)
- `-http`: Use HTTP/HTTPS timing test
- `-dns`: Use DNS query testing
- `-compare <hostname>`: Compare mode - test protocols on IPv4/IPv6 (TCP/UDP by default, or use with -icmp/-http/-dns)

### Protocol-Specific Options
- `-p <port>`: Port to test (TCP/UDP/HTTP/DNS modes, default: 53)
- `-s <size>`: Packet size in bytes (ICMP only, default: 64)
- `-dns-protocol <protocol>`: DNS protocol: udp, tcp, dot, doh (default: udp)
- `-dns-query <domain>`: Domain name to query for DNS testing (default: dns-query.qosbox.com)

### Output Options
- `-json`: Output results in JSON format instead of human-readable text
- `-v`: Verbose output

### Configuration and Daemon Options
- `-config <file>`: Configuration file (YAML or JSON format) for batch testing and daemon mode
- `-daemon`: Run in daemon mode using configuration file (requires -config)
- `-output <file>`: Output file for results (stdout if not specified, can override config file setting)

### IPv4/IPv6 Options
- `-4only`: Test IPv4 only
- `-6only`: Test IPv6 only

**Smart Protocol Selection**:
- By default, both IPv4 and IPv6 are tested using default addresses
- When you specify a custom `-4` address (without custom `-6`), only IPv4 is tested
- When you specify a custom `-6` address (without custom `-4`), only IPv6 is tested
- When you specify both custom addresses, both protocols are tested
- Explicit `-4only` or `-6only` flags override the smart selection
- IPv6 is tested first and displayed with priority to encourage IPv6 adoption

## Understanding Permissions

### Default Behavior (No Root)
- **TCP Mode**: Works without any special permissions ✅
- **UDP Mode**: Works without root (uses connected UDP sockets) ✅
- **HTTP Mode**: Works without root ✅
- **DNS Mode**: Works without root (all DNS protocols) ✅
- **ICMP Mode on Linux**: Works without root on modern Linux kernels (unprivileged ICMP) ✅

### ICMP Mode Behavior (Smart Fallback)
1. **Linux Unprivileged ICMP** (First attempt - no root needed):
   - Uses `SOCK_DGRAM` + `IPPROTO_ICMP/IPPROTO_ICMPV6`
   - Available on Linux kernels with unprivileged ICMP support
   - Kernel automatically manages ICMP packet ID field
   - Works on most modern Linux distributions out of the box
2. **Raw Socket ICMP** (Second attempt - requires root):
   - Falls back to `SOCK_RAW` if unprivileged fails
   - Requires root/administrator privileges
   - Full control over ICMP packet structure
3. **TCP Fallback** (Final fallback):
   - If both ICMP methods fail, automatically uses TCP connect
   - Verbose mode shows: "ICMP failed (no root), falling back to TCP connect test..."

### Running with Root (Optional)
```bash
# Enable true ICMP ping on all platforms
sudo ./prototester -icmp

# Root enables raw socket ICMP with larger packets
sudo ./prototester -icmp -s 1400 -v

# Note: On modern Linux, root is NOT required for basic ICMP
./prototester -icmp  # Works without sudo on Linux!
```

## Sample Output

### Default TCP Mode (No Root)
```
High-Fidelity IPv4/IPv6 Latency Tester (TCP)
===============================================

Testing IPv6 connectivity to [2001:4860:4860::8888]:53...
Testing IPv4 connectivity to 8.8.8.8:53...

============================================================
LATENCY TEST RESULTS
============================================================

IPv6 Results (2001:4860:4860::8888)
----------------------------------------
Connections: 10 sent, 10 successful, 0 failed (100.0% success)
Latency: min=8.866ms avg=9.895ms max=11.035ms stddev=0.665ms
Jitter: 0.241ms
Percentiles: P50=9.816ms P95=10.645ms P99=10.645ms

IPv4 Results (8.8.8.8)
----------------------------------------
Connections: 10 sent, 10 successful, 0 failed (100.0% success)
Latency: min=8.677ms avg=15.121ms max=61.112ms stddev=15.347ms
Jitter: 5.826ms
Percentiles: P50=10.237ms P95=11.178ms P99=11.178ms

IPv6 vs IPv4 Comparison
----------------------------------------
Average latency difference: 5.226ms (IPv6 is faster)
Success rate: IPv6=100.0% IPv4=100.0%
```

### ICMP Fallback Mode
```
High-Fidelity IPv4/IPv6 Latency Tester (ICMP)
===============================================

Testing IPv6 connectivity to 2001:4860:4860::8888...
ICMP failed (no root), falling back to TCP connect test...
IPv6 test 1: 11.087ms
Testing IPv4 connectivity to 8.8.8.8...
ICMP failed (no root), falling back to TCP connect test...
IPv4 test 1: 9.173ms
```

### Compare Mode Output
```
High-Fidelity IPv4/IPv6 Comparison Mode
=======================================

Resolving google.com...
Resolved addresses:
  IPv4 (A): 142.251.163.102
  IPv6 (AAAA): 2607:f8b0:4009:818::200e

Testing TCP IPv6 ([2607:f8b0:4009:818::200e]:53)...
Testing TCP IPv4 (142.251.163.102:53)...
Testing UDP IPv6 ([2607:f8b0:4009:818::200e]:53)...
Testing UDP IPv4 (142.251.163.102:53)...

============================================================
COMPREHENSIVE COMPARISON RESULTS
============================================================

TCP Results
----------------------------------------
IPv6 ([2607:f8b0:4009:818::200e]:53):
  Success: 100.0% (10/10)
  Latency: avg=22.038ms min=8.560ms max=124.338ms

IPv4 (142.251.163.102:53):
  Success: 100.0% (10/10)
  Latency: avg=29.358ms min=7.348ms max=129.031ms

UDP Results
----------------------------------------
IPv6 ([2607:f8b0:4009:818::200e]:53):
  Success: 100.0% (10/10)
  Latency: avg=101.460ms min=100.849ms max=101.835ms

IPv4 (142.251.163.102:53):
  Success: 100.0% (10/10)
  Latency: avg=101.516ms min=101.037ms max=102.161ms

Overall Performance Ranking
----------------------------------------
IPv6 Score: 31.17
IPv4 Score: 24.38

Winner: IPv6 (27.9% better)

Scoring: Based on success rate and latency (lower latency + higher success = higher score)
Weighting: TCP 60%, UDP 40%
```

### HTTP/HTTPS Testing
```
High-Fidelity IPv4/IPv6 Latency Tester (HTTP/HTTPS)
===============================================

Testing IPv4 connectivity to google.com:443...

============================================================
LATENCY TEST RESULTS
============================================================

IPv4 Results (google.com)
----------------------------------------
HTTP Requests: 10 sent, 10 successful, 0 failed (100.0% success)
Latency: min=121.058ms avg=147.394ms max=173.730ms stddev=26.336ms
Jitter: 52.672ms
Percentiles: P50=145.234ms P95=170.123ms P99=173.730ms
```

### DNS Query Testing
```
High-Fidelity IPv4/IPv6 Latency Tester (DNS (UDP))
===============================================

Testing IPv6 DNS to [2001:4860:4860::8888]:53 (query: dns-query.qosbox.com)...
Testing IPv4 DNS to 8.8.8.8:53 (query: dns-query.qosbox.com)...

============================================================
LATENCY TEST RESULTS
============================================================

IPv6 Results (2001:4860:4860::8888)
----------------------------------------
DNS Queries (UDP): 10 sent, 10 successful, 0 failed (100.0% success)
Latency: min=33.334ms avg=37.488ms max=45.178ms stddev=5.444ms
Jitter: 5.922ms
Percentiles: P50=35.123ms P95=44.567ms P99=45.178ms

IPv4 Results (8.8.8.8)
----------------------------------------
DNS Queries (UDP): 10 sent, 10 successful, 0 failed (100.0% success)
Latency: min=31.667ms avg=40.668ms max=46.396ms stddev=6.443ms
Jitter: 7.365ms
Percentiles: P50=39.234ms P95=45.123ms P99=46.396ms

IPv6 vs IPv4 Comparison
----------------------------------------
Average latency difference: 3.180ms (IPv6 is faster)
Success rate: IPv6=100.0% IPv4=100.0%
```

### JSON Output Format
```json
{
  "mode": "single",
  "protocol": "TCP",
  "targets": {
    "ipv4": "8.8.8.8",
    "ipv6": "2001:4860:4860::8888"
  },
  "ipv4_results": {
    "sent": 10,
    "received": 10,
    "lost": 0,
    "min_ms": 8200417,
    "max_ms": 9028750,
    "avg_ms": 8485444,
    "stddev_ms": 384330,
    "jitter_ms": 414166,
    "success_rate": 100.0
  },
  "ipv6_results": {
    "sent": 10,
    "received": 10,
    "lost": 0,
    "min_ms": 12331292,
    "max_ms": 19593625,
    "avg_ms": 16687417,
    "stddev_ms": 3137096,
    "jitter_ms": 3631166,
    "success_rate": 100.0
  },
  "test_config": {
    "count": 10,
    "interval_ms": 1000000000,
    "timeout_ms": 3000000000,
    "port": 53,
    "size": 64,
    "dns_query": "dns-query.qosbox.com",
    "dns_protocol": "udp",
    "verbose": false
  },
  "timestamp": "2025-09-29T11:53:09.71829-05:00"
}
```

#### JSON Compare Mode Output
```json
{
  "mode": "compare",
  "protocol": "DNS-UDP",
  "targets": {
    "hostname": "dns.google",
    "ipv4": "8.8.4.4",
    "ipv6": "2001:4860:4860::8844"
  },
  "comparison": {
    "dns_v4_stats": {
      "sent": 10,
      "received": 10,
      "lost": 0,
      "min_ms": 8257708,
      "max_ms": 158996250,
      "avg_ms": 47095933,
      "stddev_ms": 45818266,
      "jitter_ms": 16748726,
      "success_rate": 100.0
    },
    "dns_v6_stats": {
      "sent": 10,
      "received": 10,
      "lost": 0,
      "min_ms": 17678041,
      "max_ms": 39847375,
      "avg_ms": 26857975,
      "stddev_ms": 8140374,
      "jitter_ms": 2463259,
      "success_rate": 100.0
    },
    "ipv4_score": 2123.33,
    "ipv6_score": 3723.29,
    "winner": "IPv6",
    "resolved_ipv4": "8.8.4.4",
    "resolved_ipv6": "2001:4860:4860::8844",
    "protocol": "DNS-UDP",
    "hostname": "dns.google",
    "port": 53,
    "dns_query": "dns-query.qosbox.com",
    "timestamp": "2025-09-29T11:53:32.780339-05:00"
  },
  "test_config": {
    "count": 10,
    "interval_ms": 1000000000,
    "timeout_ms": 3000000000,
    "port": 53,
    "size": 64,
    "dns_query": "dns-query.qosbox.com",
    "dns_protocol": "udp",
    "verbose": false
  },
  "timestamp": "2025-09-29T11:53:32.780345-05:00"
}
```

## Configuration Files

ProtoTester supports YAML and JSON configuration files for defining multiple test scenarios, daemon mode operation, and batch testing.

### Configuration File Structure

```yaml
global:
  output_file: "results.log"              # Output file for all results
  log_level: "info"                       # Log level: debug, info, warn, error
  default_count: 10                       # Default test count for all tests
  timeout: "3s"                          # Default timeout for all tests
  interval: "1s"                         # Default interval between tests
  json_output: true                       # Use JSON output format (recommended)

  # InfluxDB time-series database integration
  influxdb:
    enabled: false                        # Enable InfluxDB output
    url: "http://localhost:8086"          # InfluxDB server URL
    token: "your-influxdb-token"          # InfluxDB authentication token
    organization: "your-organization"     # InfluxDB organization name
    bucket: "network-monitoring"          # InfluxDB bucket name
    measurement: "network_latency"        # InfluxDB measurement name (default: network_latency)
    batch_size: 1000                      # Number of points to batch before writing
    flush_interval: "5s"                  # How often to flush batched data to InfluxDB

# Daemon mode configuration for background service operation
daemon:
  enabled: false                          # Enable daemon mode
  run_interval: "5m"                      # How often to run complete test cycles
  output_file: "daemon.log"               # Daemon-specific output file
  log_file: "daemon.log"                  # Daemon log file for operational messages
  pid_file: "prototester.pid"             # PID file location for process management
  max_log_size: 104857600                 # Maximum log file size in bytes (100MB default)
  rotate_logs: true                       # Enable automatic log rotation
  stop_on_failure: false                  # Continue running even if individual tests fail
  max_retries: 3                          # Maximum number of retries for failed tests
  retry_interval: "30s"                   # Wait time between retry attempts

# Individual test definitions
tests:
  - name: "Google DNS TCP"                # Test identification name
    type: "tcp"                          # Protocol: tcp, udp, icmp, http, https, dns, dot, doh, compare
    target_ipv4: "8.8.8.8"              # IPv4 target address
    target_ipv6: "2001:4860:4860::8888"  # IPv6 target address (optional)
    port: 53                             # Target port number
    count: 10                            # Number of test iterations
    timeout: "3s"                        # Per-test timeout
    interval: "500ms"                    # Interval between individual tests
    size: 64                             # Packet size for applicable protocols (optional)
    ipv4_only: false                     # Test IPv4 only (optional)
    ipv6_only: false                     # Test IPv6 only (optional)
    enabled: true                        # Enable/disable this test
    schedule: "*/5 * * * *"              # Cron-like schedule (optional, daemon mode)

  - name: "DNS Query Test"
    type: "dns"                          # DNS query test
    target_ipv4: "8.8.8.8"
    target_ipv6: "2001:4860:4860::8888"
    port: 53
    dns_protocol: "udp"                  # DNS protocol: udp, tcp, dot (DNS over TLS), doh (DNS over HTTPS)
    dns_query: "google.com"              # Domain name to query
    count: 5
    enabled: true

  - name: "HTTP Performance"
    type: "http"                         # HTTP request test
    target_ipv4: "142.250.191.110"      # Web server IP
    port: 80
    count: 3
    enabled: true

  - name: "HTTPS Performance"
    type: "https"                        # HTTPS request test
    target_ipv4: "142.250.191.110"
    port: 443
    count: 3
    enabled: true

  - name: "ICMP Ping Test"
    type: "icmp"                         # ICMP ping test
    target_ipv4: "8.8.8.8"
    target_ipv6: "2001:4860:4860::8888"
    count: 10
    enabled: true

  - name: "Multi-Protocol Compare"
    type: "compare"                      # Compare multiple protocols
    hostname: "google.com"               # Hostname to resolve and test
    port: 80                             # Port for protocol tests
    count: 5
    enabled: true
```

### Configuration Parameters Reference

#### Global Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `output_file` | string | - | Output file path for test results |
| `log_level` | string | "info" | Log level: debug, info, warn, error |
| `default_count` | int | 10 | Default number of test iterations |
| `timeout` | duration | "3s" | Default timeout for all tests |
| `interval` | duration | "1s" | Default interval between tests |
| `json_output` | bool | false | Enable JSON output format |

#### InfluxDB Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | false | Enable InfluxDB output |
| `url` | string | - | InfluxDB server URL (e.g., "http://localhost:8086") |
| `token` | string | - | InfluxDB authentication token |
| `organization` | string | - | InfluxDB organization name |
| `bucket` | string | - | InfluxDB bucket for storing metrics |
| `measurement` | string | "network_latency" | InfluxDB measurement name |
| `batch_size` | int | 1000 | Number of points to batch before writing |
| `flush_interval` | duration | "5s" | How often to flush batched data |

#### Daemon Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | false | Enable daemon mode |
| `run_interval` | duration | "5m" | How often to run complete test cycles |
| `output_file` | string | - | Daemon-specific output file |
| `log_file` | string | - | Daemon log file for operational messages |
| `pid_file` | string | - | PID file location for process management |
| `max_log_size` | int | 104857600 | Maximum log file size in bytes (100MB) |
| `rotate_logs` | bool | true | Enable automatic log rotation |
| `stop_on_failure` | bool | false | Stop daemon if any test fails |
| `max_retries` | int | 3 | Maximum retries for failed tests |
| `retry_interval` | duration | "30s" | Wait time between retry attempts |

#### Test Configuration Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | **Required.** Test identification name |
| `type` | string | - | **Required.** Protocol type: tcp, udp, icmp, http, https, dns, dot, doh, compare |
| `target_ipv4` | string | - | IPv4 target address |
| `target_ipv6` | string | - | IPv6 target address (optional) |
| `hostname` | string | - | Hostname for compare mode (mutually exclusive with target_ipv4/ipv6) |
| `port` | int | 53 | Target port number |
| `count` | int | 10 | Number of test iterations |
| `timeout` | duration | "3s" | Per-test timeout |
| `interval` | duration | "1s" | Interval between individual tests |
| `size` | int | 64 | Packet size for applicable protocols |
| `ipv4_only` | bool | false | Test IPv4 only |
| `ipv6_only` | bool | false | Test IPv6 only |
| `enabled` | bool | true | Enable/disable this test |
| `schedule` | string | - | Cron-like schedule for daemon mode (optional) |
| `dns_protocol` | string | "udp" | DNS protocol: udp, tcp, dot, doh |
| `dns_query` | string | "google.com" | Domain name to query for DNS tests |

#### Protocol-Specific Notes

- **TCP/UDP**: Use `target_ipv4`/`target_ipv6` and `port`
- **ICMP**: Only uses `target_ipv4`/`target_ipv6`, `port` is ignored
- **HTTP/HTTPS**: Uses `target_ipv4`/`target_ipv6` and `port` (80/443 typically)
- **DNS**: Requires `dns_protocol` and `dns_query` parameters
- **DoT (DNS over TLS)**: Uses port 853 by default
- **DoH (DNS over HTTPS)**: Uses port 443 and HTTPS transport
- **Compare**: Uses `hostname` to resolve and test multiple protocols

### Running with Configuration Files

```bash
# Run tests from YAML configuration
./prototester -config example-config.yaml

# Run tests from JSON configuration
./prototester -config example-config.json

# Override output file
./prototester -config config.yaml -output custom-results.log

# Run in daemon mode
./prototester -config daemon-config.yaml -daemon

# Run with InfluxDB output enabled
./prototester -config influxdb-config.yaml
```

### InfluxDB Monitoring Examples

#### Production Network Monitoring with InfluxDB

```yaml
global:
  json_output: true
  influxdb:
    enabled: true
    url: "https://influxdb.company.com:8086"
    token: "${INFLUXDB_TOKEN}"              # Use environment variable
    organization: "network-ops"
    bucket: "network-monitoring"
    measurement: "network_latency"
    batch_size: 500                         # Higher frequency = smaller batches
    flush_interval: "2s"                    # Frequent flushes for real-time data

daemon:
  enabled: true
  run_interval: "30s"                       # High-frequency monitoring
  output_file: "/var/log/prototester.log"
  max_retries: 2
  retry_interval: "5s"

tests:
  # Critical infrastructure monitoring
  - name: "Primary DC Gateway"
    type: "icmp"
    target_ipv4: "10.0.1.1"
    count: 3
    enabled: true

  - name: "DNS Primary"
    type: "dns"
    target_ipv4: "10.0.10.1"
    dns_protocol: "udp"
    dns_query: "company.com"
    count: 2
    enabled: true

  - name: "Web Services"
    type: "https"
    target_ipv4: "10.0.20.100"
    port: 443
    count: 2
    enabled: true
```

#### Multi-Location Network Monitoring

```yaml
global:
  influxdb:
    enabled: true
    url: "http://localhost:8086"
    token: "monitoring-token"
    organization: "infrastructure"
    bucket: "network-metrics"
    measurement: "multi_location_latency"

daemon:
  enabled: true
  run_interval: "1m"

tests:
  # East Coast Data Center
  - name: "DC-East-Primary"
    type: "tcp"
    target_ipv4: "192.168.1.100"
    port: 22
    count: 5
    enabled: true

  - name: "DC-East-Backup"
    type: "tcp"
    target_ipv4: "192.168.1.101"
    port: 22
    count: 5
    enabled: true

  # West Coast Data Center
  - name: "DC-West-Primary"
    type: "tcp"
    target_ipv4: "192.168.2.100"
    port: 22
    count: 5
    enabled: true

  # External connectivity checks
  - name: "External-Google"
    type: "compare"
    hostname: "google.com"
    port: 443
    count: 3
    enabled: true

  - name: "External-Cloudflare"
    type: "compare"
    hostname: "cloudflare.com"
    port: 443
    count: 3
    enabled: true
```

#### DNS Performance Monitoring

```yaml
global:
  influxdb:
    enabled: true
    url: "http://localhost:8086"
    token: "dns-monitoring-token"
    organization: "dns-team"
    bucket: "dns-performance"
    measurement: "dns_latency"

daemon:
  enabled: true
  run_interval: "2m"

tests:
  # Primary DNS servers
  - name: "DNS-Primary-UDP"
    type: "dns"
    target_ipv4: "8.8.8.8"
    dns_protocol: "udp"
    dns_query: "example.com"
    count: 5
    enabled: true

  - name: "DNS-Primary-TCP"
    type: "dns"
    target_ipv4: "8.8.8.8"
    dns_protocol: "tcp"
    dns_query: "example.com"
    count: 3
    enabled: true

  - name: "DNS-Primary-DoH"
    type: "doh"
    target_ipv4: "8.8.8.8"
    dns_query: "example.com"
    count: 3
    enabled: true

  # Secondary DNS servers
  - name: "DNS-Secondary-UDP"
    type: "dns"
    target_ipv4: "1.1.1.1"
    dns_protocol: "udp"
    dns_query: "example.com"
    count: 5
    enabled: true

  - name: "DNS-Cloudflare-DoT"
    type: "dot"
    target_ipv4: "1.1.1.1"
    dns_query: "example.com"
    count: 3
    enabled: true
```

### Test Types

The configuration file supports all protocol types:

- **`tcp`**: TCP connection tests
- **`udp`**: UDP connectivity tests
- **`icmp`**: ICMP ping tests (with automatic fallback)
- **`http`**: HTTP request timing tests
- **`https`**: HTTPS request timing tests
- **`dns`**: DNS query tests (specify `dns_protocol`)
- **`dot`**: DNS-over-TLS tests (automatically sets protocol)
- **`doh`**: DNS-over-HTTPS tests (automatically sets protocol)
- **`compare`**: Protocol comparison tests (requires `hostname`)

### Configuration Examples

#### Basic Monitoring Setup
```yaml
global:
  output_file: "monitoring.log"
  json_output: true

tests:
  - name: "Primary DNS"
    type: "tcp"
    target_ipv4: "8.8.8.8"
    port: 53
    count: 5
    enabled: true

  - name: "Web Connectivity"
    type: "http"
    target_ipv4: "142.250.191.110"
    port: 80
    count: 3
    enabled: true
```

#### DNS Performance Testing
```yaml
global:
  default_count: 10

tests:
  - name: "DNS UDP"
    type: "dns"
    target_ipv4: "8.8.8.8"
    dns_protocol: "udp"
    dns_query: "example.com"
    enabled: true

  - name: "DNS over TLS"
    type: "dot"
    target_ipv4: "1.1.1.1"
    port: 853
    dns_query: "example.com"
    enabled: true

  - name: "DNS over HTTPS"
    type: "doh"
    target_ipv4: "1.1.1.1"
    port: 443
    dns_query: "example.com"
    enabled: true
```

## Daemon Mode

Daemon mode allows ProtoTester to run continuously as a background service, executing scheduled test cycles and logging results.

### Starting Daemon Mode

```bash
# Run as daemon with configuration file
./prototester -config daemon-config.yaml -daemon

# Run in background
nohup ./prototester -config daemon-config.yaml -daemon > /dev/null 2>&1 &

# With systemd (create service file)
sudo systemctl start prototester
```

### Daemon Configuration

```yaml
daemon:
  enabled: true
  run_interval: "5m"                      # Test every 5 minutes
  output_file: "/var/log/prototester.log" # Results output
  log_file: "/var/log/prototester.log"    # Daemon logs
  pid_file: "/var/run/prototester.pid"    # PID file
  max_log_size: 104857600                 # 100MB log rotation
  rotate_logs: true                       # Enable log rotation
  stop_on_failure: false                  # Continue on test failures
  max_retries: 3                          # Retry failed tests 3 times
  retry_interval: "30s"                   # Wait 30s between retries
```

### Daemon Features

- **Scheduled Execution**: Run test cycles at regular intervals
- **Graceful Shutdown**: Responds to SIGINT/SIGTERM signals
- **PID File Management**: Creates and cleans up PID files
- **Retry Logic**: Automatically retries failed tests
- **Log Rotation**: Prevents log files from growing too large
- **Error Handling**: Configurable behavior on test failures
- **Signal Handling**: Proper cleanup on shutdown

### Stopping Daemon

```bash
# Send termination signal
kill $(cat prototester.pid)

# Force stop if necessary
kill -9 $(cat prototester.pid)

# With systemd
sudo systemctl stop prototester
```

### Daemon Output

#### Human-Readable Format
```
[2025-09-29 12:00:00] Primary DNS (tcp): SUCCESS - Duration: 0.01s
[2025-09-29 12:00:01] Web Test (http): SUCCESS - Duration: 0.15s

=== Test Summary ===
Total tests: 2
Successful: 2
Failed: 0
Success rate: 100.0%
```

#### JSON Format
```json
{
  "test_name": "Primary DNS",
  "timestamp": "2025-09-29T12:00:00Z",
  "test_type": "tcp",
  "target": "IPv4:8.8.8.8 IPv6:2001:4860:4860::8888",
  "success": true,
  "results": {
    "ipv4_results": {
      "sent": 5,
      "received": 5,
      "success_rate": 100.0,
      "avg_ms": 12345678
    }
  },
  "duration_seconds": 0.01
}
```

## InfluxDB Integration

ProtoTester supports optional integration with InfluxDB for time-series storage and monitoring of network latency metrics. This enables long-term data analysis, alerting, and visualization with tools like Grafana.

### Configuration

Enable InfluxDB output by configuring the `influxdb` section in your configuration file:

```yaml
global:
  influxdb:
    enabled: true
    url: "http://localhost:8086"
    token: "your-influxdb-token"
    organization: "your-organization"
    bucket: "network-monitoring"
    measurement: "network_latency"
    batch_size: 1000
    flush_interval: "5s"
```

### InfluxDB Setup

1. **Install InfluxDB**: Follow the [official InfluxDB installation guide](https://docs.influxdata.com/influxdb/v2.0/install/)

2. **Create Token**: Generate an authentication token with write permissions to your target bucket

3. **Create Bucket**: Create a bucket for storing network monitoring data

4. **Configure ProtoTester**: Update your configuration file with the InfluxDB connection details

### Data Schema

ProtoTester writes the following metrics to InfluxDB:

**Tags**:
- `test_name`: Name of the test from configuration
- `test_type`: Protocol type (tcp, udp, icmp, http, https, dns, etc.)
- `target`: Target address being tested
- `ip_version`: IP version (4 or 6)

**Fields**:
- `sent`: Number of packets/requests sent
- `received`: Number of successful responses
- `lost`: Number of lost packets/requests
- `min_ms`: Minimum latency in milliseconds
- `max_ms`: Maximum latency in milliseconds
- `avg_ms`: Average latency in milliseconds
- `stddev_ms`: Standard deviation of latency
- `jitter_ms`: Jitter measurement
- `success_rate`: Success rate percentage (0-100)

### Usage Examples

#### Basic InfluxDB Monitoring

```yaml
global:
  influxdb:
    enabled: true
    url: "http://localhost:8086"
    token: "your-token"
    organization: "monitoring"
    bucket: "network-metrics"

tests:
  - name: "DNS Performance"
    type: "dns"
    target_ipv4: "8.8.8.8"
    port: 53
    count: 5
```

#### High-Frequency Monitoring

```yaml
global:
  influxdb:
    enabled: true
    url: "http://influxdb.example.com:8086"
    token: "your-production-token"
    organization: "ops"
    bucket: "network-monitoring"
    batch_size: 500
    flush_interval: "1s"

daemon:
  enabled: true
  run_interval: "10s"

tests:
  - name: "Critical Service"
    type: "https"
    target_ipv4: "192.168.1.100"
    port: 443
    count: 3
```

### Grafana Integration

Once data is flowing into InfluxDB, you can create Grafana dashboards to visualize network performance:

1. **Add InfluxDB Datasource**: Configure Grafana with your InfluxDB instance
2. **Create Queries**: Use Flux queries to analyze latency trends
3. **Set up Alerts**: Configure alerting based on latency thresholds

Example Flux query for average latency over time:
```flux
from(bucket: "network-monitoring")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "network_latency")
  |> filter(fn: (r) => r._field == "avg_ms")
  |> aggregateWindow(every: 1m, fn: mean)
```

### Troubleshooting

- **Connection Issues**: Verify InfluxDB URL, token, and network connectivity
- **Permission Errors**: Ensure the token has write permissions to the specified bucket
- **Data Not Appearing**: Check ProtoTester logs for InfluxDB write errors
- **Performance**: Adjust `batch_size` and `flush_interval` for your write volume

## Technical Details

### Protocol Implementation

#### TCP Mode (Default)
- Uses TCP connection establishment time as latency measurement
- Tests application-level connectivity and performance
- No special privileges required
- Measures complete connection setup time
- Most reliable for application-level connectivity testing

#### UDP Mode
- Tests UDP connectivity with write operations
- Connectionless protocol testing
- Considers successful write as indication of reachability
- Useful for testing services like DNS

#### ICMP Mode (Smart Implementation)
- **Linux Unprivileged ICMP**: Automatically tries `SOCK_DGRAM` ICMP sockets first (no root required on modern Linux)
  - Uses `syscall.Connect()` and `syscall.Write()` for packet transmission
  - Kernel manages ICMP ID field automatically
  - Only sequence number matching required for replies
- **Cross-Platform Support**: Platform-specific implementations for Linux, macOS, and Windows
  - Uses build tags to handle different syscall interfaces across platforms
  - Platform-specific socket descriptor types (int on Unix, Handle on Windows)
  - Abstracts platform differences in select(), socket timeout, and I/O operations
- **Raw Socket Fallback**: Uses raw ICMP sockets when unprivileged fails (requires root/admin)
- **TCP Fallback**: Automatic TCP mode if all ICMP methods fail
- **EINTR Handling**: Properly handles interrupted system calls with retry logic
- Provides pure network-level latency without application overhead
- Implements proper ICMP Echo Request/Reply handling for both IPv4 and IPv6

#### HTTP/HTTPS Mode
- Uses HTTP HEAD requests to minimize data transfer
- Automatically detects HTTP vs HTTPS based on port (443, 8443 = HTTPS)
- Measures full HTTP request/response cycle including TLS handshake
- Skips certificate validation for testing purposes
- Forces IPv4 or IPv6 as specified

#### DNS Mode (High-Fidelity DNS Testing)
- **UDP DNS**: Traditional DNS queries over UDP (RFC 1035)
  - Fastest DNS protocol, minimal overhead
  - Uses dns-query.qosbox.com as default test domain
  - Validates response ID matching for accuracy
- **TCP DNS**: DNS queries over TCP for larger responses
  - Handles DNS responses larger than 512 bytes
  - Includes TCP connection establishment time
  - Uses length-prefixed DNS messages
- **DoT (DNS over TLS)**: Secure DNS over TLS (RFC 7858)
  - Encrypted DNS queries for privacy
  - Typically uses port 853
  - Includes TLS handshake time in measurements
- **DoH (DNS over HTTPS)**: DNS over HTTPS (RFC 8484)
  - DNS queries over HTTPS for ultimate privacy
  - Uses POST requests with DNS wire format
  - Includes full HTTP/TLS overhead in timing

### Compare Mode
- Performs DNS resolution to obtain both A (IPv4) and AAAA (IPv6) records
- **Default Mode**: Tests both TCP and UDP protocols automatically (10 tests each by default)
- **Protocol-Specific Modes**: Use `-icmp`, `-http`, or `-dns` for focused comparison testing
- **ICMP Compare**: Compares pure ICMP ping performance between IPv4 and IPv6
- **HTTP Compare**: Compares HTTP/HTTPS request timing between protocols
- **DNS Compare**: Tests DNS query performance using specified protocol (UDP/TCP/DoT/DoH)
- Calculates performance scores: (success_rate) × (1000 / avg_latency_ms)
- **TCP/UDP Weighting**: TCP 60%, UDP 40% in default compare mode
- Provides comprehensive ranking and percentage performance difference
- Supports JSON output for programmatic analysis

### Statistics
- Calculates jitter as the average absolute difference between consecutive latencies
- Provides percentile calculations (P50, P95, P99) for latency distribution analysis
- Thread-safe result collection for concurrent testing
- High-precision nanosecond timing throughout

## Common Usage Examples

### Quick Connectivity Tests
```bash
# Quick test - works immediately, no setup needed
./prototester -c 3

# Test specific service
./prototester -t -p 22 -4 github.com

# Monitor performance over time
./prototester -c 50 -i 200ms
```

### Service-Specific Testing
```bash
# Web server testing
./prototester -http -p 443 -4 example.com

# DNS server testing (multiple protocols)
./prototester -dns -dns-protocol udp -4 1.1.1.1
./prototester -dns -dns-protocol dot -4 1.1.1.1 -p 853
./prototester -dns -dns-protocol doh -4 1.1.1.1 -p 443

# SSH connectivity
./prototester -t -p 22 -4 your-server.com

# DNS performance comparison across protocols
./prototester -dns -dns-protocol udp -c 20 -4 8.8.8.8
./prototester -dns -dns-protocol tcp -c 20 -4 8.8.8.8
./prototester -dns -dns-protocol dot -c 20 -4 8.8.8.8 -p 853
```

### Network Analysis
```bash
# Compare protocols for a service (default TCP/UDP)
./prototester -compare your-service.com -p 80

# Protocol-specific comparisons
./prototester -compare google.com -http -p 80      # HTTP performance comparison
./prototester -compare dns.google -dns             # DNS performance comparison
./prototester -compare example.com -icmp           # ICMP latency comparison

# IPv6 deployment testing
./prototester -6only -6 your-ipv6-server.com

# High-frequency testing
./prototester -c 100 -i 100ms -v

# JSON output for automation
./prototester -compare your-service.com -json > results.json
./prototester -dns -dns-protocol doh -json | jq '.ipv4_results.avg_ms'
```

## Troubleshooting

### Common Issues
- **"Cannot specify multiple protocol flags"**: Use only one of `-t`, `-u`, `-icmp`, `-http`, `-dns` at a time
- **Connection timeouts**: Increase timeout with `-timeout 10s`
- **"No A or AAAA records found"**: Hostname doesn't resolve to both IPv4 and IPv6 (for compare mode)
- **"Invalid DNS protocol"**: Must be one of: udp, tcp, dot, doh

### Permission-Related
- **"Operation not permitted" with ICMP**: This is normal - the tool automatically falls back to TCP
- **Want true ICMP?**: Run with `sudo ./prototester -icmp`
- **Linux users**: The tool automatically tries unprivileged ICMP first

### IPv6 Issues
- **IPv6 connectivity problems**: Test IPv4 only with `-4only`
- **"No route to host"**: Your network may not support IPv6
- **Verify IPv6**: Try `ping6 google.com` outside the tool

### HTTP/HTTPS Issues
- **Connection refused**: Verify the port is correct (80 for HTTP, 443 for HTTPS)
- **TLS errors**: The tool skips certificate validation, so this shouldn't occur
- **Some servers block HEAD requests**: This is expected behavior for some services

### DNS Issues
- **"DNS response too short"**: Server may not support the requested protocol
- **"DNS response ID mismatch"**: Network interference or server issues
- **DoT connection failures**: Verify server supports DNS-over-TLS on port 853
- **DoH HTTP errors**: Verify server supports DNS-over-HTTPS at /dns-query endpoint
- **Custom query domains**: Use valid domain names (avoid localhost, IP addresses)

## Migration from Root-Required Version

If you were previously running this tool with `sudo`, you can now:

1. **Remove `sudo` for most use cases**: `./prototester` works immediately
2. **Use `-icmp` for ICMP testing**: It will automatically fall back to TCP if no root
3. **Keep `sudo` only for true ICMP**: `sudo ./prototester -icmp` for raw socket ICMP
4. **Try new protocols**: `-http` mode for web service testing, `-dns` for DNS performance analysis

## DNS Testing Capabilities

The DNS testing feature provides comprehensive analysis of DNS performance across multiple protocols:

### Supported DNS Protocols
- **UDP DNS** (default): Traditional DNS, fastest with minimal overhead
- **TCP DNS**: For larger responses, includes connection establishment time
- **DoT (DNS over TLS)**: Encrypted DNS for privacy, typically port 853
- **DoH (DNS over HTTPS)**: DNS over HTTPS for maximum privacy and circumventing blocks

### DNS Testing Examples
```bash
# Compare DNS protocols performance
./prototester -dns -dns-protocol udp -c 20    # Traditional UDP
./prototester -dns -dns-protocol tcp -c 20    # TCP for reliability
./prototester -dns -dns-protocol dot -p 853 -c 20  # Encrypted DoT
./prototester -dns -dns-protocol doh -p 443 -c 20  # DoH over HTTPS

# Test specific DNS providers
./prototester -dns -4 1.1.1.1 -dns-query cloudflare.com    # Cloudflare
./prototester -dns -4 8.8.8.8 -dns-query google.com        # Google
./prototester -dns -4 9.9.9.9 -dns-query quad9.net         # Quad9

# DNS latency monitoring
./prototester -dns -c 50 -i 500ms -v                       # High frequency
./prototester -dns -dns-protocol dot -4 1.1.1.1 -p 853 -c 100  # DoT monitoring

# Privacy-focused DNS testing
./prototester -dns -dns-protocol doh -4 1.1.1.1 -p 443 -dns-query dns-query.qosbox.com
```

### Default Test Domain
The tool uses `dns-query.qosbox.com` as the default query domain, which is specifically designed for DNS performance testing and provides consistent, reliable responses across all DNS protocols.

## Use Cases

### Network Performance Analysis
- **DNS Provider Comparison**: Test multiple DNS providers to find the fastest
- **Protocol Performance**: Compare UDP vs TCP vs DoT vs DoH performance
- **Geographic Performance**: Test DNS servers in different regions
- **Privacy vs Performance**: Measure the overhead of encrypted DNS protocols

### Security and Privacy Testing
- **DoT Deployment**: Verify DNS-over-TLS is working correctly
- **DoH Testing**: Ensure DNS-over-HTTPS is functioning and performing well
- **Fallback Testing**: Test DNS resolution when certain protocols are blocked
- **Censorship Circumvention**: Verify encrypted DNS works in restricted networks

### Troubleshooting DNS Issues
- **Resolution Latency**: Identify slow DNS responses affecting application performance
- **Protocol Availability**: Test which DNS protocols are supported/blocked
- **IPv4 vs IPv6 DNS**: Compare DNS performance over different IP versions
- **DNS Load Testing**: High-frequency testing to identify capacity limits

## License

MIT License
