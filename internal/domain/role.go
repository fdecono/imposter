package domain

// Role represents a player's role in a round
type Role string

const (
	RoleImposter Role = "IMPOSTER"
	RoleVilek    Role = "VILEK"
)

// String returns the string representation of the role
func (r Role) String() string {
	return string(r)
}

// IsImposter returns true if this role is the imposter
func (r Role) IsImposter() bool {
	return r == RoleImposter
}

