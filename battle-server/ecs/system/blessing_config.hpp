#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    constexpr int normalized_blessing_level(int level) {
        return level < 1 ? 1 : level;
    }

    struct CriticalStrikeConfig {
        static constexpr int BasePercent = 25;
        static constexpr int PercentPerLevel = 5;
        static constexpr int DamageMultiplier = 2;
    };

    constexpr int critical_strike_percent(int level) {
        return CriticalStrikeConfig::BasePercent + (normalized_blessing_level(level) - 1) *
            CriticalStrikeConfig::PercentPerLevel;
    }

    struct LifeStealConfig {
        static constexpr int BasePercent = 50;
        static constexpr int PercentPerLevel = 10;
    };

    constexpr int life_steal_percent(int level) {
        return LifeStealConfig::BasePercent + (normalized_blessing_level(level) - 1) * LifeStealConfig::PercentPerLevel;
    }

    struct BurnOnHitConfig {
        static constexpr int BaseDamagePerTick = 5;
        static constexpr int DamagePerTickPerLevel = 3;
        static constexpr DeltaTime BaseDurationSeconds = DeltaTime{3.0f};
        static constexpr DeltaTime DurationSecondsPerLevel = DeltaTime{0.5f};
        static constexpr DeltaTime TickIntervalSeconds = DeltaTime{1.0f};
    };

    constexpr int burn_damage_per_tick(int level) {
        return BurnOnHitConfig::BaseDamagePerTick +
            (normalized_blessing_level(level) - 1) * BurnOnHitConfig::DamagePerTickPerLevel;
    }

    constexpr DeltaTime burn_duration_seconds(int level) {
        return BurnOnHitConfig::BaseDurationSeconds +
            BurnOnHitConfig::DurationSecondsPerLevel * static_cast<float>(normalized_blessing_level(level) - 1);
    }

    struct FreezeOnHitConfig {
        static constexpr int BasePercent = 50;
        static constexpr int PercentPerLevel = 10;
        static constexpr DeltaTime BaseDurationSeconds = DeltaTime{2.0f};
        static constexpr DeltaTime DurationSecondsPerLevel = DeltaTime{0.25f};
    };

    constexpr int freeze_percent(int level) {
        return FreezeOnHitConfig::BasePercent +
            (normalized_blessing_level(level) - 1) * FreezeOnHitConfig::PercentPerLevel;
    }

    constexpr DeltaTime freeze_duration_seconds(int level) {
        return FreezeOnHitConfig::BaseDurationSeconds +
            FreezeOnHitConfig::DurationSecondsPerLevel * static_cast<float>(normalized_blessing_level(level) - 1);
    }
}
