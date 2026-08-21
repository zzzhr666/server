#pragma once

#include "ecs/time.hpp"
#include "gameplay/gameplay_config.hpp"

namespace battle::ecs {
    constexpr int normalized_blessing_level(int level) {
        return level < 1 ? 1 : level;
    }

    constexpr int critical_strike_percent(int level) {
        return gameplay_config::blessing::critical_strike::BasePercent + (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::critical_strike::PercentPerLevel;
    }

    constexpr int critical_strike_damage_percent(int level) {
        return gameplay_config::blessing::critical_strike::BaseDamagePercent +
            (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::critical_strike::DamagePercentPerLevel;
    }

    constexpr int life_steal_percent(int level) {
        return gameplay_config::blessing::life_steal::BasePercent + (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::life_steal::PercentPerLevel;
    }

    constexpr int burn_damage_per_tick(int level) {
        return gameplay_config::blessing::burn_on_hit::BaseDamagePerTick +
            (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::burn_on_hit::DamagePerTickPerLevel;
    }

    constexpr DeltaTime burn_duration_seconds(int level) {
        return gameplay_config::blessing::burn_on_hit::BaseDuration +
            gameplay_config::blessing::burn_on_hit::DurationPerLevel *
            static_cast<float>(normalized_blessing_level(level) - 1);
    }

    constexpr int freeze_percent(int level) {
        return gameplay_config::blessing::freeze_on_hit::BasePercent +
            (normalized_blessing_level(level) - 1) * gameplay_config::blessing::freeze_on_hit::PercentPerLevel;
    }

    constexpr DeltaTime freeze_duration_seconds(int level) {
        return gameplay_config::blessing::freeze_on_hit::BaseDuration +
            gameplay_config::blessing::freeze_on_hit::DurationPerLevel *
            static_cast<float>(normalized_blessing_level(level) - 1);
    }

    constexpr int chain_lightning_damage_percent(int level) {
        return gameplay_config::blessing::chain_lightning::BaseDamagePercent +
            gameplay_config::blessing::chain_lightning::DamagePercentPerLevel *
            (normalized_blessing_level(level) - 1);
    }

    constexpr int chain_lightning_target_count(int level) {
        return gameplay_config::blessing::chain_lightning::BaseSecondaryTargets +
            (level - 1) / gameplay_config::blessing::chain_lightning::LevelsPerExtraTarget;
    }
}
