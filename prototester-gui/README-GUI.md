# ProtoTester GUI - Native macOS Application

A beautiful, native macOS application for high-fidelity network latency testing built with Wails and Svelte.

## Overview

ProtoTester GUI is a desktop application that provides a modern, user-friendly interface for the powerful ProtoTester network testing engine. It combines the performance of Go with a sleek web-based UI.

## Features

- **Modern Native macOS App**: Built with Wails framework for true native performance
- **Beautiful UI**: Gradient design with responsive layouts and smooth animations
- **All ProtoTester Features**: Full support for TCP, UDP, ICMP, HTTP/HTTPS, DNS testing
- **Multiple DNS Protocols**: UDP, TCP, DoT (DNS over TLS), DoH (DNS over HTTPS)
- **IPv4/IPv6 Comparison**: Side-by-side performance analysis
- **Real-time Results**: Live statistics display with formatted metrics
- **Lightweight**: Only 8.9MB app bundle size
- **Fast**: Direct Go-to-JavaScript bindings with no IPC overhead

## Screenshots

The application features:
- Left panel: Test configuration with dynamic fields based on selected protocol
- Right panel: Comprehensive results display with statistics grids
- Beautiful gradient background and card-based layout
- Loading states and error handling

## Installation

### Pre-built App

Simply download `prototester-gui.app` and drag it to your Applications folder.

### Building from Source

#### Prerequisites

- Go 1.21 or higher
- Node.js 14+ and npm
- Wails CLI v2.10.2+
- Xcode Command Line Tools

#### Build Steps

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone the repository
cd prototester-gui

# Build the application
wails build

# The app will be created at:
# build/bin/prototester-gui.app
```

#### Development Mode

```bash
# Run in development mode with hot reload
wails dev
```

## Usage

### Quick Start

1. Launch ProtoTester from your Applications folder
2. Select a protocol from the dropdown (TCP, UDP, ICMP, HTTP, DNS, or Compare)
3. Configure test parameters:
   - Target addresses (IPv4/IPv6)
   - Port number
   - Test count and interval
   - Protocol-specific options
4. Click "Run Test"
5. View detailed results in the right panel

### Protocol-Specific Testing

#### TCP Connect Test
- Tests TCP connectivity to specified target
- Measures connection establishment time
- No special permissions required

#### UDP Test
- Tests UDP connectivity
- Useful for DNS servers and other UDP services

#### ICMP Ping
- Traditional ping test
- Automatic fallback to TCP if no root permissions
- Configurable packet size

#### HTTP/HTTPS
- Tests web server response time
- Automatically detects HTTPS on port 443
- Measures full request/response cycle

#### DNS Query
- Supports multiple DNS protocols:
  - **UDP**: Traditional DNS (fastest)
  - **TCP**: For larger responses
  - **DoT**: DNS over TLS (encrypted, port 853)
  - **DoH**: DNS over HTTPS (maximum privacy)
- Custom query domain support

#### Compare Mode
- Resolves hostname to both IPv4 and IPv6
- Tests both protocols simultaneously
- Calculates performance scores
- Displays winner with percentage difference

## Test Results

The results panel displays:

- **Success Rate**: Percentage of successful tests
- **Sent/Received**: Packet counts
- **Min/Max/Avg Latency**: Timing statistics in milliseconds
- **Jitter**: Network stability metric
- **Standard Deviation**: Latency variance
- **Comparison Scores** (in compare mode): Overall performance ranking

## Technical Architecture

### Backend (Go)
- **Framework**: Wails v2
- **Core Library**: `internal/tester` package
  - Refactored from original CLI codebase
  - Full protocol support with platform-specific code
  - Statistics calculation and scoring algorithms
- **Dependencies**:
  - InfluxDB client (for future integration)
  - YAML parser
  - Minimal external dependencies

### Frontend (Svelte)
- **Framework**: Svelte 3 with Vite
- **Design**: Custom gradient theme with responsive grid layout
- **State Management**: Reactive Svelte stores
- **API Communication**: Direct JavaScript bindings to Go functions

### Platform Support
- **macOS**: Full native support (Darwin/arm64 and x86_64)
- **Cross-platform potential**: Can be built for Windows and Linux

## Project Structure

```
prototester-gui/
├── main.go                 # Wails application entry point
├── app.go                  # Application logic and API
├── internal/
│   └── tester/            # Core testing library
│       ├── tester.go      # Main testing functions
│       ├── types.go       # Data structures
│       ├── select_darwin.go  # macOS-specific code
│       └── select_linux.go   # Linux-specific code
├── frontend/
│   └── src/
│       ├── App.svelte     # Main UI component
│       ├── main.js        # Frontend entry point
│       └── style.css      # Global styles
└── build/
    └── bin/
        └── prototester-gui.app  # Built macOS app
```

## API Reference

### Go Functions (exposed to frontend)

#### `RunTest(req TestRequest) *TestResult`
Executes a network test with the provided configuration.

**Parameters**:
- `protocol`: "tcp", "udp", "icmp", "http", "dns", or "compare"
- `target4`: IPv4 address
- `target6`: IPv6 address
- `hostname`: For compare mode
- `port`: Port number
- `count`: Number of tests
- `interval`: Milliseconds between tests
- `timeout`: Test timeout in milliseconds
- `size`: ICMP packet size
- `dnsProtocol`: "udp", "tcp", "dot", or "doh"
- `dnsQuery`: Domain to query
- `ipv4Only`, `ipv6Only`: IP version restrictions

**Returns**: TestResult with statistics and comparison data

#### `GetDefaultConfig() TestRequest`
Returns default test configuration.

## Performance

- **App Size**: 8.9MB (compressed ~3MB)
- **Memory Usage**: ~30-50MB at runtime
- **Test Speed**: Same as CLI version (nanosecond precision)
- **UI Responsiveness**: <16ms render time (60fps)

## Future Enhancements

Potential additions:
- [ ] Test history and saved configurations
- [ ] Charts and graphs for latency trends
- [ ] Batch testing with multiple targets
- [ ] Export results to CSV/JSON
- [ ] InfluxDB integration UI
- [ ] Dark mode toggle
- [ ] Custom themes
- [ ] Notification support
- [ ] Menu bar app mode

## Troubleshooting

### App won't open
- Right-click the app and select "Open" (required for unsigned apps)
- Check System Preferences > Security & Privacy

### ICMP tests failing
- ICMP requires special permissions; the app automatically falls back to TCP
- To use true ICMP, run: `sudo open prototester-gui.app`

### Tests timing out
- Increase timeout value in configuration
- Check network connectivity
- Verify target addresses are reachable

## Development

### Adding new features

1. **Backend**: Add functions to `app.go` (will be auto-bound to frontend)
2. **Frontend**: Import from `../wailsjs/go/main/App.js`
3. **Testing**: Use `wails dev` for hot reload during development

### Building for distribution

```bash
# Standard build
wails build

# Build for specific platform
wails build -platform darwin/amd64

# Build with compression
wails build -compress

# Clean build
wails build -clean
```

## License

MIT License - Same as ProtoTester CLI

## Credits

- **Core Engine**: ProtoTester CLI by buraglio
- **Framework**: Wails (https://wails.io)
- **UI Framework**: Svelte (https://svelte.dev)
- **Icons**: Unicode emojis

## Links

- [Wails Documentation](https://wails.io/docs/introduction)
- [ProtoTester CLI](../README.md)
- [Report Issues](https://github.com/buraglio/prototester/issues)

---

**Built with ❤️ using Wails and Svelte**
