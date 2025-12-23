package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"imposter/internal/domain"
)

// Response is a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateRoomResponse is the response for room creation
type CreateRoomResponse struct {
	RoomCode   string `json:"roomCode"`
	InviteLink string `json:"inviteLink"`
}

// GetRoomResponse is the response for getting room info
type GetRoomResponse struct {
	RoomCode    string `json:"roomCode"`
	PlayerCount int    `json:"playerCount"`
	Phase       string `json:"phase"`
	CanJoin     bool   `json:"canJoin"`
}

// RoomExistsResponse is the response for checking if room exists
type RoomExistsResponse struct {
	Exists bool `json:"exists"`
}

// HealthResponse is the response for health check
type HealthResponse struct {
	Status string `json:"status"`
}

// StatsResponse is the response for stats endpoint
type StatsResponse struct {
	ActiveGames   int `json:"activeGames"`
	TotalPlayers  int `json:"totalPlayers"`
}

// handleCreateRoom handles POST /api/rooms
func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	session, err := s.hub.CreateGame()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "CREATION_FAILED", "Failed to create room")
		return
	}

	// Build invite link
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	inviteLink := scheme + "://" + host + "/join/" + session.GetRoomCode()

	s.sendSuccess(w, &CreateRoomResponse{
		RoomCode:   session.GetRoomCode(),
		InviteLink: inviteLink,
	})
}

// handleGetRoom handles GET /api/rooms/{roomCode}
func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomCode := r.PathValue("roomCode")
	if roomCode == "" {
		s.sendError(w, http.StatusBadRequest, "MISSING_ROOM_CODE", "Room code is required")
		return
	}

	session, err := s.hub.GetSession(strings.ToUpper(roomCode))
	if err != nil {
		if err == domain.ErrGameNotFound {
			s.sendError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room not found")
		} else {
			s.sendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	s.sendSuccess(w, &GetRoomResponse{
		RoomCode:    session.GetRoomCode(),
		PlayerCount: session.GetPlayerCount(),
		Phase:       string(session.GetPhase()),
		CanJoin:     session.CanJoin(),
	})
}

// handleRoomExists handles GET /api/rooms/{roomCode}/exists
func (s *Server) handleRoomExists(w http.ResponseWriter, r *http.Request) {
	roomCode := r.PathValue("roomCode")
	if roomCode == "" {
		s.sendError(w, http.StatusBadRequest, "MISSING_ROOM_CODE", "Room code is required")
		return
	}

	_, err := s.hub.GetSession(strings.ToUpper(roomCode))
	exists := err == nil

	s.sendSuccess(w, &RoomExistsResponse{
		Exists: exists,
	})
}

// handleHealth handles GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.sendSuccess(w, &HealthResponse{
		Status: "ok",
	})
}

// handleStats handles GET /api/stats
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.sendSuccess(w, &StatsResponse{
		ActiveGames:  s.hub.GetSessionCount(),
		TotalPlayers: s.hub.GetTotalPlayerCount(),
	})
}

// handleStatic serves static files
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Strip /static/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/static/")

	// Try to open from webFS
	file, err := s.webFS.Open("static/" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Get file info for content type and modification time
	stat, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
}

// handleSPA serves the single-page application
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// For all non-API, non-static, non-WS routes, serve index.html
	// This enables client-side routing (e.g., /join/ABC123)

	file, err := s.webFS.Open("index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", stat.ModTime(), file.(io.ReadSeeker))
}

// sendSuccess sends a successful JSON response
func (s *Server) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&Response{
		Success: true,
		Data:    data,
	})
}

// sendError sends an error JSON response
func (s *Server) sendError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(&Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

