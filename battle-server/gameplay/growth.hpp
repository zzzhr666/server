#pragma once


#include "ecs/world.hpp"

namespace battle {
    struct GrowthLevels {
        std::int32_t attack_level = 1;
        std::int32_t attack_speed_level = 1;
        std::int32_t health_level = 1;
        std::int32_t move_speed_level = 1;
    };

    /// @brief 将局外成长等级换算为玩家创建配置中的战斗属性。
    ecs::CreatePlayerConfig apply_growth(ecs::CreatePlayerConfig base, const GrowthLevels& levels);
}
