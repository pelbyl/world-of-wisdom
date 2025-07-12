# Word of Wisdom TCP Server

A high-performance TCP server in Go that serves random quotes ("words of wisdom") protected by a Proof-of-Work (PoW) challenge-response protocol to prevent DDoS attacks.

## Features

- **DDoS Protection**: SHA-256 based Proof-of-Work challenge prevents abuse
- **Concurrent Design**: Handles multiple clients simultaneously using Go routines
- **Configurable Difficulty**: Adjustable PoW difficulty levels (1-6)
- **Docker Support**: Containerized deployment for both server and client
- **Connection Timeouts**: Automatic cleanup of stale connections
- **Random Wisdom Quotes**: Serves from a curated collection of 25+ quotes
- **Web Visualization**: Real-time blockchain visualization with Mantine UI
- **WebSocket Updates**: Live monitoring of mining operations and connections

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

### Web Visualization

```bash
# Start all services with web UI
make dev

# Or using Docker Compose (includes web interface)
docker-compose up

# Visit http://localhost:3000 for the blockchain visualizer
```

### Command Line Interface

```bash
# Run server only
go run cmd/server/main.go -difficulty 3

# Run client
go run cmd/client/main.go -server localhost:8080 -attempts 5

# Run web server with visualization
go run cmd/webserver/main.go
```

### Docker Services

```bash
# Build all images
make docker-build

# Run complete stack
docker-compose up

# Individual services
docker run -p 8080:8080 world-of-wisdom-server
docker run -p 8081:8081 world-of-wisdom-webserver
docker run -p 3000:3000 world-of-wisdom-web
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
│   ├── server/     # TCP Server entry point
│   ├── client/     # TCP Client entry point
│   └── webserver/  # Web/WebSocket server
├── internal/
│   ├── server/     # TCP Server implementation
│   ├── client/     # TCP Client implementation
│   └── webserver/  # Web server with blockchain simulation
├── pkg/
│   ├── pow/        # Proof-of-Work logic
│   └── wisdom/     # Quote management
├── web/            # React frontend with Mantine UI
│   ├── src/
│   │   ├── components/  # Visualization components
│   │   ├── hooks/       # WebSocket hooks
│   │   └── types/       # TypeScript types
└── scripts/        # Development scripts
```

### Web Visualization Features

- **Real-time Blockchain**: Visual representation of mined blocks
- **Mining Monitor**: Live view of active PoW challenges being solved
- **Network Stats**: Current difficulty, hash rate, success rate
- **Connection Panel**: Active clients and their status
- **Interactive Simulation**: Button to simulate new clients
- **WebSocket Updates**: Real-time updates without polling

### Testing

```bash
# Unit tests
go test ./...

# Integration test with Docker
docker-compose up --abort-on-container-exit
```

## License

This project is part of a technical assessment and is provided as-is for educational purposes.