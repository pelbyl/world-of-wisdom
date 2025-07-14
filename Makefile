.PHONY: re-run clean-all demo demo-stop generate sqlc oapi-codegen

# Original targets
re-run:
	@echo "🚀 Starting fresh Docker stack rebuild..."
	@echo "Stopping all containers..."
	docker-compose down -v --remove-orphans || true
	@echo "Removing Docker volumes..."
	docker volume rm world-of-wisdom_postgres_data || true
	@echo "Removing Docker images..."
	docker-compose down --rmi all || true
	@echo "Building and starting fresh stack..."
	docker-compose up -d --build --force-recreate
	@echo "✅ Fresh Docker stack ready!"
	@echo "🌐 Access points:"
	@echo "   Web UI: http://localhost:3000"
	@echo "   TCP Server: localhost:8080"
	@echo "   API Server: http://localhost:8081"

clean-all:
	@echo "🧹 Starting full project cleanup..."
	@echo "Removing local data directories..."
	rm -rf bin/ logs/ /tmp/wisdom-data/ || true
	rm -rf web/node_modules/ web/dist/ web/.next/ || true
	@echo "Clearing Go build cache..."
	go clean -cache -modcache -i -r || true
	@echo "✅ Full cleanup complete!"

# Demo commands using docker-compose
demo:
	@echo "🚀 Starting demo client containers..."
	@echo "Starting fast clients (1x), normal clients (2x), and slow clients (1x)..."
	docker-compose -f docker-compose.demo.yml up -d --scale client-fast=1 --scale client-normal=2 --scale client-slow=1
	@echo "✅ Demo clients started!"
	@echo "📊 Monitor at http://localhost:3000"
	@echo "📝 Check server logs: docker logs world-of-wisdom-server-1 -f"
	@echo "📝 Check client logs: docker-compose -f docker-compose.demo.yml logs -f"

demo-stop:
	@echo "🛑 Stopping demo containers..."
	docker-compose -f docker-compose.demo.yml down
	@echo "✅ Demo clients stopped!"

demo-logs:
	@echo "📝 Showing demo client logs..."
	docker-compose -f docker-compose.demo.yml logs -f

demo-status:
	@echo "📊 Demo container status:"
	docker-compose -f docker-compose.demo.yml ps

# DDoS demo scenario
demo-ddos:
	@echo "⚠️  Starting DDoS demo scenario..."
	@echo "This will create 10 clients (8 aggressive attackers + 2 legitimate)"
	@echo "Watch the adaptive difficulty increase in the dashboard!"
	docker-compose -f docker-compose.ddos.yml up -d
	@echo "✅ DDoS demo started!"
	@echo "📊 Monitor at http://localhost:3000"
	@echo "📝 Use 'make demo-ddos-logs' to view attack logs"

demo-ddos-stop:
	@echo "🛑 Stopping DDoS demo..."
	docker-compose -f docker-compose.ddos.yml down
	@echo "✅ DDoS demo stopped."

demo-ddos-logs:
	@echo "📝 Showing DDoS demo logs..."
	docker-compose -f docker-compose.ddos.yml logs -f

# Code generation targets
generate: sqlc oapi-codegen
	@echo "✅ All code generation complete!"

sqlc:
	@echo "🔨 Generating SQLC code..."
	@if ! command -v sqlc &> /dev/null; then \
		echo "Installing sqlc..."; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest; \
	fi
	cd internal/database && sqlc generate

oapi-codegen:
	@echo "🔨 Generating OpenAPI server code..."
	@if ! command -v oapi-codegen &> /dev/null; then \
		echo "Installing oapi-codegen..."; \
		go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest; \
	fi
	@mkdir -p internal/apiserver
	oapi-codegen -package apiserver -generate types -o internal/apiserver/types.gen.go api/openapi.yaml