package clients

import (
	"context"
	"time"

	"github.com/forum/forum-service/internal/domain"
	auth_proto "github.com/forum/proto/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type authClient struct {
	client auth_proto.AuthServiceClient
	logger zerolog.Logger
}

// NewAuthClient creates a new auth client
func NewAuthClient(address string, logger zerolog.Logger) (*authClient, error) {
	// Set up a connection to the server
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error().Err(err).Str("address", address).Msg("Failed to connect to auth service")
		return nil, err
	}

	// Create a client
	client := auth_proto.NewAuthServiceClient(conn)

	return &authClient{
		client: client,
		logger: logger,
	}, nil
}

// ValidateToken validates a JWT token
func (c *authClient) ValidateToken(token string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make gRPC call
	resp, err := c.client.ValidateToken(ctx, &auth_proto.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to validate token")
		return nil, err
	}

	// Convert response to domain user
	user := &domain.User{
		ID:       resp.User.Id,
		Username: resp.User.Username,
		Role:     resp.User.Role,
		IsBanned: resp.User.IsBanned,
	}

	return user, nil
}

// GetUser gets a user by ID
func (c *authClient) GetUser(id int64) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make gRPC call
	resp, err := c.client.GetUser(ctx, &auth_proto.GetUserRequest{
		Id: id,
	})
	if err != nil {
		c.logger.Error().Err(err).Int64("id", id).Msg("Failed to get user")
		return nil, err
	}

	// Convert response to domain user
	user := &domain.User{
		ID:       resp.User.Id,
		Username: resp.User.Username,
		Role:     resp.User.Role,
		IsBanned: resp.User.IsBanned,
	}

	return user, nil
}
