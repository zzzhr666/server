#include "gameplay/spawn_planner.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(SpawnPlannerTest, PlayerSpawnPlacesFirstFourPlayersAroundCenter) {
    SpawnPlanner planner;

    auto first = planner.player_spawn(0);
    auto second = planner.player_spawn(1);
    auto third = planner.player_spawn(2);
    auto fourth = planner.player_spawn(3);

    EXPECT_FLOAT_EQ(first.position.x, -2.0f);
    EXPECT_FLOAT_EQ(first.position.y, 0.0f);

    EXPECT_FLOAT_EQ(second.position.x, 2.0f);
    EXPECT_FLOAT_EQ(second.position.y, 0.0f);

    EXPECT_FLOAT_EQ(third.position.x, 0.0f);
    EXPECT_FLOAT_EQ(third.position.y, -2.0f);

    EXPECT_FLOAT_EQ(fourth.position.x, 0.0f);
    EXPECT_FLOAT_EQ(fourth.position.y, 2.0f);
}

TEST(SpawnPlannerTest, PlayerSpawnReusesFourPlayerSlots) {
    SpawnPlanner planner;

    auto spawn = planner.player_spawn(4);

    EXPECT_FLOAT_EQ(spawn.position.x, -2.0f);
    EXPECT_FLOAT_EQ(spawn.position.y, 0.0f);
}

TEST(SpawnPlannerTest, PlayerSpawnUsesTunedStatsWithoutReducingHealth) {
    SpawnPlanner planner;

    auto spawn = planner.player_spawn(0);

    EXPECT_EQ(spawn.max_health, 1000);
    EXPECT_FLOAT_EQ(spawn.move_speed, 11.0f);
    EXPECT_EQ(spawn.attack.damage, 23);
    EXPECT_FLOAT_EQ(spawn.attack.cooldown_seconds.count(), 0.34f);
}

TEST(SpawnPlannerTest, MonsterSpawnPlacesMonstersOnCircle) {
    SpawnPlanner planner;

    auto first = planner.monster_spawn(0, 4);
    auto second = planner.monster_spawn(1, 4);
    auto third = planner.monster_spawn(2, 4);
    auto fourth = planner.monster_spawn(3, 4);

    EXPECT_NEAR(first.position.x, 8.0f, 0.001f);
    EXPECT_NEAR(first.position.y, 0.0f, 0.001f);

    EXPECT_NEAR(second.position.x, 0.0f, 0.001f);
    EXPECT_NEAR(second.position.y, 8.0f, 0.001f);

    EXPECT_NEAR(third.position.x, -8.0f, 0.001f);
    EXPECT_NEAR(third.position.y, 0.0f, 0.001f);

    EXPECT_NEAR(fourth.position.x, 0.0f, 0.001f);
    EXPECT_NEAR(fourth.position.y, -8.0f, 0.001f);
}

TEST(SpawnPlannerTest, MonsterSpawnTreatsZeroCountAsOne) {
    SpawnPlanner planner;

    auto spawn = planner.monster_spawn(0, 0);

    EXPECT_NEAR(spawn.position.x, 8.0f, 0.001f);
    EXPECT_NEAR(spawn.position.y, 0.0f, 0.001f);
}

}  // namespace
}  // namespace battle
