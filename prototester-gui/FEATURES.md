# ProtoTester GUI - Feature Guide

## 🎯 New Features Overview

The enhanced ProtoTester GUI now includes powerful features for test tracking, configuration management, and detailed test observation.

## ✨ Key Features

### 1. Verbose Mode 📋

**Real-time test observation with detailed logging**

- **Toggle**: Check "Enable Verbose Output" in the test configuration panel
- **What it shows**:
  - Test initialization messages
  - Protocol selection and configuration
  - Real-time test progress
  - Individual test results as they complete
  - Success/failure indicators with color coding
  - Final statistics summary

**Verbose Log Display**:
- **Terminal-style UI**: Black background with colored messages
- **Auto-scroll**: Automatically scrolls to show latest messages (toggle on/off)
- **Message Types**:
  - **Info** (Blue): General information and progress
  - **Success** (Green): Successful tests and completions
  - **Error** (Red): Failed tests and error messages
- **Timestamps**: Each message shows exact time
- **Clear button**: Reset the log at any time

**Example Verbose Output**:
```
[14:32:15] Starting tcp test...
[14:32:15] Target IPv4: 8.8.8.8, IPv6: 2001:4860:4860::8888
[14:32:15] Test count: 10, Interval: 1000ms, Timeout: 3000ms
[14:32:15] Running TCP connectivity test...
[14:32:25] Test completed in 10.523s
[14:32:25] IPv4: 10/10 successful (100.0% success rate, avg 15.23ms)
[14:32:25] IPv6: 10/10 successful (100.0% success rate, avg 12.45ms)
```

### 2. Test History 📜

**Automatic tracking of all tests with persistent storage**

**Features**:
- **Automatic Saving**: Every test is automatically saved to history
- **Persistent Storage**: History saved to `~/.prototester/history.json`
- **Last 100 Tests**: Keeps most recent 100 test results
- **Quick Access**: Three-tab interface (Test / History / Saved Configs)

**History Panel Layout**:
- **Left Side**: List of all test entries
  - Protocol badge (TCP, UDP, ICMP, etc.)
  - Timestamp
  - Target information
  - Key metrics (Avg latency, Success rate)
  - Action buttons (Rerun, Delete)

- **Right Side**: Detailed view when entry selected
  - Full test configuration
  - Complete IPv4 results
  - Complete IPv6 results
  - All statistics (min/avg/max/jitter/stddev)

**History Actions**:
- **🔄 Refresh**: Reload history from disk
- **💾 Export**: Export all history to JSON file (saved to ~/Downloads)
- **🗑️ Clear**: Remove all history entries
- **↻ Rerun**: Load test configuration and switch to Test tab
- **✕ Delete**: Remove individual history entry

**Use Cases**:
- Compare test results over time
- Track network performance changes
- Document network issues with timestamps
- Rerun previous tests quickly
- Export data for reporting

### 3. Saved Configurations 💾

**Save and manage frequently used test configurations**

**Features**:
- **Save Current Config**: Save any test configuration with a custom name
- **Persistent Storage**: Configs saved to `~/.prototester/configs.json`
- **Quick Load**: One-click loading of saved configurations
- **Unlimited Storage**: Save as many configurations as needed

**Saved Config Display**:
- Configuration name (custom)
- Creation date
- Protocol type badge
- Target/hostname and port
- Action buttons (Load, Delete)

**Workflow**:
1. Configure a test (protocol, targets, ports, etc.)
2. Click "➕ Save Current" in Saved Configs tab
3. Enter a descriptive name (e.g., "Google DNS TCP", "Cloudflare DoH")
4. Config is saved permanently
5. Load anytime by clicking "↻ Load"

**Example Saved Configs**:
- "Production DNS Monitoring" - UDP to internal DNS
- "Google DNS Performance" - TCP to 8.8.8.8
- "Cloudflare DoH Test" - DNS over HTTPS
- "Local Network Ping" - ICMP to gateway
- "Website Response Time" - HTTP to production server

**Use Cases**:
- Save production monitoring configurations
- Quick A/B testing between providers
- Team sharing (export configs file)
- Documentation of test procedures
- Standardized testing across environments

## 🔧 Data Storage

All data is stored locally in your home directory:

```
~/.prototester/
├── history.json      # Test history (last 100 tests)
└── configs.json      # Saved configurations
```

**Data Format**: JSON (human-readable, can be edited)

**Data Persistence**:
- Survives app restarts
- Cross-session availability
- Portable (copy files to another machine)
- Version control friendly

## 🎨 User Interface Improvements

### Tab-Based Navigation
- **⚡ Test**: Run tests and view results
- **📜 History**: Browse and manage test history
- **💾 Saved Configs**: Manage saved configurations

### Enhanced Test Panel
- Verbose mode toggle with visual indicator
- Real-time verbose log display
- Collapsible terminal-style output
- Color-coded message types

### Improved Results Display
- Same comprehensive statistics
- Better organized layout
- Clearer metric presentation
- Persistent across tab switches

## 📊 Export Capabilities

### Export History to JSON

**Feature**: Export all test history to a timestamped JSON file

**Location**: `~/Downloads/prototester_history_YYYY-MM-DD_HH-MM-SS.json`

**Use Cases**:
- Archive historical data
- Share results with team
- Import into analysis tools
- Generate reports
- Backup before clearing history

**JSON Structure**:
```json
[
  {
    "id": "1728938420000000000",
    "name": "tcp test - 14:33:40",
    "timestamp": "2025-10-14T14:33:40Z",
    "request": {
      "protocol": "tcp",
      "target4": "8.8.8.8",
      "target6": "2001:4860:4860::8888",
      "port": 53,
      "count": 10,
      ...
    },
    "result": {
      "ipv4_results": { ... },
      "ipv6_results": { ... },
      ...
    }
  }
]
```

## 🚀 Workflow Examples

### Daily Network Monitoring

1. **Setup** (once):
   - Create test configuration for your critical services
   - Save as "Daily DNS Check", "Production Server Ping", etc.

2. **Daily Routine**:
   - Load "Daily DNS Check" from Saved Configs
   - Enable verbose mode
   - Run test
   - Watch real-time progress
   - Results automatically saved to history

3. **Weekly Review**:
   - Switch to History tab
   - Review past week's tests
   - Export to JSON for reporting
   - Identify trends or issues

### Troubleshooting Network Issues

1. **Document Current State**:
   - Run test with verbose mode enabled
   - Test automatically saved to history
   - Verbose log shows exact sequence

2. **Test Different Configurations**:
   - Try different protocols
   - Test alternative DNS servers
   - Each test saved to history

3. **Compare Results**:
   - Review history entries side-by-side
   - Rerun previous tests to verify fixes
   - Export evidence of issue and resolution

### A/B Testing DNS Providers

1. **Create Test Configs**:
   - Save "Google DNS (8.8.8.8)"
   - Save "Cloudflare DNS (1.1.1.1)"
   - Save "Quad9 DNS (9.9.9.9)"

2. **Run Tests**:
   - Load each config
   - Run multiple times
   - All results in history

3. **Analysis**:
   - Review history
   - Compare average latencies
   - Check success rates
   - Export data for charts

## 🔍 Accuracy Verification

**Verbose Mode Benefits for Accuracy**:
- See exact test sequence
- Verify correct targets being tested
- Confirm protocol usage
- Identify timeout vs. connection issues
- Observe fallback behavior (ICMP → TCP)

**History Benefits for Accuracy**:
- Compare results across runs
- Spot anomalies or outliers
- Verify consistency
- Track long-term trends

**Tips for Accurate Results**:
1. Enable verbose mode for first run of new config
2. Verify targets in verbose output
3. Run multiple tests (count ≥ 10)
4. Use 1 second interval for normal tests
5. Check history for consistency

## 💡 Pro Tips

1. **Naming Saved Configs**: Use descriptive names
   - ✅ "Production DNS - UDP to 8.8.8.8"
   - ❌ "Test 1"

2. **Verbose Mode**: Enable when:
   - Debugging connection issues
   - Verifying new configurations
   - Documenting test procedures
   - Learning protocol behavior

3. **History Management**:
   - Export before clearing
   - Review periodically for patterns
   - Delete failed/invalid tests
   - Keep successful baseline tests

4. **Configuration Sets**:
   - Create configs for each environment (prod/staging/dev)
   - Save configs for different protocols to same target
   - Document purpose in config name

## 🐛 Troubleshooting

### History not loading
- Check `~/.prototester/history.json` exists
- Verify JSON is valid
- Click Refresh button

### Configs not persisting
- Check write permissions to `~/.prototester/`
- Verify disk space
- Check `configs.json` for corruption

### Verbose output not showing
- Ensure "Enable Verbose Output" is checked
- Restart app if needed
- Check console for errors

### Tests seem inaccurate
- Enable verbose mode to see exact test sequence
- Verify targets in verbose output
- Check network connectivity
- Increase test count for better average
- Review history for consistency

## 📝 Notes

- History is capped at 100 entries (oldest removed first)
- All data stored locally (no cloud sync)
- JSON files can be manually edited if needed
- Test results include full configuration for reproducibility
- Verbose mode adds minimal overhead (<1% impact)

---

**Need help?** Check [README-GUI.md](README-GUI.md) for full documentation or [QUICKSTART.md](QUICKSTART.md) for quick reference.
