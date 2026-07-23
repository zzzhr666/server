#include "ecs/world.hpp"

#include <cmath>

#include <gtest/gtest.h>

namespace battle::ecs {
namespace {

CreatePlayerConfig default_player_config() {
    return {
        .player_id = 1001,
        .x_position = 10.0f,
        .y_position = 20.0f,
        .max_health = 100,
        .move_speed = 5.0f,
    };
}

TEST(WorldTest, CreatePlayerReturnsLiveEntityWithInitialTransform) {
    World world;

    auto entity = world.create_player(default_player_config());

    EXPECT_NE(entity, INVALID_ENTITY);
    EXPECT_TRUE(world.has_entity(entity));

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
}

TEST(WorldTest, CreatePlayerRejectsDuplicatePlayerID) {
    World world;

    auto first = world.create_player(default_player_config());
    auto second = world.create_player(default_player_config());

    EXPECT_NE(first, INVALID_ENTITY);
    EXPECT_EQ(second, INVALID_ENTITY);
}

TEST(WorldTest, SetMoveInputReturnsFalseForUnknownPlayer) {
    World world;

    EXPECT_FALSE(world.set_move_input(404, 1.0f, 0.0f));
}

TEST(WorldTest, TickMovesPlayerByInputDirectionAndMoveSpeed) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(1001, 1.0f, 0.0f));
    world.tick(1.0f);

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 15.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickNormalizesDiagonalMoveInput) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(1001, 1.0f, 1.0f));
    world.tick(1.0f);

    const auto& transform = world.transforms().get(entity);
    const float expected_delta = 5.0f / std::sqrt(2.0f);
    EXPECT_NEAR(transform.position.x, 10.0f + expected_delta, 0.001f);
    EXPECT_NEAR(transform.position.y, 20.0f + expected_delta, 0.001f);
    EXPECT_NEAR(transform.direction.x, 1.0f / std::sqrt(2.0f), 0.001f);
    EXPECT_NEAR(transform.direction.y, 1.0f / std::sqrt(2.0f), 0.001f);
}

TEST(WorldTest, TickWithZeroMoveInputDoesNotMoveOrChangeDirection) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(1001, 1.0f, 0.0f));
    world.tick(1.0f);
    ASSERT_TRUE(world.set_move_input(1001, 0.0f, 0.0f));
    world.tick(1.0f);

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 15.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
    EXPECT_FALSE(std::isnan(transform.position.x));
    EXPECT_FALSE(std::isnan(transform.position.y));
}

}  // namespace
}  // namespace battle::ecs
