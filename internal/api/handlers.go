package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"prosig/internal/store"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CreatePostRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
}

type PostResponse struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content,omitempty"`
	CreatedAt string `json:"created_at"`
	Comments  []store.Comment `json:"comments,omitempty"`
	CommentCount int   `json:"comment_count,omitempty"`
}

type CommentResponse struct {
	ID        int    `json:"id"`
	PostID    int    `json:"post_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

var (
	requestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_request_errors_total",
			Help: "Total number of API request errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)
)

type APIHandler struct {
	store  store.StoreInterface
	logger *slog.Logger
}

func NewAPIHandler(s *store.Store, logger *slog.Logger) *APIHandler {
	return &APIHandler{
		store:  s,
		logger: logger,
	}
}

func NewAPIHandlerWithStore(storeInterface store.StoreInterface, logger *slog.Logger) *APIHandler {
	return &APIHandler{
		store:  storeInterface,
		logger: logger,
	}
}

func (h *APIHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	h.logger.Info("Listing posts", "request_id", requestID)

	posts, err := h.store.GetAllPosts()
	if err != nil {
		h.logger.Error("Failed to get posts", "request_id", requestID, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts", "database_error").Inc()
		h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve posts")
		return
	}
	
	responses := make([]PostResponse, len(posts))
	for i, post := range posts {
		commentCount, err := h.store.GetCommentCount(post.ID)
		if err != nil {
			h.logger.Warn("Failed to get comment count", "post_id", post.ID, "error", err)
			commentCount = 0
		}
		responses[i] = PostResponse{
			ID:          post.ID,
			Title:       post.Title,
			CreatedAt:   post.CreatedAt.Format(time.RFC3339),
			CommentCount: commentCount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

func (h *APIHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	h.logger.Info("Creating new post", "request_id", requestID)

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode request", "request_id", requestID, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts", "invalid_json").Inc()
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		h.logger.Warn("Missing required fields", "request_id", requestID, "title_empty", req.Title == "", "content_empty", req.Content == "")
		requestErrors.WithLabelValues(r.Method, "posts", "validation_error").Inc()
		h.respondWithError(w, http.StatusBadRequest, "Title and content are required")
		return
	}

	post, err := h.store.CreatePost(req.Title, req.Content)
	if err != nil {
		h.logger.Error("Failed to create post", "request_id", requestID, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts", "database_error").Inc()
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}
	
	response := PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Content:   post.Content,
		CreatedAt: post.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	h.logger.Info("Getting post", "request_id", requestID, "post_id", idStr)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Warn("Invalid post ID format", "request_id", requestID, "post_id", idStr, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts/{id}", "invalid_id").Inc()
		h.respondWithError(w, http.StatusBadRequest, "Invalid post ID format")
		return
	}

	post, err := h.store.GetPost(id)
	if err == store.ErrPostNotFound {
		h.logger.Info("Post not found", "request_id", requestID, "post_id", id)
		requestErrors.WithLabelValues(r.Method, "posts/{id}", "not_found").Inc()
		h.respondWithError(w, http.StatusNotFound, "Post not found")
		return
	}
	if err != nil {
		h.logger.Error("Failed to get post", "request_id", requestID, "post_id", id, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts/{id}", "database_error").Inc()
		h.respondWithError(w, http.StatusInternalServerError, "Failed to retrieve post")
		return
	}

	response := PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Content:   post.Content,
		CreatedAt: post.CreatedAt.Format(time.RFC3339),
		Comments:  post.Comments,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	postIDStr := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	postIDStr = strings.TrimSuffix(postIDStr, "/comments")
	h.logger.Info("Creating comment", "request_id", requestID, "post_id", postIDStr)

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		h.logger.Warn("Invalid post ID format", "request_id", requestID, "post_id", postIDStr, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts/{id}/comments", "invalid_id").Inc()
		h.respondWithError(w, http.StatusBadRequest, "Invalid post ID format")
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode request", "request_id", requestID, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts/{id}/comments", "invalid_json").Inc()
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		h.logger.Warn("Missing comment content", "request_id", requestID, "post_id", postID)
		requestErrors.WithLabelValues(r.Method, "posts/{id}/comments", "validation_error").Inc()
		h.respondWithError(w, http.StatusBadRequest, "Comment content is required")
		return
	}

	comment, err := h.store.AddComment(postID, req.Content)
	if err == store.ErrPostNotFound {
		h.logger.Info("Post not found for comment", "request_id", requestID, "post_id", postID)
		requestErrors.WithLabelValues(r.Method, "posts/{id}/comments", "not_found").Inc()
		h.respondWithError(w, http.StatusNotFound, "Post not found")
		return
	}
	if err != nil {
		h.logger.Error("Failed to add comment", "request_id", requestID, "post_id", postID, "error", err)
		requestErrors.WithLabelValues(r.Method, "posts/{id}/comments", "database_error").Inc()
		h.respondWithError(w, http.StatusInternalServerError, "Failed to add comment")
		return
	}

	response := CommentResponse{
		ID:        comment.ID,
		PostID:    comment.PostID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	h.logger.Info("Health check request", "request_id", requestID)

	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
		"request_id": requestID,
		"service": map[string]string{
			"name":    "prosig",
			"version": "1.0.0",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func (h *APIHandler) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
		"time": time.Now().Format(time.RFC3339),
	}

	if requestID := w.Header().Get("X-Request-ID"); requestID != "" {
		errorResponse["request_id"] = requestID
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

func (h *APIHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}
