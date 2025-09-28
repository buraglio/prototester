# prototester - High-Fidelity IPv4/IPv6 Latency Tester

A comprehensive Go program that tests IPv4 and IPv6 connectivity and performance with high precision timing and detailed comparative analysis.

## Features

- **Unified Protocol Support**: Single tool supporting ICMP, TCP, and UDP testing modes
- **Compare Mode**: Automatic hostname resolution and comprehensive IPv4 vs IPv6 performance comparison
- **High-Precision Timing**: Uses nanosecond-precision timing for accurate latency measurements
- **Comprehensive Statistics**: Provides min/max/avg latency, standard deviation, jitter, and percentiles
- **Multiple Test Types**:
  - Raw ICMP sockets (requires root) for network-level measurements
  - TCP connection-based testing (no root required) for application-level measurements
  - UDP testing for connectionless protocol analysis
- **Intelligent Scoring**: Performance ranking system based on success rate and latency
- **Flexible Configuration**: Customizable targets, connection count, intervals, timeouts, and ports
- **Detailed Reporting**: Comparative analysis between IPv4 and IPv6 performance

## Requirements

- Go 1.21 or higher
- Network connectivity to test targets
- For ICMP version: Root/Administrator privileges (required for raw ICMP sockets)

## Installation

```bash
git clone <repository>
cd prototester
go mod tidy

# Build unified version (supports ICMP, TCP, UDP, and compare modes)
go build -o prototester main.go
```

## Usage

### Compare Mode (Recommended - Comprehensive Analysis)
```bash
# Automatically resolve hostname and compare IPv4 vs IPv6 performance
./prototester -compare google.com

# Compare using HTTPS port
./prototester -compare cloudflare.com -p 443

# Compare with custom settings
./prototester -compare github.com -p 22 -v
```

### TCP-Based Testing (No Root Required)
```bash
# Test both IPv4 and IPv6 using TCP connections
./prototester -t

# Test only IPv4 with custom target
./prototester -t -4only -4 1.1.1.1 -p 443

# Custom targets and settings
./prototester -t -4 1.1.1.1 -6 2606:4700:4700::1111 -p 443 -c 20 -v
```

### UDP-Based Testing
```bash
# Test both IPv4 and IPv6 using UDP (default addresses)
./prototester -u

# Test only specific IPv6 DNS server (auto-enables IPv6-only)
./prototester -u -6 2606:4700:4700::1111 -p 53 -v

# Test only specific IPv4 DNS server (auto-enables IPv4-only)
./prototester -u -4 1.1.1.1 -p 53 -v

# Test both custom addresses
./prototester -u -4 8.8.8.8 -6 2001:4860:4860::8888 -p 53 -v
```

### ICMP-Based Testing (Requires Root)
```bash
# Test both IPv4 and IPv6 using raw ICMP (default mode)
sudo ./prototester

# Custom ICMP settings
sudo ./prototester -4 1.1.1.1 -6 2606:4700:4700::1111 -c 20 -i 500ms -v
```

## Command Line Options

### Core Options
- `-4 string`: IPv4 target address (default: "8.8.8.8", auto-enables IPv4-only if custom)
- `-6 string`: IPv6 target address (default: "2001:4860:4860::8888", auto-enables IPv6-only if custom)
- `-c int`: Number of tests to perform (default: 10)
- `-i duration`: Interval between tests (default: 1s)
- `-timeout duration`: Timeout for each test (default: 3s)
- `-4only`: Test IPv4 only (explicit override)
- `-6only`: Test IPv6 only (explicit override)
- `-v`: Verbose output

**Smart Protocol Selection**:
- By default, both IPv4 and IPv6 are tested using default addresses
- When you specify a custom `-4` address (without custom `-6`), only IPv4 is tested
- When you specify a custom `-6` address (without custom `-4`), only IPv6 is tested
- When you specify both custom addresses, both protocols are tested
- Explicit `-4only` or `-6only` flags override the smart selection
- IPv6 is tested first and displayed with priority to encourage IPv6 adoption

### Protocol Selection (Mutually Exclusive)
- `-t`: Use TCP connect test instead of ICMP
- `-u`: Use UDP test instead of ICMP
- `-compare string`: Compare mode - resolve hostname and test both TCP/UDP on IPv4/IPv6

### Protocol-Specific Options
- `-p int`: Port to test (for TCP/UDP modes, default: 53)
- `-s int`: Packet size in bytes (ICMP only, default: 64)

### Compare Mode
The compare mode (`-compare hostname`) performs comprehensive testing:
- Resolves both A and AAAA records for the hostname
- Tests both TCP and UDP connectivity (10 tests each)
- Calculates performance scores based on success rate and latency
- Provides an overall ranking of IPv4 vs IPv6 performance
- Uses weighted scoring: TCP 60%, UDP 40%

## Sample Output

### Compare Mode Output
```
High-Fidelity IPv4/IPv6 Comparison Mode
=======================================

Resolving google.com...
Resolved addresses:
  IPv4 (A): 142.251.163.102
  IPv6 (AAAA): 2607:f8b0:4009:818::200e

Testing TCP IPv6 ([2607:f8b0:4009:818::200e]:443)...
Testing TCP IPv4 (142.251.163.102:443)...
Testing UDP IPv6 ([2607:f8b0:4009:818::200e]:443)...
Testing UDP IPv4 (142.251.163.102:443)...

============================================================
COMPREHENSIVE COMPARISON RESULTS
============================================================

TCP Results
----------------------------------------
IPv6 ([2607:f8b0:4009:818::200e]:443):
  Success: 100.0% (10/10)
  Latency: avg=22.038ms min=8.560ms max=124.338ms

IPv4 (142.251.163.102:443):
  Success: 100.0% (10/10)
  Latency: avg=29.358ms min=7.348ms max=129.031ms

UDP Results
----------------------------------------
IPv6 ([2607:f8b0:4009:818::200e]:443):
  Success: 100.0% (10/10)
  Latency: avg=101.460ms min=100.849ms max=101.835ms

IPv4 (142.251.163.102:443):
  Success: 100.0% (10/10)
  Latency: avg=101.516ms min=101.037ms max=102.161ms

Overall Performance Ranking
----------------------------------------
IPv6 Score: 31.17
IPv4 Score: 24.38

🏆 Winner: IPv6 (27.9% better)

Scoring: Based on success rate and latency (lower latency + higher success = higher score)
Weighting: TCP 60%, UDP 40%
```

### Traditional Mode Output
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

## Technical Details

### Compare Mode
- Performs DNS resolution to obtain both A (IPv4) and AAAA (IPv6) records
- Tests both TCP and UDP protocols automatically (10 tests each by default)
- Calculates weighted performance scores: TCP 60%, UDP 40%
- Score formula: (success_rate) × (1000 / avg_latency_ms)
- Provides comprehensive ranking and percentage performance difference

### Protocol Support
- **ICMP Mode**: Uses raw ICMP sockets for precise network-level measurements
  - Requires root privileges for raw socket access
  - Provides pure network-level latency without application overhead
  - Implements proper ICMP Echo Request/Reply handling for both protocols

- **TCP Mode**: Uses TCP connection establishment time as latency measurement
  - Tests application-level connectivity and performance
  - No special privileges required
  - Measures complete connection setup time

- **UDP Mode**: Tests UDP connectivity with write operations
  - Connectionless protocol testing
  - Considers successful write as indication of reachability
  - Useful for testing services like DNS

### Statistics
- Calculates jitter as the average absolute difference between consecutive latencies
- Provides percentile calculations (P50, P95, P99) for latency distribution analysis
- Thread-safe result collection for concurrent testing
- High-precision nanosecond timing throughout

## Troubleshooting

### General Issues
- **"permission denied"**: Run with `sudo` for ICMP mode (raw socket access)
- **"no A or AAAA records found"**: Hostname doesn't resolve to both IPv4 and IPv6
- **"Cannot specify both -t and -u flags"**: Use only one protocol flag at a time

### Compare Mode Issues
- **"No IPv4/IPv6 address found"**: Hostname must resolve to both protocols for comparison
- **TCP connection failures**: Try different ports (443 for HTTPS, 22 for SSH, 80 for HTTP)
- **All tests failing**: Check firewall settings and network connectivity

### Protocol-Specific Issues
- **ICMP "i/o timeout"**: Some networks block ICMP, try TCP/UDP modes instead
- **TCP "connection refused"**: Target port may not be listening
- **UDP latency seems high**: UDP includes timeout waiting for response
- **IPv6 connectivity issues**: Ensure your network supports IPv6

## License

MIT License