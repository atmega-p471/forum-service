# Forum Service

Forum messaging microservice with real-time WebSocket support.

## Features

- 💬 Message creation and management
- 🔄 Real-time messaging via WebSocket
- 🔐 JWT-based authentication integration
- 📊 SQLite database storage
- 🌐 HTTP REST API
- 🔗 gRPC API for internal communication
- 📋 Swagger documentation
- ⚡ Message expiration and cleanup

## API Endpoints

### HTTP REST API (Port 8082)

#### Messages
- `GET /messages` - Get all messages
- `POST /messages` - Create new message (requires authentication)
- `GET /messages/{id}` - Get message by ID
- `PUT /messages/{id}` - Update message (requires authentication)
- `DELETE /messages/{id}` - Delete message (requires authentication)

#### WebSocket
- `GET /ws` - WebSocket connection for real-time messaging

### gRPC API

- `CreateMessage` - Create new message
- `GetMessages` - Retrieve messages
- `UpdateMessage` - Update existing message
- `DeleteMessage` - Delete message

## Quick Start

### Prerequisites

- Go 1.21+
- SQLite3
- Running auth-service (for authentication)

### Installation

```bash
# Clone repository
git clone https://github.com/YOUR_USERNAME/forum-service.git
cd forum-service

# Install dependencies
go mod tidy

# Create data directory
mkdir -p data

# Start service
go run cmd/main.go
```

### Configuration

Environment variables:
- `PORT` - HTTP server port (default: 8082)
- `GRPC_PORT` - gRPC server port (default: 9082)
- `DB_PATH` - SQLite database path (default: data/forum.db)
- `AUTH_SERVICE_GRPC` - Auth service gRPC address (default: localhost:9081)

## Database Schema

```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    author_name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME
);
```

## Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test ./tests/...
```

### Manual Testing

Test message creation (requires valid JWT token):
```bash
# Get token from auth-service first
TOKEN="your-jwt-token"

curl -X POST http://localhost:8082/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":"Hello World!"}'
```

Test WebSocket connection:
```bash
# Install websocat: https://github.com/vi/websocat
websocat ws://localhost:8082/ws
```

## Real-time Features

### WebSocket Connection

Connect to `ws://localhost:8082/ws` to receive real-time message updates:

```javascript
const ws = new WebSocket('ws://localhost:8082/ws');

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('New message:', message);
};
```

### Message Broadcasting

When a new message is created via HTTP API, it's automatically broadcast to all connected WebSocket clients.

## Architecture

```
forum-service/
├── cmd/
│   └── main.go              # HTTP + gRPC + WebSocket server
├── internal/
│   ├── clients/             # External service clients
│   ├── config/              # Configuration
│   ├── delivery/
│   │   ├── http/            # HTTP handlers
│   │   ├── grpc/            # gRPC server
│   │   └── ws/              # WebSocket hub and clients
│   ├── domain/              # Business entities
│   ├── repository/          # Data access layer
│   └── usecase/             # Business logic
├── tools/                   # Utility tools
├── tests/                   # Integration tests
└── proto/                   # gRPC definitions
```

## Integration with Auth Service

This service integrates with the auth-service for user authentication:

1. **Token Validation**: All authenticated endpoints validate JWT tokens via gRPC call to auth-service
2. **User Information**: Retrieves user details (username, role) from auth-service
3. **Authorization**: Checks user permissions for message operations

## API Documentation

When service is running, Swagger documentation is available at:
- http://localhost:8082/swagger/

## Dependencies

- **github.com/gorilla/mux** - HTTP router
- **github.com/gorilla/websocket** - WebSocket implementation
- **github.com/mattn/go-sqlite3** - SQLite driver
- **google.golang.org/grpc** - gRPC framework
- **github.com/rs/zerolog** - Structured logging
- **github.com/swaggo/swag** - Swagger generation

## Tools

### Database Check Tool
```bash
cd tools/check-db && go run main.go
```

### Message Check Tool
```bash
cd tools/check-messages && go run main.go
```

## License

MIT 