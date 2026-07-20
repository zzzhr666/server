package friend

import "time"

type Request struct {
	FromPlayerID int64
	ToPlayerID   int64
	CreatedAt    time.Time
}
