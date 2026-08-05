#pragma once


#include "ecs/world.hpp"

namespace battle {
    struct GrowthConfig {
        static constexpr float attack_incr_percent = 0.08f;
        static constexpr float attack_speed_incr_percent = 0.06f;
        static constexpr float health_incr_percent = 0.1f;
        static constexpr float move_speed_incr_percent = 0.03f;
    };
    struct GrowthLevels {
        std::int32_t attack_level = 1;
        std::int32_t attack_speed_level = 1;
        std::int32_t health_level = 1;
        std::int32_t move_speed_level = 1;
    };

    ecs::CreatePlayerConfig apply_growth(ecs::CreatePlayerConfig base, const GrowthLevels& levels);
}