# Word of Wisdom - TCP Server with Advanced PoW Protection

![land](images/land.jpeg)

A production-ready TCP server that serves wisdom quotes protected by Proof-of-Work (PoW) challenges. Features Argon2 memory-hard puzzles, adaptive difficulty, real-time visualization, PostgreSQL/TimescaleDB persistence, and comprehensive monitoring.

## 🚀 Quick Start

```bash
# Start complete system with databases
make re-run

# Access services
# - Web UI: http://localhost:3000  
# - TCP Server: localhost:8080
# - REST API: http://localhost:8082/api/v1
# - API Docs: http://localhost:8082/api/v1/docs
# - Metrics: http://localhost:2112/metrics
```

## ✨ Features

- **🛡️ Advanced Security**: Argon2 memory-hard PoW puzzles with adaptive difficulty
- **💾 Data Persistence**: PostgreSQL/TimescaleDB for metrics, Redis for caching  
- **📊 Real-time Monitoring**: Interactive React dashboard with live WebSocket updates
- **🚀 REST API Gateway**: Type-safe database operations with sqlc-generated queries
- **🔄 Auto-Recovery**: Robust error handling with automatic reconnection
- **📈 Comprehensive Metrics**: Prometheus integration with 10+ metrics
- **🐳 Production Ready**: Docker deployment with health checks and restart policies

## 📊 Proof-of-Work Algorithm Comparison

![khajiit](images/khajiit.jpeg)

### SHA-256 vs Argon2 Performance Analysis

| Aspect | SHA-256 PoW | Argon2 PoW |
|--------|-------------|------------|
| **Solve Time** | ~0.33s (20 bits) | ~0.07s (t=3, m=64MB, p=4) |
| **Memory Usage** | Negligible | 64MB (adjustable) |
| **GPU/ASIC Advantage** | >100× speedup | ≤32× (memory limits parallelism) |
| **Verification Cost** | <1ms (single hash) | ~70ms (memory-hard hash) |
| **Difficulty Tuning** | Leading-zero bits (coarse) | (time, memory, parallelism) parameters |
| **Implementation** | Very simple | Moderate (existing libraries) |

### Why Argon2 for DDoS Protection?

**SHA-256 Limitations:**
- CPU-bound only with negligible memory footprint
- Highly parallelizable on GPUs/ASICs (>100× speedup)
- Attackers can achieve millions of hashes per second
- Trivial solve times under specialized hardware attack

**Argon2 Advantages:**
- **Memory Hardness**: Forces 64MB+ per parallel thread
- **GPU Resistance**: Limited by available RAM (≤32× vs >100× for SHA-256)
- **Tunable Parameters**: Fine control via (time, memory, parallelism)
- **Future-Proof**: Maintains security as hardware improves

### Performance Benchmarks

**SHA-256 Performance:**
- ~1.56 million hashes/second per CPU core
- 20-bit difficulty: ~0.33s average solve time
- 24-bit difficulty: ~10s average solve time

**Argon2 Performance:**
- ~14 hashes/second with t=3, m=64MB, p=4
- Memory bandwidth becomes bottleneck
- Parallel scaling limited by RAM availability

**Conclusion:** While SHA-256 offers simplicity and minimal server cost, Argon2 provides superior resistance to large-scale, GPU-accelerated attacks through memory hardness, making it the preferred choice for robust DDoS mitigation.

## 🏗️ Architecture

![arch](images/arch.jpeg)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           Word of Wisdom System                          │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────────────┤
│   Database  │ TCP Server  │ Web Server  │ API Server  │ React Frontend  │
│             │ (Port 8080) │ (Port 8081) │ (Port 8082) │  (Port 3000)    │
│ PostgreSQL  │             │             │             │                 │
│ TimescaleDB │ ┌─────────┐ │ ┌─────────┐ │ ┌─────────┐ │ ┌─────────────┐ │
│ Redis       │ │ Argon2  │ │ │WebSocket│ │ │ REST    │ │ │ Blockchain  │ │
│             │ │ PoW     │ │ │ API     │ │ │ API     │ │ │ Visualizer  │ │
│ ┌─────────┐ │ │ Engine  │ │ │ Mining  │ │ │ sqlc    │ │ │ Live Logs   │ │
│ │Metrics  │ │ │ Adaptive│ │ │ Sim     │ │ │ Queries │ │ │ Statistics  │ │
│ │Storage  │ │ │ Diff    │ │ │ Control │ │ │ CRUD    │ │ │ Monitoring  │ │
│ └─────────┘ │ └─────────┘ │ └─────────┘ │ └─────────┘ │ └─────────────┘ │
└─────────────┴─────────────┴─────────────┴─────────────┴─────────────────┘
```

## 🔧 Configuration

### Server Options

```bash
go run cmd/server/main.go [options]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-algorithm` | `argon2` | PoW algorithm: `sha256` or `argon2` |
| `-port` | `:8080` | TCP port to listen on |
| `-difficulty` | `2` | Initial PoW difficulty (1-6) |
| `-timeout` | `30s` | Client connection timeout |
| `-adaptive` | `true` | Enable adaptive difficulty |
| `-metrics-port` | `:2112` | Prometheus metrics port |

### Environment Variables

```bash
# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=wisdom
POSTGRES_PASSWORD=wisdom123
REDIS_HOST=localhost
REDIS_PORT=6379

# Algorithm Selection
ALGORITHM=argon2  # or sha256
DIFFICULTY=2
ADAPTIVE_MODE=true
```

## 📈 Monitoring & Metrics

### Key Metrics (Prometheus)

```shell
# Connection metrics
wisdom_connections_total{status="accepted"}
wisdom_active_connections

# Challenge metrics  
wisdom_puzzles_solved_total{difficulty="N"}
wisdom_current_difficulty

# Performance metrics
wisdom_solve_time_seconds{difficulty="N"}
wisdom_average_solve_time_seconds
```

### Web Dashboard Features

- **📊 Live Metrics**: Real-time difficulty and performance tracking
- **🔗 Blockchain View**: Visual representation of solved challenges
- **📋 Enhanced Logs**: Paginated activity logs (latest first, 600px height)
- **🎮 Interactive Controls**: Demo mode with progress tracking
- **🔄 Connection Status**: WebSocket state with auto-reconnection
- **💾 Persistent Stats**: Data survives page refresh

## 🧪 Testing

```bash
# Unit tests
go test ./...

# Integration tests  
go test -v ./tests/

# Load testing
docker-compose up --scale client1=10

# Web interface testing
# 1. Visit http://localhost:3000
# 2. Try Demo Mode (60s simulation)  
# 3. Test Extreme Mode (resource-safe)
# 4. Verify logs show latest first
# 5. Check connection recovery
```

## 🐳 Production Deployment

### Docker Compose Setup

```bash
# Start all services with databases
docker-compose up -d

# Scale for load testing
docker-compose up --scale client1=5 --scale client2=3

# Check service health
docker-compose ps
```

### VPS Deployment (Planned)

```bash
# SSH to VPS instances
ssh vps_1_1  # Primary server
ssh vps_1_2  # Secondary server

# Deploy with GitHub Actions (planned)
# Or manual deployment:
make run-server
```

## 📁 Project Structure

![lib](images/lib.jpeg)

```
world-of-wisdom/
├── cmd/                    # Executables
│   ├── server/            # TCP server (Argon2 PoW)
│   ├── client/            # Test client
│   ├── webserver/         # WebSocket API
│   └── apiserver/         # REST API server
├── internal/              # Application logic
│   ├── server/            # TCP server implementation
│   ├── webserver/         # WebSocket server implementation
│   └── apiserver/         # REST API implementation
├── api/db/                # Generated database code (sqlc)
├── pkg/                   # Libraries
│   ├── pow/               # PoW algorithms (SHA-256 + Argon2)
│   ├── database/          # PostgreSQL/Redis integration
│   ├── config/            # Environment configuration
│   └── metrics/           # Prometheus metrics
├── web/                   # React frontend
│   ├── src/components/    # Enhanced UI components
│   ├── src/hooks/         # WebSocket with reconnection
│   └── src/utils/         # Persistence utilities
├── db/                    # Database layer
│   ├── migrations/        # Database schema
│   └── queries/           # SQL queries for sqlc
├── sqlc.yaml              # sqlc configuration
├── docker-compose.yml     # Full stack deployment
└── STABILITY-IMPROVEMENTS.md # Technical details
```

## 🔄 Recent Improvements

- ✅ **Enhanced Security**: SHA-256 → Argon2 memory-hard PoW
- ✅ **Database Integration**: PostgreSQL + TimescaleDB + Redis
- ✅ **Frontend Stability**: Persistent stats, enhanced logs, auto-recovery
- ✅ **Resource Safety**: Conservative limits prevent crashes under extreme load
- ✅ **Live Metrics**: Real-time difficulty tracking and updates
- ✅ **REST API Gateway**: Type-safe database operations with comprehensive endpoints
- ✅ **Production Ready**: Restart policies, health checks, monitoring

### 🖼️ Frontend Demo

![front-demo](images/front-demo.png)

## 📜 License

Educational project demonstrating advanced Go programming, cryptographic PoW systems, real-time web applications, and production monitoring solutions.

---

**Built with:** Go, React, TypeScript, PostgreSQL, TimescaleDB, Redis, Docker, Prometheus, Mantine UI