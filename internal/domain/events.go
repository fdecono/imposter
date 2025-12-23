package domain

import "time"

// EventType represents the type of game event
type EventType string

const (
	EventPlayerJoined      EventType = "PLAYER_JOINED"
	EventPlayerLeft        EventType = "PLAYER_LEFT"
	EventPlayerReconnected EventType = "PLAYER_RECONNECTED"
	EventGameStarted       EventType = "GAME_STARTED"
	EventRolesAssigned     EventType = "ROLES_ASSIGNED"
	EventSubmissionMade    EventType = "SUBMISSION_MADE"
	EventAllSubmitted      EventType = "ALL_SUBMITTED"
	EventVotingStarted     EventType = "VOTING_STARTED"
	EventVoteCast          EventType = "VOTE_CAST"
	EventRoundEnded        EventType = "ROUND_ENDED"
	EventGameEnded         EventType = "GAME_ENDED"
	EventError             EventType = "ERROR"
)

// GameEvent represents an event that occurred in the game
type GameEvent struct {
	Type      EventType   `json:"type"`
	GameID    string      `json:"gameId"`
	PlayerID  string      `json:"playerId,omitempty"` // If event is player-specific
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewEvent creates a new game event
func NewEvent(eventType EventType, gameID string, payload interface{}) *GameEvent {
	return &GameEvent{
		Type:      eventType,
		GameID:    gameID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// NewPlayerEvent creates a new player-specific game event
func NewPlayerEvent(eventType EventType, gameID, playerID string, payload interface{}) *GameEvent {
	return &GameEvent{
		Type:      eventType,
		GameID:    gameID,
		PlayerID:  playerID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// Payload types for different events

// LobbyUpdatePayload is sent when lobby state changes
type LobbyUpdatePayload struct {
	Players  []PlayerInfo `json:"players"`
	HostID   string       `json:"hostId"`
	CanStart bool         `json:"canStart"`
}

// RoleAssignedPayload is sent to each player with their role
type RoleAssignedPayload struct {
	Role       Role   `json:"role"`
	SecretWord string `json:"secretWord,omitempty"` // Only for VILEKs
}

// SubmissionPhasePayload is sent when submission phase starts
type SubmissionPhasePayload struct {
	CurrentPlayerID string        `json:"currentPlayerId"`
	PlayerOrder     []PlayerInfo  `json:"playerOrder"`
	Submissions     []*Submission `json:"submissions"`
}

// SubmissionUpdatePayload is sent when a new submission is made
type SubmissionUpdatePayload struct {
	Submissions     []*Submission `json:"submissions"`
	CurrentPlayerID string        `json:"currentPlayerId"`
	IsComplete      bool          `json:"isComplete"`
}

// VotingPhasePayload is sent when voting phase starts
type VotingPhasePayload struct {
	RemainingSeconds int          `json:"remainingSeconds"`
	Players          []PlayerInfo `json:"players"`
}

// VotingCountdownPayload is sent every second during voting
type VotingCountdownPayload struct {
	RemainingSeconds int `json:"remainingSeconds"`
}

// VoteUpdatePayload is sent when a vote is cast (without revealing who)
type VoteUpdatePayload struct {
	VotedCount   int `json:"votedCount"`
	TotalPlayers int `json:"totalPlayers"`
}

// RoundResultsPayload is sent when a round ends
type RoundResultsPayload struct {
	Votes      []VoteResult `json:"votes"`
	ImposterID string       `json:"imposterId"`
	Winner     Role         `json:"winner"`
	SecretWord string       `json:"secretWord"`
}

// ErrorPayload is sent when an error occurs
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

