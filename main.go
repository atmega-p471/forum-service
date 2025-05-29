package main

import (
	"database/sql"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/atmega-p471/forum-service/internal/config"
	"github.com/atmega-p471/forum-service/internal/delivery/grpc"
	"github.com/atmega-p471/forum-service/internal/delivery/grpc/client"
	httpHandler "github.com/atmega-p471/forum-service/internal/delivery/http"
	"github.com/atmega-p471/forum-service/internal/delivery/ws"
	"github.com/atmega-p471/forum-service/internal/repository"
	"github.com/atmega-p471/forum-service/internal/usecase"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Swagger docs
	_ "github.com/atmega-p471/forum-service/docs"
)

// @title Forum Service API
// @version 1.0
// @description Forum service for forum application
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8082
// @BasePath /api/v1
func main() {
	// Initialize logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger.Info().Msg("Starting forum service")

	// Load configuration
	cfg := config.NewConfig()

	// Connect to SQLite database
	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Check database connection
	if err := db.Ping(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to ping database")
	}

	// Initialize database schema
	if err := repository.InitSchema(db); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database schema")
	}

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Initialize repositories
	repo := repository.NewRepository(db)

	// Initialize auth client
	authConn, err := grpclib.Dial(cfg.AuthServiceAddr, grpclib.WithInsecure())
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to auth service")
	}
	defer authConn.Close()
	authClient := client.NewAuthClient(authConn)

	// Initialize use cases
	messageUsecase := usecase.NewUseCase(repo, authClient, hub, cfg)

	// Initialize gRPC server
	grpcServer := grpclib.NewServer()
	forumServer := grpc.NewForumServer(messageUsecase, logger)
	forumServer.Register(grpcServer)
	reflection.Register(grpcServer)

	go func() {
		lis, err := net.Listen("tcp", ":9092")
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to listen for gRPC")
		}
		logger.Info().Msg("gRPC server is running on :9092")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal().Err(err).Msg("Failed to serve gRPC")
		}
	}()

	// Initialize HTTP server
	router := http.NewServeMux()

	// Swagger
	router.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8082/swagger/doc.json"), //The url pointing to API definition
	))

	// Initialize HTTP handler
	handler := httpHandler.NewHandler(messageUsecase, hub, authClient)
	handler.RegisterRoutes(router)

	// Start HTTP server
	go func() {
		logger.Info().Msg("HTTP server is running on :8082")
		if err := http.ListenAndServe(":8082", router); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down servers...")

	// Stop gRPC server
	grpcServer.GracefulStop()

	logger.Info().Msg("Server exited properly")
}
