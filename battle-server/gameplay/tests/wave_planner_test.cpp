#include "gameplay/wave_planner.hpp"

#include <array>

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(WavePlannerTest, PlanWaveCreatesMonsterConfigsForGroupCount) {
    WavePlanner planner;

    auto configs = planner.plan_wave(WaveDefinition{
        .groups = {
            WaveMonsterGroup{
                .kind = MonsterKind::Melee,
                .count = 3,
            },
        },
    });

    EXPECT_EQ(configs.size(), 3);
}

TEST(WavePlannerTest, PlanWaveAppliesHealthAndMoveSpeedMultipliers) {
    WavePlanner planner;

    auto configs = planner.plan_wave(WaveDefinition{
        .groups = {
            WaveMonsterGroup{
                .kind = MonsterKind::Melee,
                .count = 1,
            },
        },
        .health_multiplier = 2.0f,
        .move_speed_multiplier = 1.5f,
    });

    ASSERT_EQ(configs.size(), 1);
    EXPECT_EQ(configs[0].max_health, 100);
    EXPECT_FLOAT_EQ(configs[0].move_speed, 4.5f);
}

TEST(WavePlannerTest, PlanWaveAppliesMonsterAttackDefinition) {
    WavePlanner planner;

    auto configs = planner.plan_wave(WaveDefinition{
        .groups = {
            WaveMonsterGroup{
                .kind = MonsterKind::Melee,
                .count = 1,
            },
        },
    });

    ASSERT_EQ(configs.size(), 1);
    EXPECT_EQ(configs[0].kind, MonsterKind::Melee);
    EXPECT_EQ(configs[0].attack.kind, ecs::AttackKind::Melee);
    EXPECT_EQ(configs[0].attack.damage, 10);
    EXPECT_FLOAT_EQ(configs[0].attack.range, 0.7f);
    EXPECT_FLOAT_EQ(configs[0].attack.cooldown_seconds.count(), 1.6f);
    EXPECT_FLOAT_EQ(configs[0].attack.windup_seconds.count(), 0.45f);
    EXPECT_FLOAT_EQ(configs[0].attack.active_seconds.count(), 0.10f);
    EXPECT_FLOAT_EQ(configs[0].attack.recovery_seconds.count(), 1.05f);
    EXPECT_FLOAT_EQ(configs[0].attack.movement_multiplier, 0.0f);
    EXPECT_FLOAT_EQ(configs[0].attack.projectile_speed, 0.0f);
}

TEST(WavePlannerTest, PlanWaveCarriesRangedMonsterKitingConfiguration) {
    WavePlanner planner;

    const auto configs = planner.plan_wave(WaveDefinition{
        .groups = {
            WaveMonsterGroup{
                .kind = MonsterKind::Ranged,
                .count = 1,
            },
        },
    });

    ASSERT_EQ(configs.size(), 1);
    EXPECT_EQ(configs[0].kind, MonsterKind::Ranged);
    EXPECT_EQ(configs[0].attack.kind, ecs::AttackKind::Projectile);
    EXPECT_FLOAT_EQ(configs[0].attack.range, 10.5f);
    EXPECT_FLOAT_EQ(configs[0].attack.cooldown_seconds.count(), 2.0f);
    EXPECT_FLOAT_EQ(configs[0].attack.windup_seconds.count(), 0.60f);
    EXPECT_FLOAT_EQ(configs[0].attack.active_seconds.count(), 0.05f);
    EXPECT_FLOAT_EQ(configs[0].attack.recovery_seconds.count(), 1.35f);
    EXPECT_FLOAT_EQ(configs[0].attack.movement_multiplier, 0.0f);
    EXPECT_FLOAT_EQ(configs[0].attack.projectile_speed, 11.0f);
    ASSERT_TRUE(configs[0].kiting_ai.has_value());
    EXPECT_FLOAT_EQ(configs[0].kiting_ai->retreat_distance, 7.0f);
}

TEST(WavePlannerTest, DefaultWaveConfigCreatesTenIncreasingWaves) {
    auto config = default_wave_config();
    constexpr std::array<std::size_t, 10> expected_melee_counts{
        7, 7, 8, 8, 9, 10, 11, 11, 12, 12,
    };
    constexpr std::array<std::size_t, 10> expected_ranged_counts{
        0, 1, 2, 3, 4, 4, 5, 6, 7, 8,
    };

    ASSERT_EQ(config.waves.size(), 10);
    for (std::size_t i = 0; i < expected_melee_counts.size(); ++i) {
        ASSERT_FALSE(config.waves[i].groups.empty()) << "wave " << i + 1;
        EXPECT_EQ(config.waves[i].groups[0].kind, MonsterKind::Melee) << "wave " << i + 1;
        EXPECT_EQ(config.waves[i].groups[0].count, expected_melee_counts[i]) << "wave " << i + 1;

        if (expected_ranged_counts[i] == 0) {
            EXPECT_EQ(config.waves[i].groups.size(), 1) << "wave " << i + 1;
            continue;
        }

        ASSERT_EQ(config.waves[i].groups.size(), 2) << "wave " << i + 1;
        EXPECT_EQ(config.waves[i].groups[1].kind, MonsterKind::Ranged) << "wave " << i + 1;
        EXPECT_EQ(config.waves[i].groups[1].count, expected_ranged_counts[i]) << "wave " << i + 1;
    }
    EXPECT_GT(config.waves[9].health_multiplier, config.waves[0].health_multiplier);
}

}  // namespace
}  // namespace battle
