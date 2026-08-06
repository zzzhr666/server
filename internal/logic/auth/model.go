package auth

import (
	"server/internal/logic/player"
	"time"
)

// Account 保存登录凭据及绑定到用户名的玩家。
type Account struct {
	Username     string
	PasswordHash string
	PlayerID     int64
}

// Session 表示由不透明令牌标识的登录会话。
type Session struct {
	Token     string
	PlayerID  int64
	ExpiresAt time.Time
}

type AuthorizeResult struct {
	Session *Session
	Player  *player.Player
}

// RegisterInput 包含注册账号及创建玩家档案所需的信息。
type RegisterInput struct {
	Username      string
	PlainPassword string
	Nickname      string
	Avatar        string
	Email         string
	Phone         string
}

// RegisterAccountInput 包含原子注册所需的已校验账号、玩家和会话数据。
type RegisterAccountInput struct {
	Username         string
	PasswordHash     string
	Nickname         string
	Avatar           string
	Email            string
	Phone            string
	SessionToken     string
	SessionExpiresAt time.Time
}

// RegisterAccountResult 包含注册过程创建的账号、玩家和会话。
type RegisterAccountResult struct {
	Account *Account
	Player  *player.Player
	Session *Session
}

// LoginInput 包含登录时提交的账号凭据。
type LoginInput struct {
	Username      string
	PlainPassword string
}
