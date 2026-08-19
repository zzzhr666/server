#include "growth.hpp"
#include <cmath>

battle::ecs::CreatePlayerConfig battle::apply_growth(ecs::CreatePlayerConfig base, const GrowthLevels& levels) {
    const float attack_speed_multiplier = 1 + (static_cast<float>(levels.attack_speed_level) - 1) *
        GrowthConfig::attack_speed_incr_percent;
    base.attack.damage = static_cast<int>(std::lround(static_cast<float>(base.attack.damage) *
        ((static_cast<float>(levels.attack_level) - 1) * GrowthConfig::attack_incr_percent + 1)));
    base.attack.cooldown_seconds = base.attack.cooldown_seconds / attack_speed_multiplier;
    base.attack.windup_seconds = base.attack.windup_seconds / attack_speed_multiplier;
    base.attack.active_seconds = base.attack.active_seconds / attack_speed_multiplier;
    base.attack.recovery_seconds = base.attack.recovery_seconds / attack_speed_multiplier;
    base.max_health = static_cast<int>(std::lround(static_cast<float>(base.max_health) *
        (1 + (static_cast<float>(levels.health_level) - 1) * GrowthConfig::health_incr_percent)));
    base.move_speed = base.move_speed * (GrowthConfig::move_speed_incr_percent *
        (static_cast<float>(levels.move_speed_level) - 1) + 1);

    return base;
}
