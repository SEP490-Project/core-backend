#!/bin/bash

# Test Report Generator for Campaign Suggestion Refactoring
# Generates a comprehensive HTML report from test results

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
REPORTS_DIR="./test-reports"
COVERAGE_DIR="$REPORTS_DIR/coverage"
HTML_REPORT="$REPORTS_DIR/test-report.html"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Test Report Generator${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Create reports directory
mkdir -p "$REPORTS_DIR"
mkdir -p "$COVERAGE_DIR"

# Run tests and collect results
echo -e "${YELLOW}Running tests and collecting results...${NC}"

# Unit tests
echo -e "${BLUE}→ Running unit tests...${NC}"
go test ./tests/unit_tests/... -v -json -count=1 > "$REPORTS_DIR/unit-tests.json" 2>&1 || true
UNIT_EXIT_CODE=$?

# Integration tests
echo -e "${BLUE}→ Running integration tests...${NC}"
go test ./tests/integration_tests/... -v -json -count=1 > "$REPORTS_DIR/integration-tests.json" 2>&1 || true
INTEGRATION_EXIT_CODE=$?

# Coverage
echo -e "${BLUE}→ Generating coverage report...${NC}"
go test ./tests/... -coverprofile="$COVERAGE_DIR/coverage.out" -covermode=atomic -count=1 > /dev/null 2>&1 || true
go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html" 2>&1 || true
go tool cover -func="$COVERAGE_DIR/coverage.out" > "$COVERAGE_DIR/coverage-summary.txt" 2>&1 || true

# Parse test results
parse_json_results() {
    local json_file=$1
    local test_type=$2
    
    if [ ! -f "$json_file" ]; then
        echo "0|0|0|0" # passed|failed|skipped|total
        return
    fi
    
    local passed=$(grep -o '"Action":"pass"' "$json_file" 2>/dev/null | wc -l || echo "0")
    local failed=$(grep -o '"Action":"fail"' "$json_file" 2>/dev/null | wc -l || echo "0")
    local skipped=$(grep -o '"Action":"skip"' "$json_file" 2>/dev/null | wc -l || echo "0")
    local total=$((passed + failed + skipped))
    
    echo "$passed|$failed|$skipped|$total"
}

UNIT_RESULTS=$(parse_json_results "$REPORTS_DIR/unit-tests.json" "unit")
INTEGRATION_RESULTS=$(parse_json_results "$REPORTS_DIR/integration-tests.json" "integration")

IFS='|' read -r UNIT_PASSED UNIT_FAILED UNIT_SKIPPED UNIT_TOTAL <<< "$UNIT_RESULTS"
IFS='|' read -r INT_PASSED INT_FAILED INT_SKIPPED INT_TOTAL <<< "$INTEGRATION_RESULTS"

TOTAL_PASSED=$((UNIT_PASSED + INT_PASSED))
TOTAL_FAILED=$((UNIT_FAILED + INT_FAILED))
TOTAL_SKIPPED=$((UNIT_SKIPPED + INT_SKIPPED))
TOTAL_TESTS=$((TOTAL_PASSED + TOTAL_FAILED + TOTAL_SKIPPED))

# Get coverage percentage
if [ -f "$COVERAGE_DIR/coverage-summary.txt" ]; then
    COVERAGE=$(tail -1 "$COVERAGE_DIR/coverage-summary.txt" | grep -oE '[0-9]+\.[0-9]+%' | head -1 || echo "0.0%")
else
    COVERAGE="N/A"
fi

# Generate HTML report
cat > "$HTML_REPORT" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Test Report - Campaign Suggestion Refactoring</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif; background: #f5f7fa; color: #2c3e50; line-height: 1.6; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 40px 20px; text-align: center; border-radius: 10px; margin-bottom: 30px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        header h1 { font-size: 2.5em; margin-bottom: 10px; }
        header p { font-size: 1.1em; opacity: 0.9; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; padding: 25px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); border-left: 4px solid #667eea; }
        .stat-card h3 { font-size: 0.9em; text-transform: uppercase; color: #7f8c8d; margin-bottom: 10px; }
        .stat-card .value { font-size: 2.5em; font-weight: bold; color: #2c3e50; }
        .stat-card.success { border-left-color: #2ecc71; }
        .stat-card.success .value { color: #27ae60; }
        .stat-card.failure { border-left-color: #e74c3c; }
        .stat-card.failure .value { color: #c0392b; }
        .stat-card.warning { border-left-color: #f39c12; }
        .stat-card.warning .value { color: #d68910; }
        .stat-card.info { border-left-color: #3498db; }
        .stat-card.info .value { color: #2980b9; }
        .section { background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
        .section h2 { color: #2c3e50; margin-bottom: 20px; border-bottom: 2px solid #ecf0f1; padding-bottom: 10px; }
        .test-suite { margin-bottom: 20px; }
        .test-suite h3 { color: #34495e; margin-bottom: 15px; display: flex; align-items: center; }
        .badge { display: inline-block; padding: 4px 12px; border-radius: 12px; font-size: 0.85em; font-weight: 600; margin-left: 10px; }
        .badge.passed { background: #d4edda; color: #155724; }
        .badge.failed { background: #f8d7da; color: #721c24; }
        .badge.skipped { background: #fff3cd; color: #856404; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ecf0f1; }
        th { background: #f8f9fa; font-weight: 600; color: #2c3e50; }
        tr:hover { background: #f8f9fa; }
        .progress-bar { width: 100%; height: 30px; background: #ecf0f1; border-radius: 15px; overflow: hidden; margin-top: 10px; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #2ecc71 0%, #27ae60 100%); transition: width 0.3s ease; display: flex; align-items: center; justify-content: center; color: white; font-weight: bold; font-size: 0.9em; }
        .timestamp { color: #7f8c8d; font-size: 0.9em; text-align: center; margin-top: 30px; }
        .links { display: flex; gap: 15px; justify-content: center; margin-top: 20px; }
        .link-button { display: inline-block; padding: 12px 24px; background: #667eea; color: white; text-decoration: none; border-radius: 6px; transition: background 0.3s; }
        .link-button:hover { background: #5568d3; }
        .status-icon { display: inline-block; width: 12px; height: 12px; border-radius: 50%; margin-right: 8px; }
        .status-icon.pass { background: #2ecc71; }
        .status-icon.fail { background: #e74c3c; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🧪 Test Report</h1>
            <p>Campaign Suggestion Refactoring - Phase 7 Testing</p>
        </header>

        <div class="stats">
            <div class="stat-card success">
                <h3>✓ Passed</h3>
                <div class="value">$TOTAL_PASSED</div>
            </div>
            <div class="stat-card failure">
                <h3>✗ Failed</h3>
                <div class="value">$TOTAL_FAILED</div>
            </div>
            <div class="stat-card warning">
                <h3>⊘ Skipped</h3>
                <div class="value">$TOTAL_SKIPPED</div>
            </div>
            <div class="stat-card info">
                <h3>📊 Total Tests</h3>
                <div class="value">$TOTAL_TESTS</div>
            </div>
        </div>

        <div class="section">
            <h2>📈 Coverage</h2>
            <p style="font-size: 1.2em; margin-bottom: 10px;">Overall Coverage: <strong>$COVERAGE</strong></p>
            <div class="progress-bar">
                <div class="progress-fill" style="width: ${COVERAGE/\%/}%">$COVERAGE</div>
            </div>
            <div class="links">
                <a href="coverage/coverage.html" class="link-button" target="_blank">View Detailed Coverage Report</a>
            </div>
        </div>

        <div class="section">
            <h2>🧩 Test Suites</h2>
            
            <div class="test-suite">
                <h3>
                    <span class="status-icon $([ $UNIT_FAILED -eq 0 ] && echo 'pass' || echo 'fail')"></span>
                    Unit Tests
                    <span class="badge passed">$UNIT_PASSED passed</span>
                    $([ $UNIT_FAILED -gt 0 ] && echo "<span class='badge failed'>$UNIT_FAILED failed</span>" || echo "")
                </h3>
                <p>Tests for utility functions and helpers (payment cycle calculator, campaign service helpers)</p>
                <p style="margin-top: 10px;"><strong>Total:</strong> $UNIT_TOTAL tests</p>
            </div>

            <div class="test-suite">
                <h3>
                    <span class="status-icon $([ $INT_FAILED -eq 0 ] && echo 'pass' || echo 'fail')"></span>
                    Integration Tests
                    <span class="badge passed">$INT_PASSED passed</span>
                    $([ $INT_FAILED -gt 0 ] && echo "<span class='badge failed'>$INT_FAILED failed</span>" || echo "")
                </h3>
                <p>End-to-end tests for contract type extraction (ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING)</p>
                <p style="margin-top: 10px;"><strong>Total:</strong> $INT_TOTAL tests</p>
            </div>
        </div>

        <div class="section">
            <h2>📁 Report Files</h2>
            <table>
                <thead>
                    <tr>
                        <th>Report Type</th>
                        <th>File Location</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Unit Test Log</td>
                        <td><code>test-reports/unit-tests.log</code></td>
                    </tr>
                    <tr>
                        <td>Unit Test JSON</td>
                        <td><code>test-reports/unit-tests.json</code></td>
                    </tr>
                    <tr>
                        <td>Integration Test Log</td>
                        <td><code>test-reports/integration-tests.log</code></td>
                    </tr>
                    <tr>
                        <td>Integration Test JSON</td>
                        <td><code>test-reports/integration-tests.json</code></td>
                    </tr>
                    <tr>
                        <td>Coverage HTML</td>
                        <td><code>test-reports/coverage/coverage.html</code></td>
                    </tr>
                    <tr>
                        <td>Coverage Summary</td>
                        <td><code>test-reports/coverage/coverage-summary.txt</code></td>
                    </tr>
                </tbody>
            </table>
        </div>

        <p class="timestamp">Generated on $(date '+%Y-%m-%d %H:%M:%S')</p>
    </div>
</body>
</html>
EOF

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  ✓ Report Generation Complete${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}Test Summary:${NC}"
echo -e "  Passed:  ${GREEN}$TOTAL_PASSED${NC}"
echo -e "  Failed:  ${RED}$TOTAL_FAILED${NC}"
echo -e "  Skipped: ${YELLOW}$TOTAL_SKIPPED${NC}"
echo -e "  Total:   ${BLUE}$TOTAL_TESTS${NC}"
echo -e "  Coverage: ${BLUE}$COVERAGE${NC}"
echo ""
echo -e "${YELLOW}Report Files:${NC}"
echo -e "  • HTML Report:    ${BLUE}$HTML_REPORT${NC}"
echo -e "  • Coverage HTML:  ${BLUE}$COVERAGE_DIR/coverage.html${NC}"
echo ""
echo -e "${BLUE}Open HTML report:${NC}"
echo -e "  open $HTML_REPORT"
echo ""

# Exit with error if tests failed
if [ $TOTAL_FAILED -gt 0 ]; then
    exit 1
fi

exit 0
