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

    EXPECT_EQ(instance.phase(), BattlePhase::RewardSelection);
    EXPECT_FALSE(instance.ended());

    instance.tick(SelectionTime);

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

TEST(BattleInstanceTest, TickStartsRewardSelectionWhenCurrentWaveHasNoLivingMonsters) {
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
    ASSERT_EQ(snapshot.entities.size(), 1);
    EXPECT_EQ(instance.current_wave(), 1);
    EXPECT_EQ(instance.phase(), BattlePhase::RewardSelection);
    EXPECT_FLOAT_EQ(instance.reward_selection_remaining().count(), SelectionTime.count());
}

TEST(BattleInstanceTest, TickSpawnsNextWaveAfterRewardSelectionEnds) {
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
    EXPECT_EQ(instance.phase(), BattlePhase::RewardSelection);

    instance.tick(SelectionTime);

    auto snapshot = instance.snapshot();
    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_EQ(instance.current_wave(), 2);
    EXPECT_EQ(instance.phase(), BattlePhase::Fighting);
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
    ASSERT_EQ(player_stats.size(), 1);
    ASSERT_TRUE(player_stats.contains(1001));
    const auto& stats = player_stats.at(1001);
    EXPECT_EQ(stats.total_kills, 1);
    ASSERT_TRUE(stats.kills_by_kind.contains(MonsterKind::Melee));
    EXPECT_EQ(stats.kills_by_kind.at(MonsterKind::Melee), 1);
}

TEST(BattleInstanceTest, TickGrantsExperienceForPlayerKill) {
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

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->level, 1);
    EXPECT_EQ(progress->experience, 35);
    EXPECT_EQ(progress->experience_to_next_level, 100);
    EXPECT_EQ(progress->pending_upgrade_choices, 0);
}

TEST(BattleInstanceTest, TickLevelsUpAndKeepsOverflowExperience) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->level, 2);
    EXPECT_EQ(progress->experience, 5);
    EXPECT_EQ(progress->experience_to_next_level, 40);
    EXPECT_EQ(progress->pending_upgrade_choices, 1);
}

TEST(BattleInstanceTest, TickCanGrantMultiplePendingUpgradeChoices) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 20,
            .melee_experience = 100,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->level, 3);
    EXPECT_EQ(progress->experience, 20);
    EXPECT_EQ(progress->experience_to_next_level, 70);
    EXPECT_EQ(progress->pending_upgrade_choices, 2);
}

TEST(BattleInstanceTest, SnapshotIncludesProgressAndBlessingState) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1002, 1001},
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    const auto snapshot = instance.snapshot();

    EXPECT_EQ(snapshot.current_wave, 1);
    EXPECT_EQ(snapshot.phase, BattlePhase::RewardSelection);
    EXPECT_FLOAT_EQ(snapshot.reward_selection_remaining.count(), SelectionTime.count());

    ASSERT_EQ(snapshot.player_progress.size(), 2);
    EXPECT_EQ(snapshot.player_progress[0].player_id, 1001);
    EXPECT_EQ(snapshot.player_progress[0].level, 2);
    EXPECT_EQ(snapshot.player_progress[0].experience, 5);
    EXPECT_EQ(snapshot.player_progress[0].experience_to_next_level, 40);
    EXPECT_EQ(snapshot.player_progress[0].pending_upgrade_choices, 1);
    EXPECT_EQ(snapshot.player_progress[1].player_id, 1002);
    EXPECT_EQ(snapshot.player_progress[1].level, 1);
    EXPECT_EQ(snapshot.player_progress[1].pending_upgrade_choices, 0);

    ASSERT_EQ(snapshot.player_blessings.size(), 2);
    EXPECT_EQ(snapshot.player_blessings[0].player_id, 1001);
    ASSERT_EQ(snapshot.player_blessings[0].current_options.size(), 3);
    EXPECT_EQ(snapshot.player_blessings[0].current_options[0].option_id, 0);
    EXPECT_EQ(snapshot.player_blessings[0].current_options[0].blessing_id, BlessingID::BurnOnHit);
    EXPECT_EQ(snapshot.player_blessings[1].player_id, 1002);
    EXPECT_TRUE(snapshot.player_blessings[1].current_options.empty());
}

TEST(BattleInstanceTest, RewardSelectionGeneratesBlessingOptionsForPlayersWithPendingChoices) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->current_options.size(), 3);
    EXPECT_EQ(blessing_state->current_options[0].option_id, 0);
    EXPECT_EQ(blessing_state->current_options[0].blessing_id, BlessingID::BurnOnHit);
    EXPECT_EQ(blessing_state->current_options[1].option_id, 1);
    EXPECT_EQ(blessing_state->current_options[1].blessing_id, BlessingID::LifeSteal);
    EXPECT_EQ(blessing_state->current_options[2].option_id, 2);
    EXPECT_EQ(blessing_state->current_options[2].blessing_id, BlessingID::FreezeOnHit);
}

TEST(BattleInstanceTest, RewardSelectionClearsBlessingOptionsForPlayersWithoutPendingChoices) {
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
            },
        },
    });

    instance.tick(ecs::DeltaTime{0.0f});

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    EXPECT_TRUE(blessing_state->current_options.empty());
}

TEST(BattleInstanceTest, ChooseBlessingReturnsFalseOutsideRewardSelection) {
    BattleInstance instance({
        .room_name = "room-1",
        .player_ids = {1001},
    });

    EXPECT_FALSE(instance.choose_blessing(1001, 0));
}

TEST(BattleInstanceTest, ChooseBlessingReturnsFalseForInvalidOption) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    EXPECT_FALSE(instance.choose_blessing(1001, 99));

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 1);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    EXPECT_TRUE(blessing_state->blessings.empty());
    EXPECT_EQ(blessing_state->current_options.size(), 3);
}

TEST(BattleInstanceTest, ChooseBlessingAddsOwnedBlessingAndClearsOptions) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    ASSERT_TRUE(instance.choose_blessing(1001, 1));

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 0);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->blessings.size(), 1);
    EXPECT_EQ(blessing_state->blessings[0].blessing_id, BlessingID::LifeSteal);
    EXPECT_EQ(blessing_state->blessings[0].level, 1);
    EXPECT_TRUE(blessing_state->current_options.empty());
}

TEST(BattleInstanceTest, ChooseBlessingRegeneratesOptionsWhenMoreChoicesRemain) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 20,
            .melee_experience = 100,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    ASSERT_TRUE(instance.choose_blessing(1001, 2));

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 1);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->blessings.size(), 1);
    EXPECT_EQ(blessing_state->blessings[0].blessing_id, BlessingID::FreezeOnHit);
    EXPECT_EQ(blessing_state->current_options.size(), 3);
}

TEST(BattleInstanceTest, ChooseBlessingLevelsExistingBlessing) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 20,
            .melee_experience = 100,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    ASSERT_TRUE(instance.choose_blessing(1001, 0));
    ASSERT_TRUE(instance.choose_blessing(1001, 0));

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 0);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->blessings.size(), 1);
    EXPECT_EQ(blessing_state->blessings[0].blessing_id, BlessingID::BurnOnHit);
    EXPECT_EQ(blessing_state->blessings[0].level, 2);
    EXPECT_TRUE(blessing_state->current_options.empty());
}

TEST(BattleInstanceTest, RewardSelectionTimeoutChoosesFirstOptionByDefault) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    instance.tick(SelectionTime);

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 0);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->blessings.size(), 1);
    EXPECT_EQ(blessing_state->blessings[0].blessing_id, BlessingID::BurnOnHit);
    EXPECT_TRUE(blessing_state->current_options.empty());
}

TEST(BattleInstanceTest, RewardSelectionTimeoutChoosesDefaultsForAllPendingChoices) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 20,
            .melee_experience = 100,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});

    instance.tick(SelectionTime);

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 0);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->blessings.size(), 1);
    EXPECT_EQ(blessing_state->blessings[0].blessing_id, BlessingID::BurnOnHit);
    EXPECT_EQ(blessing_state->blessings[0].level, 2);
    EXPECT_TRUE(blessing_state->current_options.empty());
}

TEST(BattleInstanceTest, RewardSelectionTimeoutDoesNotChooseAgainAfterManualChoice) {
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
        .progression_config = ProgressionConfig{
            .base_experience_to_next_level = 30,
            .experience_to_next_level_growth = 10,
            .melee_experience = 35,
        },
    });

    ASSERT_TRUE(instance.receive_input(1001, PlayerInput{
                                                 .attack_requested = true,
                                             }));
    instance.tick(ecs::DeltaTime{0.0f});
    ASSERT_TRUE(instance.choose_blessing(1001, 1));

    instance.tick(SelectionTime);

    const auto progress = instance.player_progress(1001);
    ASSERT_TRUE(progress.has_value());
    EXPECT_EQ(progress->pending_upgrade_choices, 0);

    const auto blessing_state = instance.player_blessing_state(1001);
    ASSERT_TRUE(blessing_state.has_value());
    ASSERT_EQ(blessing_state->blessings.size(), 1);
    EXPECT_EQ(blessing_state->blessings[0].blessing_id, BlessingID::LifeSteal);
    EXPECT_TRUE(blessing_state->current_options.empty());
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

    instance.tick(SelectionTime);

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
    instance.tick(SelectionTime);

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
    EXPECT_EQ(instance.phase(), BattlePhase::RewardSelection);

    instance.tick(SelectionTime);

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
