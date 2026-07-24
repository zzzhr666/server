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

    EXPECT_FLOAT_EQ(first.x_position, -2.0f);
    EXPECT_FLOAT_EQ(first.y_position, 0.0f);

    EXPECT_FLOAT_EQ(second.x_position, 2.0f);
    EXPECT_FLOAT_EQ(second.y_position, 0.0f);

    EXPECT_FLOAT_EQ(third.x_position, 0.0f);
    EXPECT_FLOAT_EQ(third.y_position, -2.0f);

    EXPECT_FLOAT_EQ(fourth.x_position, 0.0f);
    EXPECT_FLOAT_EQ(fourth.y_position, 2.0f);
}

TEST(SpawnPlannerTest, PlayerSpawnReusesFourPlayerSlots) {
    SpawnPlanner planner;

    auto spawn = planner.player_spawn(4);

    EXPECT_FLOAT_EQ(spawn.x_position, -2.0f);
    EXPECT_FLOAT_EQ(spawn.y_position, 0.0f);
}

TEST(SpawnPlannerTest, PlayerSpawnUsesDefaultStats) {
    SpawnPlanner planner;

    auto spawn = planner.player_spawn(0);

    EXPECT_EQ(spawn.max_health, 100);
    EXPECT_FLOAT_EQ(spawn.move_speed, 5.0f);
}

TEST(SpawnPlannerTest, MonsterSpawnPlacesMonstersOnCircle) {
    SpawnPlanner planner;

    auto first = planner.monster_spawn(0, 4);
    auto second = planner.monster_spawn(1, 4);
    auto third = planner.monster_spawn(2, 4);
    auto fourth = planner.monster_spawn(3, 4);

    EXPECT_NEAR(first.x_position, 8.0f, 0.001f);
    EXPECT_NEAR(first.y_position, 0.0f, 0.001f);

    EXPECT_NEAR(second.x_position, 0.0f, 0.001f);
    EXPECT_NEAR(second.y_position, 8.0f, 0.001f);

    EXPECT_NEAR(third.x_position, -8.0f, 0.001f);
    EXPECT_NEAR(third.y_position, 0.0f, 0.001f);

    EXPECT_NEAR(fourth.x_position, 0.0f, 0.001f);
    EXPECT_NEAR(fourth.y_position, -8.0f, 0.001f);
}

TEST(SpawnPlannerTest, MonsterSpawnTreatsZeroCountAsOne) {
    SpawnPlanner planner;

    auto spawn = planner.monster_spawn(0, 0);

    EXPECT_NEAR(spawn.x_position, 8.0f, 0.001f);
    EXPECT_NEAR(spawn.y_position, 0.0f, 0.001f);
}

TEST(SpawnPlannerTest, MonsterSpawnUsesDefaultStats) {
    SpawnPlanner planner;

    auto spawn = planner.monster_spawn(0, 1);

    EXPECT_EQ(spawn.max_health, 50);
    EXPECT_FLOAT_EQ(spawn.move_speed, 3.0f);
}

}  // namespace
}  // namespace battle
