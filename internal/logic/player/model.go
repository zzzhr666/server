package player

// Player 表示登录和游戏系统使用的玩家档案。
type Player struct {
	ID       int64
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Coins    int64
}
