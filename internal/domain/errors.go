package domain

import "errors"

// Domain errors
var (
	ErrGameNotFound       = errors.New("game not found")
	ErrGameFull           = errors.New("game is full")
	ErrGameAlreadyStarted = errors.New("game already started")
	ErrNotEnoughPlayers   = errors.New("not enough players to start")
	ErrNotYourTurn        = errors.New("not your turn to submit")
	ErrAlreadySubmitted   = errors.New("already submitted this round")
	ErrAlreadyVoted       = errors.New("already voted this round")
	ErrInvalidPhase       = errors.New("invalid action for current phase")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrNotHost            = errors.New("only host can perform this action")
	ErrCannotVoteSelf     = errors.New("cannot vote for yourself")
	ErrInvalidTransition  = errors.New("invalid phase transition")
	ErrEmptyWord          = errors.New("word cannot be empty")
	ErrInvalidTargetID    = errors.New("invalid vote target")
)

