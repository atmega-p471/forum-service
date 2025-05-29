.PHONY: proto build run test clean

# Generate proto files
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/forum/forum.proto

# Build the application
build:
	go build -o bin/forum-service

# Run the application
run:
	go run main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/ 