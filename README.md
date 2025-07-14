# Word of Wisdom - TCP Server with PoW Protection

![land](images/land.jpeg)

A simple TCP server that serves wisdom quotes protected by Proof-of-Work (PoW) challenges. Features Argon2 memory-hard puzzles, adaptive difficulty, real-time visualization, and PostgreSQL persistence.

## 🚀 Quick Start

```bash
# Setup environment (first time only)
cp .env.example .env

# Start complete system with docker-compose
docker-compose up -d

# Run demo with multiple clients
make demo

# Access services
# - Web Dashboard: http://localhost:3000  
# - TCP Server: localhost:8080
# - REST API: http://localhost:8081
```

## ✨ Features

- **🛡️ Security**: Argon2 memory-hard PoW puzzles with adaptive difficulty
- **💾 Data Persistence**: PostgreSQL TimescaleDB for metrics and application data (with sqlc-generated queries)
- **📊 Real-time Monitoring**: Mantine UI dashboard with Polling
- **🚀 REST API**: Type-safe database operations with sqlc-generated queries
- **🔄 Auto-Recovery**: Robust error handling with automatic reconnection
- **🐳 Docker Ready**: Simple docker-compose setup for local development

## ⚙️ Environment Configuration

The application uses environment variables for configuration management.

The `.env` file contains all configurable variables with sensible defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | TCP server port |
| `API_SERVER_PORT` | 8081 | REST API server port |
| `WEB_PORT` | 3000 | Frontend port |
| `POSTGRES_*` | Various | Database configuration |
| `ALGORITHM` | argon2 | PoW algorithm (sha256/argon2) |
| `DIFFICULTY` | 2 | Mining difficulty |
| `ADAPTIVE_MODE` | true | Enable adaptive difficulty |

**Note:** Environment variables are automatically loaded by docker-compose from the `.env` file.

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

### System Overview

The Word of Wisdom system is a simple, clean architecture with three core components running in Docker containers.

```shell
┌──────────────────────────────────────────────────────────────────────┐
│                       Word of Wisdom System                          │
└──────────────────────────────────────────────────────────────────────┘

┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ TCP Server  │    │  Database   │    │ API Server  │    │  Frontend   │
│             │    │             │    │             │    │             │
│ Argon2 PoW  │◄──►│ PostgreSQL  │◄──►│ REST API    │◄──►│ React App   │
│ DDoS Guard  │    │ TimescaleDB │    │             │    │ Dashboard   │
│ Adaptive    │    │ Metrics     │    │             │    │ Visualizer  │
│             │    │ Logs        │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

### Core Components

#### 1. **Database Layer** (PostgreSQL)

**PostgreSQL Database:**

- **Primary Storage**: All application data with ACID compliance
- **Type Safety**: Integration with sqlc for compile-time SQL validation

**Key Tables:**

```sql
-- Challenges (PoW puzzle tracking)
challenges(id, seed, difficulty, algorithm, client_id, status, created_at, solved_at)

-- Solutions (completed puzzles)  
solutions(id, challenge_id, nonce, hash, attempts, solve_time_ms, verified)

-- Connections (client session tracking)
connections(id, client_id, remote_addr, status, algorithm, connected_at)

-- Blocks (blockchain-like storage)
blocks(id, block_index, challenge_id, solution_id, quote, previous_hash, block_hash)

-- Metrics (system metrics)
metrics(time, metric_name, metric_value, labels, server_instance)

-- Logs (activity logs)  
logs(id, timestamp, level, message, icon, metadata, created_at)
```

#### 2. **TCP Server** (Port 8080) - Core PoW Engine

**Responsibilities:**

- Handle raw TCP connections from clients
- Generate and validate Proof-of-Work challenges
- Serve wisdom quotes to successful solvers
- Adaptive difficulty adjustment based on network load

**PoW Algorithm Implementation:**

```go
// Argon2 (Default - Memory-Hard)
type Argon2Challenge struct {
    Seed      string `json:"seed"`
    Difficulty int   `json:"difficulty"` 
    Time      uint32 `json:"time"`      // t=3
    Memory    uint32 `json:"memory"`    // m=64MB  
    Threads   uint8  `json:"threads"`   // p=4
    KeyLength uint32 `json:"keyLength"` // 32 bytes
}
```

**Adaptive Security Features:**

- **Dynamic Difficulty**: Adjusts 1-6 based on solve times and connection rate
- **Resource Protection**: CPU and memory usage monitoring
- **Connection Limits**: Per-IP rate limiting and concurrent connection caps

#### 3. **API Server** (Port 8081) - REST API

**REST API Features:**

- **Database Access**: Serves real-time data to frontend via HTTP endpoints
- **OpenAPI Specification**: Documented API with typed responses
- **Real-Time Polling**: Frontend polls for live data updates
- **Echo Framework**: High-performance HTTP server with CORS support

**Available Endpoints:**

```openapi
GET  /health               - Health check
GET  /api/v1/stats         - System statistics
GET  /api/v1/challenges    - Challenge list (with filters)
GET  /api/v1/connections   - Active connections
GET  /api/v1/metrics       - System metrics
GET  /api/v1/recent-solves - Recent blockchain blocks
GET  /api/v1/logs          - Activity logs
```

**Database Integration:**

- **SQLC Generated Queries**: Type-safe database operations
- **Automatic Logging**: All events stored to database
- **Metrics Recording**: Difficulty adjustments, connection stats, performance data

#### 4. **React Frontend** (Port 3000) - Interactive Dashboard

**Component Architecture:**

```typescript
App.tsx                          // Main application container
├── StatsPanel.tsx               // System statistics
├── ChallengePanel.tsx           // Active challenges display
├── ConnectionsPanel.tsx         // Active client connections
├── MetricsDashboard.tsx         // Performance metrics
└── LogsPanel.tsx                // System activity logs
```

**Real-Time Features:**

- **Auto-Refresh**: Real-time data updates via polling
- **Persistent State**: All data stored in PostgreSQL
- **Responsive Design**: Clean, modern UI with status indicators
- **Error Handling**: Automatic retry with graceful degradation

**Key Visualizations:**

```typescript
// Live Metrics Dashboard
interface MetricsData {
  timestamp: number
  connectionsTotal: number
  currentDifficulty: number
  puzzlesSolvedTotal: number
  puzzlesFailedTotal: number
  averageSolveTime: number
  connectionRate: number
  difficultyAdjustments: number
  activeConnections: number
}
```

## 🔧 Configuration

### Environment Variables

```bash
# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=wisdom
POSTGRES_PASSWORD=wisdom123

# Algorithm Selection
ALGORITHM=argon2  # or sha256
DIFFICULTY=2
ADAPTIVE_MODE=true
```

## 📈 Web Dashboard Features

- **📊 System Stats**: Real-time performance metrics and difficulty tracking
- **🔍 Challenge Monitor**: View active and completed challenges
- **👥 Connection Tracking**: Monitor active client connections
- **📋 Activity Logs**: System event logging with timestamps
- **💾 Persistent Data**: All data stored in PostgreSQL database
- **🔄 Auto-refresh**: Real-time updates via API polling

## 🧪 Testing & Demo

### Running the Demo

```bash
# Start the main system
docker-compose up -d

# Run demo with multiple client types
make demo

# Monitor demo activity
make demo-logs

# Check demo status
make demo-status

# Stop demo clients
make demo-stop
```

### Demo Client Types

- **Fast Clients**: Solve challenges quickly (100ms delay)
- **Normal Clients**: Standard solve time (1000ms delay)
- **Slow Clients**: Slower solving (3000ms delay)

### Testing API Endpoints

```bash
# Test API endpoints
curl -s "http://localhost:8081/health" | jq '.'
curl -s "http://localhost:8081/api/v1/stats" | jq '.'
curl -s "http://localhost:8081/api/v1/challenges?limit=10" | jq '.'
```

## 🐳 Docker Deployment

### Docker Compose Setup

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 📁 Project Structure

![lib](images/lib.jpeg)

```shell
world-of-wisdom/
├── cmd/                          # Executable entry points
│   ├── server/                   # TCP server (Argon2 PoW)
│   ├── client/                   # Demo client
│   └── apiserver/                # REST API server
├── internal/                     # Application logic
│   ├── server/                   # TCP server implementation
│   ├── apiserver/                # API server implementation
│   ├── client/                   # Client implementation
│   ├── blockchain/               # Blockchain implementation
│   └── database/                 # Database layer
│       ├── generated/            # SQLC generated code
│       ├── migrations/           # Database schema
│       ├── queries/              # SQL source files
│       └── repository/           # Repository pattern
├── pkg/                          # Shared libraries
│   ├── pow/                      # PoW algorithms (Argon2/SHA256)
│   ├── config/                   # Configuration management
│   ├── metrics/                  # Metrics collection
│   └── wisdom/                   # Quote management
├── web/                          # React frontend
│   ├── src/
│   │   ├── components/           # UI components
│   │   │   ├── StatsPanel.tsx
│   │   │   ├── ChallengePanel.tsx
│   │   │   ├── ConnectionsPanel.tsx
│   │   │   ├── MetricsDashboard.tsx
│   │   │   └── LogsPanel.tsx
│   │   ├── hooks/                # Custom React hooks
│   │   │   └── useAPI.ts         # API polling hook
│   │   ├── types/                # TypeScript definitions
│   │   └── api/                  # API client
│   └── nginx.conf                # Nginx proxy config
├── api/                          # API specification
│   └── openapi.yaml              # OpenAPI 3.0 spec
├── images/                       # Documentation assets
├── docker-compose.yml            # Main services
├── docker-compose.demo.yml       # Demo client setup
├── Dockerfile.server             # TCP server image
├── Dockerfile.apiserver          # API server image
├── Dockerfile.client             # Client image
├── Dockerfile.web                # Frontend image
├── Makefile                      # Build automation
├── .env.example                  # Environment template
└── README.md                     # This file
```

### Key Directories Explained

**Core Services:**

- `cmd/` - Entry points for server, client, and API server
- `internal/` - Private application code and business logic
- `pkg/` - Shared libraries (PoW, config, metrics)

**Database Layer:**

- `internal/database/generated/` - SQLC type-safe query code
- `internal/database/queries/` - SQL source for code generation
- `internal/database/repository/` - Clean repository pattern

**Frontend & API:**

- `web/` - React dashboard with real-time data display
- `api/openapi.yaml` - OpenAPI 3.0 specification
- REST API endpoints for all data access

**Docker Infrastructure:**

- `docker-compose.yml` - Main services (server, API, web, DB)
- `docker-compose.demo.yml` - Demo client orchestration
- Multiple Dockerfiles for each service

## 🔄 Key Features & Improvements

- ✅ **Argon2 PoW**: Memory-hard proof-of-work for DDoS protection
- ✅ **PostgreSQL Integration**: Full database persistence with type-safe queries
- ✅ **REST API**: Comprehensive endpoints with OpenAPI documentation
- ✅ **Real-time Dashboard**: Live metrics and system monitoring
- ✅ **Demo System**: Multiple client types for testing and demonstration
- ✅ **Docker Compose**: Complete containerized deployment
- ✅ **CORS Support**: Proper cross-origin request handling
- ✅ **Adaptive Difficulty**: Dynamic adjustment based on network conditions

### 🖼️ Frontend Demo

![front-demo](images/front-demo.png)

## 📜 License

Educational project demonstrating advanced Go programming, cryptographic PoW systems, real-time web applications, and clean architecture.

---

**Built with:** Go, React, TypeScript, PostgreSQL, Docker, Mantine UI
