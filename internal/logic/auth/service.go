package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"server/internal/logic/player"
	"server/internal/platform/logging"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service 定义账号认证与会话管理操作。
type Service interface {
	// Register 创建账号和绑定玩家，并返回登录会话。
	Register(ctx context.Context, input RegisterInput) (*AuthorizeResult, error)
	// Login 校验凭据并创建新的登录会话。
	Login(ctx context.Context, input LoginInput) (*AuthorizeResult, error)
	// Logout 按令牌删除登录会话。
	Logout(ctx context.Context, token string) error
	// GetSession 按令牌返回有效的登录会话。
	GetSession(ctx context.Context, token string) (*Session, error)
	// GetCurrentPlayer 返回有效会话令牌绑定的玩家。
	GetCurrentPlayer(ctx context.Context, token string) (*player.Player, error)
}

// Repository 定义认证服务依赖的持久化操作。
type Repository interface {
	// RegisterAccount 在一次状态操作中创建账号、玩家和首个会话。
	RegisterAccount(ctx context.Context, input RegisterAccountInput) (*RegisterAccountResult, error)
	// GetAccount 按用户名读取账号。
	GetAccount(ctx context.Context, username string) (*Account, error)
	// CreateSession 持久化带过期时间的登录会话。
	CreateSession(ctx context.Context, session *Session) error
	// GetSession 按令牌读取未过期会话。
	GetSession(ctx context.Context, token string) (*Session, error)
	// DeleteSession 删除一个会话令牌。
	DeleteSession(ctx context.Context, token string) error
}

// NewService 使用认证仓储、玩家服务和会话 TTL 创建认证服务。
func NewService(authRepo Repository, playerService player.Service, sessionTTL time.Duration) *GameAuthService {
	return &GameAuthService{
		authRepo:      authRepo,
		playerService: playerService,
		sessionTTL:    sessionTTL,
	}
}

// GameAuthService 实现账号注册、登录和会话规则。
type GameAuthService struct {
	authRepo      Repository
	playerService player.Service
	sessionTTL    time.Duration
}

// Register 创建绑定玩家的账号，并立即创建登录会话。
func (g *GameAuthService) Register(ctx context.Context, input RegisterInput) (*AuthorizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Username == "" {
		return nil, ErrInvalidUsername
	}
	if input.PlainPassword == "" {
		return nil, ErrInvalidPassword
	}
	if input.Nickname == "" {
		return nil, player.ErrInvalidNickname
	}

	// 明文密码只在当前调用栈中存在；写入 state-server 的 RegisterAccountInput 仅包含
	// bcrypt 哈希。账号、玩家和首个会话由状态层原子创建，避免注册成功后无法登录。
	passwordHash, err := hashPassword(input.PlainPassword)
	if err != nil {
		return nil, err
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	result, err := g.authRepo.RegisterAccount(ctx, RegisterAccountInput{
		Username:         input.Username,
		PasswordHash:     passwordHash,
		Nickname:         input.Nickname,
		Avatar:           input.Avatar,
		Email:            input.Email,
		Phone:            input.Phone,
		SessionToken:     token,
		SessionExpiresAt: time.Now().Add(g.sessionTTL),
	})
	if err != nil {
		logging.Error("register account failed username=%s: %v", input.Username, err)
		return nil, err
	}
	logging.Info("account registered username=%s player_id=%d", input.Username, result.Player.ID)
	return &AuthorizeResult{
		Session: result.Session,
		Player:  result.Player,
	}, nil
}

// Login 校验用户名和密码后创建新的登录会话。
func (g *GameAuthService) Login(ctx context.Context, input LoginInput) (*AuthorizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Username == "" {
		return nil, ErrInvalidUsername
	}
	if input.PlainPassword == "" {
		return nil, ErrInvalidPassword
	}
	// 对账号不存在和密码不匹配返回同一个错误，避免通过登录接口枚举已注册用户名。
	account, err := g.authRepo.GetAccount(ctx, input.Username)
	if errors.Is(err, ErrAccountNotFound) {
		logging.Warn("login rejected username=%s reason=invalid_credentials", input.Username)
		return nil, ErrInvalidCredentials
	} else if err != nil {
		logging.Error("load account failed username=%s: %v", input.Username, err)
		return nil, err
	}
	if correct := checkPassword(account.PasswordHash, input.PlainPassword); !correct {
		logging.Warn("login rejected username=%s reason=invalid_credentials", input.Username)
		return nil, ErrInvalidCredentials
	}

	p, err := g.playerService.Get(ctx, account.PlayerID)
	if err != nil {
		logging.Error("load player for login failed player_id=%d: %v", account.PlayerID, err)
		return nil, err
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	session := &Session{
		Token:     token,
		PlayerID:  account.PlayerID,
		ExpiresAt: time.Now().Add(g.sessionTTL),
	}

	// 每次登录创建独立随机令牌，不复用旧 session；TTL 由 state-server/Redis 执行，
	// 即使 logic-server 重启也不会让过期会话重新有效。
	if err := g.authRepo.CreateSession(ctx, session); err != nil {
		logging.Error("create session failed player_id=%d: %v", account.PlayerID, err)
		return nil, err
	}
	logging.Info("login succeeded player_id=%d", account.PlayerID)

	return &AuthorizeResult{
		Session: session,
		Player:  p,
	}, nil
}

// Logout 删除令牌标识的会话。
func (g *GameAuthService) Logout(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return ErrSessionNotFound
	}
	if err := g.authRepo.DeleteSession(ctx, token); err != nil {
		logging.Error("logout failed: %v", err)
		return err
	}
	logging.Info("logout succeeded")
	return nil
}

// GetSession 返回非空令牌对应的已存储会话。
func (g *GameAuthService) GetSession(ctx context.Context, token string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if token == "" {
		return nil, ErrSessionNotFound
	}
	return g.authRepo.GetSession(ctx, token)
}

// GetCurrentPlayer 返回有效会话令牌绑定的玩家。
func (g *GameAuthService) GetCurrentPlayer(ctx context.Context, token string) (*player.Player, error) {
	session, err := g.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return g.playerService.Get(ctx, session.PlayerID)
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func checkPassword(passwordHash, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plainPassword))
	return err == nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
