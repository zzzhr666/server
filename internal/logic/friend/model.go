package friend

import "time"

// Request 表示一条待处理的好友申请。
type Request struct {
	// FromPlayerID 是申请发起者的玩家 ID。
	FromPlayerID int64
	// ToPlayerID 是申请接收者的玩家 ID。
	ToPlayerID int64
	// CreatedAt 是申请创建时间。
	CreatedAt time.Time
}
