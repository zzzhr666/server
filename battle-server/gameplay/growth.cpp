#include "growth.hpp"
#include <cmath>
#include "gameplay_config.hpp"

battle::ecs::CreatePlayerConfig battle::apply_growth(ecs::CreatePlayerConfig base, const GrowthLevels& levels) {
    const float attack_speed_multiplier = 1 + (static_cast<float>(levels.attack_speed_level) - 1) *
        gameplay_config::growth::AttackSpeedIncreasePerLevel;
    base.attack.damage = static_cast<int>(std::lround(static_cast<float>(base.attack.damage) *
        ((static_cast<float>(levels.attack_level) - 1) * gameplay_config::growth::AttackIncreasePerLevel + 1)));
    base.attack.cooldown_seconds = base.attack.cooldown_seconds / attack_speed_multiplier;
    base.attack.windup_seconds = base.attack.windup_seconds / attack_speed_multiplier;
    base.attack.active_seconds = base.attack.active_seconds / attack_speed_multiplier;
    base.attack.recovery_seconds = base.attack.recovery_seconds / attack_speed_multiplier;
    base.max_health = static_cast<int>(std::lround(static_cast<float>(base.max_health) *
        (1 + (static_cast<float>(levels.health_level) - 1) * gameplay_config::growth::HealthIncreasePerLevel)));
    base.move_speed = base.move_speed * (gameplay_config::growth::MoveSpeedIncreasePerLevel *
        (static_cast<float>(levels.move_speed_level) - 1) + 1);

    return base;
}
