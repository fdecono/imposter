package ws

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"imposter/internal/app"
)

// Handler handles WebSocket connections
type Handler struct {
	hub      *app.GameHub
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *app.GameHub, logger *slog.Logger) *Handler {
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				// In production, you should validate the origin
				return true
			},
		},
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get room code from query params
	roomCode := r.URL.Query().Get("roomCode")
	if roomCode == "" {
		http.Error(w, "roomCode is required", http.StatusBadRequest)
		return
	}

	// Get or create player ID
	playerID := r.URL.Query().Get("playerId")
	isReconnect := playerID != ""
	if !isReconnect {
		playerID = uuid.New().String()
	}

	// Get the game session
	session, err := h.hub.GetSession(roomCode)
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Check if can join (for new players)
	if !isReconnect && !session.CanJoin() {
		http.Error(w, "Cannot join this game", http.StatusForbidden)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	// Create client
	client := NewClient(conn, session, playerID, h.logger)

	// Register client with session
	session.RegisterClient(playerID, client)

	h.logger.Info("websocket connected",
		"roomCode", roomCode,
		"playerID", playerID,
		"isReconnect", isReconnect,
	)

	// Handle reconnection
	if isReconnect {
		_, err := session.ReconnectPlayer(playerID)
		if err != nil {
			// Player not found, treat as new connection
			h.logger.Debug("reconnect failed, treating as new", "playerID", playerID, "error", err)
		} else {
			// Send current game state
			client.sendConnected()
		}
	}

	// Start the client
	client.Run()
}

