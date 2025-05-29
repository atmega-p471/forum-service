#!/bin/bash

# Create proto directory if it doesn't exist
mkdir -p proto/forum

# Generate Go files from proto definitions
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/forum/forum.proto 