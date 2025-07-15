# World of Wisdom - Development Guide

## Per-Client Adaptive Difficulty System

### Current Status

The backend implementation for per-client adaptive difficulty is **complete**! The system includes:

✅ **Completed Backend Features:**
1. Client Behavior Tracking - Database table and Go structs tracking per-IP metrics
2. Adaptive Difficulty Algorithm - SQL function calculating difficulty (1-6) based on behavior
3. Reputation System - Tracks and decays reputation over time
4. API Endpoints - `/api/v1/client-behaviors` returns client data
5. Server Integration - TCP server uses per-client difficulty for challenges

✅ **Completed Frontend Features:**
1. Client Behavior Dashboard - Real-time monitoring in MetricsDashboard.tsx
2. Experiment Analytics UI - Comprehensive testing dashboard
3. API-driven components - All data served from backend

### System Architecture

The adaptive difficulty system tracks each client's behavior and adjusts challenge difficulty accordingly:

```go
type ClientBehavior struct {
    IP              string
    ConnectionCount int
    FailureRate     float64
    AvgSolveTime    time.Duration
    LastConnection  time.Time
    ReconnectRate   float64
    Difficulty      int
}
```

## Experiment Execution Guide

### Prerequisites

1. Ensure Docker and Docker Compose are installed
2. Clone the repository and navigate to the project directory
3. Build all required images:
   ```bash
   docker-compose build
   ```

### Quick Start

1. **Start base services:**
   ```bash
   make demo
   ```

2. **Run a scenario:**
   ```bash
   make scenario-morning-rush    # Or any other scenario
   ```

3. **Monitor in the UI:**
   ```bash
   make monitor    # Opens http://localhost:3000
   ```
   Navigate to Experiment Analytics tab for scenario-specific monitoring

### Available Scenarios

#### 1. Morning Rush (Legitimate Traffic Spike)
- Gradual increase of normal users
- Power users joining during peak hours
- System should maintain low difficulty for all legitimate users

#### 2. Script Kiddie Attack
- Normal baseline traffic
- Single attacker with basic automation
- System should quickly identify and increase attacker difficulty to 5-6
- Normal users should remain unaffected

#### 3. Sophisticated DDoS
- Coordinated attack from multiple sources
- Optimized solvers attempting to bypass defenses
- System should identify patterns and penalize all attackers
- Reputation system should prevent evasion attempts

#### 4. Botnet Simulation
- Multiple attacking nodes with varying patterns
- Partial botnet takedown simulation
- System should handle dynamic attack patterns

#### 5. Mixed Reality
- Continuous normal and power user traffic
- Intermittent script kiddie attacks
- Sophisticated attacker probes
- Random botnet nodes

### Client Personas

1. **Normal User (Green)**
   - Connection frequency: 1-2 connections per minute
   - Solve time: 1-3 seconds
   - Expected difficulty: 1-2

2. **Power User (Green-Yellow)**
   - Connection frequency: 5-10 connections per minute
   - Solve time: 0.5-2 seconds
   - Expected difficulty: 2-3

3. **Script Kiddie (Orange-Red)**
   - Connection frequency: 20-50 connections per minute
   - Solve time: 50-200ms (using basic solver)
   - Expected difficulty: 4-5

4. **Sophisticated Attacker (Red)**
   - Connection frequency: 100+ connections per minute
   - Solve time: 10-100ms (optimized solver)
   - Expected difficulty: 5-6

5. **Botnet Node (Red)**
   - Connection frequency: 10-30 connections per minute
   - Solve time: 100-500ms
   - Expected difficulty: 3-5

### Success Criteria

**Protection Effectiveness**
- ✅ Normal users maintain difficulty 1-2 throughout all scenarios
- ✅ Attackers reach difficulty 5-6 within 2-3 minutes
- ✅ System remains responsive under 100+ concurrent attackers
- ✅ Difficulty adjusts within 30 seconds of behavior change

**User Experience**
- ✅ Legitimate users solve challenges in <3 seconds
- ✅ Power users complete tasks despite higher difficulty
- ✅ No false positives for normal usage patterns
- ✅ Clear visual feedback in monitoring UI

**Behavioral Adaptation**
- ✅ Reputation decay reduces difficulty for reformed clients
- ✅ Burst detection catches sudden attack spikes
- ✅ Pattern recognition identifies coordinated attacks
- ✅ IP tracking prevents simple evasion tactics

## UI Analytics Components

The system includes comprehensive analytics visualizations:

### 1. Experiment Analytics Dashboard
- **Location**: `/web/src/components/ExperimentAnalytics.tsx`
- **Features**: 
  - Scenario tabs for different attack types
  - Real-time success criteria evaluation
  - Live mode for active monitoring
  - Performance metrics by difficulty level

### 2. Client Behavior Monitoring
- **Real-time Updates**: Fetches from `/api/v1/client-behaviors`
- **Visual Indicators**: Color-coded difficulty levels
- **Metrics**: Connection count, failure rate, solve time, reputation

### 3. Attack Mitigation Analysis
- **Detection Rate**: Percentage of identified attackers
- **False Positive Rate**: Incorrectly flagged legitimate users
- **Normal User Impact**: Average solve time for legitimate traffic
- **Effectiveness Score**: Overall system performance

### 4. API Endpoints

```
GET /api/v1/experiment/summary?scenario={scenario}
GET /api/v1/experiment/success-criteria
GET /api/v1/experiment/timeline?scenario={scenario}
GET /api/v1/experiment/performance
GET /api/v1/experiment/mitigation
GET /api/v1/experiment/comparison
```

## Experiment Tools

### 1. Scenario Commands (Makefile)
Run predefined scenarios using make commands:
```bash
# Start scenarios
make scenario-morning-rush     # Legitimate traffic spike
make scenario-script-kiddie    # Basic automated attack
make scenario-ddos            # Sophisticated DDoS attack
make scenario-botnet          # Distributed botnet simulation
make scenario-mixed           # Mixed reality scenario

# Control scenarios
make scenario-add-attackers   # Add attackers to mixed scenario
make scenario-status          # Check scenario status
make scenario-logs           # View scenario logs
make scenario-stop           # Stop all scenarios
```

### 2. Monitoring Dashboard
All monitoring is available through the web UI:
```bash
make monitor                  # Opens http://localhost:3000
```

The UI provides comprehensive real-time monitoring:
- **Metrics Dashboard**: Live system metrics and charts
- **Client Behaviors Table**: Per-IP difficulty with color coding
- **Attack Detection**: Visual indicators for aggressive clients
- **Experiment Analytics**: Scenario-specific analysis and success criteria

## Docker Compose Configuration

The `docker-compose.scenario.yml` file defines all client types with appropriate parameters:

```yaml
services:
  normal-user:
    environment:
      - CLIENT_TYPE=normal
      - SOLVE_DELAY_MS=1500
      - CONNECTION_DELAY_MS=30000
    deploy:
      replicas: 10

  script-kiddie:
    environment:
      - CLIENT_TYPE=attacker
      - SOLVE_DELAY_MS=125
      - CONNECTION_DELAY_MS=1200
      - ATTACK_MODE=true
    deploy:
      replicas: 3
```

## Troubleshooting

1. **Services not starting:**
   ```bash
   docker-compose logs -f server
   docker-compose logs -f apiserver
   ```

2. **Monitor shows no data:**
   - Ensure API server is running on port 8081
   - Check: `curl http://localhost:8081/api/v1/client-behaviors`

3. **Clients not connecting:**
   - Verify server is running on port 8080
   - Check client logs: `docker-compose logs -f normal-user`

## Best Practices

1. **Always start fresh:**
   ```bash
   docker-compose down -v
   make demo
   ```

2. **Monitor resource usage during experiments:**
   ```bash
   docker stats
   ```

3. **Save experiment results for comparison**

4. **Adjust parameters in `docker-compose.scenario.yml` to test edge cases**

## Important Reminders

- Do what has been asked; nothing more, nothing less
- NEVER create files unless they're absolutely necessary
- ALWAYS prefer editing existing files to creating new ones
- NEVER proactively create documentation files unless explicitly requested