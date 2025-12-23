package ws

import "time"

// MessageType represents the type of WebSocket message
type MessageType string

// Client → Server message types
const (
	MsgJoinLobby       MessageType = "join_lobby"
	MsgStartGame       MessageType = "start_game"
	MsgSubmitWord      MessageType = "submit_word"
	MsgCastVote        MessageType = "cast_vote"
	MsgRequestNewRound MessageType = "request_new_round"
	MsgPing            MessageType = "ping"
)

// Server → Client message types
const (
	MsgConnected          MessageType = "connected"
	MsgError              MessageType = "error"
	MsgLobbyUpdate        MessageType = "lobby_update"
	MsgGameStarted        MessageType = "game_started"
	MsgRoleAssigned       MessageType = "role_assigned"
	MsgSubmissionPhase    MessageType = "submission_phase"
	MsgSubmissionUpdate   MessageType = "submission_update"
	MsgVotingPhase        MessageType = "voting_phase"
	MsgVotingCountdown    MessageType = "voting_countdown"
	MsgVoteUpdate         MessageType = "vote_update"
	MsgRoundResults       MessageType = "round_results"
	MsgPlayerDisconnected MessageType = "player_disconnected"
	MsgPlayerReconnected  MessageType = "player_reconnected"
	MsgPong               MessageType = "pong"
)

// ClientMessage represents a message from client to server
type ClientMessage struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// ServerMessage represents a message from server to client
type ServerMessage struct {
	Type      MessageType `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// NewServerMessage creates a new server message with current timestamp
func NewServerMessage(msgType MessageType, payload interface{}) *ServerMessage {
	return &ServerMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Client message payloads

// JoinLobbyPayload is the payload for join_lobby message
type JoinLobbyPayload struct {
	Nickname string `json:"nickname"`
}

// SubmitWordPayload is the payload for submit_word message
type SubmitWordPayload struct {
	Word string `json:"word"`
}

// CastVotePayload is the payload for cast_vote message
type CastVotePayload struct {
	TargetPlayerID string `json:"targetPlayerId"`
}

// Server message payloads

// ConnectedPayload is the payload for connected message
type ConnectedPayload struct {
	PlayerID  string                 `json:"playerId"`
	GameID    string                 `json:"gameId"`
	GameState map[string]interface{} `json:"gameState"`
}

// ErrorPayload is the payload for error message
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes
const (
	ErrCodeInvalidMessage  = "INVALID_MESSAGE"
	ErrCodeGameNotFound    = "GAME_NOT_FOUND"
	ErrCodeGameFull        = "GAME_FULL"
	ErrCodeNotYourTurn     = "NOT_YOUR_TURN"
	ErrCodeInvalidAction   = "INVALID_ACTION"
	ErrCodeNotHost         = "NOT_HOST"
	ErrCodeAlreadyVoted    = "ALREADY_VOTED"
	ErrCodeCannotVoteSelf  = "CANNOT_VOTE_SELF"
	ErrCodeInternalError   = "INTERNAL_ERROR"
)

