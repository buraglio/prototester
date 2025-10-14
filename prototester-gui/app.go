package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"prototester-gui/internal/tester"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	testHistory     []HistoryEntry
	savedConfigs    []SavedConfig
	historyMu       sync.RWMutex
	configMu        sync.RWMutex
	verboseCallback func(string)
}

// HistoryEntry represents a test result in history
type HistoryEntry struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Timestamp time.Time           `json:"timestamp"`
	Request   TestRequest         `json:"request"`
	Result    *tester.TestResult  `json:"result"`
}

// SavedConfig represents a saved test configuration
type SavedConfig struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	CreatedAt time.Time   `json:"createdAt"`
	Config    TestRequest `json:"config"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		testHistory:  make([]HistoryEntry, 0, 100),
		savedConfigs: make([]SavedConfig, 0, 20),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Load saved configurations and history from disk
	a.loadHistory()
	a.loadConfigs()
}

// TestRequest represents a test request from the frontend
type TestRequest struct {
	Protocol    string `json:"protocol"`    // "tcp", "udp", "icmp", "http", "dns", "compare"
	Target4     string `json:"target4"`     // IPv4 target
	Target6     string `json:"target6"`     // IPv6 target
	Hostname    string `json:"hostname"`    // For compare mode
	Port        int    `json:"port"`        // Port number
	Count       int    `json:"count"`       // Number of tests
	Interval    int    `json:"interval"`    // Interval in milliseconds
	Timeout     int    `json:"timeout"`     // Timeout in milliseconds
	Size        int    `json:"size"`        // ICMP packet size
	DNSProtocol string `json:"dnsProtocol"` // "udp", "tcp", "dot", "doh"
	DNSQuery    string `json:"dnsQuery"`    // Domain to query
	IPv4Only    bool   `json:"ipv4Only"`
	IPv6Only    bool   `json:"ipv6Only"`
	Verbose     bool   `json:"verbose"`     // Enable verbose output
}

// VerboseMessage represents a verbose log message
type VerboseMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // "info", "success", "error"
}

// EmitVerbose sends a verbose message to the frontend
func (a *App) EmitVerbose(msg string, msgType string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "verbose", VerboseMessage{
			Timestamp: time.Now(),
			Message:   msg,
			Type:      msgType,
		})
	}
}

// RunTest executes a network test based on the provided configuration
func (a *App) RunTest(req TestRequest) *tester.TestResult {
	if req.Verbose {
		a.EmitVerbose(fmt.Sprintf("Starting %s test...", req.Protocol), "info")
		a.EmitVerbose(fmt.Sprintf("Target IPv4: %s, IPv6: %s", req.Target4, req.Target6), "info")
		a.EmitVerbose(fmt.Sprintf("Test count: %d, Interval: %dms, Timeout: %dms", req.Count, req.Interval, req.Timeout), "info")
	}

	// Create test configuration
	config := tester.TestConfig{
		Target4:     req.Target4,
		Target6:     req.Target6,
		Hostname:    req.Hostname,
		Port:        req.Port,
		Count:       req.Count,
		Interval:    time.Duration(req.Interval) * time.Millisecond,
		Timeout:     time.Duration(req.Timeout) * time.Millisecond,
		Size:        req.Size,
		DNSProtocol: req.DNSProtocol,
		DNSQuery:    req.DNSQuery,
		IPv4Only:    req.IPv4Only,
		IPv6Only:    req.IPv6Only,
		Verbose:     req.Verbose,
	}

	// Create tester instance
	t := tester.NewTester(config)

	// Run appropriate test based on protocol
	var result *tester.TestResult
	startTime := time.Now()

	switch req.Protocol {
	case "tcp":
		if req.Verbose {
			a.EmitVerbose("Running TCP connectivity test...", "info")
		}
		result = t.RunTCPTest()
	case "udp":
		if req.Verbose {
			a.EmitVerbose("Running UDP connectivity test...", "info")
		}
		result = t.RunUDPTest()
	case "icmp":
		if req.Verbose {
			a.EmitVerbose("Running ICMP ping test...", "info")
		}
		result = t.RunICMPTest()
	case "http", "https":
		if req.Verbose {
			a.EmitVerbose("Running HTTP/HTTPS timing test...", "info")
		}
		result = t.RunHTTPTest()
	case "dns":
		if req.Verbose {
			a.EmitVerbose(fmt.Sprintf("Running DNS query test (%s protocol)...", req.DNSProtocol), "info")
		}
		result = t.RunDNSTest()
	case "compare":
		if req.Verbose {
			a.EmitVerbose(fmt.Sprintf("Resolving %s and comparing IPv4 vs IPv6...", req.Hostname), "info")
		}
		compareProtocol := "TCP/UDP"
		result = t.RunCompareTest(compareProtocol)
	default:
		result = &tester.TestResult{
			ErrorMessage: "Unknown protocol: " + req.Protocol,
			Timestamp:    time.Now(),
		}
	}

	duration := time.Since(startTime)

	if req.Verbose {
		if result.ErrorMessage != "" {
			a.EmitVerbose(fmt.Sprintf("Test failed: %s", result.ErrorMessage), "error")
		} else {
			a.EmitVerbose(fmt.Sprintf("Test completed in %s", duration.Round(time.Millisecond)), "success")
			if result.IPv4Results != nil {
				a.EmitVerbose(fmt.Sprintf("IPv4: %d/%d successful (%.1f%% success rate, avg %.2fms)",
					result.IPv4Results.Received, result.IPv4Results.Sent,
					result.IPv4Results.SuccessRate,
					float64(result.IPv4Results.Avg)/1e6), "success")
			}
			if result.IPv6Results != nil {
				a.EmitVerbose(fmt.Sprintf("IPv6: %d/%d successful (%.1f%% success rate, avg %.2fms)",
					result.IPv6Results.Received, result.IPv6Results.Sent,
					result.IPv6Results.SuccessRate,
					float64(result.IPv6Results.Avg)/1e6), "success")
			}
		}
	}

	// Add to history
	a.addToHistory("", req, result)

	return result
}

// GetDefaultConfig returns default test configuration
func (a *App) GetDefaultConfig() TestRequest {
	return TestRequest{
		Protocol:    "tcp",
		Target4:     "8.8.8.8",
		Target6:     "2001:4860:4860::8888",
		Port:        53,
		Count:       10,
		Interval:    1000, // 1 second
		Timeout:     3000, // 3 seconds
		Size:        64,
		DNSProtocol: "udp",
		DNSQuery:    "dns-query.qosbox.com",
		IPv4Only:    false,
		IPv6Only:    false,
		Verbose:     false,
	}
}

// ============================================================================
// HISTORY MANAGEMENT
// ============================================================================

// addToHistory adds a test result to history
func (a *App) addToHistory(name string, req TestRequest, result *tester.TestResult) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()

	if name == "" {
		name = fmt.Sprintf("%s test - %s", req.Protocol, result.Timestamp.Format("15:04:05"))
	}

	entry := HistoryEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:      name,
		Timestamp: result.Timestamp,
		Request:   req,
		Result:    result,
	}

	// Prepend to history (newest first)
	a.testHistory = append([]HistoryEntry{entry}, a.testHistory...)

	// Keep only last 100 entries
	if len(a.testHistory) > 100 {
		a.testHistory = a.testHistory[:100]
	}

	// Save to disk
	a.saveHistory()
}

// GetHistory returns the test history
func (a *App) GetHistory() []HistoryEntry {
	a.historyMu.RLock()
	defer a.historyMu.RUnlock()
	return a.testHistory
}

// GetHistoryEntry returns a specific history entry by ID
func (a *App) GetHistoryEntry(id string) *HistoryEntry {
	a.historyMu.RLock()
	defer a.historyMu.RUnlock()

	for _, entry := range a.testHistory {
		if entry.ID == id {
			return &entry
		}
	}
	return nil
}

// DeleteHistoryEntry removes an entry from history
func (a *App) DeleteHistoryEntry(id string) bool {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()

	for i, entry := range a.testHistory {
		if entry.ID == id {
			a.testHistory = append(a.testHistory[:i], a.testHistory[i+1:]...)
			a.saveHistory()
			return true
		}
	}
	return false
}

// ClearHistory clears all history
func (a *App) ClearHistory() {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	a.testHistory = make([]HistoryEntry, 0, 100)
	a.saveHistory()
}

// getHistoryPath returns the path to the history file
func (a *App) getHistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prototester", "history.json")
}

// loadHistory loads history from disk
func (a *App) loadHistory() {
	path := a.getHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist yet, that's okay
	}

	var history []HistoryEntry
	if err := json.Unmarshal(data, &history); err == nil {
		a.testHistory = history
	}
}

// saveHistory saves history to disk
func (a *App) saveHistory() {
	path := a.getHistoryPath()
	dir := filepath.Dir(path)

	// Create directory if it doesn't exist
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(a.testHistory, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(path, data, 0644)
}

// ============================================================================
// SAVED CONFIGURATIONS
// ============================================================================

// SaveConfig saves a test configuration
func (a *App) SaveConfig(name string, config TestRequest) string {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	saved := SavedConfig{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:      name,
		CreatedAt: time.Now(),
		Config:    config,
	}

	a.savedConfigs = append(a.savedConfigs, saved)
	a.saveConfigs()

	return saved.ID
}

// GetSavedConfigs returns all saved configurations
func (a *App) GetSavedConfigs() []SavedConfig {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.savedConfigs
}

// GetSavedConfig returns a specific saved configuration
func (a *App) GetSavedConfig(id string) *SavedConfig {
	a.configMu.RLock()
	defer a.configMu.RUnlock()

	for _, config := range a.savedConfigs {
		if config.ID == id {
			return &config
		}
	}
	return nil
}

// UpdateSavedConfig updates an existing saved configuration
func (a *App) UpdateSavedConfig(id string, name string, config TestRequest) bool {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	for i, saved := range a.savedConfigs {
		if saved.ID == id {
			a.savedConfigs[i].Name = name
			a.savedConfigs[i].Config = config
			a.saveConfigs()
			return true
		}
	}
	return false
}

// DeleteSavedConfig removes a saved configuration
func (a *App) DeleteSavedConfig(id string) bool {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	for i, config := range a.savedConfigs {
		if config.ID == id {
			a.savedConfigs = append(a.savedConfigs[:i], a.savedConfigs[i+1:]...)
			a.saveConfigs()
			return true
		}
	}
	return false
}

// getConfigPath returns the path to the saved configs file
func (a *App) getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prototester", "configs.json")
}

// loadConfigs loads saved configurations from disk
func (a *App) loadConfigs() {
	path := a.getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist yet, that's okay
	}

	var configs []SavedConfig
	if err := json.Unmarshal(data, &configs); err == nil {
		a.savedConfigs = configs
	}
}

// saveConfigs saves configurations to disk
func (a *App) saveConfigs() {
	path := a.getConfigPath()
	dir := filepath.Dir(path)

	// Create directory if it doesn't exist
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(a.savedConfigs, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(path, data, 0644)
}

// ExportHistoryToJSON exports history to a JSON file
func (a *App) ExportHistoryToJSON() (string, error) {
	a.historyMu.RLock()
	defer a.historyMu.RUnlock()

	data, err := json.MarshalIndent(a.testHistory, "", "  ")
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("prototester_history_%s.json", timestamp)
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Downloads", filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	return path, nil
}
