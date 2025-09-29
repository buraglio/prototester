# ProtoTester - High-Fidelity IPv4/IPv6 Latency Tester

A comprehensive Go program that tests IPv4 and IPv6 connectivity and performance with high precision timing and detailed comparative analysis. **Now works without root privileges by default!**

## Features

- **Multiple Protocol Support**: TCP, UDP, ICMP, HTTP/HTTPS latency testing
- **No Root Required**: Defaults to TCP mode, works out-of-the-box for all users
- **Smart Fallbacks**: Automatically falls back from ICMP to TCP when permissions are insufficient
- **Linux Optimization**: Uses unprivileged ICMP sockets on Linux when available
- **Compare Mode**: Automatic hostname resolution and comprehensive IPv4 vs IPv6 performance comparison
- **High-Precision Timing**: Uses nanosecond-precision timing for accurate latency measurements
- **Comprehensive Statistics**: Provides min/max/avg latency, standard deviation, jitter, and percentiles
- **Cross-Platform**: Works on Linux, macOS, and other Unix-like systems
- **IPv4/IPv6 Dual Stack**: Tests both protocols simultaneously or individually
- **Intelligent Scoring**: Performance ranking system based on success rate and latency
- **Flexible Configuration**: Customizable targets, connection count, intervals, timeouts, and ports

## Requirements

- Go 1.21 or higher
- Network connectivity to test targets
- **No special privileges required for default operation**
- Optional: Root/Administrator privileges for true ICMP testing

## Installation

```bash
git clone <repository>
cd prototester
go mod tidy
go build -o prototester main.go
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
# Automatically resolve hostname and compare IPv4 vs IPv6 performance
./prototester -compare google.com

# Compare using HTTPS port
./prototester -compare cloudflare.com -p 443

# Compare with verbose output
./prototester -compare github.com -p 22 -v
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
- `-compare <hostname>`: Compare mode - test both TCP/UDP on IPv4/IPv6

### Protocol-Specific Options
- `-p <port>`: Port to test (TCP/UDP/HTTP modes, default: 53)
- `-s <size>`: Packet size in bytes (ICMP only, default: 64)

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

### ICMP Mode Behavior (Smart Fallback)
1. **Linux**: First tries unprivileged ICMP sockets (`SOCK_DGRAM`)
2. **All Systems**: Falls back to raw sockets (requires root)
3. **Final Fallback**: If ICMP fails due to permissions, automatically uses TCP
4. **Verbose Feedback**: Shows "ICMP failed (no root), falling back to TCP connect test..."

### Running with Root (Optional)
```bash
# Enable true ICMP ping on all platforms
sudo ./prototester -icmp

# Root enables raw socket ICMP with larger packets
sudo ./prototester -icmp -s 1400 -v
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

🏆 Winner: IPv6 (27.9% better)

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
- **Linux**: First tries unprivileged ICMP sockets (`SOCK_DGRAM` with `IPPROTO_ICMP`)
- **Fallback**: Uses raw ICMP sockets (requires root)
- **Final Fallback**: Automatic TCP mode if permissions insufficient
- Provides pure network-level latency without application overhead
- Implements proper ICMP Echo Request/Reply handling for both protocols

#### HTTP/HTTPS Mode
- Uses HTTP HEAD requests to minimize data transfer
- Automatically detects HTTP vs HTTPS based on port (443, 8443 = HTTPS)
- Measures full HTTP request/response cycle including TLS handshake
- Skips certificate validation for testing purposes
- Forces IPv4 or IPv6 as specified

### Compare Mode
- Performs DNS resolution to obtain both A (IPv4) and AAAA (IPv6) records
- Tests both TCP and UDP protocols automatically (10 tests each by default)
- Calculates weighted performance scores: TCP 60%, UDP 40%
- Score formula: (success_rate) × (1000 / avg_latency_ms)
- Provides comprehensive ranking and percentage performance difference

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

# DNS server testing
./prototester -u -p 53 -4 1.1.1.1

# SSH connectivity
./prototester -t -p 22 -4 your-server.com
```

### Network Analysis
```bash
# Compare protocols for a service
./prototester -compare your-service.com -p 80

# IPv6 deployment testing
./prototester -6only -6 your-ipv6-server.com

# High-frequency testing
./prototester -c 100 -i 100ms -v
```

## Troubleshooting

### Common Issues
- **"Cannot specify multiple protocol flags"**: Use only one of `-t`, `-u`, `-icmp`, `-http` at a time
- **Connection timeouts**: Increase timeout with `-timeout 10s`
- **"No A or AAAA records found"**: Hostname doesn't resolve to both IPv4 and IPv6 (for compare mode)

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

## Migration from Root-Required Version

If you were previously running this tool with `sudo`, you can now:

1. **Remove `sudo` for most use cases**: `./prototester` works immediately
2. **Use `-icmp` for ICMP testing**: It will automatically fall back to TCP if no root
3. **Keep `sudo` only for true ICMP**: `sudo ./prototester -icmp` for raw socket ICMP
4. **Try new protocols**: `-http` mode for web service testing

## License

MIT License