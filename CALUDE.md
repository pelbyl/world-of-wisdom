# Word of Wisdom Implementation Guide

## Architecture

```shell
Browser ←─REST─→ API Server ←─SQL─→ PostgreSQL ←─SQL─← TCP Server
(React)          (Port 8081)        (TimescaleDB)      (Port 8080)
```

- **Browser**: Polls API for data visualization
- **API Server**: Read-only database access, serves REST API
- **TCP Server**: Write-only database access, handles PoW protocol
- **Database**: Single source of truth with time-series optimization

## Requirements

### 1. Database (PostgreSQL + TimescaleDB)

- Use SQLC for all queries (no raw SQL in code)
- Define queries in `db/queries/*.sql`
- Use TimescaleDB hypertables for time-series data

### 2. API Server

- Use Go Echo web framework"github.com/labstack/echo/v4"
- Use OpenAPI 3.0 spec in `api/openapi.yaml`
- Generate server code with `oapi-codegen`
- Implement handlers in `internal/api/handlers.go`
- Read-only database access
- Delete all code that is not related to serving data to the frontend.
- Delete all simulations, demos related code.

Required endpoints:

- `GET /api/v1/stats` - Current system statistics
- `GET /api/v1/challenges` - Recent challenges list
- `GET /api/v1/metrics` - Time-series metrics
- `GET /api/v1/connections` - Active connections

### 3. TCP Server

- Write-only database access
- Batch writes for performance
- Log all events: challenges, solutions, connections

### 4. Frontend

- Mantine UI
- No WebSockets - use REST polling only
- Poll the API server
- Poll intervals: stats (1s), challenges (2s), metrics (5s)

## Implementation Steps

1. **Database Setup**

    - Create migrations in db/migrations/
    - Generate SQLC: sqlc generate

2. **API Server Development**

    - Define OpenAPI spec
    - Generate code: oapi-codegen api/openapi.yaml
    - Implement handlers

3. **Run System**

    ```bash
    make re-run
    ```

## Key Rules

- TCP server only writes to database
- API server only reads from database
- Frontend only accesses API (no direct DB)
- Use code generation (SQLC, OpenAPI)
- No WebSockets - REST polling is sufficient

## Demo Control

I control all demo manually. Don't do anything automatically. Remove any code that is for demos, except demo docker compose and make demo.

## General Rules

- Keep it simple, concise and clean.
- Keep it readable
- Keep it efficient
- Keep it easy to understand
- Keep it easy to maintain
- Keep it consice
- Don't think about CI/CD or anything else. Just focus on local docker compose.
