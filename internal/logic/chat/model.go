package chat

import "time"

type ChannelType string

const (
	ChannelWorld  ChannelType = "world"
	ChannelDirect ChannelType = "direct"
)

type Message struct {
	MessageKey       string
	ChannelType      ChannelType
	ChannelKey       string
	SenderID         int64
	ReceiverID       int64
	Content          string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	ClientMessageKey string
	SenderNickname   string
}

type SendWorldMessageInput struct {
	SenderID         int64
	Content          string
	ClientMessageKey string
	SenderNickname   string
}

// SendDirectMessageInput 描述发送私聊消息的请求。
type SendDirectMessageInput struct {
	SenderID         int64
	ReceiverID       int64
	Content          string
	ClientMessageKey string
	SenderNickname   string
}

// ListWorldMessagesInput 描述读取世界频道历史消息的请求。
type ListWorldMessagesInput struct {
	PlayerID         int64
	Limit            int64
	BeforeMessageKey string
}

// ListDirectMessagesInput 描述读取私聊历史消息的请求。
type ListDirectMessagesInput struct {
	PlayerID         int64
	FriendID         int64
	Limit            int64
	BeforeMessageKey string
}

// SaveMessageInput 描述聊天仓储保存消息所需的数据。
type SaveMessageInput struct {
	ChannelType      ChannelType
	ChannelKey       string
	SenderID         int64
	ReceiverID       int64
	Content          string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	MaxMessages      int64
	ClientMessageKey string
	SenderNickname   string
}

// ListMessagesInput 描述聊天仓储查询消息所需的数据。
type ListMessagesInput struct {
	ChannelType      ChannelType
	ChannelKey       string
	Limit            int64
	BeforeMessageKey string
}
