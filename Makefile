.PHONY: build test run-server run-client run-webserver run-web dev docker-build docker-run clean

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