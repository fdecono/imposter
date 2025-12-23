package domain

import "time"

// Vote represents a vote cast by a player
type Vote struct {
	VoterID   string    `json:"voterId"`
	TargetID  string    `json:"targetId"`
	Timestamp time.Time `json:"timestamp"`
}

// NewVote creates a new vote
func NewVote(voterID, targetID string) *Vote {
	return &Vote{
		VoterID:   voterID,
		TargetID:  targetID,
		Timestamp: time.Now(),
	}
}

// VoteResult represents the voting results for display
type VoteResult struct {
	PlayerID   string   `json:"playerId"`
	Nickname   string   `json:"nickname"`
	VoteCount  int      `json:"voteCount"`
	VotedBy    []string `json:"votedBy"` // Nicknames of voters
	IsImposter bool     `json:"isImposter"`
}

