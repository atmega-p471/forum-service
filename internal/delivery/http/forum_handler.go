package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/forum/forum-service/internal/domain"
	"github.com/gorilla/mux"
)

// ForumHandler handles HTTP requests for forum operations
type ForumHandler struct {
	usecase domain.MessageUseCase
}

// NewForumHandler creates a new forum handler
func NewForumHandler(usecase domain.MessageUseCase) *ForumHandler {
	return &ForumHandler{
		usecase: usecase,
	}
}

// ListMessages handles GET /api/v1/messages
func (h *ForumHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int64(10) // default
	offset := int64(0) // default

	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			offset = o
		}
	}

	messages, total, err := h.usecase.GetMessages(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"messages": messages,
		"total":    total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateMessage handles POST /api/v1/messages
func (h *ForumHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// For testing, use anonymous user
	message, err := h.usecase.CreateMessage(0, "anonymous", req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

// GetComments handles GET /api/v1/messages/{id}/comments
func (h *ForumHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	messageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	comments, err := h.usecase.GetComments(messageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// CreateComment handles POST /api/v1/messages/{id}/comments
func (h *ForumHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	messageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	comment, err := h.usecase.CreateComment(messageID, 0, "anonymous", req.Content)
	if err != nil {
		if err.Error() == "message not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// DeleteComment handles DELETE /api/v1/comments/{id}
func (h *ForumHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	commentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	err = h.usecase.DeleteComment(commentID)
	if err != nil {
		if err.Error() == "comment not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
