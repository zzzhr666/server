package room

type Status string

const (
	StatusWaiting Status = "waiting"
	StatusPlaying Status = "playing"
	StatusClosed  Status = "closed"
)

type Room struct {
	ID           int64
	OwnerID      int64
	Status       Status
	MaxPlayers   int
	Players      map[int64]struct{}
	ReadyPlayers map[int64]struct{}
}
