package server

import (
	"context"
	"time"

	"github.com/forum/forum-service/internal/domain"
	"github.com/forum/forum-service/proto/forum"
	"github.com/rs/zerolog"
)

type ForumServer struct {
	forum.UnimplementedForumServiceServer
	uc     domain.MessageUseCase
	logger zerolog.Logger
}

func NewForumServer(uc domain.MessageUseCase, logger zerolog.Logger) *ForumServer {
	return &ForumServer{
		uc:     uc,
		logger: logger,
	}
}

func (s *ForumServer) GetMessages(ctx context.Context, req *forum.GetMessagesRequest) (*forum.GetMessagesResponse, error) {
	messages, total, err := s.uc.GetMessages(req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	var protoMessages []*forum.Message
	for _, msg := range messages {
		protoMessages = append(protoMessages, &forum.Message{
			Id:        msg.ID,
			UserId:    msg.UserID,
			Username:  msg.Username,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
			IsBanned:  msg.IsBanned,
		})
	}

	return &forum.GetMessagesResponse{
		Messages: protoMessages,
		Total:    total,
	}, nil
}

func (s *ForumServer) CreateMessage(ctx context.Context, req *forum.CreateMessageRequest) (*forum.CreateMessageResponse, error) {
	msg, err := s.uc.CreateMessage(req.UserId, req.Username, req.Content)
	if err != nil {
		return nil, err
	}

	return &forum.CreateMessageResponse{
		Message: &forum.Message{
			Id:        msg.ID,
			UserId:    msg.UserID,
			Username:  msg.Username,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
			IsBanned:  msg.IsBanned,
		},
	}, nil
}

func (s *ForumServer) BanMessage(ctx context.Context, req *forum.BanMessageRequest) (*forum.BanMessageResponse, error) {
	err := s.uc.BanMessage(req.Id)
	if err != nil {
		return nil, err
	}
	return &forum.BanMessageResponse{Success: true}, nil
}

func (s *ForumServer) UnbanMessage(ctx context.Context, req *forum.UnbanMessageRequest) (*forum.UnbanMessageResponse, error) {
	err := s.uc.UnbanMessage(req.Id)
	if err != nil {
		return nil, err
	}
	return &forum.UnbanMessageResponse{Success: true}, nil
}
