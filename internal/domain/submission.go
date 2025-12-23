package domain

import "time"

// Submission represents a word submitted by a player during the submission phase
type Submission struct {
	PlayerID  string    `json:"playerId"`
	Nickname  string    `json:"nickname"`
	Word      string    `json:"word"`
	Order     int       `json:"order"` // 1-based order in submission sequence
	Timestamp time.Time `json:"timestamp"`
}

// NewSubmission creates a new submission
func NewSubmission(playerID, nickname, word string, order int) *Submission {
	return &Submission{
		PlayerID:  playerID,
		Nickname:  nickname,
		Word:      word,
		Order:     order,
		Timestamp: time.Now(),
	}
}

