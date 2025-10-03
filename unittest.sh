#!/bin/bash

# ProtoTester Unit Test Script
# Tests all functionality using 'go run' without requiring compilation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_pattern="$3"
    local timeout_seconds="${4:-10}"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    echo -n "Testing $test_name... "

    # Run the command with timeout and capture output
    if timeout "${timeout_seconds}s" bash -c "$command" > /tmp/unittest_output.tmp 2>&1; then
        if [ -n "$expected_pattern" ]; then
            if grep -q "$expected_pattern" /tmp/unittest_output.tmp; then
                echo -e "${GREEN}PASS${NC}"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}FAIL${NC} (expected pattern not found: $expected_pattern)"
                FAILED_TESTS=$((FAILED_TESTS + 1))
                echo "Output: $(head -3 /tmp/unittest_output.tmp)"
            fi
        else
            echo -e "${GREEN}PASS${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        fi
    else
        echo -e "${RED}FAIL${NC} (command failed or timed out)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo "Output: $(head -3 /tmp/unittest_output.tmp)"
    fi
}

# Function to run JSON test (special handling)
run_json_test() {
    local test_name="$1"
    local command="$2"
    local timeout_seconds="${3:-10}"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    echo -n "Testing $test_name... "

    # Run the command with timeout and capture output
    if timeout "${timeout_seconds}s" bash -c "$command" > /tmp/unittest_json.tmp 2>&1; then
        # Try to extract the JSON portion by finding the opening brace and everything after
        if awk '/^{/,0' /tmp/unittest_json.tmp > /tmp/unittest_json_only.tmp 2>/dev/null && [ -s /tmp/unittest_json_only.tmp ]; then
            # Validate the extracted JSON
            if python3 -m json.tool < /tmp/unittest_json_only.tmp > /dev/null 2>&1; then
                echo -e "${GREEN}PASS${NC}"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}FAIL${NC} (invalid JSON structure)"
                FAILED_TESTS=$((FAILED_TESTS + 1))
                echo "JSON output: $(head -3 /tmp/unittest_json_only.tmp)"
            fi
        else
            # Fallback: check if the entire output contains valid JSON structure
            if grep -q '"mode".*"protocol".*"targets"' /tmp/unittest_json.tmp; then
                echo -e "${GREEN}PASS${NC}"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}FAIL${NC} (no JSON found)"
                FAILED_TESTS=$((FAILED_TESTS + 1))
                echo "Output: $(head -3 /tmp/unittest_json.tmp)"
            fi
        fi
    else
        echo -e "${RED}FAIL${NC} (command failed or timed out)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo "Output: $(head -3 /tmp/unittest_json.tmp)"
    fi
}

echo "======================================"
echo "ProtoTester Unit Test Suite"
echo "======================================"
echo

# Basic functionality tests
echo -e "${YELLOW}=== Basic Functionality Tests ===${NC}"

run_test "Help output" "go run . -h" "Usage of"
run_test "Default TCP mode (IPv4 only)" "go run . -4only -c 2" "High-Fidelity IPv4/IPv6 Latency Tester"
run_test "Default TCP mode (IPv6 only)" "go run . -6only -c 2" "High-Fidelity IPv4/IPv6 Latency Tester"
run_test "Verbose output" "go run . -4only -c 2 -v" "IPv4 test"

echo

# Protocol-specific tests
echo -e "${YELLOW}=== Protocol-Specific Tests ===${NC}"

run_test "TCP mode explicit" "go run . -t -4only -c 2" "TCP"
run_test "UDP mode" "go run . -u -4only -c 2" "UDP"
run_test "ICMP mode (with fallback)" "go run . -icmp -4only -c 2" "ICMP"
run_test "HTTP mode (port 80)" "go run . -http -p 80 -4 google.com -c 2" "HTTP"
run_test "HTTPS mode (port 443)" "go run . -http -p 443 -4 google.com -c 2" "HTTPS"
run_test "DNS UDP mode" "go run . -dns -4only -c 2" "DNS"
run_test "DNS TCP mode" "go run . -dns -dns-protocol tcp -4only -c 2" "DNS"

echo

# Port and configuration tests
echo -e "${YELLOW}=== Configuration Tests ===${NC}"

run_test "Custom port" "go run . -t -p 80 -4only -c 2" ":80"
run_test "Custom packet count" "go run . -4only -c 3" "3 sent"
run_test "Custom interval" "go run . -4only -c 2 -i 500ms" "Latency Tester"
run_test "Custom timeout" "go run . -4only -c 2 -timeout 5s" "Latency Tester"
run_test "Custom ICMP size" "go run . -icmp -s 128 -4only -c 2" "ICMP"

echo

# DNS-specific tests
echo -e "${YELLOW}=== DNS Protocol Tests ===${NC}"

run_test "DNS UDP" "go run . -dns -dns-protocol udp -4only -c 2" "DNS.*UDP"
run_test "DNS TCP" "go run . -dns -dns-protocol tcp -4only -c 2" "DNS.*TCP"
run_test "Custom DNS query" "go run . -dns -dns-query google.com -4only -c 2" "google.com"

echo

# Compare mode tests
echo -e "${YELLOW}=== Compare Mode Tests ===${NC}"

run_test "Compare TCP/UDP mode" "go run . -compare google.com -c 2" "COMPREHENSIVE COMPARISON RESULTS" 30
run_test "Compare DNS mode" "go run . -compare dns.google -dns -c 2" "DNS.*COMPARISON RESULTS" 15
run_test "Compare HTTP mode" "go run . -compare google.com -http -p 80 -c 2" "HTTP.*COMPARISON RESULTS" 15
run_test "Compare ICMP mode" "go run . -compare google.com -icmp -c 2" "ICMP COMPARISON RESULTS" 15

echo

# JSON output tests
echo -e "${YELLOW}=== JSON Output Tests ===${NC}"

run_json_test "JSON single mode" "go run . -json -4only -c 2"
run_json_test "JSON TCP mode" "go run . -t -json -4only -c 2"
run_json_test "JSON UDP mode" "go run . -u -json -4only -c 2"
run_json_test "JSON DNS mode" "go run . -dns -json -4only -c 2"
run_json_test "JSON HTTP mode" "go run . -http -p 80 -4 google.com -json -c 2"
run_json_test "JSON compare mode" "go run . -compare google.com -json -c 2" 30

echo

# Target specification tests
echo -e "${YELLOW}=== Target Specification Tests ===${NC}"

run_test "Custom IPv4 target" "go run . -4 1.1.1.1 -4only -c 2" "1.1.1.1"
run_test "Custom IPv6 target" "go run . -6 2606:4700:4700::1111 -6only -c 2" "2606:4700:4700::1111"
run_test "Both protocols with defaults" "go run . -c 2" "IPv4.*IPv6"

echo

# Error condition tests
echo -e "${YELLOW}=== Error Condition Tests ===${NC}"

run_test "Multiple protocol flags error" "go run . -t -u -c 1 2>&1 || true" "Cannot specify multiple protocol flags"
run_test "Invalid DNS protocol error" "go run . -dns -dns-protocol invalid 2>&1 || true" "Invalid DNS protocol"
run_test "Compare with explicit flags error" "go run . -compare google.com -t 2>&1 || true" "Compare mode cannot be used"

echo

# Edge case tests
echo -e "${YELLOW}=== Edge Case Tests ===${NC}"

run_test "Very short test count" "go run . -4only -c 1" "1 sent"
run_test "DNS with custom port" "go run . -dns -p 8053 -4only -c 2" "8053"
run_test "HTTP with custom port" "go run . -http -p 8080 -4 httpbin.org -c 2" "8080" 15

echo

# Cleanup
rm -f /tmp/unittest_output.tmp /tmp/unittest_json.tmp /tmp/unittest_json_only.tmp

# Final summary
echo "======================================"
echo -e "${YELLOW}Test Summary${NC}"
echo "======================================"
echo "Total tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed.${NC}"
    exit 1
fi