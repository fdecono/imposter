package domain

import (
	"math/rand"
	"time"
)

// Round represents a single round of the game
type Round struct {
	Number           int           `json:"number"`
	SecretWord       string        `json:"secretWord"`
	ImposterID       string        `json:"imposterId"`
	Submissions      []*Submission `json:"submissions"`
	Votes            []*Vote       `json:"votes"`
	CurrentPlayerIdx int           `json:"currentPlayerIdx"` // Index in PlayerOrder
	PlayerOrder      []string      `json:"playerOrder"`      // Order of player IDs for submission
	Winner           Role          `json:"winner,omitempty"`
	StartedAt        time.Time     `json:"startedAt"`
	EndedAt          time.Time     `json:"endedAt,omitempty"`
}

// NewRound creates a new round with the given parameters
func NewRound(number int, secretWord string, playerIDs []string) *Round {
	// Shuffle player order for submission
	order := make([]string, len(playerIDs))
	copy(order, playerIDs)
	rand.Shuffle(len(order), func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})

	// Pick a random imposter
	imposterIdx := rand.Intn(len(playerIDs))
	imposterID := playerIDs[imposterIdx]

	return &Round{
		Number:           number,
		SecretWord:       secretWord,
		ImposterID:       imposterID,
		Submissions:      make([]*Submission, 0),
		Votes:            make([]*Vote, 0),
		CurrentPlayerIdx: 0,
		PlayerOrder:      order,
		StartedAt:        time.Now(),
	}
}

// GetCurrentPlayerID returns the ID of the player whose turn it is to submit
func (r *Round) GetCurrentPlayerID() string {
	if r.CurrentPlayerIdx >= len(r.PlayerOrder) {
		return ""
	}
	return r.PlayerOrder[r.CurrentPlayerIdx]
}

// IsPlayerTurn checks if it's the given player's turn to submit
func (r *Round) IsPlayerTurn(playerID string) bool {
	return r.GetCurrentPlayerID() == playerID
}

// AddSubmission adds a word submission from a player
func (r *Round) AddSubmission(playerID, nickname, word string) error {
	if !r.IsPlayerTurn(playerID) {
		return ErrNotYourTurn
	}

	submission := NewSubmission(playerID, nickname, word, len(r.Submissions)+1)
	r.Submissions = append(r.Submissions, submission)
	r.CurrentPlayerIdx++

	return nil
}

// AllSubmitted returns true if all players have submitted
func (r *Round) AllSubmitted() bool {
	return r.CurrentPlayerIdx >= len(r.PlayerOrder)
}

// AddVote adds a vote from a player
func (r *Round) AddVote(voterID, targetID string) error {
	// Check if already voted
	for _, v := range r.Votes {
		if v.VoterID == voterID {
			return ErrAlreadyVoted
		}
	}

	vote := NewVote(voterID, targetID)
	r.Votes = append(r.Votes, vote)

	return nil
}

// AllVoted returns true if all players have voted
func (r *Round) AllVoted(totalPlayers int) bool {
	return len(r.Votes) >= totalPlayers
}

// GetVotedCount returns the number of players who have voted
func (r *Round) GetVotedCount() int {
	return len(r.Votes)
}

// CalculateResults calculates the voting results and determines the winner
func (r *Round) CalculateResults(players map[string]*Player) ([]VoteResult, Role) {
	// Count votes per player
	voteCounts := make(map[string]int)
	voterNames := make(map[string][]string) // targetID -> voter nicknames

	for _, vote := range r.Votes {
		voteCounts[vote.TargetID]++
		voterNickname := ""
		if voter, ok := players[vote.VoterID]; ok {
			voterNickname = voter.Nickname
		}
		voterNames[vote.TargetID] = append(voterNames[vote.TargetID], voterNickname)
	}

	// Build results
	results := make([]VoteResult, 0, len(players))
	maxVotes := 0
	maxVotedPlayerID := ""

	for playerID, player := range players {
		count := voteCounts[playerID]
		result := VoteResult{
			PlayerID:   playerID,
			Nickname:   player.Nickname,
			VoteCount:  count,
			VotedBy:    voterNames[playerID],
			IsImposter: playerID == r.ImposterID,
		}
		results = append(results, result)

		if count > maxVotes {
			maxVotes = count
			maxVotedPlayerID = playerID
		}
	}

	// Determine winner
	var winner Role
	if maxVotedPlayerID == r.ImposterID {
		winner = RoleVilek // Vileks caught the imposter!
	} else {
		winner = RoleImposter // Imposter wasn't caught
	}

	r.Winner = winner
	r.EndedAt = time.Now()

	return results, winner
}

// HasPlayerVoted checks if a player has already voted
func (r *Round) HasPlayerVoted(playerID string) bool {
	for _, v := range r.Votes {
		if v.VoterID == playerID {
			return true
		}
	}
	return false
}

