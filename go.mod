module github.com/atmega-p471/forum-service

go 1.23.0

toolchain go1.24.2

require (
	github.com/atmega-p471/forum-auth-service v0.0.0-20250529135858-15be6351fc4d
	github.com/atmega-p471/forum-proto v0.1.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/rs/zerolog v1.34.0
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.16.4
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.36.5
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/swaggo/files v1.0.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Временно используем локальный replace для разработки
// После того как GitHub обновит индексы, можно будет убрать эту строку
replace github.com/atmega-p471/forum-proto => ../proto
