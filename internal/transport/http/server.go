package http

import (
	"bufio"
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"imposter/internal/app"
	"imposter/internal/config"
	"imposter/internal/transport/ws"
)

// Server represents the HTTP server
type Server struct {
	server  *http.Server
	hub     *app.GameHub
	config  *config.Config
	logger  *slog.Logger
	webFS   fs.FS
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, hub *app.GameHub, logger *slog.Logger, webFS embed.FS) *Server {
	// Get the web subdirectory from embed FS
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		logger.Error("failed to get web subdirectory", "error", err)
	}

	s := &Server{
		hub:    hub,
		config: cfg,
		logger: logger,
		webFS:  webContent,
	}

	// Set up routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	s.server = &http.Server{
		Addr:         cfg.GetAddr(),
		Handler:      s.middleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("POST /api/rooms", s.handleCreateRoom)
	mux.HandleFunc("GET /api/rooms/{roomCode}", s.handleGetRoom)
	mux.HandleFunc("GET /api/rooms/{roomCode}/exists", s.handleRoomExists)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/stats", s.handleStats)

	// WebSocket
	wsHandler := ws.NewHandler(s.hub, s.logger)
	mux.Handle("GET /ws", wsHandler)

	// Static files and SPA
	mux.HandleFunc("GET /static/", s.handleStatic)
	mux.HandleFunc("GET /", s.handleSPA)
}

// middleware wraps the handler with logging and other middleware
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// Log request (skip static files in production)
		if s.config.IsDevelopment() || !isStaticRequest(r.URL.Path) {
			s.logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration", time.Since(start),
			)
		}
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("server starting", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server shutting down")
	return s.server.Shutdown(ctx)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements http.Flusher
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// isStaticRequest checks if the request is for a static file
func isStaticRequest(path string) bool {
	return len(path) > 8 && path[:8] == "/static/"
}

