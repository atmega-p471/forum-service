package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/atmega-p471/forum-service/internal/delivery/ws"
	"github.com/atmega-p471/forum-service/internal/domain"
)

// Handler handles HTTP requests
type Handler struct {
	useCase    domain.MessageUseCase
	hub        *ws.Hub
	authClient AuthClient
}

// AuthClient interface for auth service client
type AuthClient interface {
	ValidateToken(token string) (*domain.User, error)
}

// NewHandler creates a new handler
func NewHandler(useCase domain.MessageUseCase, hub *ws.Hub, authClient AuthClient) *Handler {
	return &Handler{
		useCase:    useCase,
		hub:        hub,
		authClient: authClient,
	}
}

// RegisterRoutes registers the routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Register specific routes first
	mux.HandleFunc("/api/v1/messages/ban", h.handleBanMessage)
	mux.HandleFunc("/api/v1/messages/unban", h.handleUnbanMessage)

	// Register exact match for messages list
	mux.HandleFunc("/api/v1/messages", h.handleMessages)

	// Register specific message operations
	mux.HandleFunc("/api/v1/messages/", h.handleMessageWithID)
	mux.HandleFunc("/api/v1/comments/", h.handleCommentWithID)
}

// authMiddleware extracts user info from token
func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		log.Printf("Auth header: '%s'", authHeader)

		if authHeader == "" {
			log.Printf("No authorization header")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			log.Printf("Invalid authorization header format")
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		log.Printf("Validating token: %s...", token[:min(len(token), 20)])

		// Validate token and get user info
		user, err := h.authClient.ValidateToken(token)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		log.Printf("Token validated successfully for user: ID=%d, Username='%s'", user.ID, user.Username)

		// Add user to request context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getUserFromContext extracts user from request context
func getUserFromContext(r *http.Request) (*domain.User, bool) {
	user, ok := r.Context().Value("user").(*domain.User)
	return user, ok
}

// handleMessages handles GET and POST requests to /api/v1/messages
func (h *Handler) handleMessages(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "3600")

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getMessages(w, r)
	case http.MethodPost:
		h.authMiddleware(h.createMessage)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getMessages returns a list of messages
func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int64(10) // default limit
	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		}
	}

	offset := int64(0) // default offset
	if offsetStr != "" {
		if o, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			offset = o
		}
	}

	log.Printf("Getting messages with limit: %d, offset: %d", limit, offset)

	// Get messages
	messages, total, err := h.useCase.GetMessages(limit, offset)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return messages
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"total":    total,
	}); err != nil {
		log.Printf("Error encoding messages response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// createMessage creates a new message
func (h *Handler) createMessage(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user, ok := getUserFromContext(r)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Parse request
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding message request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Creating message for user %d (%s): %s", user.ID, user.Username, req.Content)

	// Create message using user info from token
	message, err := h.useCase.CreateMessage(user.ID, user.Username, req.Content)
	if err != nil {
		log.Printf("Error creating message: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(message); err != nil {
		log.Printf("Error encoding message response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleBanMessage handles POST requests to /api/v1/messages/ban
func (h *Handler) handleBanMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		ID int64 `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ban message
	if err := h.useCase.BanMessage(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleUnbanMessage handles POST requests to /api/v1/messages/unban
func (h *Handler) handleUnbanMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		ID int64 `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Unban message
	if err := h.useCase.UnbanMessage(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// authAdminMiddleware checks for admin role
func (h *Handler) authAdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return h.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user, ok := getUserFromContext(r)
		if !ok {
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}

		if user.Role != "admin" {
			log.Printf("Access denied: user %s (role: %s) is not admin", user.Username, user.Role)
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleMessageWithID handles operations on specific messages
func (h *Handler) handleMessageWithID(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	log.Printf("Handling message with ID: %s %s", r.Method, path)

	// Skip if it's ban/unban which are handled separately
	if strings.HasSuffix(path, "/ban") || strings.HasSuffix(path, "/unban") {
		http.Error(w, "Route handled elsewhere", http.StatusBadRequest)
		return
	}

	// Handle comments endpoint: /api/v1/messages/{id}/comments
	if strings.Contains(path, "/comments") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/messages/"), "/")
		if len(parts) != 2 || parts[1] != "comments" {
			http.Error(w, "Invalid comments path", http.StatusBadRequest)
			return
		}

		messageID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "Invalid message ID for comments", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getComments(w, r, messageID)
		case http.MethodPost:
			h.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
				h.createComment(w, r, messageID)
			})(w, r)
		default:
			http.Error(w, "Method not allowed for comments", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle single message operations: /api/v1/messages/{id}
	idStr := strings.TrimPrefix(path, "/api/v1/messages/")
	idStr = strings.TrimSuffix(idStr, "/")

	if idStr == "" {
		http.Error(w, "Message ID required", http.StatusBadRequest)
		return
	}

	messageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("Failed to parse message ID '%s': %v", idStr, err)
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	log.Printf("Processing message ID: %d, method: %s", messageID, r.Method)

	switch r.Method {
	case http.MethodGet:
		h.getSingleMessage(w, r, messageID)
	case http.MethodDelete:
		// Check if this is a permanent delete (admin only)
		if r.URL.Query().Get("action") == "delete" {
			log.Printf("Permanent delete requested for message %d", messageID)
			h.authAdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
				h.deleteMessage(w, r, messageID)
			})(w, r)
		} else {
			// Regular delete = ban
			log.Printf("Ban requested for message %d", messageID)
			h.authAdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
				h.banMessage(w, r, messageID)
			})(w, r)
		}
	default:
		http.Error(w, "Method not allowed for message", http.StatusMethodNotAllowed)
	}
}

// handleCommentWithID handles operations on specific comments
func (h *Handler) handleCommentWithID(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	log.Printf("Handling comment with ID: %s %s", r.Method, path)

	// Extract comment ID from path: /api/v1/comments/{id}
	idStr := strings.TrimPrefix(path, "/api/v1/comments/")
	idStr = strings.TrimSuffix(idStr, "/")

	if idStr == "" {
		http.Error(w, "Comment ID required", http.StatusBadRequest)
		return
	}

	commentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("Failed to parse comment ID '%s': %v", idStr, err)
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	log.Printf("Processing comment ID: %d, method: %s", commentID, r.Method)

	switch r.Method {
	case http.MethodDelete:
		h.authAdminMiddleware(func(w http.ResponseWriter, r *http.Request) {
			h.deleteComment(w, r, commentID)
		})(w, r)
	default:
		http.Error(w, "Method not allowed for comment", http.StatusMethodNotAllowed)
	}
}

// getSingleMessage gets a single message by ID
func (h *Handler) getSingleMessage(w http.ResponseWriter, r *http.Request, messageID int64) {
	log.Printf("Getting single message ID: %d", messageID)

	// For now, just return a simple response
	// In a real implementation, you'd get the message from the use case
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      messageID,
		"message": "Single message endpoint",
	})
}

// banMessage bans a message (soft delete)
func (h *Handler) banMessage(w http.ResponseWriter, r *http.Request, messageID int64) {
	log.Printf("Admin banning message ID: %d", messageID)

	if err := h.useCase.BanMessage(messageID); err != nil {
		log.Printf("Error banning message %d: %v", messageID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message banned successfully",
	})
}

// deleteMessage deletes a message (admin only)
func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request, messageID int64) {
	log.Printf("Admin deleting message ID: %d", messageID)

	if err := h.useCase.DeleteMessage(messageID); err != nil {
		log.Printf("Error deleting message %d: %v", messageID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message deleted successfully",
	})
}

// deleteComment deletes a comment (admin only)
func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request, commentID int64) {
	log.Printf("Admin deleting comment ID: %d", commentID)

	if err := h.useCase.DeleteComment(commentID); err != nil {
		log.Printf("Error deleting comment %d: %v", commentID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Comment deleted successfully",
	})
}

// getComments returns comments for a message
func (h *Handler) getComments(w http.ResponseWriter, r *http.Request, messageID int64) {
	comments, err := h.useCase.GetComments(messageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"comments": comments,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// createComment creates a new comment
func (h *Handler) createComment(w http.ResponseWriter, r *http.Request, messageID int64) {
	// Get user from context
	user, ok := getUserFromContext(r)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Parse request
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Creating comment for user %d (%s) on message %d: %s", user.ID, user.Username, messageID, req.Content)

	// Create comment using user info from token
	comment, err := h.useCase.CreateComment(messageID, user.ID, user.Username, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return comment
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(comment); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleWebsocket handles WebSocket connections
func (h *Handler) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	ws.ServeWs(h.hub, w, r, nil)
}
