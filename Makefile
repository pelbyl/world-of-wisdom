.PHONY: build test run-server run-client run-webserver run-web dev docker-build docker-run clean clean-all clean-full re-run rebuild run-server-vps deploy-vps

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
	docker-compose -f docker-compose.test.yml down -v --remove-orphans || true
	@echo "Removing all project volumes..."
	docker volume rm wisdom-data postgres_data redis_data service-registry-data gateway-config || true
	docker volume rm world-of-wisdom_wisdom-data world-of-wisdom_postgres_data world-of-wisdom_redis_data || true
	docker volume rm world-of-wisdom_service-registry-data world-of-wisdom_gateway-config || true
	@echo "Removing all project images..."
	docker rmi world-of-wisdom-server world-of-wisdom-client world-of-wisdom-webserver world-of-wisdom-web || true
	docker rmi world-of-wisdom-apiserver || true
	@echo "Removing local data directories..."
	rm -rf bin/ logs/ /tmp/wisdom-data/ || true
	@echo "Pruning all unused Docker resources..."
	docker volume prune -f
	docker image prune -f
	docker container prune -f
	docker network prune -f
	@echo "✅ Full cleanup complete!"

clean-full: 
	@echo "🗑️  Starting COMPLETE project cleanup (including databases)..."
	@echo "Stopping all containers..."
	docker-compose down -v --remove-orphans || true
	docker-compose -f docker-compose.test.yml down -v --remove-orphans || true
	@echo "Removing ALL project-specific volumes and data..."
	docker volume rm wisdom-data postgres_data redis_data service-registry-data gateway-config || true
	docker volume rm world-of-wisdom_wisdom-data world-of-wisdom_postgres_data world-of-wisdom_redis_data || true
	docker volume rm world-of-wisdom_service-registry-data world-of-wisdom_gateway-config || true
	@echo "Removing ALL project images..."
	docker rmi world-of-wisdom-server world-of-wisdom-client world-of-wisdom-webserver world-of-wisdom-web || true
	docker rmi world-of-wisdom-apiserver || true
	docker rmi world-of-wisdom-client1 world-of-wisdom-client2 || true
	@echo "Removing local data directories and caches..."
	rm -rf bin/ logs/ /tmp/wisdom-data/ || true
	rm -rf web/node_modules/ web/dist/ web/.next/ || true
	@echo "Clearing Go build cache..."
	go clean -cache -modcache -i -r || true
	@echo "Removing any dangling project containers..."
	docker container prune -f
	@echo "Removing any dangling project volumes..."
	docker volume prune -f
	@echo "✅ COMPLETE project cleanup finished!"


clean-full-system: clean-full
	@echo "🚨 WARNING: This will remove ALL Docker data (affects all projects)!"
	@read -p "Are you sure you want to remove ALL Docker data? [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1
	docker system prune -a --volumes -f
	@echo "🚨 COMPLETE system cleanup finished - ALL Docker data removed!"

re-run: clean-full
	@echo "🚀 Starting fresh project build and run..."
	@echo "Building all Docker images..."
	docker-compose build --no-cache
	@echo "Starting services..."
	docker-compose up -d --remove-orphans
	@echo "Waiting for services to be ready..."
	sleep 10
	@echo "Clearing all application data for fresh start..."
	@if [ "$$(docker ps -q -f name=wisdom-postgres)" ]; then \
		echo "Clearing database data..."; \
		docker exec wisdom-postgres psql -U wisdom -d wisdom -c " \
			SET session_replication_role = replica; \
			TRUNCATE TABLE blocks RESTART IDENTITY CASCADE; \
			TRUNCATE TABLE solutions RESTART IDENTITY CASCADE; \
			TRUNCATE TABLE challenges RESTART IDENTITY CASCADE; \
			TRUNCATE TABLE connections RESTART IDENTITY CASCADE; \
			TRUNCATE TABLE metrics RESTART IDENTITY CASCADE; \
			SET session_replication_role = DEFAULT; \
			ALTER SEQUENCE blocks_id_seq RESTART WITH 1; \
			INSERT INTO metrics (metric_name, metric_value, labels) VALUES ('server_started', 1, '{\"version\": \"1.0.0\", \"algorithm\": \"argon2\", \"cleaned\": true}'); \
		" >/dev/null 2>&1; \
	fi
	@echo "Checking service status..."
	docker-compose ps
	@echo "✅ Project restarted with completely clean state!"
	@echo "🌐 Access points:"
	@echo "   Web UI: http://localhost:3000 (empty blockchain & logs)"
	@echo "   TCP Server: localhost:8080"
	@echo "   REST API: http://localhost:8082/api/v1"
	@echo "   OpenAPI: http://localhost:8082/swagger/index.html"

# VPS deployment commands (for public IP addresses)
run-server-vps:
	@echo "🌍 Starting server for VPS deployment (public IP access)..."
	@echo "Setting up for public IP access..."
	docker-compose -f docker-compose.yml -e PUBLIC_IP=true up -d
	@echo "📝 Note: Configure your firewall to allow ports 8080-8085"
	@echo "📝 Update frontend config to use your public IP address"

deploy-vps:
	@echo "🚀 Preparing for VPS deployment..."
	@echo "Building optimized production images..."
	docker-compose build --no-cache
	@echo "Exporting images for VPS transfer..."
	mkdir -p vps-deploy
	docker save world-of-wisdom-server:latest | gzip > vps-deploy/server.tar.gz
	docker save world-of-wisdom-webserver:latest | gzip > vps-deploy/webserver.tar.gz
	docker save world-of-wisdom-apiserver:latest | gzip > vps-deploy/apiserver.tar.gz
	docker save world-of-wisdom-web:latest | gzip > vps-deploy/web.tar.gz
	@echo "Copying deployment files..."
	cp docker-compose.yml vps-deploy/
	cp docker-compose.test.yml vps-deploy/
	cp -r db/ vps-deploy/
	@echo "📦 VPS deployment package ready in ./vps-deploy/"
	@echo "📝 Next steps:"
	@echo "   1. Transfer vps-deploy/ to your VPS"
	@echo "   2. Load images: docker load < server.tar.gz"
	@echo "   3. Update environment variables for public IP"
	@echo "   4. Run: docker-compose up -d"