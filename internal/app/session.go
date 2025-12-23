package app

import (
	"log/slog"
	"sync"
	"time"

	"imposter/internal/domain"
)

// ClientConnection represents a connected client
type ClientConnection interface {
	Send(message interface{}) error
	GetPlayerID() string
	Close() error
}

// GameSession wraps a game with concurrency control and client management
type GameSession struct {
	game      *domain.Game
	mu        sync.RWMutex
	clients   map[string]ClientConnection // playerID -> client
	clientsMu sync.RWMutex
	logger    *slog.Logger

	// Timers
	votingTimer   *time.Timer
	countdownDone chan struct{}

	// Event channel for broadcasting
	events chan *domain.GameEvent
	done   chan struct{}
}

// NewGameSession creates a new game session
func NewGameSession(game *domain.Game, logger *slog.Logger) *GameSession {
	session := &GameSession{
		game:    game,
		clients: make(map[string]ClientConnection),
		logger:  logger,
		events:  make(chan *domain.GameEvent, 100),
		done:    make(chan struct{}),
	}

	// Start event broadcaster
	go session.eventLoop()

	return session
}

// GetGame returns the underlying game (read-only operations should use specific methods)
func (s *GameSession) GetGame() *domain.Game {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.game
}

// GetRoomCode returns the room code
func (s *GameSession) GetRoomCode() string {
	return s.game.ID
}

// GetCreatedAt returns when the game was created
func (s *GameSession) GetCreatedAt() time.Time {
	return s.game.CreatedAt
}

// GetPlayerCount returns the number of players
func (s *GameSession) GetPlayerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.game.Players)
}

// GetPhase returns the current game phase
func (s *GameSession) GetPhase() domain.Phase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.game.Phase
}

// CanJoin checks if a new player can join the game
func (s *GameSession) CanJoin() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.game.Phase == domain.PhaseLobby && len(s.game.Players) < s.game.Settings.MaxPlayers
}

// RegisterClient registers a client connection for a player
func (s *GameSession) RegisterClient(playerID string, client ClientConnection) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[playerID] = client
}

// UnregisterClient removes a client connection
func (s *GameSession) UnregisterClient(playerID string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	delete(s.clients, playerID)
}

// GetClient returns the client for a player
func (s *GameSession) GetClient(playerID string) (ClientConnection, bool) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	client, ok := s.clients[playerID]
	return client, ok
}

// AddPlayer adds a player to the game
func (s *GameSession) AddPlayer(playerID, nickname string) (*domain.Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, err := s.game.AddPlayer(playerID, nickname)
	if err != nil {
		return nil, err
	}

	// Broadcast lobby update
	s.queueEvent(domain.NewEvent(domain.EventPlayerJoined, s.game.ID, s.game.GetLobbyState()))

	return player, nil
}

// RemovePlayer removes a player from the game
func (s *GameSession) RemovePlayer(playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.game.RemovePlayer(playerID)
	if err != nil {
		return err
	}

	// Broadcast lobby update
	s.queueEvent(domain.NewEvent(domain.EventPlayerLeft, s.game.ID, s.game.GetLobbyState()))

	return nil
}

// DisconnectPlayer marks a player as disconnected
func (s *GameSession) DisconnectPlayer(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if player, err := s.game.GetPlayer(playerID); err == nil {
		player.Disconnect()
		s.queueEvent(domain.NewEvent(domain.EventPlayerLeft, s.game.ID, s.game.GetLobbyState()))
	}
}

// ReconnectPlayer marks a player as reconnected
func (s *GameSession) ReconnectPlayer(playerID string) (*domain.Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, err := s.game.GetPlayer(playerID)
	if err != nil {
		return nil, err
	}

	player.Reconnect()
	s.queueEvent(domain.NewEvent(domain.EventPlayerReconnected, s.game.ID, s.game.GetLobbyState()))

	return player, nil
}

// StartGame starts the game (host only)
func (s *GameSession) StartGame(playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.game.IsHost(playerID) {
		return domain.ErrNotHost
	}

	secretWord := GetRandomWord()
	err := s.game.StartRound(secretWord)
	if err != nil {
		return err
	}

	// Send role assignments to each player
	for pid, player := range s.game.Players {
		payload := &domain.RoleAssignedPayload{
			Role: player.Role,
		}
		if player.Role == domain.RoleVilek {
			payload.SecretWord = s.game.CurrentRound.SecretWord
		}
		s.queueEvent(domain.NewPlayerEvent(domain.EventRolesAssigned, s.game.ID, pid, payload))
	}

	// Schedule transition to submission phase
	go func() {
		time.Sleep(s.game.Settings.RoleRevealTime)
		s.transitionToSubmission()
	}()

	return nil
}

// transitionToSubmission moves to submission phase
func (s *GameSession) transitionToSubmission() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.game.Phase != domain.PhaseRoleAssignment {
		return
	}

	s.game.TransitionToSubmission()

	// Build player order info
	playerOrder := make([]domain.PlayerInfo, 0, len(s.game.CurrentRound.PlayerOrder))
	for _, pid := range s.game.CurrentRound.PlayerOrder {
		if p, err := s.game.GetPlayer(pid); err == nil {
			playerOrder = append(playerOrder, p.ToInfo())
		}
	}

	payload := &domain.SubmissionPhasePayload{
		CurrentPlayerID: s.game.CurrentRound.GetCurrentPlayerID(),
		PlayerOrder:     playerOrder,
		Submissions:     s.game.CurrentRound.Submissions,
	}

	s.queueEvent(domain.NewEvent(domain.EventSubmissionMade, s.game.ID, payload))
}

// SubmitWord submits a word for a player
func (s *GameSession) SubmitWord(playerID, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.game.SubmitWord(playerID, word)
	if err != nil {
		return err
	}

	// Broadcast submission update
	s.queueEvent(domain.NewEvent(domain.EventSubmissionMade, s.game.ID, s.game.GetSubmissionState()))

	// Check if all submitted
	if s.game.AllSubmitted() {
		s.game.TransitionToVoting()
		s.startVotingPhase()
	}

	return nil
}

// startVotingPhase starts the voting phase with countdown
func (s *GameSession) startVotingPhase() {
	// Already holding lock from caller

	votingDuration := s.game.Settings.VotingDuration
	remainingSeconds := int(votingDuration.Seconds())

	// Broadcast voting phase start
	payload := &domain.VotingPhasePayload{
		RemainingSeconds: remainingSeconds,
		Players:          s.game.GetPlayerInfoList(),
	}
	s.queueEvent(domain.NewEvent(domain.EventVotingStarted, s.game.ID, payload))

	// Start countdown
	s.countdownDone = make(chan struct{})
	go s.votingCountdown(remainingSeconds)
}

// votingCountdown runs the voting countdown
func (s *GameSession) votingCountdown(seconds int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	remaining := seconds

	for {
		select {
		case <-s.countdownDone:
			return
		case <-s.done:
			return
		case <-ticker.C:
			remaining--
			if remaining <= 0 {
				s.endVotingPhase()
				return
			}

			// Broadcast countdown
			s.queueEvent(domain.NewEvent(domain.EventVoteCast, s.game.ID, &domain.VotingCountdownPayload{
				RemainingSeconds: remaining,
			}))
		}
	}
}

// CastVote casts a vote for a player
func (s *GameSession) CastVote(voterID, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.game.CastVote(voterID, targetID)
	if err != nil {
		return err
	}

	// Broadcast vote progress (without revealing who voted for whom)
	s.queueEvent(domain.NewEvent(domain.EventVoteCast, s.game.ID, s.game.GetVoteProgress()))

	// Check if all voted - end early
	if s.game.AllVoted() {
		// Stop the countdown
		if s.countdownDone != nil {
			close(s.countdownDone)
			s.countdownDone = nil
		}
		s.endVotingPhaseUnlocked()
	}

	return nil
}

// endVotingPhase ends the voting phase and shows results
func (s *GameSession) endVotingPhase() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endVotingPhaseUnlocked()
}

// endVotingPhaseUnlocked ends voting phase (caller must hold lock)
func (s *GameSession) endVotingPhaseUnlocked() {
	if s.game.Phase != domain.PhaseVoting {
		return
	}

	results, winner, err := s.game.EndRound()
	if err != nil {
		s.logger.Error("failed to end round", "error", err)
		return
	}

	payload := &domain.RoundResultsPayload{
		Votes:      results,
		ImposterID: s.game.CurrentRound.ImposterID,
		Winner:     winner,
		SecretWord: s.game.CurrentRound.SecretWord,
	}

	s.queueEvent(domain.NewEvent(domain.EventRoundEnded, s.game.ID, payload))
}

// StartNewRound starts a new round (host only)
func (s *GameSession) StartNewRound(playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.game.IsHost(playerID) {
		return domain.ErrNotHost
	}

	if s.game.Phase != domain.PhaseResults {
		return domain.ErrInvalidPhase
	}

	// Get words used in previous rounds to avoid repeats
	usedWords := make([]string, 0, len(s.game.RoundHistory))
	for _, round := range s.game.RoundHistory {
		usedWords = append(usedWords, round.SecretWord)
	}

	secretWord := GetRandomWordExcluding(usedWords)
	err := s.game.StartRound(secretWord)
	if err != nil {
		return err
	}

	// Send role assignments
	for pid, player := range s.game.Players {
		payload := &domain.RoleAssignedPayload{
			Role: player.Role,
		}
		if player.Role == domain.RoleVilek {
			payload.SecretWord = s.game.CurrentRound.SecretWord
		}
		s.queueEvent(domain.NewPlayerEvent(domain.EventRolesAssigned, s.game.ID, pid, payload))
	}

	// Schedule transition to submission
	go func() {
		time.Sleep(s.game.Settings.RoleRevealTime)
		s.transitionToSubmission()
	}()

	return nil
}

// GetGameState returns the current game state for a reconnecting player
func (s *GameSession) GetGameState(playerID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := map[string]interface{}{
		"phase":    s.game.Phase,
		"players":  s.game.GetPlayerInfoList(),
		"hostId":   s.game.HostID,
		"canStart": s.game.CanStart(),
	}

	// Add phase-specific state
	switch s.game.Phase {
	case domain.PhaseSubmission:
		if s.game.CurrentRound != nil {
			state["submissions"] = s.game.CurrentRound.Submissions
			state["currentPlayerId"] = s.game.CurrentRound.GetCurrentPlayerID()
		}
	case domain.PhaseVoting:
		state["voteProgress"] = s.game.GetVoteProgress()
	case domain.PhaseResults:
		if s.game.CurrentRound != nil {
			results, _ := s.game.CurrentRound.CalculateResults(s.game.Players)
			state["results"] = results
			state["winner"] = s.game.CurrentRound.Winner
			state["imposterId"] = s.game.CurrentRound.ImposterID
			state["secretWord"] = s.game.CurrentRound.SecretWord
		}
	}

	// Add player's role if in game
	if player, err := s.game.GetPlayer(playerID); err == nil && player.Role != "" {
		state["role"] = player.Role
		if player.Role == domain.RoleVilek && s.game.CurrentRound != nil {
			state["secretWord"] = s.game.CurrentRound.SecretWord
		}
	}

	return state
}

// queueEvent adds an event to the broadcast queue
func (s *GameSession) queueEvent(event *domain.GameEvent) {
	select {
	case s.events <- event:
	default:
		s.logger.Warn("event queue full, dropping event", "type", event.Type)
	}
}

// eventLoop processes events and broadcasts to clients
func (s *GameSession) eventLoop() {
	for {
		select {
		case <-s.done:
			return
		case event := <-s.events:
			s.broadcastEvent(event)
		}
	}
}

// broadcastEvent sends an event to appropriate clients
func (s *GameSession) broadcastEvent(event *domain.GameEvent) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	// If player-specific, send only to that player
	if event.PlayerID != "" {
		if client, ok := s.clients[event.PlayerID]; ok {
			if err := client.Send(event); err != nil {
				s.logger.Debug("failed to send to client", "playerID", event.PlayerID, "error", err)
			}
		}
		return
	}

	// Broadcast to all clients
	for playerID, client := range s.clients {
		if err := client.Send(event); err != nil {
			s.logger.Debug("failed to send to client", "playerID", playerID, "error", err)
		}
	}
}

// Close shuts down the session
func (s *GameSession) Close() {
	select {
	case <-s.done:
		return // Already closed
	default:
		close(s.done)
	}

	if s.countdownDone != nil {
		close(s.countdownDone)
	}

	// Close all client connections
	s.clientsMu.Lock()
	for _, client := range s.clients {
		client.Close()
	}
	s.clients = make(map[string]ClientConnection)
	s.clientsMu.Unlock()
}

