.PHONY: build test run-server run-client run-webserver run-web dev docker-build docker-run clean clean-all rebuild

build:
	mkdir -p bin
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go
	go build -o bin/webserver cmd/webserver/main.go

test:
	go test -v ./...

run-server:
	go run cmd/server/main.go

run-client:
	go run cmd/client/main.go

run-webserver:
	go run cmd/webserver/main.go

run-web:
	cd web && npm run dev

dev:
	./scripts/dev.sh

docker-build:
	docker build -f Dockerfile.server -t world-of-wisdom-server .
	docker build -f Dockerfile.client -t world-of-wisdom-client .
	docker build -f Dockerfile.webserver -t world-of-wisdom-webserver .
	docker build -f Dockerfile.web -t world-of-wisdom-web .

docker-run:
	docker-compose up

docker-test:
	docker-compose up --abort-on-container-exit

clean:
	rm -rf bin/ logs/
	docker-compose down
	docker rmi world-of-wisdom-server world-of-wisdom-client world-of-wisdom-webserver world-of-wisdom-web || true

clean-all:
	@echo "🧹 Starting full project cleanup..."
	@echo "Stopping all containers..."
	docker-compose down -v --remove-orphans || true
	docker-compose -f docker-compose.db.yml down -v --remove-orphans || true
	@echo "Removing all project volumes..."
	docker volume rm wisdom-data postgres_data redis_data || true
	@echo "Pruning all unused volumes..."
	docker volume prune -f
	@echo "Removing all project images..."
	docker rmi world-of-wisdom-server world-of-wisdom-client world-of-wisdom-webserver world-of-wisdom-web world-of-wisdom-apiserver || true
	@echo "Pruning all unused images..."
	docker image prune -f
	@echo "Removing all unused containers..."
	docker container prune -f
	@echo "Removing all unused networks..."
	docker network prune -f
	@echo "✅ Full cleanup complete!"

re-run: clean-all
	@echo "🚀 Starting fresh project build and run..."
	@echo "Building all Docker images..."
	docker-compose build --no-cache
	@echo "Starting services..."
	docker-compose up -d --remove-orphans
	@echo "✅ Project restarted with clean state!"