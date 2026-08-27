#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace battle {
    struct ShopOffer {
        std::uint32_t item_id{};
        int price{};
    };

    enum class ShopBuffKind : std::uint8_t {
        AttackDamage,
        MaxHealth,
        Armor,
        MoveSpeed
    };

    struct ShopBuff {
        ShopBuffKind kind{};
        float value{};
    };

    struct ShopItemDefinition {
        std::uint32_t item_id{};
        std::string item_name{};
        std::vector<ShopBuff> buffs{};
    };
}
