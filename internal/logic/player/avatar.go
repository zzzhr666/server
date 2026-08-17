package player

import "strings"

// AvatarType 表示客户端内置资源使用的稳定头像标识。
type AvatarType string

const (
	// AvatarAdventurer 是冒险者头像标识。
	AvatarAdventurer AvatarType = "adventurer"
	// AvatarWarrior 是战士头像标识。
	AvatarWarrior AvatarType = "warrior"
	// AvatarMage 是法师头像标识。
	AvatarMage AvatarType = "mage"
	// AvatarPriest 是牧师头像标识。
	AvatarPriest AvatarType = "priest"
	// AvatarSummoner 是召唤师头像标识。
	AvatarSummoner AvatarType = "summoner"
)

// DefaultAvatar 是注册时未选择头像所使用的默认标识。
const DefaultAvatar = AvatarAdventurer

// ResolveAvatar 将客户端提交的字符串解析为受支持的头像标识。
func ResolveAvatar(value string) (AvatarType, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultAvatar, nil
	}
	switch value {
	case string(AvatarAdventurer):
		return AvatarAdventurer, nil
	case string(AvatarWarrior):
		return AvatarWarrior, nil
	case string(AvatarMage):
		return AvatarMage, nil
	case string(AvatarPriest):
		return AvatarPriest, nil
	case string(AvatarSummoner):
		return AvatarSummoner, nil
	}
	return DefaultAvatar, ErrInvalidAvatar
}
