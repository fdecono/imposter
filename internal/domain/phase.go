package domain

// Phase represents the current phase of a game
type Phase string

const (
	PhaseLobby          Phase = "LOBBY"           // Waiting for players to join
	PhaseRoleAssignment Phase = "ROLE_ASSIGNMENT" // Showing roles to players
	PhaseSubmission     Phase = "SUBMISSION"      // Players submitting words one by one
	PhaseVoting         Phase = "VOTING"          // 20s countdown, everyone votes
	PhaseResults        Phase = "RESULTS"         // Show votes & winner
)

// String returns the string representation of the phase
func (p Phase) String() string {
	return string(p)
}

// CanTransitionTo checks if a transition from current phase to target phase is valid
func (p Phase) CanTransitionTo(target Phase) bool {
	validTransitions := map[Phase][]Phase{
		PhaseLobby:          {PhaseRoleAssignment},
		PhaseRoleAssignment: {PhaseSubmission},
		PhaseSubmission:     {PhaseVoting},
		PhaseVoting:         {PhaseResults},
		PhaseResults:        {PhaseRoleAssignment, PhaseLobby}, // Can start new round or go back to lobby
	}

	allowed, ok := validTransitions[p]
	if !ok {
		return false
	}

	for _, phase := range allowed {
		if phase == target {
			return true
		}
	}
	return false
}

