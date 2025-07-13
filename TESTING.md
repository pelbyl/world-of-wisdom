# Testing Guide

This guide covers various testing approaches for the World of Wisdom project.

## Overview

The project includes multiple testing tools and strategies:

1. **Unit Tests**: Standard Go tests for individual components
2. **Load Testing**: Multiple client simulation and stress testing
3. **Integration Testing**: End-to-end service testing
4. **Performance Monitoring**: Real-time metrics and analysis

## Unit Testing

### Running Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test ./pkg/pow/
go test ./internal/server/
```

### Test Structure

Tests are located alongside source code:
- `pkg/pow/pow_test.go` - PoW algorithm tests
- `internal/server/server_test.go` - Server logic tests
- `pkg/database/database_test.go` - Database tests

## Load Testing

### Quick Start

```bash
# Basic load test with 5 clients for 5 minutes
./scripts/load-test.sh baseline

# Stress test with 10 clients
./scripts/load-test.sh -s 10 stress

# Full test suite
./scripts/load-test.sh all
```

### Load Test Types

#### 1. Baseline Test
Tests normal operation with steady load:
```bash
./scripts/load-test.sh -d 300 -s 5 baseline
```

#### 2. Burst Test
Simulates traffic spikes:
```bash
./scripts/load-test.sh -d 600 burst
```

#### 3. Endurance Test
Long-running stability test:
```bash
./scripts/load-test.sh -d 3600 endurance
```

#### 4. Mixed Algorithm Test
Tests both SHA-256 and Argon2:
```bash
./scripts/load-test.sh mixed
```

#### 5. Scalability Test
Progressive load increase:
```bash
./scripts/load-test.sh scalability
```

#### 6. Stress Test
High-concurrency testing:
```bash
./scripts/load-test.sh -s 20 stress
```

### Docker Compose Load Testing

Using the test configuration:

```bash
# Start basic load test
docker-compose -f docker-compose.yml -f docker-compose.test.yml up --scale client-load=10

# Mixed testing with different client types
docker-compose -f docker-compose.yml -f docker-compose.test.yml up \
  --scale client-load=5 \
  --scale client-burst=3 \
  --scale client-mixed=2

# Monitor with Prometheus and Grafana
docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d test-monitor test-dashboard
```

Access monitoring:
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001 (admin/admin)

### Advanced Load Client

The dedicated load client provides detailed statistics:

```bash
# Build load client
go build -o bin/load-client cmd/load-client/main.go

# Continuous testing
./bin/load-client -server localhost:8080 -mode continuous -duration 5m -concurrency 10

# Burst testing
./bin/load-client -server localhost:8080 -mode burst -burst-size 20 -concurrency 5

# Mixed mode testing
./bin/load-client -server localhost:8080 -mode mixed -duration 10m -concurrency 8
```

#### Load Client Options

| Option | Default | Description |
|--------|---------|-------------|
| `-server` | localhost:8080 | Target server address |
| `-algorithm` | argon2 | PoW algorithm (sha256/argon2) |
| `-mode` | continuous | Test mode (continuous/burst/mixed) |
| `-duration` | 5m | Test duration |
| `-concurrency` | 1 | Number of concurrent clients |
| `-interval` | 5s | Connection interval (continuous mode) |
| `-burst-size` | 10 | Connections per burst |
| `-timeout` | 30s | Connection timeout |

## Integration Testing

### Service Health Checks

Test all service endpoints:

```bash
# Health check script
curl -f http://localhost:8082/health  # API Server
curl -f http://localhost:8083/health  # Service Registry
curl -f http://localhost:8084/health  # Gateway
curl -f http://localhost:8085/health  # Monitor

# TCP server check
timeout 5 bash -c '</dev/tcp/localhost/8080'
```

### End-to-End Testing

Test complete workflow:

```bash
# 1. Start all services
make re-run

# 2. Wait for services to be ready
sleep 30

# 3. Test basic functionality
go run cmd/client/main.go -server localhost:8080 -algorithm argon2

# 4. Test web interface
curl -f http://localhost:3000

# 5. Test API endpoints
curl -f http://localhost:8082/api/v1/metrics
curl -f http://localhost:8082/api/v1/challenges
```

### Microservices Integration

Test service discovery and communication:

```bash
# Start microservices
docker-compose -f docker-compose.yml -f docker-compose.microservices.yml up -d

# Test service registry
curl http://localhost:8083/api/v1/services

# Test gateway routing
curl http://localhost:8084/api/v1/health
curl http://localhost:8084/services

# Test monitoring
curl http://localhost:8085/system
```

## Performance Testing

### Metrics Collection

Key metrics to monitor:

1. **Connection Metrics**:
   - `wisdom_connections_total`
   - `wisdom_active_connections`
   - `wisdom_connection_duration_seconds`

2. **Challenge Metrics**:
   - `wisdom_puzzles_solved_total`
   - `wisdom_current_difficulty`
   - `wisdom_solve_time_seconds`

3. **System Metrics**:
   - CPU usage
   - Memory usage
   - Network I/O

### Prometheus Queries

Useful queries for analysis:

```promql
# Average solve time by difficulty
avg(wisdom_solve_time_seconds) by (difficulty)

# Connection rate
rate(wisdom_connections_total[5m])

# Success rate
rate(wisdom_puzzles_solved_total[5m]) / rate(wisdom_connections_total[5m])

# Current load
wisdom_active_connections
```

### Performance Benchmarks

Expected performance characteristics:

#### SHA-256 Algorithm
- **Solve Time**: ~100-500ms (depending on difficulty)
- **Throughput**: 50-200 solves/sec per CPU core
- **Memory**: Minimal impact
- **Scalability**: Excellent parallel performance

#### Argon2 Algorithm
- **Solve Time**: ~50-200ms (difficulty dependent)
- **Throughput**: 10-50 solves/sec per CPU core
- **Memory**: 64MB-1GB per thread
- **Scalability**: Limited by available RAM

### Load Testing Scenarios

#### Scenario 1: Normal Operation
- **Load**: 10 concurrent clients
- **Algorithm**: Mixed (50% Argon2, 50% SHA-256)
- **Duration**: 10 minutes
- **Expected**: 95%+ success rate, <1s average response time

#### Scenario 2: Peak Traffic
- **Load**: 50 concurrent clients
- **Algorithm**: SHA-256 (faster)
- **Duration**: 5 minutes
- **Expected**: 90%+ success rate, adaptive difficulty increase

#### Scenario 3: Memory Stress
- **Load**: 20 concurrent clients
- **Algorithm**: Argon2 only
- **Duration**: 15 minutes
- **Expected**: Monitor memory usage, potential difficulty reduction

#### Scenario 4: Burst Traffic
- **Pattern**: 100 connections every 30 seconds
- **Algorithm**: Mixed
- **Duration**: 30 minutes
- **Expected**: System handles bursts without failure

## Automated Testing

### GitHub Actions

The CI/CD pipeline includes:

1. **Unit Tests**: Run on every PR
2. **Build Tests**: Verify all services build
3. **Integration Tests**: Basic service health checks
4. **Security Scans**: Vulnerability analysis
5. **Performance Tests**: Load testing on staging

### Local CI Simulation

Run the same tests locally:

```bash
# Unit tests
go test ./...

# Build all services
docker-compose build

# Integration test
docker-compose up -d
sleep 30
./scripts/load-test.sh -d 60 baseline
docker-compose down
```

## Troubleshooting

### Common Issues

1. **High Memory Usage (Argon2)**:
   - Reduce concurrent Argon2 clients
   - Lower difficulty setting
   - Monitor with `docker stats`

2. **Connection Timeouts**:
   - Increase client timeout
   - Check server resource limits
   - Verify network connectivity

3. **Poor Performance**:
   - Check CPU usage
   - Verify algorithm selection
   - Monitor adaptive difficulty

### Debug Commands

```bash
# Service logs
docker-compose logs -f server
docker-compose logs -f webserver

# Resource usage
docker stats

# Network connectivity
telnet localhost 8080
curl -v http://localhost:8082/health

# Database status
docker-compose exec postgres psql -U wisdom -d wisdom -c "\dt"
docker-compose exec redis redis-cli ping
```

### Performance Tuning

1. **For High Throughput**:
   - Use SHA-256 algorithm
   - Lower initial difficulty
   - Increase server resources

2. **For Security**:
   - Use Argon2 algorithm
   - Higher difficulty settings
   - Monitor for attack patterns

3. **For Stability**:
   - Enable adaptive difficulty
   - Set resource limits
   - Monitor system metrics

## Test Results Analysis

### Interpreting Results

Key metrics to evaluate:

1. **Success Rate**: Should be >95% under normal load
2. **Response Time**: 
   - SHA-256: <500ms average
   - Argon2: <200ms average
3. **Throughput**: Depends on hardware and algorithm
4. **Resource Usage**: Monitor CPU and memory limits

### Reporting

Generate test reports:

```bash
# Load test generates automatic reports
./scripts/load-test.sh baseline
# Check test-results/ directory

# Manual metrics collection
curl http://localhost:9090/api/v1/query?query=wisdom_connections_total
```

### Continuous Monitoring

Set up ongoing monitoring:

1. **Prometheus**: Metrics collection
2. **Grafana**: Visualization
3. **Alerting**: Set thresholds for critical metrics
4. **Log Analysis**: Monitor for errors and patterns

## Best Practices

1. **Test Environment**: Use dedicated testing infrastructure
2. **Baseline**: Establish performance baselines
3. **Progressive**: Gradually increase load
4. **Monitor**: Watch system resources during tests
5. **Document**: Record test configurations and results
6. **Automate**: Use CI/CD for consistent testing
7. **Real-world**: Test with realistic scenarios