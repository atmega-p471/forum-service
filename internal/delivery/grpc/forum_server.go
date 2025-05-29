package grpc

import (
	"context"
	"time"

	"github.com/atmega-p471/forum-service/internal/domain"
	"github.com/atmega-p471/forum-service/proto/forum"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ForumServer struct {
	forum.UnimplementedForumServiceServer
	messageUsecase domain.MessageUseCase
	logger         zerolog.Logger
}

// NewForumServer creates a new forum gRPC server
func NewForumServer(messageUsecase domain.MessageUseCase, logger zerolog.Logger) *ForumServer {
	return &ForumServer{
		messageUsecase: messageUsecase,
		logger:         logger,
	}
}

// Register registers the server with the gRPC server
func (s *ForumServer) Register(server *grpc.Server) {
	forum.RegisterForumServiceServer(server, s)
}

// GetMessages gets messages from the general chat
func (s *ForumServer) GetMessages(ctx context.Context, req *forum.GetMessagesRequest) (*forum.GetMessagesResponse, error) {
	messages, total, err := s.messageUsecase.GetMessages(req.Limit, req.Offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get messages")
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &forum.GetMessagesResponse{
		Messages: make([]*forum.Message, 0, len(messages)),
		Total:    total,
	}

	for _, message := range messages {
		if !message.IsBanned {
			response.Messages = append(response.Messages, &forum.Message{
				Id:        message.ID,
				UserId:    message.UserID,
				Username:  message.Username,
				Content:   message.Content,
				CreatedAt: message.CreatedAt.Format(time.RFC3339),
				IsBanned:  message.IsBanned,
			})
		}
	}

	return response, nil
}

// CreateMessage creates a new message
func (s *ForumServer) CreateMessage(ctx context.Context, req *forum.CreateMessageRequest) (*forum.CreateMessageResponse, error) {
	message, err := s.messageUsecase.CreateMessage(req.UserId, req.Username, req.Content)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create message")
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &forum.CreateMessageResponse{
		Message: &forum.Message{
			Id:        message.ID,
			UserId:    message.UserID,
			Username:  message.Username,
			Content:   message.Content,
			CreatedAt: message.CreatedAt.Format(time.RFC3339),
			IsBanned:  message.IsBanned,
		},
	}, nil
}

// BanMessage bans a message by ID
func (s *ForumServer) BanMessage(ctx context.Context, req *forum.BanMessageRequest) (*forum.BanMessageResponse, error) {
	if err := s.messageUsecase.BanMessage(req.Id); err != nil {
		s.logger.Error().Err(err).Int64("id", req.Id).Msg("Failed to ban message")
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &forum.BanMessageResponse{
		Success: true,
	}, nil
}

// UnbanMessage unbans a message by ID
func (s *ForumServer) UnbanMessage(ctx context.Context, req *forum.UnbanMessageRequest) (*forum.UnbanMessageResponse, error) {
	if err := s.messageUsecase.UnbanMessage(req.Id); err != nil {
		s.logger.Error().Err(err).Int64("id", req.Id).Msg("Failed to unban message")
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &forum.UnbanMessageResponse{
		Success: true,
	}, nil
}
