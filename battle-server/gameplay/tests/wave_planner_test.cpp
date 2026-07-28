#include "gameplay/wave_planner.hpp"

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
    EXPECT_FLOAT_EQ(configs[0].attack.range, 1.0f);
    EXPECT_FLOAT_EQ(configs[0].attack.cooldown_seconds.count(), 1.0f);
    EXPECT_FLOAT_EQ(configs[0].attack.projectile_speed, 0.0f);
}

TEST(WavePlannerTest, DefaultWaveConfigCreatesTenIncreasingWaves) {
    auto config = default_wave_config();

    ASSERT_EQ(config.waves.size(), 10);
    ASSERT_EQ(config.waves[0].groups.size(), 1);
    ASSERT_EQ(config.waves[9].groups.size(), 1);
    EXPECT_EQ(config.waves[0].groups[0].count, 3);
    EXPECT_EQ(config.waves[9].groups[0].count, 12);
    EXPECT_GT(config.waves[9].health_multiplier, config.waves[0].health_multiplier);
}

}  // namespace
}  // namespace battle
