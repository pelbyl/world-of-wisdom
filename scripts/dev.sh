#!/bin/bash

# Development script to run all services

set -e

echo "Starting Word of Wisdom development environment..."

# Kill any existing processes
pkill -f "go run cmd/server/main.go" || true
pkill -f "go run cmd/webserver/main.go" || true
pkill -f "npm run dev" || true

sleep 2

# Start TCP server
echo "Starting TCP server..."
go run cmd/server/main.go -difficulty 2 > logs/server.log 2>&1 &
SERVER_PID=$!

sleep 2

# Start web server
echo "Starting web server..."
go run cmd/webserver/main.go > logs/webserver.log 2>&1 &
WEBSERVER_PID=$!

sleep 2

# Start frontend
echo "Starting frontend..."
cd web && npm run dev > ../logs/frontend.log 2>&1 &
FRONTEND_PID=$!

echo "Services started:"
echo "  TCP Server: http://localhost:8080 (PID: $SERVER_PID)"
echo "  Web Server: http://localhost:8081 (PID: $WEBSERVER_PID)"
echo "  Frontend: http://localhost:3000 (PID: $FRONTEND_PID)"

echo ""
echo "Press Ctrl+C to stop all services..."

# Cleanup function
cleanup() {
    echo "Stopping services..."
    kill $SERVER_PID $WEBSERVER_PID $FRONTEND_PID 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT

# Wait for all services
wait