package tests

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/forum/forum-service/internal/domain"
	"github.com/forum/forum-service/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

// TestDatabase sets up a test database
func setupTestDatabase(t *testing.T) *sql.DB {
	// Create temporary database file
	tempDB := fmt.Sprintf("test_%d.db", time.Now().UnixNano())

	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Initialize schema
	err = repository.InitSchema(db)
	if err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	// Clean up database file when test completes
	t.Cleanup(func() {
		db.Close()
		os.Remove(tempDB)
	})

	return db
}

// MockAuthClient for integration tests
type MockAuthClient struct {
	users map[int64]User
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	IsBanned bool   `json:"is_banned"`
}

func NewMockAuthClient() *MockAuthClient {
	return &MockAuthClient{
		users: map[int64]User{
			1: {ID: 1, Username: "testuser", Role: "user", IsBanned: false},
			2: {ID: 2, Username: "admin", Role: "admin", IsBanned: false},
			3: {ID: 3, Username: "banned", Role: "user", IsBanned: true},
		},
	}
}

func (m *MockAuthClient) GetUser(userID int64) (*domain.User, error) {
	if user, exists := m.users[userID]; exists && !user.IsBanned {
		return &domain.User{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			IsBanned: user.IsBanned,
		}, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockAuthClient) ValidateToken(token string) (*domain.User, error) {
	switch token {
	case "valid_user_token":
		user := m.users[1]
		return &domain.User{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			IsBanned: user.IsBanned,
		}, nil
	case "valid_admin_token":
		user := m.users[2]
		return &domain.User{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			IsBanned: user.IsBanned,
		}, nil
	default:
		return nil, fmt.Errorf("invalid token")
	}
}

// MockHub for integration tests
type MockHub struct {
	messages []*domain.Message
}

func NewMockHub() *MockHub {
	return &MockHub{
		messages: make([]*domain.Message, 0),
	}
}

func (h *MockHub) BroadcastMessage(message *domain.Message) {
	h.messages = append(h.messages, message)
}

// TestMessageUseCase wraps MessageUseCase to accept interface
type TestMessageUseCase struct {
	repo       domain.MessageRepository
	authClient AuthClientInterface
	hub        Hub
}

// AuthClientInterface defines the interface for auth client
type AuthClientInterface interface {
	GetUser(id int64) (*domain.User, error)
	ValidateToken(token string) (*domain.User, error)
}

// Hub defines a minimal interface for the WebSocket hub
type Hub interface {
	BroadcastMessage(*domain.Message)
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
		return nil, fmt.Errorf("content is required")
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
			return nil, fmt.Errorf("user is banned")
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
		return fmt.Errorf("message not found")
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
		return nil, fmt.Errorf("content is required")
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
			return nil, fmt.Errorf("user is banned")
		}
	}

	// Check if message exists
	_, err := u.repo.GetByID(messageID)
	if err != nil {
		return nil, fmt.Errorf("message not found")
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

func (u *TestMessageUseCase) GetComments(messageID int64) ([]*domain.Comment, error) {
	return u.repo.GetComments(messageID)
}

func (u *TestMessageUseCase) DeleteComment(id int64) error {
	return u.repo.DeleteComment(id)
}

func (u *TestMessageUseCase) GetAllMessages() ([]*domain.Message, error) {
	return u.repo.GetAllMessages()
}

func (u *TestMessageUseCase) UnbanMessage(id int64) error {
	return u.repo.Unban(id)
}

func (u *TestMessageUseCase) DeleteMessage(id int64) error {
	return u.repo.Delete(id)
}

func TestIntegration_MessageFlow(t *testing.T) {
	// Setup test components
	db := setupTestDatabase(t)
	repo := repository.NewRepository(db)
	authClient := NewMockAuthClient()
	hub := NewMockHub()

	// Create usecase with proper interface
	messageUseCase := NewTestMessageUseCase(repo.Message, authClient, hub)

	// Test creating a message through usecase
	message, err := messageUseCase.CreateMessage(1, "testuser", "Integration test message")
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	if message.ID == 0 {
		t.Error("Message ID should be set")
	}

	if message.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", message.Username)
	}

	// Test retrieving messages
	messages, total, err := messageUseCase.GetMessages(10, 0)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Test creating a comment
	comment, err := messageUseCase.CreateComment(message.ID, 2, "admin", "Test comment")
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}

	if comment.MessageID != message.ID {
		t.Errorf("Expected comment message ID %d, got %d", message.ID, comment.MessageID)
	}

	// Test getting comments
	comments, err := messageUseCase.GetComments(message.ID)
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}

	// Test banning message
	err = messageUseCase.BanMessage(message.ID)
	if err != nil {
		t.Fatalf("Failed to ban message: %v", err)
	}

	// Verify message is banned
	bannedMessage, err := messageUseCase.GetByID(message.ID)
	if err != nil {
		t.Fatalf("Failed to get banned message: %v", err)
	}

	if !bannedMessage.IsBanned {
		t.Error("Expected message to be banned")
	}

	// Test that banned messages don't appear in list
	messages, total, err = messageUseCase.GetMessages(10, 0)
	if err != nil {
		t.Fatalf("Failed to get messages after ban: %v", err)
	}

	if total != 0 {
		t.Errorf("Expected total 0 after ban, got %d", total)
	}

	// Test deleting comment
	err = messageUseCase.DeleteComment(comment.ID)
	if err != nil {
		t.Fatalf("Failed to delete comment: %v", err)
	}

	// Verify comment is deleted
	comments, err = messageUseCase.GetComments(message.ID)
	if err != nil {
		t.Fatalf("Failed to get comments after deletion: %v", err)
	}

	if len(comments) != 0 {
		t.Errorf("Expected 0 comments after deletion, got %d", len(comments))
	}
}

func TestIntegration_DatabasePersistence(t *testing.T) {
	// Setup test database
	db := setupTestDatabase(t)
	repo := repository.NewRepository(db)

	// Test message persistence
	testMessage := &domain.Message{
		UserID:    1,
		Username:  "testuser",
		Content:   "Persistence test message",
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	// Create message
	messageID, err := repo.Message.Create(testMessage)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Retrieve message
	retrievedMessage, err := repo.Message.GetByID(messageID)
	if err != nil {
		t.Fatalf("Failed to retrieve message: %v", err)
	}

	// Verify data
	if retrievedMessage.Username != testMessage.Username {
		t.Errorf("Expected username '%s', got '%s'", testMessage.Username, retrievedMessage.Username)
	}

	if retrievedMessage.Content != testMessage.Content {
		t.Errorf("Expected content '%s', got '%s'", testMessage.Content, retrievedMessage.Content)
	}

	// Test comment persistence
	testComment := &domain.Comment{
		MessageID: messageID,
		UserID:    2,
		Username:  "commenter",
		Content:   "Test comment",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Create comment
	commentID, err := repo.Message.CreateComment(testComment)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}

	// Retrieve comment
	retrievedComment, err := repo.Message.GetCommentByID(commentID)
	if err != nil {
		t.Fatalf("Failed to retrieve comment: %v", err)
	}

	// Verify comment data
	if retrievedComment.Content != testComment.Content {
		t.Errorf("Expected comment content '%s', got '%s'", testComment.Content, retrievedComment.Content)
	}

	if retrievedComment.MessageID != messageID {
		t.Errorf("Expected comment message ID %d, got %d", messageID, retrievedComment.MessageID)
	}

	// Test foreign key constraint (comments should be deleted when message is deleted)
	err = repo.Message.Delete(messageID)
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}

	// Verify comment is also deleted due to foreign key constraint
	_, err = repo.Message.GetCommentByID(commentID)
	if err == nil {
		t.Error("Expected comment to be deleted when message is deleted")
	}
}

func TestIntegration_AuthValidation(t *testing.T) {
	// Setup test components
	db := setupTestDatabase(t)
	repo := repository.NewRepository(db)
	authClient := NewMockAuthClient()
	hub := NewMockHub()

	messageUseCase := NewTestMessageUseCase(repo.Message, authClient, hub)

	// Test with valid user
	message, err := messageUseCase.CreateMessage(1, "testuser", "Valid user message")
	if err != nil {
		t.Fatalf("Failed to create message with valid user: %v", err)
	}

	if message == nil {
		t.Error("Expected message to be created")
	}

	// Test with banned user (should fail)
	_, err = messageUseCase.CreateMessage(3, "banned", "Banned user message")
	if err == nil {
		t.Error("Expected error when creating message with banned user")
	}

	// Test with non-existent user (should fail)
	_, err = messageUseCase.CreateMessage(999, "nonexistent", "Non-existent user message")
	if err == nil {
		t.Error("Expected error when creating message with non-existent user")
	}

	// Test anonymous user (should succeed)
	anonymousMessage, err := messageUseCase.CreateMessage(0, "anonymous", "Anonymous message")
	if err != nil {
		t.Fatalf("Failed to create anonymous message: %v", err)
	}

	if anonymousMessage == nil {
		t.Error("Expected anonymous message to be created")
	}
}

func TestIntegration_CommentExpiration(t *testing.T) {
	// Setup test components
	db := setupTestDatabase(t)
	repo := repository.NewRepository(db)

	// Create test message
	testMessage := &domain.Message{
		UserID:    1,
		Username:  "testuser",
		Content:   "Test message",
		CreatedAt: time.Now(),
		IsBanned:  false,
	}

	messageID, err := repo.Message.Create(testMessage)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Create expired comment
	expiredComment := &domain.Comment{
		MessageID: messageID,
		UserID:    2,
		Username:  "commenter",
		Content:   "Expired comment",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	expiredCommentID, err := repo.Message.CreateComment(expiredComment)
	if err != nil {
		t.Fatalf("Failed to create expired comment: %v", err)
	}

	// Create valid comment
	validComment := &domain.Comment{
		MessageID: messageID,
		UserID:    2,
		Username:  "commenter",
		Content:   "Valid comment",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	validCommentID, err := repo.Message.CreateComment(validComment)
	if err != nil {
		t.Fatalf("Failed to create valid comment: %v", err)
	}

	// Get comments - should only return non-expired ones
	comments, err := repo.Message.GetComments(messageID)
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	// Should only have 1 valid comment
	if len(comments) != 1 {
		t.Errorf("Expected 1 valid comment, got %d", len(comments))
	}

	if len(comments) > 0 && comments[0].ID != validCommentID {
		t.Error("Expected to get only the valid comment")
	}

	// Test cleanup of expired comments
	err = repo.Message.DeleteExpiredComments()
	if err != nil {
		t.Fatalf("Failed to delete expired comments: %v", err)
	}

	// Verify expired comment is deleted
	_, err = repo.Message.GetCommentByID(expiredCommentID)
	if err == nil {
		t.Error("Expected expired comment to be deleted")
	}

	// Verify valid comment still exists
	_, err = repo.Message.GetCommentByID(validCommentID)
	if err != nil {
		t.Errorf("Expected valid comment to still exist: %v", err)
	}
}
