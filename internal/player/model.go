package player

// Player represents a user profile that can join rooms and play games.
type Player struct {
	ID     int64
	Name   string
	Avatar string
	Email  string
	Phone  string
}
