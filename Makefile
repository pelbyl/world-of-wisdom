.PHONY: re-run clean-all

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
	@echo "   WebSocket API: http://localhost:8081"

clean-all:
	@echo "🧹 Starting full project cleanup..."
	@echo "Removing local data directories..."
	rm -rf bin/ logs/ /tmp/wisdom-data/ || true
	rm -rf web/node_modules/ web/dist/ web/.next/ || true
	@echo "Clearing Go build cache..."
	go clean -cache -modcache -i -r || true
	@echo "✅ Full cleanup complete!"