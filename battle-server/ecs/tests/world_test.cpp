#include "ecs/world.hpp"

#include <cmath>

#include <gtest/gtest.h>

namespace battle::ecs {
namespace {

CreatePlayerConfig default_player_config() {
    return {
        .x_position = 10.0f,
        .y_position = 20.0f,
        .max_health = 100,
        .move_speed = 5.0f,
    };
}

CreateMonsterConfig default_monster_config() {
    return {
        .x_position = 30.0f,
        .y_position = 40.0f,
        .max_health = 50,
        .move_speed = 3.0f,
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

TEST(WorldTest, CreatePlayerAllowsMultiplePlayerControlledEntities) {
    World world;

    auto first = world.create_player(default_player_config());
    auto second = world.create_player(default_player_config());

    EXPECT_NE(first, INVALID_ENTITY);
    EXPECT_NE(second, INVALID_ENTITY);
    EXPECT_NE(first, second);
    EXPECT_TRUE(world.player_controllers().has(first));
    EXPECT_TRUE(world.player_controllers().has(second));
}

TEST(WorldTest, SetMoveInputReturnsFalseForUnknownEntity) {
    World world;

    EXPECT_FALSE(world.set_move_input(404, 1.0f, 0.0f));
}

TEST(WorldTest, TickMovesPlayerByInputDirectionAndMoveSpeed) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(entity, 1.0f, 0.0f));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 15.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickUsesDeltaSecondsOnce) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(entity, 1.0f, 0.0f));
    world.tick(DeltaTime{0.5f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 12.5f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
}

TEST(WorldTest, TickMovesPlayerInNegativeInputDirection) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(entity, -1.0f, 0.0f));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 5.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, -1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickNormalizesDiagonalMoveInput) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(entity, 1.0f, 1.0f));
    world.tick(DeltaTime{1.0f});

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

    ASSERT_TRUE(world.set_move_input(entity, 1.0f, 0.0f));
    world.tick(DeltaTime{1.0f});
    ASSERT_TRUE(world.set_move_input(entity, 0.0f, 0.0f));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 15.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
    EXPECT_FALSE(std::isnan(transform.position.x));
    EXPECT_FALSE(std::isnan(transform.position.y));
}

TEST(WorldTest, DestroyPlayerRemovesEntityAndPlayerComponents) {
    World world;
    auto entity = world.create_player(default_player_config());

    EXPECT_TRUE(world.destroy_entity(entity));

    EXPECT_FALSE(world.has_entity(entity));
    EXPECT_FALSE(world.transforms().has(entity));
    EXPECT_FALSE(world.player_controllers().has(entity));
    EXPECT_FALSE(world.set_move_input(entity, 1.0f, 0.0f));
}

TEST(WorldTest, DestroyUnknownEntityReturnsFalse) {
    World world;

    EXPECT_FALSE(world.destroy_entity(999));
}

TEST(WorldTest, DestroyEntityReturnsFalseWhenCalledTwice) {
    World world;
    auto entity = world.create_player(default_player_config());

    EXPECT_TRUE(world.destroy_entity(entity));
    EXPECT_FALSE(world.destroy_entity(entity));
}

TEST(WorldTest, CreateMonsterReturnsLiveEntityWithInitialTransform) {
    World world;

    auto entity = world.create_monster(default_monster_config());

    EXPECT_NE(entity, INVALID_ENTITY);
    EXPECT_TRUE(world.has_entity(entity));

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 30.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 40.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
    EXPECT_FALSE(world.player_controllers().has(entity));
}

TEST(WorldTest, MonsterDoesNotHaveMoveInput) {
    World world;
    auto entity = world.create_monster(default_monster_config());

    EXPECT_FALSE(world.set_move_input(entity, 1.0f, 0.0f));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 30.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 40.0f);
}

TEST(WorldTest, DestroyMonsterRemovesEntityComponents) {
    World world;
    auto entity = world.create_monster(default_monster_config());

    EXPECT_TRUE(world.destroy_entity(entity));

    EXPECT_FALSE(world.has_entity(entity));
    EXPECT_FALSE(world.transforms().has(entity));
}

TEST(WorldTest, SnapshotReturnsEmptyEntitiesForEmptyWorld) {
    World world;

    auto snapshot = world.snapshot();

    EXPECT_TRUE(snapshot.entities.empty());
}

TEST(WorldTest, SnapshotIncludesMovedPlayerTransformAndHealth) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_move_input(entity, 1.0f, 0.0f));
    world.tick(DeltaTime{1.0f});

    auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 1);
    const auto& entity_snapshot = snapshot.entities[0];
    EXPECT_EQ(entity_snapshot.entity, entity);
    EXPECT_FLOAT_EQ(entity_snapshot.x_position, 15.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.y_position, 20.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.x_direction, 1.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.y_direction, 0.0f);
    EXPECT_EQ(entity_snapshot.current_health, 100);
    EXPECT_EQ(entity_snapshot.max_health, 100);
}

TEST(WorldTest, SnapshotIncludesPlayersAndMonsters) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_TRUE((snapshot.entities[0].entity == player && snapshot.entities[1].entity == monster) ||
                (snapshot.entities[0].entity == monster && snapshot.entities[1].entity == player));
}

TEST(WorldTest, SnapshotExcludesDestroyedEntities) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    ASSERT_TRUE(world.destroy_entity(player));

    auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 1);
    EXPECT_EQ(snapshot.entities[0].entity, monster);
}

}  // namespace
}  // namespace battle::ecs
