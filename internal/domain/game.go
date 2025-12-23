package domain

import (
	"strings"
	"time"
)

// GameSettings holds configurable game parameters
type GameSettings struct {
	MinPlayers     int           `json:"minPlayers"`
	MaxPlayers     int           `json:"maxPlayers"`
	VotingDuration time.Duration `json:"votingDuration"`
	RoleRevealTime time.Duration `json:"roleRevealTime"`
}

// DefaultGameSettings returns the default game settings
func DefaultGameSettings() GameSettings {
	return GameSettings{
		MinPlayers:     4,
		MaxPlayers:     10,
		VotingDuration: 20 * time.Second,
		RoleRevealTime: 5 * time.Second,
	}
}

// Game represents a game room
type Game struct {
	ID           string             `json:"id"`
	HostID       string             `json:"hostId"`
	Players      map[string]*Player `json:"players"`
	CurrentRound *Round             `json:"currentRound,omitempty"`
	RoundHistory []*Round           `json:"roundHistory"`
	Phase        Phase              `json:"phase"`
	Settings     GameSettings       `json:"settings"`
	CreatedAt    time.Time          `json:"createdAt"`
}

// NewGame creates a new game with the given ID
func NewGame(id string) *Game {
	return &Game{
		ID:           id,
		HostID:       "",
		Players:      make(map[string]*Player),
		CurrentRound: nil,
		RoundHistory: make([]*Round, 0),
		Phase:        PhaseLobby,
		Settings:     DefaultGameSettings(),
		CreatedAt:    time.Now(),
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(playerID, nickname string) (*Player, error) {
	if g.Phase != PhaseLobby {
		return nil, ErrGameAlreadyStarted
	}

	if len(g.Players) >= g.Settings.MaxPlayers {
		return nil, ErrGameFull
	}

	player := NewPlayer(playerID, nickname)
	g.Players[playerID] = player

	// First player becomes the host
	if g.HostID == "" {
		g.HostID = playerID
	}

	return player, nil
}

// RemovePlayer removes a player from the game
func (g *Game) RemovePlayer(playerID string) error {
	if _, ok := g.Players[playerID]; !ok {
		return ErrPlayerNotFound
	}

	delete(g.Players, playerID)

	// If host left, assign new host
	if g.HostID == playerID && len(g.Players) > 0 {
		for id := range g.Players {
			g.HostID = id
			break
		}
	}

	return nil
}

// GetPlayer returns a player by ID
func (g *Game) GetPlayer(playerID string) (*Player, error) {
	player, ok := g.Players[playerID]
	if !ok {
		return nil, ErrPlayerNotFound
	}
	return player, nil
}

// GetPlayerIDs returns a slice of all player IDs
func (g *Game) GetPlayerIDs() []string {
	ids := make([]string, 0, len(g.Players))
	for id := range g.Players {
		ids = append(ids, id)
	}
	return ids
}

// GetConnectedPlayerCount returns the number of connected players
func (g *Game) GetConnectedPlayerCount() int {
	count := 0
	for _, p := range g.Players {
		if p.IsConnected() {
			count++
		}
	}
	return count
}

// CanStart checks if the game can be started
func (g *Game) CanStart() bool {
	return g.Phase == PhaseLobby && len(g.Players) >= g.Settings.MinPlayers
}

// StartRound starts a new round with the given secret word
func (g *Game) StartRound(secretWord string) error {
	if g.Phase != PhaseLobby && g.Phase != PhaseResults {
		return ErrInvalidPhase
	}

	if len(g.Players) < g.Settings.MinPlayers {
		return ErrNotEnoughPlayers
	}

	// Reset all players for new round
	for _, player := range g.Players {
		player.ResetForNewRound()
	}

	// Create new round
	roundNumber := len(g.RoundHistory) + 1
	g.CurrentRound = NewRound(roundNumber, secretWord, g.GetPlayerIDs())

	// Assign roles to players
	for playerID, player := range g.Players {
		if playerID == g.CurrentRound.ImposterID {
			player.Role = RoleImposter
		} else {
			player.Role = RoleVilek
		}
	}

	g.Phase = PhaseRoleAssignment

	return nil
}

// TransitionToSubmission moves to submission phase
func (g *Game) TransitionToSubmission() error {
	if g.Phase != PhaseRoleAssignment {
		return ErrInvalidTransition
	}
	g.Phase = PhaseSubmission
	return nil
}

// SubmitWord submits a word for the current player
func (g *Game) SubmitWord(playerID, word string) error {
	if g.Phase != PhaseSubmission {
		return ErrInvalidPhase
	}

	if g.CurrentRound == nil {
		return ErrInvalidPhase
	}

	word = strings.TrimSpace(word)
	if word == "" {
		return ErrEmptyWord
	}

	player, err := g.GetPlayer(playerID)
	if err != nil {
		return err
	}

	if player.HasSubmitted {
		return ErrAlreadySubmitted
	}

	err = g.CurrentRound.AddSubmission(playerID, player.Nickname, word)
	if err != nil {
		return err
	}

	player.HasSubmitted = true

	return nil
}

// AllSubmitted checks if all players have submitted
func (g *Game) AllSubmitted() bool {
	if g.CurrentRound == nil {
		return false
	}
	return g.CurrentRound.AllSubmitted()
}

// TransitionToVoting moves to voting phase
func (g *Game) TransitionToVoting() error {
	if g.Phase != PhaseSubmission {
		return ErrInvalidTransition
	}
	g.Phase = PhaseVoting
	return nil
}

// CastVote casts a vote from one player for another
func (g *Game) CastVote(voterID, targetID string) error {
	if g.Phase != PhaseVoting {
		return ErrInvalidPhase
	}

	if g.CurrentRound == nil {
		return ErrInvalidPhase
	}

	if voterID == targetID {
		return ErrCannotVoteSelf
	}

	voter, err := g.GetPlayer(voterID)
	if err != nil {
		return err
	}

	if voter.HasVoted {
		return ErrAlreadyVoted
	}

	// Verify target exists
	if _, err := g.GetPlayer(targetID); err != nil {
		return ErrInvalidTargetID
	}

	err = g.CurrentRound.AddVote(voterID, targetID)
	if err != nil {
		return err
	}

	voter.HasVoted = true

	return nil
}

// AllVoted checks if all players have voted
func (g *Game) AllVoted() bool {
	if g.CurrentRound == nil {
		return false
	}
	return g.CurrentRound.AllVoted(len(g.Players))
}

// EndRound ends the current round and calculates results
func (g *Game) EndRound() ([]VoteResult, Role, error) {
	if g.Phase != PhaseVoting {
		return nil, "", ErrInvalidPhase
	}

	if g.CurrentRound == nil {
		return nil, "", ErrInvalidPhase
	}

	results, winner := g.CurrentRound.CalculateResults(g.Players)
	g.RoundHistory = append(g.RoundHistory, g.CurrentRound)
	g.Phase = PhaseResults

	return results, winner, nil
}

// GetLobbyState returns the current lobby state for broadcasting
func (g *Game) GetLobbyState() *LobbyUpdatePayload {
	players := make([]PlayerInfo, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p.ToInfo())
	}

	return &LobbyUpdatePayload{
		Players:  players,
		HostID:   g.HostID,
		CanStart: g.CanStart(),
	}
}

// GetSubmissionState returns the current submission phase state
func (g *Game) GetSubmissionState() *SubmissionUpdatePayload {
	if g.CurrentRound == nil {
		return nil
	}

	return &SubmissionUpdatePayload{
		Submissions:     g.CurrentRound.Submissions,
		CurrentPlayerID: g.CurrentRound.GetCurrentPlayerID(),
		IsComplete:      g.CurrentRound.AllSubmitted(),
	}
}

// GetVoteProgress returns the current voting progress
func (g *Game) GetVoteProgress() *VoteUpdatePayload {
	if g.CurrentRound == nil {
		return nil
	}

	return &VoteUpdatePayload{
		VotedCount:   g.CurrentRound.GetVotedCount(),
		TotalPlayers: len(g.Players),
	}
}

// IsHost checks if the given player is the host
func (g *Game) IsHost(playerID string) bool {
	return g.HostID == playerID
}

// GetPlayerInfoList returns a list of all players as PlayerInfo
func (g *Game) GetPlayerInfoList() []PlayerInfo {
	players := make([]PlayerInfo, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p.ToInfo())
	}
	return players
}

