#include "runtime/battle_instance.hpp"

#include <gtest/gtest.h>

namespace battle {
namespace {

TEST(BattleInstanceTest, ConstructorCreatesPlayersAtPlannedSpawns) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001, 1002},
    });

    auto snapshot = instance.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_EQ(snapshot.entities[0].entity, 1);
    EXPECT_FLOAT_EQ(snapshot.entities[0].x_position, -2.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[0].y_position, 0.0f);
    EXPECT_EQ(snapshot.entities[1].entity, 2);
    EXPECT_FLOAT_EQ(snapshot.entities[1].x_position, 2.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].y_position, 0.0f);
}

TEST(BattleInstanceTest, ReceiveInputReturnsFalseForUnknownPlayer) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
    });

    EXPECT_FALSE(instance.receive_input(404, 1.0f, 0.0f));
}

TEST(BattleInstanceTest, ReceiveInputAndTickMovesOnlyTargetPlayer) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001, 1002},
    });

    ASSERT_TRUE(instance.receive_input(1002, 1.0f, 0.0f));
    instance.tick(ecs::DeltaTime{1.0f});

    auto snapshot = instance.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_FLOAT_EQ(snapshot.entities[0].x_position, -2.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[0].y_position, 0.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].x_position, 7.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].y_position, 0.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].x_direction, 1.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].y_direction, 0.0f);
}

}  // namespace
}  // namespace battle
