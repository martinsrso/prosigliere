package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RequestIDMiddleware struct {
	next http.Handler
}

func NewRequestIDMiddleware(next http.Handler) http.Handler {
	return &RequestIDMiddleware{next: next}
}

func (m *RequestIDMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	r.Header.Set("X-Request-ID", requestID)
	w.Header().Set("X-Request-ID", requestID)

	ctx := r.Context()
	ctx = context.WithValue(ctx, "request_id", requestID)
	r = r.WithContext(ctx)

	m.next.ServeHTTP(w, r)
}

func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

type LoggingMiddleware struct {
	logger *slog.Logger
	next   http.Handler
}

func NewLoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return &LoggingMiddleware{
		logger: logger,
		next:   next,
	}
}

func (m *LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	recorder := &statusRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	m.next.ServeHTTP(recorder, r)

	duration := time.Since(start)
	userAgent := r.Header.Get("User-Agent")
	referer := r.Header.Get("Referer")

	m.logger.Info("HTTP request",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"status", recorder.statusCode,
		"duration_ms", duration.Milliseconds(),
		"remote_addr", r.RemoteAddr,
		"user_agent", userAgent,
		"referer", referer,
		"content_length", r.ContentLength,
	)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

type Session struct {
	ID        string
	Username  string
	ExpiresAt time.Time
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (sm *SessionManager) CreateSession(username string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID := generateSessionID()
	session := &Session{
		ID:        sessionID,
		Username:  username,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	sm.sessions[sessionID] = session
	return sessionID
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

func generateSessionID() string {
	return base64.URLEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
}

type BasicAuthMiddleware struct {
	username string
	password string
	sessions *SessionManager
	next     http.Handler
	logger   *slog.Logger
}

func NewBasicAuthMiddleware(username, password string, sessions *SessionManager, logger *slog.Logger, next http.Handler) http.Handler {
	return &BasicAuthMiddleware{
		username: username,
		password: password,
		sessions: sessions,
		next:     next,
		logger:   logger,
	}
}

func (m *BasicAuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.isPublicEndpoint(r.URL.Path) {
		m.next.ServeHTTP(w, r)
		return
	}

	sessionCookie, err := r.Cookie("session_id")
	if err == nil && sessionCookie != nil {
		if session, exists := m.sessions.GetSession(sessionCookie.Value); exists {
			r.Header.Set("X-Username", session.Username)
			m.next.ServeHTTP(w, r)
			return
		}
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		m.requireAuth(w)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		m.requireAuth(w)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		m.requireAuth(w)
		return
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		m.requireAuth(w)
		return
	}

	providedUsername := credentials[0]
	providedPassword := credentials[1]

	if subtle.ConstantTimeCompare([]byte(providedUsername), []byte(m.username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(providedPassword), []byte(m.password)) != 1 {
		m.requireAuth(w)
		return
	}

	sessionID := m.sessions.CreateSession(providedUsername)
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
	}
	http.SetCookie(w, cookie)

	r.Header.Set("X-Username", providedUsername)
	m.next.ServeHTTP(w, r)
}

func (m *BasicAuthMiddleware) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{"/health", "/metrics"}
	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}
	return false
}

func (m *BasicAuthMiddleware) requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

type MetricsMiddleware struct {
	next http.Handler
}

func NewMetricsMiddleware(next http.Handler) http.Handler {
	return &MetricsMiddleware{next: next}
}

func (m *MetricsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	endpoint := normalizeEndpoint(r.URL.Path)

	recorder := &statusRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	m.next.ServeHTTP(recorder, r)

	duration := time.Since(start).Seconds()
	status := http.StatusText(recorder.statusCode)

	httpRequestsTotal.WithLabelValues(r.Method, endpoint, status).Inc()
	httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
}

func normalizeEndpoint(path string) string {
	path = strings.TrimPrefix(path, "/api/")
	if strings.HasPrefix(path, "posts/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return "posts/{id}"
		}
	}
	return path
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
