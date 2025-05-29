package client

import (
	"context"

	"github.com/forum/auth-service/proto/auth"
	"github.com/forum/forum-service/internal/domain"
	"google.golang.org/grpc"
)

// AuthClient is a client for the auth service
type AuthClient struct {
	client auth.AuthServiceClient
}

// NewAuthClient creates a new auth client
func NewAuthClient(conn *grpc.ClientConn) *AuthClient {
	return &AuthClient{
		client: auth.NewAuthServiceClient(conn),
	}
}

// ValidateToken validates a JWT token against the auth service
func (c *AuthClient) ValidateToken(token string) (*domain.User, error) {
	resp, err := c.client.ValidateToken(context.Background(), &auth.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       resp.User.Id,
		Username: resp.User.Username,
		Role:     resp.User.Role,
		IsBanned: resp.User.IsBanned,
	}, nil
}

// GetUser gets a user by ID from the auth service
func (c *AuthClient) GetUser(id int64) (*domain.User, error) {
	resp, err := c.client.GetUser(context.Background(), &auth.GetUserRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       resp.User.Id,
		Username: resp.User.Username,
		Role:     resp.User.Role,
		IsBanned: resp.User.IsBanned,
	}, nil
}
