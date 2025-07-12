# Word of Wisdom TCP Server - Development Plan

## Project Overview

Build a high-performance TCP server in Go that serves random quotes ("words of wisdom") but only after clients successfully complete a Proof-of-Work (PoW) challenge. This protects against DDoS attacks by requiring computational effort from each client.

### Key Requirements
- **Language**: Go (1.22+)
- **Protocol**: Plain TCP (no HTTP)
- **Security**: Proof-of-Work challenge-response
- **Performance**: Handle high concurrent load
- **Adaptability**: Dynamic difficulty adjustment
- **Monitoring**: Prometheus metrics
- **Deployment**: Docker containers
- **Testing Environment**: MacBook Pro

## Technical Architecture

### Technology Stack
| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.22+ | Main application |
| Networking | TCP (net package) | Client-server communication |
| PoW Algorithm | SHA-256 | Challenge-response puzzle |
| Metrics | Prometheus | Monitoring and observability |
| Containerization | Docker | Deployment packaging |
| Logging | Standard log package | Structured logging |

### Project Structure
```
world-of-wisdom/
├── cmd/
│   ├── server/main.go      # Server entry point
│   └── client/main.go      # Client entry point
├── internal/
│   ├── server/server.go    # Server logic
│   └── client/client.go    # Client logic
├── pkg/
│   ├── pow/pow.go          # PoW implementation
│   └── wisdom/wisdom.go    # Quote management
├── Dockerfile.server       # Server container
├── Dockerfile.client       # Client container
├── Makefile               # Build automation
└── go.mod                 # Go module
```

## Proof-of-Work Mechanism

### Algorithm: SHA-256 Hash Puzzle
- **Challenge**: Server sends random seed + difficulty level
- **Solution**: Client finds nonce where SHA256(seed + nonce) has N leading zeros
- **Verification**: Server validates solution in single hash operation
- **Difficulty**: Exponential scaling (each zero bit doubles difficulty)

### Protocol Flow
1. Client connects to TCP server
2. Server sends challenge: `"Solve PoW: <seed> with prefix <zeros>"`
3. Client brute-forces nonce until hash meets difficulty
4. Client sends nonce back
5. Server verifies and responds with quote (or error)
6. Connection closes

## Development Stages

### Stage 1: Project Foundation
**Goal**: Set up Go module and basic TCP networking

**Tasks**:
- Initialize Go module: `go mod init world-of-wisdom`
- Create directory structure
- Implement basic TCP server (port 8080)
- Implement basic TCP client
- Test connection establishment

**Deliverables**:
- Working TCP server that accepts connections
- Client that can connect to server
- Basic logging setup

### Stage 2: PoW Implementation
**Goal**: Core Proof-of-Work logic

**Tasks**:
- Implement `GenerateChallenge(difficulty int) (seed string, prefix string)`
- Implement `VerifyPoW(seed, nonce string, difficulty int) bool`
- Use SHA-256 for hashing
- Support difficulty levels 1-6 (hex zeros)
- Write comprehensive unit tests

**Deliverables**:
- `pkg/pow/pow.go` with PoW functions
- Unit tests proving correctness
- Performance benchmarks

### Stage 3: Quote Management
**Goal**: Wisdom quotes storage and retrieval

**Tasks**:
- Create static quote collection (minimum 20 quotes)
- Implement `GetRandomQuote() string`
- Thread-safe random selection
- Unit tests for quote retrieval

**Deliverables**:
- `pkg/wisdom/wisdom.go` with quote functions
- Diverse collection of wisdom quotes
- Unit tests for randomness

### Stage 4: Server Implementation
**Goal**: Complete TCP server with PoW protection

**Tasks**:
- Implement connection handling in goroutines
- Challenge generation and verification
- Connection timeouts (30 seconds)
- Proper error handling and logging
- Graceful connection cleanup

**Protocol Implementation**:
```go
// Per connection:
// 1. Generate challenge
// 2. Send: "Solve PoW: <seed> with prefix <zeros>\n"
// 3. Read client response (with timeout)
// 4. Verify solution
// 5. Send quote or error
// 6. Close connection
```

**Deliverables**:
- `internal/server/server.go` with full logic
- Concurrent connection handling
- Timeout protection
- Integration tests

### Stage 5: Client Implementation
**Goal**: PoW-solving TCP client

**Tasks**:
- Parse server challenges
- Implement brute-force nonce search
- Send solutions back to server
- Handle server responses
- Error handling and retries

**Algorithm**:
```go
// Brute force approach:
for nonce := 0; ; nonce++ {
    hash := sha256.Sum256([]byte(seed + strconv.Itoa(nonce)))
    if hasRequiredZeros(hash, difficulty) {
        return strconv.Itoa(nonce)
    }
}
```

**Deliverables**:
- `internal/client/client.go` with solver logic
- End-to-end working client-server interaction
- Performance optimization for MacBook Pro

### Stage 6: Adaptive Difficulty
**Goal**: Dynamic difficulty adjustment based on load

**Tasks**:
- Track average solve times per client
- Implement difficulty adjustment algorithm
- Set reasonable bounds (difficulty 1-6)
- Monitor connection patterns
- Thread-safe difficulty updates

**Adaptation Logic**:
```go
// Example thresholds:
// - If avg solve time < 1s: increase difficulty
// - If avg solve time > 5s: decrease difficulty
// - If connection rate spikes: increase difficulty
// - Bounds: min=1, max=6
```

**Deliverables**:
- Adaptive difficulty system
- Configurable parameters
- Load testing validation

### Stage 7: Prometheus Metrics
**Goal**: Observability and monitoring

**Tasks**:
- Integrate Prometheus Go client
- Expose metrics on port 2112
- Track key performance indicators
- HTTP metrics endpoint: `/metrics`

**Metrics to Track**:
- `wisdom_connections_total` (Counter)
- `wisdom_puzzles_solved_total` (Counter) 
- `wisdom_puzzles_failed_total` (Counter)
- `wisdom_current_difficulty` (Gauge)
- `wisdom_solve_time_seconds` (Histogram)

**Deliverables**:
- Prometheus metrics integration
- HTTP metrics server
- Metric validation tests

### Stage 8: Docker Containerization
**Goal**: Containerized deployment

**Tasks**:
- Multi-stage Dockerfile for server
- Multi-stage Dockerfile for client
- Optimize image sizes
- Configure port exposure
- Environment variable support

**Server Dockerfile**:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server cmd/server/main.go

FROM alpine:latest
COPY --from=builder /app/server /usr/local/bin/
EXPOSE 8080 2112
CMD ["server"]
```

**Deliverables**:
- `Dockerfile.server` and `Dockerfile.client`
- Docker images that build successfully
- Container networking configuration
- Docker Compose for easy testing

### Stage 9: Integration Testing & Documentation
**Goal**: Comprehensive testing and documentation

**Tasks**:
- Load testing with multiple concurrent clients
- Performance benchmarking on MacBook Pro
- Stress testing adaptive difficulty
- Metrics validation
- Complete documentation

**Test Scenarios**:
- Single client request
- 50 concurrent clients
- Simulated attack (rapid connections)
- Difficulty adaptation validation
- Container deployment testing

**Deliverables**:
- Comprehensive test suite
- Performance benchmarks
- Usage documentation
- Configuration guide

## Configuration Options

### Server Configuration
- `--port`: TCP port (default: 8080)
- `--metrics-port`: Metrics port (default: 2112)
- `--difficulty`: Initial difficulty (default: 2)
- `--timeout`: Client timeout (default: 30s)
- `--adaptive`: Enable adaptive difficulty (default: true)

### Client Configuration
- `--server`: Server address (default: localhost:8080)
- `--attempts`: Number of quote requests (default: 1)

## Expected Performance

### Target Metrics (MacBook Pro)
- **Concurrent Connections**: 1000+
- **Solve Time**: 0.5-3 seconds (difficulty 2-4)
- **Throughput**: 100+ quotes/minute
- **Memory Usage**: <100MB
- **CPU Usage**: Moderate under normal load

### Difficulty Scaling
- **Difficulty 1**: ~0.1 seconds
- **Difficulty 2**: ~0.5 seconds  
- **Difficulty 3**: ~2 seconds
- **Difficulty 4**: ~10 seconds
- **Difficulty 5**: ~1 minute
- **Difficulty 6**: ~5 minutes

## Success Criteria

✅ **Functional Requirements**:
- TCP server accepts connections and serves quotes
- PoW protection prevents easy abuse
- Adaptive difficulty responds to load
- Prometheus metrics provide visibility
- Docker containers work in isolation

✅ **Performance Requirements**:
- Handle 100+ concurrent connections
- Maintain sub-second response times for legitimate users
- Gracefully handle connection timeouts
- Demonstrate DDoS protection effectiveness

✅ **Quality Requirements**:
- Comprehensive unit tests
- Integration tests
- Clear documentation
- Production-ready code structure
- Proper error handling and logging

## Development Timeline

**Total Duration** 9 stages

- **Stages 1-3** (Foundation, PoW, Quotes)
- **Stages 4-5** (Server, Client)
- **Stages 6-7** (Adaptive, Metrics)
- **Stages 8-9** (Docker, Testing)

This plan provides a structured approach to building a production-ready PoW-protected TCP server that demonstrates both security concepts and high-performance Go networking.