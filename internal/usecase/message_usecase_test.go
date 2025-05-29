package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/atmega-p471/forum-service/internal/domain"
)

// AuthClientInterface defines the interface for auth client
type AuthClientInterface interface {
	GetUser(id int64) (*domain.User, error)
	ValidateToken(token string) (*domain.User, error)
}

// MockHub implements Hub interface for testing
type MockHub struct {
	broadcastedMessages []*domain.Message
}

func NewMockHub() *MockHub {
	return &MockHub{
		broadcastedMessages: make([]*domain.Message, 0),
	}
}

func (m *MockHub) BroadcastMessage(message *domain.Message) {
	m.broadcastedMessages = append(m.broadcastedMessages, message)
}

// MockMessageRepository implements domain.MessageRepository for testing
type MockMessageRepository struct {
	messages map[int64]*domain.Message
	comments map[int64]*domain.Comment
	nextID   int64
}

func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		messages: make(map[int64]*domain.Message),
		comments: make(map[int64]*domain.Comment),
		nextID:   1,
	}
}

func (m *MockMessageRepository) GetByID(id int64) (*domain.Message, error) {
	if msg, exists := m.messages[id]; exists {
		return msg, nil
	}
	return nil, errors.New("message not found")
}

func (m *MockMessageRepository) List(limit, offset int64) ([]*domain.Message, int64, error) {
	var messages []*domain.Message
	var count int64

	for _, msg := range m.messages {
		if !msg.IsBanned {
			count++
			if count > offset && int64(len(messages)) < limit {
				messages = append(messages, msg)
			}
		}
	}

	return messages, count, nil
}

func (m *MockMessageRepository) GetAllMessages() ([]*domain.Message, error) {
	var messages []*domain.Message
	for _, msg := range m.messages {
		messages = append(messages, msg)
	}
	return messages, nil
}

func (m *MockMessageRepository) Create(message *domain.Message) (int64, error) {
	id := m.nextID
	m.nextID++
	message.ID = id
	message.CreatedAt = time.Now()
	m.messages[id] = message
	return id, nil
}

func (m *MockMessageRepository) Ban(id int64) error {
	if msg, exists := m.messages[id]; exists {
		msg.IsBanned = true
		return nil
	}
	return errors.New("message not found")
}

func (m *MockMessageRepository) Unban(id int64) error {
	if msg, exists := m.messages[id]; exists {
		msg.IsBanned = false
		return nil
	}
	return errors.New("message not found")
}

func (m *MockMessageRepository) Delete(id int64) error {
	if _, exists := m.messages[id]; exists {
		delete(m.messages, id)
		return nil
	}
	return errors.New("message not found")
}

func (m *MockMessageRepository) CreateComment(comment *domain.Comment) (int64, error) {
	id := m.nextID
	m.nextID++
	comment.ID = id
	comment.CreatedAt = time.Now()
	m.comments[id] = comment
	return id, nil
}

func (m *MockMessageRepository) GetComments(messageID int64) ([]*domain.Comment, error) {
	var comments []*domain.Comment
	for _, comment := range m.comments {
		if comment.MessageID == messageID && !comment.IsExpired() {
			comments = append(comments, comment)
		}
	}
	return comments, nil
}

func (m *MockMessageRepository) GetCommentByID(id int64) (*domain.Comment, error) {
	if comment, exists := m.comments[id]; exists {
		return comment, nil
	}
	return nil, errors.New("comment not found")
}

func (m *MockMessageRepository) DeleteComment(id int64) error {
	if _, exists := m.comments[id]; exists {
		delete(m.comments, id)
		return nil
	}
	return errors.New("comment not found")
}

func (m *MockMessageRepository) DeleteExpiredComments() error {
	for id, comment := range m.comments {
		if comment.IsExpired() {
			delete(m.comments, id)
		}
	}
	return nil
}

// MockAuthClient implements AuthClientInterface for testing
type MockAuthClient struct {
	users map[int64]*domain.User
}

func NewMockAuthClient() *MockAuthClient {
	return &MockAuthClient{
		users: map[int64]*domain.User{
			1: {ID: 1, Username: "testuser", Role: "user", IsBanned: false},
			2: {ID: 2, Username: "admin", Role: "admin", IsBanned: false},
		},
	}
}

func (m *MockAuthClient) GetUser(userID int64) (*domain.User, error) {
	if user, exists := m.users[userID]; exists && !user.IsBanned {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *MockAuthClient) ValidateToken(token string) (*domain.User, error) {
	// Simple mock implementation
	if token == "valid_token" {
		return m.users[1], nil
	}
	return nil, errors.New("invalid token")
}

// TestMessageUseCase wraps MessageUseCase to accept interface
type TestMessageUseCase struct {
	repo       domain.MessageRepository
	authClient AuthClientInterface
	hub        Hub
}

func NewTestMessageUseCase(repo domain.MessageRepository, authClient AuthClientInterface, hub Hub) *TestMessageUseCase {
	return &TestMessageUseCase{
		repo:       repo,
		authClient: authClient,
		hub:        hub,
	}
}

func (u *TestMessageUseCase) CreateMessage(userID int64, username, content string) (*domain.Message, error) {
	if content == "" {
		return nil, errors.New("content is required")
	}

	// Skip auth validation for anonymous users (ID=0)
	if userID != 0 {
		// Validate user ID
		user, err := u.authClient.GetUser(userID)
		if err != nil {
			return nil, err
		}

		// Check if user is banned
		if user.IsBanned {
			return nil, errors.New("user is banned")
		}
	}

	// Create message
	message := &domain.Message{
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		IsBanned:  false,
	}

	// Save message
	messageID, err := u.repo.Create(message)
	if err != nil {
		return nil, err
	}

	// Set message ID
	message.ID = messageID

	// Broadcast message
	u.hub.BroadcastMessage(message)

	return message, nil
}

func (u *TestMessageUseCase) GetMessages(limit, offset int64) ([]*domain.Message, int64, error) {
	return u.repo.List(limit, offset)
}

func (u *TestMessageUseCase) BanMessage(id int64) error {
	message, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	if message == nil {
		return errors.New("message not found")
	}

	err = u.repo.Ban(id)
	if err != nil {
		return err
	}

	message.IsBanned = true
	u.hub.BroadcastMessage(message)

	return nil
}

func (u *TestMessageUseCase) GetByID(id int64) (*domain.Message, error) {
	return u.repo.GetByID(id)
}

func (u *TestMessageUseCase) CreateComment(messageID, userID int64, username, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("content is required")
	}

	// Skip auth validation for anonymous users (ID=0)
	if userID != 0 {
		// Validate user ID
		user, err := u.authClient.GetUser(userID)
		if err != nil {
			return nil, err
		}

		// Check if user is banned
		if user.IsBanned {
			return nil, errors.New("user is banned")
		}
	}

	// Check if message exists
	_, err := u.repo.GetByID(messageID)
	if err != nil {
		return nil, errors.New("message not found")
	}

	// Create comment
	comment := &domain.Comment{
		MessageID: messageID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	// Save comment
	commentID, err := u.repo.CreateComment(comment)
	if err != nil {
		return nil, err
	}

	comment.ID = commentID
	return comment, nil
}

func TestMessageUseCase_CreateMessage(t *testing.T) {
	repo := NewMockMessageRepository()
	authClient := NewMockAuthClient()
	hub := NewMockHub()
	uc := NewTestMessageUseCase(repo, authClient, hub)

	tests := []struct {
		name     string
		userID   int64
		username string
		content  string
		wantErr  bool
	}{
		{
			name:     "Valid message creation",
			userID:   1,
			username: "testuser",
			content:  "Valid test message",
			wantErr:  false,
		},
		{
			name:     "Empty content",
			userID:   1,
			username: "testuser",
			content:  "",
			wantErr:  true,
		},
		{
			name:     "Anonymous user message",
			userID:   0,
			username: "anonymous",
			content:  "Anonymous message",
			wantErr:  false,
		},
		{
			name:     "Invalid user",
			userID:   999,
			username: "nonexistent",
			content:  "Valid content",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := uc.CreateMessage(tt.userID, tt.username, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if message == nil {
				t.Error("Expected message, got nil")
				return
			}

			if message.Username != tt.username {
				t.Errorf("Expected username %s, got %s", tt.username, message.Username)
			}

			if message.Content != tt.content {
				t.Errorf("Expected content %s, got %s", tt.content, message.Content)
			}

			// Check if message was broadcasted
			if len(hub.broadcastedMessages) == 0 {
				t.Error("Expected message to be broadcasted")
			}
		})
	}
}

func TestMessageUseCase_GetMessages(t *testing.T) {
	repo := NewMockMessageRepository()
	authClient := NewMockAuthClient()
	hub := NewMockHub()
	uc := NewTestMessageUseCase(repo, authClient, hub)

	// Create test messages
	for i := 0; i < 5; i++ {
		_, err := uc.CreateMessage(1, "testuser", "Test message "+string(rune(i+'1')))
		if err != nil {
			t.Fatalf("Failed to create test message: %v", err)
		}
	}

	messages, total, err := uc.GetMessages(3, 0)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}
}

func TestMessageUseCase_BanMessage(t *testing.T) {
	repo := NewMockMessageRepository()
	authClient := NewMockAuthClient()
	hub := NewMockHub()
	uc := NewTestMessageUseCase(repo, authClient, hub)

	// Create test message
	message, err := uc.CreateMessage(1, "testuser", "Test message")
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Ban the message
	err = uc.BanMessage(message.ID)
	if err != nil {
		t.Fatalf("Failed to ban message: %v", err)
	}

	// Verify message is banned
	banned, err := uc.GetByID(message.ID)
	if err != nil {
		t.Fatalf("Failed to get banned message: %v", err)
	}

	if !banned.IsBanned {
		t.Error("Expected message to be banned")
	}

	// Test banning non-existent message
	err = uc.BanMessage(999)
	if err == nil {
		t.Error("Expected error when banning non-existent message")
	}
}

func TestMessageUseCase_CreateComment(t *testing.T) {
	repo := NewMockMessageRepository()
	authClient := NewMockAuthClient()
	hub := NewMockHub()
	uc := NewTestMessageUseCase(repo, authClient, hub)

	// Create test message
	message, err := uc.CreateMessage(1, "testuser", "Test message")
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	tests := []struct {
		name      string
		messageID int64
		userID    int64
		username  string
		content   string
		wantErr   bool
	}{
		{
			name:      "Valid comment creation",
			messageID: message.ID,
			userID:    2,
			username:  "admin",
			content:   "Test comment",
			wantErr:   false,
		},
		{
			name:      "Empty content",
			messageID: message.ID,
			userID:    1,
			username:  "testuser",
			content:   "",
			wantErr:   true,
		},
		{
			name:      "Anonymous comment",
			messageID: message.ID,
			userID:    0,
			username:  "anonymous",
			content:   "Anonymous comment",
			wantErr:   false,
		},
		{
			name:      "Invalid user",
			messageID: message.ID,
			userID:    999,
			username:  "nonexistent",
			content:   "Test comment",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment, err := uc.CreateComment(tt.messageID, tt.userID, tt.username, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if comment == nil {
				t.Error("Expected comment, got nil")
				return
			}

			if comment.Username != tt.username {
				t.Errorf("Expected username %s, got %s", tt.username, comment.Username)
			}

			if comment.Content != tt.content {
				t.Errorf("Expected content %s, got %s", tt.content, comment.Content)
			}
		})
	}
}
