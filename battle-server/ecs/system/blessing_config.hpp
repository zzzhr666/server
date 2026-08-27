#pragma once

#include "ecs/time.hpp"
#include "gameplay/gameplay_config.hpp"

namespace battle::ecs {
    /// @brief 将祝福等级限制为至少一级，供数值公式统一使用。
    constexpr int normalized_blessing_level(int level) {
        return level < 1 ? 1 : level;
    }

    /// @brief 返回指定等级暴击祝福的触发概率。
    constexpr int critical_strike_percent(int level) {
        return gameplay_config::blessing::critical_strike::BasePercent + (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::critical_strike::PercentPerLevel;
    }

    /// @brief 返回指定等级暴击祝福的伤害倍率百分比。
    constexpr int critical_strike_damage_percent(int level) {
        return gameplay_config::blessing::critical_strike::BaseDamagePercent +
            (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::critical_strike::DamagePercentPerLevel;
    }

    /// @brief 返回指定等级吸血祝福的伤害转化比例。
    constexpr int life_steal_percent(int level) {
        return gameplay_config::blessing::life_steal::BasePercent + (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::life_steal::PercentPerLevel;
    }

    /// @brief 返回指定等级燃烧祝福的每跳伤害。
    constexpr int burn_damage_per_tick(int level) {
        return gameplay_config::blessing::burn_on_hit::BaseDamagePerTick +
            (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::burn_on_hit::DamagePerTickPerLevel;
    }

    /// @brief 返回指定等级燃烧祝福的持续时间。
    constexpr DeltaTime burn_duration_seconds(int level) {
        return gameplay_config::blessing::burn_on_hit::BaseDuration +
            gameplay_config::blessing::burn_on_hit::DurationPerLevel *
            static_cast<float>(normalized_blessing_level(level) - 1);
    }

    /// @brief 返回指定等级冰冻祝福的触发概率。
    constexpr int freeze_percent(int level) {
        return gameplay_config::blessing::freeze_on_hit::BasePercent +
            (normalized_blessing_level(level) - 1) * gameplay_config::blessing::freeze_on_hit::PercentPerLevel;
    }

    /// @brief 返回指定等级冰冻祝福的持续时间。
    constexpr DeltaTime freeze_duration_seconds(int level) {
        return gameplay_config::blessing::freeze_on_hit::BaseDuration +
            gameplay_config::blessing::freeze_on_hit::DurationPerLevel *
            static_cast<float>(normalized_blessing_level(level) - 1);
    }

    constexpr int freeze_damage_per_tick(int level) {
        return gameplay_config::blessing::freeze_on_hit::BaseDamagePerTick +
            (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::freeze_on_hit::DamagePerTickPerLevel;
    }

    /// @brief 返回指定等级连锁闪电的伤害比例。
    constexpr int chain_lightning_damage_percent(int level) {
        return gameplay_config::blessing::chain_lightning::BaseDamagePercent +
            gameplay_config::blessing::chain_lightning::DamagePercentPerLevel *
            (normalized_blessing_level(level) - 1);
    }

    /// @brief 返回指定等级连锁闪电的最大目标数量。
    constexpr int chain_lightning_target_count(int level) {
        return gameplay_config::blessing::chain_lightning::BaseSecondaryTargets +
            (normalized_blessing_level(level) - 1) / gameplay_config::blessing::chain_lightning::LevelsPerExtraTarget;
    }

    constexpr int frenzy_cooldown_reduction_percent(int level) {
        return gameplay_config::blessing::frenzy::CooldownReductionPercentPerLevel * normalized_blessing_level(level);
    }

    constexpr float swift_move_speed_increase(int level) {
        return gameplay_config::blessing::swift::MoveSpeedIncreasePerLevel * static_cast<float>(
            normalized_blessing_level(level));
    }

    constexpr int toughness_armor_increase(int level) {
        return gameplay_config::blessing::toughness::ArmorIncreasePerLevel * normalized_blessing_level(level);
    }

    constexpr int heavy_strike_extra_damage_percent(int level) {
        return gameplay_config::blessing::heavy_strike::ExtraDamageBasePercent + (normalized_blessing_level(level) - 1)
            * gameplay_config::blessing::heavy_strike::PercentPerLevel;
    }

    constexpr int revenge_extra_damage_percent(int level) {
        return gameplay_config::blessing::revenge::ExtraDamageBasePercent + (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::revenge::PercentPerLevel;
    }

    constexpr int armor_break_armors(int level) {
        return gameplay_config::blessing::armor_break::BaseArmorBreak + (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::armor_break::ArmorBreakPerLevel;
    }

    constexpr int soul_harvest_move_speed_increase_percent(int level) {
        return gameplay_config::blessing::soul_harvest::BaseMoveSpeedIncreasePercent +
            (normalized_blessing_level(level) - 1) *
            gameplay_config::blessing::soul_harvest::MoveSpeedIncreasePercentPerLevel;
    }
}
