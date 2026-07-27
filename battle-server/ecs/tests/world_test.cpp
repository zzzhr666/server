#include "ecs/world.hpp"

#include <cmath>

#include <gtest/gtest.h>

namespace battle::ecs {
namespace {

CreatePlayerConfig default_player_config() {
    return {
        .position = Position{.x = 10.0f, .y = 20.0f},
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

TEST(WorldTest, CreatePlayerInitializesAttackComponentsFromConfig) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 30,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.75f},
        },
    });

    ASSERT_TRUE(world.attack_intents().has(entity));
    ASSERT_TRUE(world.attack_requests().has(entity));
    ASSERT_TRUE(world.melee_attacks().has(entity));
    ASSERT_TRUE(world.attack_cooldowns().has(entity));

    const auto& attack_intent = world.attack_intents().get(entity);
    const auto& melee_attack = world.melee_attacks().get(entity);
    const auto& cooldown = world.attack_cooldowns().get(entity);
    EXPECT_FALSE(attack_intent.active);
    EXPECT_EQ(attack_intent.damage, 0);
    EXPECT_FLOAT_EQ(attack_intent.range, 0.0f);
    EXPECT_FALSE(world.attack_requests().get(entity).requested);
    EXPECT_EQ(melee_attack.damage, 30);
    EXPECT_FLOAT_EQ(melee_attack.range, 2.0f);
    EXPECT_FLOAT_EQ(melee_attack.cooldown_seconds.count(), 0.75f);
    EXPECT_FLOAT_EQ(cooldown.remaining_seconds.count(), 0.0f);
}

TEST(WorldTest, SetPlayerCommandReturnsFalseForUnknownEntity) {
    World world;

    EXPECT_FALSE(world.set_player_command(404, PlayerCommand{
                                                   .move_x = 1.0f,
                                                   .move_y = 0.0f,
                                                   .attack_requested = false,
                                                     .dash_requested = false,
                                               }));
}

TEST(WorldTest, SetPlayerCommandWritesMoveAndAttackRequests) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));

    EXPECT_FLOAT_EQ(world.move_requests().get(entity).x, 1.0f);
    EXPECT_FLOAT_EQ(world.move_requests().get(entity).y, 0.0f);
    EXPECT_TRUE(world.attack_requests().get(entity).requested);
    EXPECT_FALSE(world.attack_intents().get(entity).active);
}

TEST(WorldTest, SetPlayerCommandReturnsFalseForMonster) {
    World world;
    auto monster = world.create_monster(default_monster_config());

    EXPECT_FALSE(world.set_player_command(monster, PlayerCommand{
                                                       .move_x = 1.0f,
                                                       .move_y = 0.0f,
                                                       .attack_requested = true,
                                                     .dash_requested = false,
                                                   }));
}

TEST(WorldTest, TickResolvesAttackRequestIntoAttackIntent) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 30,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.75f},
        },
    });

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    const auto& intent = world.attack_intents().get(entity);
    EXPECT_TRUE(intent.active);
    EXPECT_EQ(intent.damage, 30);
    EXPECT_FLOAT_EQ(intent.range, 2.0f);
    EXPECT_FALSE(world.attack_requests().get(entity).requested);
    EXPECT_FLOAT_EQ(world.attack_cooldowns().get(entity).remaining_seconds.count(), 0.75f);
}

TEST(WorldTest, TickDoesNotResolveAttackRequestDuringCooldown) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 30,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.75f},
        },
    });
    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});
    ASSERT_TRUE(world.attack_intents().get(entity).active);

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.1f});

    const auto& intent = world.attack_intents().get(entity);
    EXPECT_FALSE(intent.active);
    EXPECT_EQ(intent.damage, 0);
    EXPECT_FLOAT_EQ(intent.range, 0.0f);
    EXPECT_FALSE(world.attack_requests().get(entity).requested);
    EXPECT_FLOAT_EQ(world.attack_cooldowns().get(entity).remaining_seconds.count(), 0.65f);
}

TEST(WorldTest, TickAllowsAttackAfterCooldownExpires) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 30,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.75f},
        },
    });
    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.75f});

    const auto& intent = world.attack_intents().get(entity);
    EXPECT_TRUE(intent.active);
    EXPECT_EQ(intent.damage, 30);
    EXPECT_FLOAT_EQ(intent.range, 2.0f);
    EXPECT_FALSE(world.attack_requests().get(entity).requested);
    EXPECT_FLOAT_EQ(world.attack_cooldowns().get(entity).remaining_seconds.count(), 0.75f);
}

TEST(WorldTest, TickDamagesMonsterInsideAttackRange) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 1.0f,
        .y_position = 0.0f,
        .max_health = 50,
        .move_speed = 3.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    EXPECT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.health().get(monster).current_health, 30);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, TickDoesNotDamageMonsterOutsideAttackRange) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 3.0f,
        .y_position = 0.0f,
        .max_health = 50,
        .move_speed = 3.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.health().get(monster).current_health, 50);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, TickDoesNotDamageOtherPlayers) {
    World world;
    auto attacker = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto teammate = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });

    ASSERT_TRUE(world.set_player_command(attacker, PlayerCommand{
                                                       .move_x = 0.0f,
                                                       .move_y = 0.0f,
                                                       .attack_requested = true,
                                                     .dash_requested = false,
                                                   }));
    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.health().get(teammate).current_health, 100);
}

TEST(WorldTest, TickDestroysMonsterWhenHealthReachesZero) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .melee_attack = MeleeAttack{
            .damage = 50,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 1.0f,
        .y_position = 0.0f,
        .max_health = 50,
        .move_speed = 3.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    EXPECT_FALSE(world.has_entity(monster));
    EXPECT_FALSE(world.health().has(monster));
    EXPECT_FALSE(world.transforms().has(monster));
    EXPECT_FALSE(world.monster_controllers().has(monster));
}

TEST(WorldTest, DamageEventReducesHealthAndIsCleared) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 15,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.health().get(monster).current_health, 35);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, DamageEventWithNegativeDamageDoesNotChangeHealth) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = -10,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.health().get(monster).current_health, 50);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, DamageEventDestroysTargetWhenDamageExceedsHealth) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 999,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_FALSE(world.has_entity(monster));
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, TickMovesPlayerByInputDirectionAndMoveSpeed) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 15.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickDashMovesPlayerWithDashSpeedMultiplierOnce) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = true,
                                                 }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 35.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
    EXPECT_FALSE(world.dash_requests().get(entity).requested);
    EXPECT_FLOAT_EQ(world.dash_cooldowns().get(entity).remaining_seconds.count(), 1.0f);
}

TEST(WorldTest, TickDoesNotDashAgainDuringCooldown) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = true,
                                                 }));
    world.tick(DeltaTime{1.0f});

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = true,
                                                 }));
    world.tick(DeltaTime{0.5f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 37.5f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FALSE(world.dash_requests().get(entity).requested);
    EXPECT_FLOAT_EQ(world.dash_cooldowns().get(entity).remaining_seconds.count(), 0.5f);
}

TEST(WorldTest, TickUsesDeltaSecondsOnce) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.5f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 12.5f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
}

TEST(WorldTest, TickMovesPlayerInNegativeInputDirection) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = -1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 5.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, -1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickNormalizesDiagonalMoveIntent) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 1.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    const float expected_delta = 5.0f / std::sqrt(2.0f);
    EXPECT_NEAR(transform.position.x, 10.0f + expected_delta, 0.001f);
    EXPECT_NEAR(transform.position.y, 20.0f + expected_delta, 0.001f);
    EXPECT_NEAR(transform.direction.x, 1.0f / std::sqrt(2.0f), 0.001f);
    EXPECT_NEAR(transform.direction.y, 1.0f / std::sqrt(2.0f), 0.001f);
}

TEST(WorldTest, TickWithZeroMoveIntentDoesNotMoveOrChangeDirection) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{1.0f});
    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
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
    EXPECT_FALSE(world.attack_intents().has(entity));
    EXPECT_FALSE(world.attack_requests().has(entity));
    EXPECT_FALSE(world.melee_attacks().has(entity));
    EXPECT_FALSE(world.attack_cooldowns().has(entity));
    EXPECT_FALSE(world.move_requests().has(entity));
    EXPECT_FALSE(world.move_intents().has(entity));
    EXPECT_FALSE(world.dash_requests().has(entity));
    EXPECT_FALSE(world.dash_intents().has(entity));
    EXPECT_FALSE(world.dashes().has(entity));
    EXPECT_FALSE(world.dash_cooldowns().has(entity));
    EXPECT_FALSE(world.set_player_command(entity, PlayerCommand{
                                                      .move_x = 1.0f,
                                                      .move_y = 0.0f,
                                                      .attack_requested = false,
                                                     .dash_requested = false,
                                                  }));
}

TEST(WorldTest, HasLivingPlayersReflectsPlayerControllerEntities) {
    World world;

    EXPECT_FALSE(world.has_living_players());

    auto player = world.create_player(default_player_config());
    EXPECT_TRUE(world.has_living_players());

    ASSERT_TRUE(world.destroy_entity(player));
    EXPECT_FALSE(world.has_living_players());
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
    EXPECT_TRUE(world.monster_controllers().has(entity));
}

TEST(WorldTest, MonsterDoesNotHavePlayerCommand) {
    World world;
    auto entity = world.create_monster(default_monster_config());

    EXPECT_FALSE(world.set_player_command(entity, PlayerCommand{
                                                      .move_x = 1.0f,
                                                      .move_y = 0.0f,
                                                      .attack_requested = false,
                                                     .dash_requested = false,
                                                  }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 30.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 40.0f);
}

TEST(WorldTest, TickMovesMonsterTowardPlayerUsingMonsterMoveSpeed) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 0.0f,
        .y_position = 0.0f,
        .max_health = 50,
        .move_speed = 3.0f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 3.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickMovesMonsterTowardNearestPlayer) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 100.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 5.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 0.0f,
        .y_position = 0.0f,
        .max_health = 50,
        .move_speed = 2.0f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 2.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
}

TEST(WorldTest, TickDoesNotMoveMonsterWithoutPlayerTarget) {
    World world;
    auto monster = world.create_monster(default_monster_config());

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 30.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 40.0f);
}

TEST(WorldTest, TickStopsMonsterWhenPlayerTargetIsDestroyed) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 0.0f,
        .y_position = 0.0f,
        .max_health = 50,
        .move_speed = 3.0f,
    });
    world.tick(DeltaTime{1.0f});
    ASSERT_TRUE(world.destroy_entity(player));

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 3.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
}

TEST(WorldTest, TickDoesNotProduceNanWhenMonsterOverlapsPlayer) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .x_position = 10.0f,
        .y_position = 20.0f,
        .max_health = 50,
        .move_speed = 3.0f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.transforms().get(monster);
    EXPECT_FALSE(std::isnan(transform.position.x));
    EXPECT_FALSE(std::isnan(transform.position.y));
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
}

TEST(WorldTest, DestroyMonsterRemovesEntityComponents) {
    World world;
    auto entity = world.create_monster(default_monster_config());

    EXPECT_TRUE(world.destroy_entity(entity));

    EXPECT_FALSE(world.has_entity(entity));
    EXPECT_FALSE(world.transforms().has(entity));
    EXPECT_FALSE(world.monster_controllers().has(entity));
}

TEST(WorldTest, HasLivingMonstersReflectsMonsterControllerEntities) {
    World world;

    EXPECT_FALSE(world.has_living_monsters());

    auto monster = world.create_monster(default_monster_config());
    EXPECT_TRUE(world.has_living_monsters());

    ASSERT_TRUE(world.destroy_entity(monster));
    EXPECT_FALSE(world.has_living_monsters());
}

TEST(WorldTest, SnapshotReturnsEmptyEntitiesForEmptyWorld) {
    World world;

    auto snapshot = world.snapshot();

    EXPECT_TRUE(snapshot.entities.empty());
}

TEST(WorldTest, SnapshotIncludesMovedPlayerTransformAndHealth) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
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
