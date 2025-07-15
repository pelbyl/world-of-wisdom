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

# Experiment Scenarios
scenario-morning-rush:
	@echo "🌅 Starting Morning Rush scenario..."
	@echo "Simulating legitimate traffic spike with normal and power users"
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale normal-user=0 --scale power-user=0 --scale script-kiddie=0 --scale sophisticated-attacker=0 --scale botnet-node=0
	@sleep 2
	@echo "Phase 1: 10 normal users connecting gradually..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale normal-user=$$i --no-recreate; \
		sleep 30; \
	done
	@echo "Phase 2: 5 power users joining..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale power-user=5 --no-recreate
	@echo "✅ Morning Rush scenario started!"
	@echo "📊 Monitor at http://localhost:3000"

scenario-script-kiddie:
	@echo "🐛 Starting Script Kiddie Attack scenario..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale normal-user=5 --scale power-user=0 --scale script-kiddie=0 --scale sophisticated-attacker=0 --scale botnet-node=0
	@sleep 120
	@echo "Attack starting: 1 script kiddie..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale script-kiddie=1 --no-recreate
	@echo "✅ Script Kiddie scenario started!"
	@echo "📊 Monitor at http://localhost:3000"

scenario-ddos:
	@echo "🚨 Starting Sophisticated DDoS scenario..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale normal-user=10 --scale power-user=2 --scale script-kiddie=0 --scale sophisticated-attacker=0 --scale botnet-node=0
	@sleep 180
	@echo "DDoS attack beginning: 3 sophisticated attackers..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale sophisticated-attacker=3 --no-recreate
	@echo "✅ DDoS scenario started!"
	@echo "📊 Monitor at http://localhost:3000"

scenario-botnet:
	@echo "🤖 Starting Botnet Simulation..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale normal-user=8 --scale power-user=0 --scale script-kiddie=0 --scale sophisticated-attacker=0 --scale botnet-node=0
	@sleep 120
	@echo "Botnet activating: 20 nodes..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale botnet-node=20 --no-recreate
	@echo "✅ Botnet scenario started!"
	@echo "📊 Monitor at http://localhost:3000"

scenario-mixed:
	@echo "🌐 Starting Mixed Reality scenario..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale normal-user=7 --scale power-user=2 --scale script-kiddie=0 --scale sophisticated-attacker=0 --scale botnet-node=0
	@echo "✅ Mixed scenario started with baseline traffic"
	@echo "📊 Monitor at http://localhost:3000"
	@echo "ℹ️  Use 'make scenario-add-attackers' to add various attack types"

scenario-add-attackers:
	@echo "Adding attackers to mixed scenario..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml up -d --scale script-kiddie=2 --scale sophisticated-attacker=1 --scale botnet-node=2 --no-recreate

scenario-stop:
	@echo "🛑 Stopping all scenario containers..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml stop normal-user power-user script-kiddie sophisticated-attacker botnet-node
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml rm -f normal-user power-user script-kiddie sophisticated-attacker botnet-node
	@echo "✅ All scenario clients stopped!"

scenario-status:
	@echo "📊 Scenario container status:"
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml ps

scenario-logs:
	@echo "📝 Showing scenario logs..."
	@docker-compose -f docker-compose.yml -f docker-compose.scenario.yml logs -f

# Monitor experiment
monitor:
	@echo "📊 Opening monitoring dashboard..."
	@echo "Dashboard available at http://localhost:3000"
	@echo ""
	@echo "The UI provides real-time monitoring of:"
	@echo "  - Client behaviors with per-IP difficulty"
	@echo "  - System metrics and performance"
	@echo "  - Attack detection and alerts"
	@echo "  - Success criteria evaluation"
	@echo ""
	@echo "For experiment analytics, navigate to the 'Experiment Analytics' tab"
	@open http://localhost:3000 || xdg-open http://localhost:3000 || echo "Please open http://localhost:3000 in your browser"

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