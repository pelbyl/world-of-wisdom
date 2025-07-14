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

# Simple demo commands
demo:
	@echo "🚀 Starting demo containers..."
	@curl -s -X POST http://localhost:8081/api/v1/demo/scenario \
		-H "Content-Type: application/json" \
		-d '{"scenario": "normal"}' | jq -r 'if .status == "success" then "✅ Started demo containers" else "❌ Error: \(.message // "Failed to start")" end'
	@echo "📊 Monitor at http://localhost:3000"
	@echo "📝 Check logs: docker logs world-of-wisdom-server-1 -f"

demo-stop:
	@echo "🛑 Stopping demo containers..."
	@curl -s -X POST http://localhost:8081/api/v1/demo/stop | jq -r 'if .status == "success" then "✅ Stopped" else "❌ Error: \(.message // "Failed to stop")" end'

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