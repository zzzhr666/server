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

    EXPECT_FALSE(instance.receive_input(404, PlayerInput{
                                                 .move_x = 1.0f,
                                                 .move_y = 0.0f,
                                             }));
}

TEST(BattleInstanceTest, ReceiveInputAndTickMovesOnlyTargetPlayer) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001, 1002},
        .wave_config = WaveConfig{},
    });

    ASSERT_TRUE(instance.receive_input(1002, PlayerInput{
                                                 .move_x = 1.0f,
                                                 .move_y = 0.0f,
                                             }));
    instance.tick(ecs::DeltaTime{1.0f});

    auto snapshot = instance.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_FLOAT_EQ(snapshot.entities[0].x_position, -2.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[0].y_position, 0.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].x_position, 9.5f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].y_position, 0.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].x_direction, 1.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[1].y_direction, 0.0f);
}

TEST(BattleInstanceTest, TickClampsPlayerToConfiguredWorldBounds) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{},
        .world_bounds = ecs::WorldBounds{
            .min_x = -1.0f,
            .max_x = 1.0f,
            .min_y = -1.0f,
            .max_y = 1.0f,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .move_x = 1.0f,
                                                 .move_y = 0.0f,
                                             }));
    instance.tick(ecs::DeltaTime{1.0f});

    auto snapshot = instance.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 1);
    EXPECT_FLOAT_EQ(snapshot.entities[0].x_position, 1.0f);
    EXPECT_FLOAT_EQ(snapshot.entities[0].y_position, 0.0f);
}

TEST(BattleInstanceTest, ConstructorAppliesConfiguredPlayerWeapon) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 1.0f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
        .player_weapons = {
            {1001, WeaponKind::Axe},
        },
    });

    instance.tick(ecs::DeltaTime{2.8f});

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    EXPECT_TRUE(instance.ended());
    EXPECT_EQ(instance.end_reason(), BattleEndReason::Victory);
}

TEST(BattleInstanceTest, FirstTickSpawnsFirstConfiguredWave) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 2,
                        },
                    },
                    .health_multiplier = 2.0f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
    });

    ASSERT_EQ(instance.snapshot().entities.size(), 1);
    EXPECT_EQ(instance.current_wave(), 0);

    instance.tick(ecs::DeltaTime{0.0f});

    auto snapshot = instance.snapshot();
    ASSERT_EQ(snapshot.entities.size(), 3);
    EXPECT_EQ(instance.current_wave(), 1);
    EXPECT_EQ(snapshot.entities[1].max_health, 100);
    EXPECT_EQ(snapshot.entities[2].max_health, 100);
}

TEST(BattleInstanceTest, TickSpawnsNextWaveWhenCurrentWaveHasNoLivingMonsters) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 0,
                        },
                    },
                },
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 3.0f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
    });

    instance.tick(ecs::DeltaTime{0.0f});

    auto snapshot = instance.snapshot();
    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_EQ(instance.current_wave(), 2);
    EXPECT_EQ(snapshot.entities[1].max_health, 150);
}

TEST(BattleInstanceTest, TickSpawnsNextWaveAfterPlayerKillsCurrentWave) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 0.5f,
                    .move_speed_multiplier = 1.0f,
                },
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 2.0f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
        .player_config_override = ecs::CreatePlayerConfig{
            .max_health = 100,
            .move_speed = 5.0f,
            .attack = ecs::AttackDefinition{
                .damage = 25,
                .range = 20.0f,
                .cooldown_seconds = ecs::DeltaTime{0.5f},
            },
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    auto snapshot = instance.snapshot();
    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_EQ(instance.current_wave(), 2);
    EXPECT_EQ(snapshot.entities[1].max_health, 100);
}

TEST(BattleInstanceTest, TickRecordsPlayerKillsByMonsterKind) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 0.5f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
        .player_config_override = ecs::CreatePlayerConfig{
            .max_health = 100,
            .move_speed = 5.0f,
            .attack = ecs::AttackDefinition{
                .damage = 25,
                .range = 20.0f,
                .cooldown_seconds = ecs::DeltaTime{0.5f},
            },
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    const auto& player_stats = instance.player_battle_stats();
    ASSERT_TRUE(player_stats.contains(1001));
    const auto& stats = player_stats.at(1001);
    EXPECT_EQ(stats.total_kills, 1);
    ASSERT_TRUE(stats.kills_by_kind.contains(MonsterKind::Melee));
    EXPECT_EQ(stats.kills_by_kind.at(MonsterKind::Melee), 1);
}

TEST(BattleInstanceTest, SettlementIncludesEndReasonAndPlayerStats) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 0.5f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
        .player_config_override = ecs::CreatePlayerConfig{
            .max_health = 100,
            .move_speed = 5.0f,
            .attack = ecs::AttackDefinition{
                .damage = 25,
                .range = 20.0f,
                .cooldown_seconds = ecs::DeltaTime{0.5f},
            },
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    const auto settlement = instance.settlement();

    EXPECT_EQ(settlement.reason, BattleEndReason::Victory);
    ASSERT_EQ(settlement.players.size(), 1);
    const auto& player = settlement.players[0];
    EXPECT_EQ(player.player_id, 1001);
    EXPECT_EQ(player.total_kills, 1);
    ASSERT_EQ(player.kills.size(), 1);
    EXPECT_EQ(player.kills[0].monster_kind, MonsterKind::Melee);
    EXPECT_EQ(player.kills[0].count, 1);
}

TEST(BattleInstanceTest, TickEndsWithVictoryWhenWaveConfigIsEmpty) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{},
    });

    instance.tick(ecs::DeltaTime{0.0f});

    EXPECT_EQ(instance.state(), BattleState::Ended);
    EXPECT_EQ(instance.end_reason(), BattleEndReason::Victory);
}

TEST(BattleInstanceTest, TickEndsWithVictoryAfterPlayerKillsFinalWave) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                    .health_multiplier = 0.5f,
                    .move_speed_multiplier = 1.0f,
                },
            },
        },
        .player_config_override = ecs::CreatePlayerConfig{
            .max_health = 100,
            .move_speed = 5.0f,
            .attack = ecs::AttackDefinition{
                .damage = 25,
                .range = 20.0f,
                .cooldown_seconds = ecs::DeltaTime{0.5f},
            },
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    EXPECT_EQ(instance.state(), BattleState::Ended);
    EXPECT_EQ(instance.end_reason(), BattleEndReason::Victory);
    EXPECT_EQ(instance.current_wave(), 1);
    EXPECT_EQ(instance.snapshot().entities.size(), 1);
}

TEST(BattleInstanceTest, TickEndsWithDefeatWhenNoPlayersAreAlive) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {},
        .wave_config = WaveConfig{
            .waves = {
                WaveDefinition{
                    .groups = {
                        WaveMonsterGroup{
                            .kind = MonsterKind::Melee,
                            .count = 1,
                        },
                    },
                },
            },
        },
    });

    instance.tick(ecs::DeltaTime{0.0f});

    EXPECT_EQ(instance.state(), BattleState::Ended);
    EXPECT_EQ(instance.end_reason(), BattleEndReason::Defeat);
}

}  // namespace
}  // namespace battle
