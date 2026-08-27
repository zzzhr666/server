#include "shop_item_catalog.hpp"

std::vector<battle::ShopItemDefinition> battle::default_shop_item_definitions() {
    return {
        ShopItemDefinition{
            .item_id = 1001,
            .item_name = "重甲",
            .buffs = {
                {.kind = ShopBuffKind::MaxHealth, .value = 120.0f},
                {.kind = ShopBuffKind::Armor, .value = 20.0f},
                {.kind = ShopBuffKind::MoveSpeed, .value = -1.5f},
            }
        },
        ShopItemDefinition{
            .item_id = 1002,
            .item_name = "飞鞋",
            .buffs = {
                {.kind = ShopBuffKind::Armor, .value = -5.0f},
                {.kind = ShopBuffKind::MoveSpeed, .value = 5.0f},
            }
        },
        ShopItemDefinition{
            .item_id = 1003,
            .item_name = "护手",
            .buffs = {
                {.kind = ShopBuffKind::AttackDamage, .value = 10.0f},
                {.kind = ShopBuffKind::MaxHealth, .value = 20.0f},
            }
        },
        ShopItemDefinition{
            .item_id = 1004, .item_name = "锋刃", .buffs = {
                {.kind = ShopBuffKind::AttackDamage, .value = 18.0f}
            }
        },
        ShopItemDefinition{
            .item_id = 1005, .item_name = "生命核心", .buffs = {{ShopBuffKind::MaxHealth, 180.0f}}
        },
        ShopItemDefinition{
            .item_id = 1006, .item_name = "钢铁壁垒",
            .buffs = {{ShopBuffKind::Armor, 30.0f}, {ShopBuffKind::MoveSpeed, -1.0f}}
        },
        ShopItemDefinition{
            .item_id = 1007, .item_name = "迅捷之魂",
            .buffs = {
                {.kind = ShopBuffKind::MoveSpeed, .value = 3.0f},
                {.kind = ShopBuffKind::AttackDamage, .value = 12.0f}
            }
        },
        ShopItemDefinition{
            .item_id = 1008, .item_name = "战争圣物", .buffs = {
                {.kind = ShopBuffKind::AttackDamage, .value = 35.0f},
                {.kind = ShopBuffKind::MaxHealth, .value = 220.0f},
                {.kind = ShopBuffKind::Armor, .value = 15.0f}
            }
        },
    };
}

std::vector<battle::ShopOffer> battle::default_shop_offers() {
    return early_reward_shop_offers();
}

std::vector<battle::ShopOffer> battle::early_reward_shop_offers() {
    return {{1001, 55}, {1002, 45}, {1003, 60}, {1004, 85}};
}

std::vector<battle::ShopOffer> battle::late_reward_shop_offers() {
    return {{1005, 130}, {1006, 160}, {1007, 145}, {1008, 240}};
}
