package main

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/forum/forum-service/internal/config"
	grpcClient "github.com/forum/forum-service/internal/delivery/grpc/client"
	"github.com/forum/forum-service/internal/delivery/grpc/server"
	httpHandler "github.com/forum/forum-service/internal/delivery/http"
	wsHandler "github.com/forum/forum-service/internal/delivery/ws"
	"github.com/forum/forum-service/internal/repository"
	"github.com/forum/forum-service/internal/usecase"
	"github.com/forum/forum-service/proto/forum"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// @title Forum Service API
// @version 1.0
// @description Forum service for forum application
// @host localhost:8082
// @BasePath /api/v1
func main() {
	// Initialize logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// Load config
	cfg := config.NewConfig()

	// Connect to Auth service
	authConn, err := grpc.Dial(cfg.AuthServiceAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Auth service")
	}
	defer authConn.Close()

	authClient := grpcClient.NewAuthClient(authConn)

	// Create repository layer
	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize database schema
	if err := repository.InitSchema(db); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database schema")
	}

	// Create WebSocket hub
	hub := wsHandler.NewHub()

	// Create usecase layer
	messageRepo := repository.NewMessageRepository(db)
	messageUseCase := usecase.NewMessageUseCase(messageRepo, authClient, hub)

	// Start expired comments cleanup scheduler
	if uc, ok := messageUseCase.(*usecase.MessageUseCase); ok {
		uc.StartCleanupScheduler()
	}

	// Start WebSocket hub
	go hub.Run()

	// Create delivery layer - HTTP and WebSocket with AUTHENTICATION
	handler := httpHandler.NewHandler(messageUseCase, hub, authClient)

	// Create HTTP server
	router := http.NewServeMux()
	handler.RegisterRoutes(router)

	// Serve Swagger UI
	router.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// --- CORS middleware ---
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpHandler.CORSMiddleware(router),
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	forumServer := server.NewForumServer(messageUseCase, log.Logger)
	forum.RegisterForumServiceServer(grpcServer, forumServer)
	reflection.Register(grpcServer)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen for gRPC")
	}

	go func() {
		log.Info().Str("address", cfg.GRPCAddr).Msg("Starting gRPC server")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()

	// Start HTTP server
	go func() {
		log.Info().Str("address", cfg.HTTPAddr).Msg("Starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutting down servers...")

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to shutdown HTTP server gracefully")
	}

	log.Info().Msg("Servers stopped")
}
