package room

type Room struct {
	ID      int64
	OwnerID int64
	Players map[int64]struct{}
}
