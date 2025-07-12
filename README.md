# Word of Wisdom TCP Server

A high-performance TCP server in Go that serves random quotes ("words of wisdom") protected by a Proof-of-Work (PoW) challenge-response protocol to prevent DDoS attacks.

## Features

- **DDoS Protection**: SHA-256 based Proof-of-Work challenge prevents abuse
- **Concurrent Design**: Handles multiple clients simultaneously using Go routines
- **Configurable Difficulty**: Adjustable PoW difficulty levels (1-6)
- **Docker Support**: Containerized deployment for both server and client
- **Connection Timeouts**: Automatic cleanup of stale connections
- **Random Wisdom Quotes**: Serves from a curated collection of 25+ quotes

## Architecture

### Proof-of-Work Algorithm

The server uses SHA-256 hash puzzles:
1. Server generates a random seed and difficulty level
2. Client must find a nonce where `SHA256(seed + nonce)` has N leading zeros
3. Difficulty scales exponentially - each additional zero doubles the work

### Protocol Flow

```
Client connects → Server sends challenge → Client solves PoW → Server verifies → Server sends quote
```

## Quick Start

### Using Docker Compose

```bash
# Build and run both server and client
docker-compose up

# Run server only
docker run -p 8080:8080 world-of-wisdom-server

# Run client
docker run world-of-wisdom-client -server host.docker.internal:8080
```

### Using Go

```bash
# Run server
go run cmd/server/main.go -difficulty 3

# Run client
go run cmd/client/main.go -server localhost:8080 -attempts 5
```

## Configuration

### Server Options

- `--port`: TCP port (default: `:8080`)
- `--difficulty`: Initial PoW difficulty 1-6 (default: `2`)
- `--timeout`: Client timeout (default: `30s`)

### Client Options

- `--server`: Server address (default: `localhost:8080`)
- `--attempts`: Number of quote requests (default: `1`)
- `--timeout`: Request timeout (default: `30s`)

## Building

```bash
# Build binaries
make build

# Run tests
make test

# Build Docker images
make docker-build

# Clean up
make clean
```

## Performance

On a MacBook Pro (M3):
- Difficulty 1: ~10μs
- Difficulty 2: ~100μs
- Difficulty 3: ~1ms
- Difficulty 4: ~10ms
- Difficulty 5: ~100ms
- Difficulty 6: ~1s

## Development

### Project Structure

```
├── cmd/
│   ├── server/     # Server entry point
│   └── client/     # Client entry point
├── internal/
│   ├── server/     # Server implementation
│   └── client/     # Client implementation
├── pkg/
│   ├── pow/        # Proof-of-Work logic
│   └── wisdom/     # Quote management
└── docker/         # Dockerfiles
```

### Testing

```bash
# Unit tests
go test ./...

# Integration test with Docker
docker-compose up --abort-on-container-exit
```

## License

This project is part of a technical assessment and is provided as-is for educational purposes.