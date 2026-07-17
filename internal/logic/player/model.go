package player

// Player represents a user profile used by login and gameplay systems.
type Player struct {
	ID       int64
	Nickname string
	Avatar   string
	Email    string
	Phone    string
}
