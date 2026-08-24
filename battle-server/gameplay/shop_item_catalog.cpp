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
        }
    };
}

std::vector<battle::ShopOffer> battle::default_shop_offers() {
    return {
        ShopOffer{
            .item_id = 1001,
            .price = 30,
        },
        ShopOffer{
            .item_id = 1002,
            .price = 25,
        },
        ShopOffer{
            .item_id = 1003,
            .price = 30,
        },
    };
}
