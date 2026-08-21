#include "runtime/battle_instance.hpp"

#include <utility>

#include <gtest/gtest.h>

#include "gameplay/gameplay_config.hpp"

namespace battle {
namespace {

std::unique_ptr<BattleInstance> create_test_instance(BattleInstanceConfig config) {
    auto instance = BattleInstance::create(std::move(config));
    EXPECT_NE(instance, nullptr);
    return instance;
}

TEST(BattleInstanceTest, CreateBuildsPlayersAtRoomLayoutSpawns) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .player_ids = {1001, 1002},
    });

    auto snapshot = instance->snapshot();
    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_FLOAT_EQ(snapshot.entities[0].position.x, 0.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[0].position.y, gameplay_config::room::PlayerSpawnY);
    EXPECT_FLOAT_EQ(snapshot.entities[1].position.x,
                    -2.0f * gameplay_config::combat::DefaultCharacterCollisionRadius);
    EXPECT_FLOAT_EQ(snapshot.entities[1].position.y,
                    gameplay_config::room::PlayerSpawnY +
                        2.0f * gameplay_config::combat::DefaultCharacterCollisionRadius);
}

TEST(BattleInstanceTest, CreateEntersConfiguredInitialRoom) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .dungeon_room_graph = DungeonRoomGraph{
            .start_room_id = 42,
            .rooms = {
                DungeonRoomNode{
                    .room_id = 42,
                    .kind = DungeonRoomKind::Start,
                    .layout_id = "start",
                    .next_room_ids = {43},
                },
                DungeonRoomNode{
                    .room_id = 43,
                    .kind = DungeonRoomKind::Boss,
                    .layout_id = "boss_small",
                    .encounter = RoomEncounter{
                        .monster_groups = {
                            RoomMonsterGroup{.kind = MonsterKind::Melee, .count = 1},
                        },
                    },
                },
            },
        },
    });

    EXPECT_EQ(instance->current_room_id(), 42);
    EXPECT_EQ(instance->room_state(), RoomFlowState::Fighting);
}

TEST(BattleInstanceTest, CreateRejectsMissingInitialRoom) {
    auto instance = BattleInstance::create({
        .room_name = "room-1",
        .dungeon_room_graph = DungeonRoomGraph{
            .start_room_id = 42,
        },
    });

    EXPECT_EQ(instance, nullptr);
}

TEST(BattleInstanceTest, SnapshotUsesConfiguredTickRate) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .tick_rate = 30,
    });

    EXPECT_EQ(instance->snapshot().tick_rate, 30);
}

TEST(BattleInstanceTest, SnapshotIncludesEveryPlayerHero) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .player_ids = {1001, 1002},
        .player_loadouts = {
            {1001, std::make_pair(HeroKind::Rock, GrowthLevels{})},
            {1002, std::make_pair(HeroKind::Nature, GrowthLevels{})},
        },
    });

    const auto snapshot = instance->snapshot();
    ASSERT_EQ(snapshot.entities.size(), 2);
    ASSERT_TRUE(snapshot.entities[0].hero.has_value());
    ASSERT_TRUE(snapshot.entities[1].hero.has_value());
    EXPECT_EQ(snapshot.entities[0].hero.value(), HeroKind::Rock);
    EXPECT_EQ(snapshot.entities[1].hero.value(), HeroKind::Nature);
}

TEST(BattleInstanceTest, SnapshotFallsBackToDefaultTickRateForZeroConfig) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .tick_rate = 0,
    });

    EXPECT_EQ(instance->snapshot().tick_rate, DefaultBattleTickRate);
}

TEST(BattleInstanceTest, ReceiveInputReturnsFalseForUnknownPlayer) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .player_ids = {1001},
    });

    EXPECT_FALSE(instance->receive_input(404, PlayerInput{
                                                 .move_x = 1.0f,
                                                 .move_y = 0.0f,
                                             }));
}

TEST(BattleInstanceTest, ChooseBlessingReturnsFalseOutsideRewardSelection) {
    auto instance = create_test_instance({
        .room_name = "room-1",
        .player_ids = {1001},
    });

    EXPECT_FALSE(instance->choose_blessing(1001, 0));
}

}  // namespace
}  // namespace battle
