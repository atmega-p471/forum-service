module github.com/atmega-p471/forum-service

go 1.21

require (
	github.com/atmega-p471/forum-proto v0.1.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/rs/zerolog v1.34.0
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.8.10
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.36.5
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.8 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/swaggo/files v1.0.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Временно используем локальный replace для разработки
// После того как GitHub обновит индексы, можно будет убрать эту строку
replace github.com/atmega-p471/forum-proto => ../proto
