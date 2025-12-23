package domain

import "time"

// ConnectionStatus represents a player's connection state
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "CONNECTED"
	StatusDisconnected ConnectionStatus = "DISCONNECTED"
)

// Player represents a player in the game
type Player struct {
	ID           string           `json:"id"`
	Nickname     string           `json:"nickname"`
	Role         Role             `json:"role,omitempty"`
	HasVoted     bool             `json:"hasVoted"`
	HasSubmitted bool             `json:"hasSubmitted"`
	Status       ConnectionStatus `json:"status"`
	JoinedAt     time.Time        `json:"joinedAt"`
}

// NewPlayer creates a new player with the given ID and nickname
func NewPlayer(id, nickname string) *Player {
	return &Player{
		ID:           id,
		Nickname:     nickname,
		Role:         "",
		HasVoted:     false,
		HasSubmitted: false,
		Status:       StatusConnected,
		JoinedAt:     time.Now(),
	}
}

// ResetForNewRound resets the player's state for a new round
func (p *Player) ResetForNewRound() {
	p.Role = ""
	p.HasVoted = false
	p.HasSubmitted = false
}

// IsConnected returns true if the player is currently connected
func (p *Player) IsConnected() bool {
	return p.Status == StatusConnected
}

// Disconnect marks the player as disconnected
func (p *Player) Disconnect() {
	p.Status = StatusDisconnected
}

// Reconnect marks the player as connected
func (p *Player) Reconnect() {
	p.Status = StatusConnected
}

// PlayerInfo is a safe view of player data (hides role from other players)
type PlayerInfo struct {
	ID           string           `json:"id"`
	Nickname     string           `json:"nickname"`
	HasVoted     bool             `json:"hasVoted"`
	HasSubmitted bool             `json:"hasSubmitted"`
	Status       ConnectionStatus `json:"status"`
}

// ToInfo converts a Player to PlayerInfo (without role)
func (p *Player) ToInfo() PlayerInfo {
	return PlayerInfo{
		ID:           p.ID,
		Nickname:     p.Nickname,
		HasVoted:     p.HasVoted,
		HasSubmitted: p.HasSubmitted,
		Status:       p.Status,
	}
}

