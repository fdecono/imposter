package app

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"imposter/internal/domain"
)

const (
	// DefaultRoomCodeLength is the default length for room codes
	DefaultRoomCodeLength = 6

	// StaleGameTimeout is how long before an inactive game is cleaned up
	StaleGameTimeout = 2 * time.Hour
)

// RoomCodeChars are characters used for room codes (no ambiguous chars)
const RoomCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GameHub manages all active game sessions
type GameHub struct {
	sessions       map[string]*GameSession
	mu             sync.RWMutex
	roomCodeLength int
	logger         *slog.Logger
	done           chan struct{}
}

// NewGameHub creates a new game hub
func NewGameHub(logger *slog.Logger) *GameHub {
	hub := &GameHub{
		sessions:       make(map[string]*GameSession),
		roomCodeLength: DefaultRoomCodeLength,
		logger:         logger,
		done:           make(chan struct{}),
	}

	// Start cleanup goroutine
	go hub.cleanupLoop()

	return hub
}

// CreateGame creates a new game and returns its session
func (h *GameHub) CreateGame() (*GameSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate unique room code
	var roomCode string
	for attempts := 0; attempts < 10; attempts++ {
		roomCode = h.generateRoomCode()
		if _, exists := h.sessions[roomCode]; !exists {
			break
		}
	}

	// Check if we found a unique code
	if _, exists := h.sessions[roomCode]; exists {
		return nil, fmt.Errorf("failed to generate unique room code")
	}

	game := domain.NewGame(roomCode)
	session := NewGameSession(game, h.logger)
	h.sessions[roomCode] = session

	h.logger.Info("game created", "roomCode", roomCode)

	return session, nil
}

// GetSession returns a game session by room code
func (h *GameHub) GetSession(roomCode string) (*GameSession, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	session, ok := h.sessions[roomCode]
	if !ok {
		return nil, domain.ErrGameNotFound
	}

	return session, nil
}

// DeleteSession removes a game session
func (h *GameHub) DeleteSession(roomCode string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if session, ok := h.sessions[roomCode]; ok {
		session.Close()
		delete(h.sessions, roomCode)
		h.logger.Info("game deleted", "roomCode", roomCode)
	}
}

// GetSessionCount returns the number of active sessions
func (h *GameHub) GetSessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// GetTotalPlayerCount returns the total number of players across all sessions
func (h *GameHub) GetTotalPlayerCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, session := range h.sessions {
		total += session.GetPlayerCount()
	}
	return total
}

// Close shuts down the hub and all sessions
func (h *GameHub) Close() {
	close(h.done)

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, session := range h.sessions {
		session.Close()
	}
	h.sessions = make(map[string]*GameSession)
}

// generateRoomCode generates a random room code
func (h *GameHub) generateRoomCode() string {
	b := make([]byte, h.roomCodeLength)
	rand.Read(b)

	code := make([]byte, h.roomCodeLength)
	for i := range code {
		code[i] = RoomCodeChars[int(b[i])%len(RoomCodeChars)]
	}

	return string(code)
}

// cleanupLoop periodically cleans up stale games
func (h *GameHub) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.cleanupStaleGames()
		}
	}
}

// cleanupStaleGames removes games that have been inactive for too long
func (h *GameHub) cleanupStaleGames() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	stale := make([]string, 0)

	for roomCode, session := range h.sessions {
		// Check if game has no players and is old
		if session.GetPlayerCount() == 0 && now.Sub(session.GetCreatedAt()) > StaleGameTimeout {
			stale = append(stale, roomCode)
		}
	}

	for _, roomCode := range stale {
		if session, ok := h.sessions[roomCode]; ok {
			session.Close()
			delete(h.sessions, roomCode)
			h.logger.Info("stale game cleaned up", "roomCode", roomCode)
		}
	}
}

