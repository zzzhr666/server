#include "gameplay/growth.hpp"
#include "gameplay/hero.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(GrowthTest, LevelOnePreservesCompleteAttackTimeline) {
    ecs::CreatePlayerConfig base{
        .attack = hero_definition(HeroKind::Fire).attack,
    };

    const auto result = apply_growth(base, GrowthLevels{});

    EXPECT_FLOAT_EQ(result.attack.cooldown_seconds.count(), base.attack.cooldown_seconds.count());
    EXPECT_FLOAT_EQ(result.attack.windup_seconds.count(), base.attack.windup_seconds.count());
    EXPECT_FLOAT_EQ(result.attack.active_seconds.count(), base.attack.active_seconds.count());
    EXPECT_FLOAT_EQ(result.attack.recovery_seconds.count(), base.attack.recovery_seconds.count());
    EXPECT_FLOAT_EQ(result.attack.movement_multiplier, base.attack.movement_multiplier);
}

TEST(GrowthTest, AttackSpeedLevelScalesCompleteTimelineBySameMultiplier) {
    ecs::CreatePlayerConfig base{
        .attack = hero_definition(HeroKind::Fire).attack,
    };
    constexpr std::int32_t AttackSpeedLevel = 6;
    constexpr float ExpectedMultiplier = 1.30f;

    const auto result = apply_growth(base, GrowthLevels{
                                               .attack_speed_level = AttackSpeedLevel,
                                           });

    EXPECT_NEAR(result.attack.cooldown_seconds.count(), base.attack.cooldown_seconds.count() / ExpectedMultiplier,
                0.000001f);
    EXPECT_NEAR(result.attack.windup_seconds.count(), base.attack.windup_seconds.count() / ExpectedMultiplier,
                0.000001f);
    EXPECT_NEAR(result.attack.active_seconds.count(), base.attack.active_seconds.count() / ExpectedMultiplier,
                0.000001f);
    EXPECT_NEAR(result.attack.recovery_seconds.count(), base.attack.recovery_seconds.count() / ExpectedMultiplier,
                0.000001f);
    EXPECT_FLOAT_EQ(result.attack.movement_multiplier, base.attack.movement_multiplier);

    const float timeline_seconds = result.attack.windup_seconds.count() + result.attack.active_seconds.count() +
                                   result.attack.recovery_seconds.count();
    EXPECT_LE(timeline_seconds, result.attack.cooldown_seconds.count() + 0.000001f);
}

}  // namespace
}  // namespace battle
