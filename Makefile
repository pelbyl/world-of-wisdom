.PHONY: build test run-server run-client docker-build docker-run clean

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go

test:
	go test -v ./...

run-server:
	go run cmd/server/main.go

run-client:
	go run cmd/client/main.go

docker-build:
	docker build -f Dockerfile.server -t world-of-wisdom-server .
	docker build -f Dockerfile.client -t world-of-wisdom-client .

docker-run:
	docker-compose up

docker-test:
	docker-compose up --abort-on-container-exit

clean:
	rm -rf bin/
	docker-compose down
	docker rmi world-of-wisdom-server world-of-wisdom-client || true