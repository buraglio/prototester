# ProtoTester GUI - Quick Start Guide

## Running the App

### Option 1: Launch from Finder
1. Navigate to `build/bin/`
2. Double-click `prototester-gui.app`
3. If macOS blocks it: Right-click → Open → Open

### Option 2: Launch from Terminal
```bash
open build/bin/prototester-gui.app
```

## Running Tests

### Basic TCP Test (Default)
1. Protocol: **TCP Connect** (already selected)
2. Target IPv4: `8.8.8.8` (Google DNS)
3. Target IPv6: `2001:4860:4860::8888`
4. Port: `53`
5. Click **▶️ Run Test**

### ICMP Ping Test
1. Select Protocol: **ICMP Ping**
2. Enter target addresses
3. Adjust packet size if needed (default: 64 bytes)
4. Click **▶️ Run Test**
   - Note: Falls back to TCP if no root permissions

### DNS Query Test
1. Select Protocol: **DNS Query**
2. Choose DNS Protocol:
   - **UDP** - Fastest, traditional DNS
   - **TCP** - For larger responses
   - **DoT** - Encrypted via TLS (port 853)
   - **DoH** - Encrypted via HTTPS (port 443)
3. Enter domain to query (default: `dns-query.qosbox.com`)
4. Click **▶️ Run Test**

### Compare IPv4 vs IPv6
1. Select Protocol: **Compare IPv4/IPv6**
2. Enter hostname (e.g., `google.com`)
3. The app will:
   - Resolve to both IPv4 and IPv6 addresses
   - Test both protocols
   - Calculate performance scores
   - Show which is better
4. Click **▶️ Run Test**

## Understanding Results

### Statistics Explained

- **Success Rate**: % of packets that received responses
- **Min Latency**: Fastest response time
- **Avg Latency**: Average response time (most important)
- **Max Latency**: Slowest response time
- **Jitter**: Network stability (lower is better)
- **Std Dev**: How much latency varies

### What's Good?

| Metric | Excellent | Good | Fair | Poor |
|--------|-----------|------|------|------|
| Latency | < 10ms | 10-50ms | 50-100ms | > 100ms |
| Jitter | < 5ms | 5-20ms | 20-50ms | > 50ms |
| Success Rate | 100% | 95-99% | 90-95% | < 90% |

### Comparison Scores

In compare mode:
- **Score** = (Success Rate) × (1000 / Average Latency)
- Higher score = better performance
- Winner is displayed with % difference

## Common Test Scenarios

### Test Your Internet Connection
```
Protocol: TCP Connect
IPv4: 1.1.1.1 (Cloudflare)
IPv6: 2606:4700:4700::1111
Port: 443
```

### Test Local Network
```
Protocol: ICMP Ping
IPv4: 192.168.1.1 (your router)
Count: 20
Interval: 100ms
```

### Test DNS Performance
```
Protocol: DNS Query
DNS Protocol: UDP
Target: 1.1.1.1
Query: google.com
Count: 10
```

### Compare DNS Providers
Run multiple tests with different targets:
- Google DNS: `8.8.8.8`
- Cloudflare: `1.1.1.1`
- Quad9: `9.9.9.9`

## Tips & Tricks

### For Accurate Results
- Run at least 10 tests (`count: 10`)
- Use 1 second intervals for normal testing
- Use longer timeouts (5-10s) for slow connections

### For Quick Tests
- Reduce count to 3-5
- Decrease interval to 100-500ms
- Good for initial connectivity checks

### For Detailed Analysis
- Increase count to 50-100
- Use 1-2 second intervals
- Watch for patterns in latency

## Keyboard Shortcuts

- **Enter** when in any input field: Runs the test
- **Tab**: Navigate between fields
- **Space**: Toggle checkboxes

## Troubleshooting

### "Connection refused"
- Check if target is reachable
- Verify port number is correct
- Try different protocol

### "Timeout"
- Increase timeout value
- Check internet connection
- Target may be blocking requests

### ICMP shows "fallback to TCP"
- This is normal without root permissions
- For true ICMP: `sudo open prototester-gui.app`
- TCP fallback gives similar results

### High jitter/latency
- Network congestion
- WiFi interference
- Try wired connection
- Run test multiple times

## Advanced Usage

### Testing Web Servers
```
Protocol: HTTP/HTTPS
IPv4: example.com
Port: 443
```

### Testing SSH Connectivity
```
Protocol: TCP Connect
IPv4: your-server.com
Port: 22
```

### Testing DoH Providers
```
Protocol: DNS Query
DNS Protocol: DoH
Target: 1.1.1.1
Port: 443
Query: example.com
```

## Development Mode

To run in development mode with hot reload:

```bash
cd prototester-gui
wails dev
```

Access at: `http://localhost:34115`

## Building

```bash
# Production build
wails build

# Development build (faster)
wails build -debug

# Build for distribution
wails build -clean
```

## Support

- Check [README-GUI.md](README-GUI.md) for full documentation
- Review original [README.md](../README.md) for protocol details
- Report issues on GitHub

---

**Happy Testing! 🚀**
