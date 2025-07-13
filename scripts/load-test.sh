#!/bin/bash

# Load Testing Script for World of Wisdom
# This script orchestrates various load testing scenarios

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default configuration
TEST_DURATION=300
RAMP_UP_TIME=60
CLIENT_SCALE=5
TARGET_HOST="localhost"
TARGET_PORT=8080
ALGORITHM="argon2"
DIFFICULTY=2
RESULTS_DIR="test-results"
PROMETHEUS_URL="http://localhost:9090"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] TEST_TYPE

Load testing tool for World of Wisdom TCP server.

TEST_TYPES:
    baseline        Basic load test with constant connections
    burst           Burst traffic simulation
    endurance       Long-running stability test
    mixed           Mixed algorithm and difficulty testing
    scalability     Progressive load increase test
    stress          High-load stress testing
    all            Run all test scenarios

OPTIONS:
    -d, --duration SECONDS      Test duration in seconds (default: 300)
    -r, --ramp-up SECONDS       Ramp-up time in seconds (default: 60)
    -s, --scale NUMBER          Number of client containers (default: 5)
    -h, --host HOST             Target server host (default: localhost)
    -p, --port PORT             Target server port (default: 8080)
    -a, --algorithm ALG         PoW algorithm: sha256|argon2 (default: argon2)
    --difficulty LEVEL          PoW difficulty level 1-6 (default: 2)
    --results-dir DIR           Results directory (default: test-results)
    --prometheus URL            Prometheus URL for metrics (default: http://localhost:9090)
    --help                     Show this help message

EXAMPLES:
    # Basic load test
    $0 baseline

    # Stress test with 10 clients for 10 minutes
    $0 -s 10 -d 600 stress

    # Mixed algorithm test against remote server
    $0 -h 198.51.100.42 -p 8080 mixed

    # Full test suite
    $0 all

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--duration)
            TEST_DURATION="$2"
            shift 2
            ;;
        -r|--ramp-up)
            RAMP_UP_TIME="$2"
            shift 2
            ;;
        -s|--scale)
            CLIENT_SCALE="$2"
            shift 2
            ;;
        -h|--host)
            TARGET_HOST="$2"
            shift 2
            ;;
        -p|--port)
            TARGET_PORT="$2"
            shift 2
            ;;
        -a|--algorithm)
            ALGORITHM="$2"
            shift 2
            ;;
        --difficulty)
            DIFFICULTY="$2"
            shift 2
            ;;
        --results-dir)
            RESULTS_DIR="$2"
            shift 2
            ;;
        --prometheus)
            PROMETHEUS_URL="$2"
            shift 2
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            if [[ -z "$TEST_TYPE" ]]; then
                TEST_TYPE="$1"
            else
                log_error "Unknown option: $1"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$TEST_TYPE" ]]; then
    log_error "TEST_TYPE is required"
    usage
    exit 1
fi

# Create results directory
mkdir -p "$RESULTS_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
TEST_ID="${TEST_TYPE}_${TIMESTAMP}"
TEST_DIR="$RESULTS_DIR/$TEST_ID"
mkdir -p "$TEST_DIR"

log_info "Starting load test: $TEST_TYPE"
log_info "Target: $TARGET_HOST:$TARGET_PORT"
log_info "Algorithm: $ALGORITHM, Difficulty: $DIFFICULTY"
log_info "Duration: ${TEST_DURATION}s, Scale: $CLIENT_SCALE"
log_info "Results will be saved to: $TEST_DIR"

# Function to check if server is reachable
check_server() {
    log_info "Checking server connectivity..."
    if timeout 10 bash -c "</dev/tcp/$TARGET_HOST/$TARGET_PORT"; then
        log_success "Server is reachable"
    else
        log_error "Cannot connect to $TARGET_HOST:$TARGET_PORT"
        exit 1
    fi
}

# Function to start monitoring
start_monitoring() {
    log_info "Starting test monitoring..."
    
    # Start Prometheus metrics collection if available
    if curl -s "$PROMETHEUS_URL/api/v1/label/__name__/values" >/dev/null 2>&1; then
        log_info "Prometheus detected, enabling metrics collection"
        cat > "$TEST_DIR/prometheus-query.sh" << EOF
#!/bin/bash
# Collect metrics during test
while true; do
    timestamp=\$(date +%s)
    
    # Connection metrics
    curl -s "$PROMETHEUS_URL/api/v1/query?query=wisdom_active_connections" | jq -r '.data.result[0].value[1]' > "$TEST_DIR/connections_\$timestamp.txt" 2>/dev/null
    
    # Difficulty metrics
    curl -s "$PROMETHEUS_URL/api/v1/query?query=wisdom_current_difficulty" | jq -r '.data.result[0].value[1]' > "$TEST_DIR/difficulty_\$timestamp.txt" 2>/dev/null
    
    # Solve time metrics
    curl -s "$PROMETHEUS_URL/api/v1/query?query=wisdom_average_solve_time_seconds" | jq -r '.data.result[0].value[1]' > "$TEST_DIR/solve_time_\$timestamp.txt" 2>/dev/null
    
    sleep 5
done
EOF
        chmod +x "$TEST_DIR/prometheus-query.sh"
        "$TEST_DIR/prometheus-query.sh" &
        PROMETHEUS_PID=$!
    else
        log_warning "Prometheus not available, skipping metrics collection"
        PROMETHEUS_PID=""
    fi
    
    # Start system monitoring
    cat > "$TEST_DIR/system-monitor.sh" << EOF
#!/bin/bash
while true; do
    echo "\$(date): \$(docker stats --no-stream --format 'table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}')" >> "$TEST_DIR/docker-stats.log"
    sleep 10
done
EOF
    chmod +x "$TEST_DIR/system-monitor.sh"
    "$TEST_DIR/system-monitor.sh" &
    SYSTEM_PID=$!
}

# Function to stop monitoring
stop_monitoring() {
    log_info "Stopping monitoring..."
    
    if [[ -n "$PROMETHEUS_PID" ]]; then
        kill $PROMETHEUS_PID 2>/dev/null || true
    fi
    
    if [[ -n "$SYSTEM_PID" ]]; then
        kill $SYSTEM_PID 2>/dev/null || true
    fi
}

# Function to generate test report
generate_report() {
    log_info "Generating test report..."
    
    cat > "$TEST_DIR/test-report.md" << EOF
# Load Test Report: $TEST_TYPE

## Test Configuration
- **Test ID**: $TEST_ID
- **Date**: $(date)
- **Target**: $TARGET_HOST:$TARGET_PORT
- **Algorithm**: $ALGORITHM
- **Difficulty**: $DIFFICULTY
- **Duration**: ${TEST_DURATION}s
- **Client Scale**: $CLIENT_SCALE

## Test Results

### Docker Compose Logs
\`\`\`
$(docker-compose -f docker-compose.yml -f docker-compose.test.yml logs --tail=100 2>/dev/null || echo "Logs not available")
\`\`\`

### System Resources
\`\`\`
$(tail -20 "$TEST_DIR/docker-stats.log" 2>/dev/null || echo "System stats not available")
\`\`\`

### Metrics Summary
EOF

    # Add metrics summary if available
    if [[ -d "$TEST_DIR" ]]; then
        echo "- Connection files: $(ls "$TEST_DIR"/connections_*.txt 2>/dev/null | wc -l || echo 0)" >> "$TEST_DIR/test-report.md"
        echo "- Difficulty files: $(ls "$TEST_DIR"/difficulty_*.txt 2>/dev/null | wc -l || echo 0)" >> "$TEST_DIR/test-report.md"
        echo "- Solve time files: $(ls "$TEST_DIR"/solve_time_*.txt 2>/dev/null | wc -l || echo 0)" >> "$TEST_DIR/test-report.md"
    fi
    
    log_success "Test report generated: $TEST_DIR/test-report.md"
}

# Function to cleanup
cleanup() {
    log_info "Cleaning up test environment..."
    stop_monitoring
    docker-compose -f docker-compose.yml -f docker-compose.test.yml down --remove-orphans 2>/dev/null || true
    generate_report
}

# Trap cleanup on exit
trap cleanup EXIT

# Test implementations
run_baseline_test() {
    log_info "Running baseline load test..."
    
    export SERVER_HOST="$TARGET_HOST"
    export SERVER_PORT="$TARGET_PORT"
    export ALGORITHM="$ALGORITHM"
    export DIFFICULTY="$DIFFICULTY"
    export CONNECT_INTERVAL="5s"
    
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d --scale client-load=$CLIENT_SCALE
    
    log_info "Waiting for ramp-up period (${RAMP_UP_TIME}s)..."
    sleep $RAMP_UP_TIME
    
    log_info "Running test for ${TEST_DURATION}s..."
    sleep $TEST_DURATION
}

run_burst_test() {
    log_info "Running burst load test..."
    
    export SERVER_HOST="$TARGET_HOST"
    export SERVER_PORT="$TARGET_PORT"
    export ALGORITHM="$ALGORITHM"
    export DIFFICULTY="$DIFFICULTY"
    export BURST_SIZE="20"
    export BURST_INTERVAL="30s"
    
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d --scale client-burst=$CLIENT_SCALE
    
    log_info "Running burst test for ${TEST_DURATION}s..."
    sleep $TEST_DURATION
}

run_endurance_test() {
    log_info "Running endurance test..."
    
    export SERVER_HOST="$TARGET_HOST"
    export SERVER_PORT="$TARGET_PORT"
    export ALGORITHM="$ALGORITHM"
    export DIFFICULTY="$DIFFICULTY"
    export CONNECTION_DURATION="600s"
    export RESTART_INTERVAL="900s"
    
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d --scale client-endurance=$CLIENT_SCALE
    
    log_info "Running endurance test for ${TEST_DURATION}s..."
    sleep $TEST_DURATION
}

run_mixed_test() {
    log_info "Running mixed algorithm test..."
    
    export SERVER_HOST="$TARGET_HOST"
    export SERVER_PORT="$TARGET_PORT"
    export ALGORITHMS="sha256,argon2"
    export DIFFICULTIES="1,2,3"
    export CONNECT_INTERVAL="10s"
    
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d --scale client-mixed=$CLIENT_SCALE
    
    log_info "Running mixed test for ${TEST_DURATION}s..."
    sleep $TEST_DURATION
}

run_scalability_test() {
    log_info "Running scalability test..."
    
    for scale in 1 2 5 10 15; do
        if [[ $scale -gt $CLIENT_SCALE ]]; then
            break
        fi
        
        log_info "Testing with $scale clients..."
        
        export SERVER_HOST="$TARGET_HOST"
        export SERVER_PORT="$TARGET_PORT"
        export ALGORITHM="$ALGORITHM"
        export DIFFICULTY="$DIFFICULTY"
        export CONNECT_INTERVAL="5s"
        
        docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d --scale client-load=$scale
        sleep 60
        docker-compose -f docker-compose.yml -f docker-compose.test.yml down client-load
        sleep 30
    done
}

run_stress_test() {
    log_info "Running stress test with high concurrency..."
    
    export SERVER_HOST="$TARGET_HOST"
    export SERVER_PORT="$TARGET_PORT"
    export ALGORITHM="sha256"  # Use faster algorithm for stress test
    export DIFFICULTY="1"
    export CONNECT_INTERVAL="1s"
    
    # Start multiple test types simultaneously
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d \
        --scale client-load=$CLIENT_SCALE \
        --scale client-burst=$(($CLIENT_SCALE / 2)) \
        --scale client-mixed=$(($CLIENT_SCALE / 3))
    
    log_info "Running stress test for ${TEST_DURATION}s..."
    sleep $TEST_DURATION
}

# Check server connectivity
check_server

# Start monitoring
start_monitoring

# Run the specified test
case $TEST_TYPE in
    baseline)
        run_baseline_test
        ;;
    burst)
        run_burst_test
        ;;
    endurance)
        run_endurance_test
        ;;
    mixed)
        run_mixed_test
        ;;
    scalability)
        run_scalability_test
        ;;
    stress)
        run_stress_test
        ;;
    all)
        log_info "Running comprehensive test suite..."
        
        # Run each test type with shorter duration
        SHORT_DURATION=$((TEST_DURATION / 6))
        
        TEST_DURATION=$SHORT_DURATION run_baseline_test
        docker-compose -f docker-compose.yml -f docker-compose.test.yml down --remove-orphans
        sleep 30
        
        TEST_DURATION=$SHORT_DURATION run_burst_test
        docker-compose -f docker-compose.yml -f docker-compose.test.yml down --remove-orphans
        sleep 30
        
        TEST_DURATION=$SHORT_DURATION run_mixed_test
        docker-compose -f docker-compose.yml -f docker-compose.test.yml down --remove-orphans
        sleep 30
        
        TEST_DURATION=$SHORT_DURATION run_stress_test
        ;;
    *)
        log_error "Unknown test type: $TEST_TYPE"
        usage
        exit 1
        ;;
esac

log_success "Load test completed: $TEST_TYPE"
log_info "Check results in: $TEST_DIR"
log_info "View logs with: docker-compose -f docker-compose.yml -f docker-compose.test.yml logs"