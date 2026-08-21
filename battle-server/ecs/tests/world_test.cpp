#include "ecs/world.hpp"
#include "ecs/system/blessing_config.hpp"
#include "ecs/system/blessing_trigger_system.hpp"
#include "ecs/system/damage_modify_system.hpp"
#include "ecs/system/damage_system.hpp"
#include "ecs/system/hit_resolve_system.hpp"
#include "ecs/system/status_effect_system.hpp"

#include <cmath>

#include <gtest/gtest.h>

namespace battle::ecs {
namespace {

CreatePlayerConfig default_player_config() {
    return {
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = gameplay_config::player::MaxHealth,
        .move_speed = gameplay_config::player::MoveSpeed,
    };
}

CreateMonsterConfig default_monster_config() {
    return {
        .position = Position{.x = 30.0f, .y = 40.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    };
}

int attack_event_count_from(World& world, Entity attacker) {
    int count = 0;
    for (const auto& event : world.attack_events()) {
        if (event.attacker == attacker) {
            ++count;
        }
    }
    return count;
}

TEST(WorldTest, CreatePlayerReturnsLiveEntityWithInitialTransform) {
    World world;

    auto entity = world.create_player(default_player_config());

    EXPECT_TRUE(entity);
    EXPECT_TRUE(world.has_entity(entity));

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
}

TEST(WorldTest, CreatePlayerAllowsMultiplePlayerControlledEntities) {
    World world;

    auto first = world.create_player(default_player_config());
    auto second = world.create_player(default_player_config());

    EXPECT_TRUE(first);
    EXPECT_TRUE(second);
    EXPECT_NE(first, second);
    EXPECT_TRUE(world.registry().has<PlayerController>(first));
    EXPECT_TRUE(world.registry().has<PlayerController>(second));
}

TEST(WorldTest, CreateProjectileInitializesMovementAndCombatContext) {
    World world;
    auto owner = world.create_player(default_player_config());
    auto action_state = world.create_combat_action();

    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .direction = Direction{.x = 0.6f, .y = 0.8f},
        .speed = 25.0f,
        .damage = 30,
        .max_distance = 15.0f,
        .hit_radius = 0.85f,
        .context = CombatContext{
            .owner = owner,
            .action_state = action_state,
            .effect_id = 7,
        },
    });

    ASSERT_TRUE(world.has_entity(projectile));
    const auto& transform = world.registry().get<Transform>(projectile);
    const auto& velocity = world.registry().get<Velocity>(projectile);
    const auto& projectile_component = world.registry().get<Projectile>(projectile);
    const auto& collider = world.registry().get<Collider>(projectile);

    EXPECT_FLOAT_EQ(transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.6f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.8f);
    EXPECT_FLOAT_EQ(velocity.x, 15.0f);
    EXPECT_FLOAT_EQ(velocity.y, 20.0f);
    EXPECT_EQ(projectile_component.damage, 30);
    EXPECT_FLOAT_EQ(projectile_component.current_distance, 0.0f);
    EXPECT_FLOAT_EQ(projectile_component.max_distance, 15.0f);
    EXPECT_FLOAT_EQ(collider.radius, 0.85f);
    EXPECT_EQ(projectile_component.context.owner, owner);
    EXPECT_EQ(projectile_component.context.emitter, projectile);
    EXPECT_EQ(projectile_component.context.action_state, action_state);
    EXPECT_EQ(projectile_component.context.effect_id, 7);
}

TEST(WorldTest, TickSpawnsOneProjectileForProjectileAttack) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Projectile,
            .damage = 30,
            .range = 15.0f,
            .cooldown_seconds = DeltaTime{0.5f},
            .projectile_speed = 25.0f,
            .projectile_hit_radius = 0.85f,
        },
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    ASSERT_EQ(world.registry().pool<Projectile>().entities().size(), 1);
    const auto projectile = world.registry().pool<Projectile>().entities().front();
    const auto& transform = world.registry().get<Transform>(projectile);
    const auto& projectile_component = world.registry().get<Projectile>(projectile);
    const auto& collider = world.registry().get<Collider>(projectile);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
    EXPECT_EQ(projectile_component.damage, 30);
    EXPECT_FLOAT_EQ(projectile_component.max_distance, 15.0f);
    EXPECT_FLOAT_EQ(collider.radius, 0.85f);
    EXPECT_EQ(projectile_component.context.owner, player);
    EXPECT_EQ(projectile_component.context.emitter, projectile);

    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.registry().pool<Projectile>().entities().size(), 1);
}

TEST(WorldTest, TickMovesProjectileAndAccumulatesTravelDistance) {
    World world;
    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .direction = Direction{.x = 1.0f, .y = 0.0f},
        .speed = 10.0f,
        .damage = 30,
        .max_distance = 20.0f,
    });

    world.tick(DeltaTime{1.0f});

    ASSERT_TRUE(world.has_entity(projectile));
    const auto& transform = world.registry().get<Transform>(projectile);
    const auto& projectile_component = world.registry().get<Projectile>(projectile);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
    EXPECT_FLOAT_EQ(projectile_component.current_distance, 10.0f);
}

TEST(WorldTest, TickDestroysProjectileAtMaximumTravelDistance) {
    World world;
    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .direction = Direction{.x = 1.0f, .y = 0.0f},
        .speed = 10.0f,
        .damage = 30,
        .max_distance = 10.0f,
    });

    world.tick(DeltaTime{1.0f});

    EXPECT_FALSE(world.has_entity(projectile));
    EXPECT_FALSE(world.registry().has<Projectile>(projectile));
}

TEST(WorldTest, TickProjectileDamagesEnemyAndIsDestroyedOnHit) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });
    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
        .direction = Direction{.x = 1.0f, .y = 0.0f},
        .speed = 0.0f,
        .damage = 30,
        .max_distance = 20.0f,
        .context = CombatContext{.owner = player},
    });

    world.tick(DeltaTime{0.0f});

    EXPECT_FALSE(world.has_entity(projectile));
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 20);
}

TEST(WorldTest, TickProjectileUsesConfiguredHitRadius) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.75f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });
    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .direction = Direction{.x = 1.0f, .y = 0.0f},
        .speed = 0.0f,
        .damage = 30,
        .max_distance = 20.0f,
        .hit_radius = 0.85f,
        .context = CombatContext{.owner = player},
    });

    world.tick(DeltaTime{0.0f});

    EXPECT_FALSE(world.has_entity(projectile));
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 20);
}

TEST(WorldTest, TickProjectileDoesNotDamageFriendlyPlayer) {
    World world;
    auto owner = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto teammate = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
        .direction = Direction{.x = 1.0f, .y = 0.0f},
        .speed = 0.0f,
        .damage = 30,
        .max_distance = 20.0f,
        .context = CombatContext{.owner = owner},
    });

    world.tick(DeltaTime{0.0f});

    EXPECT_TRUE(world.has_entity(projectile));
    EXPECT_EQ(world.registry().get<Health>(teammate).current_health, 100);
}

TEST(WorldTest, CreatePlayerInitializesBlessingInventory) {
    World world;

    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.registry().has<BlessingInventory>(entity));
    EXPECT_TRUE(world.registry().get<BlessingInventory>(entity).blessings.empty());
}

TEST(WorldTest, CreatePlayerInitializesStatusEffects) {
    World world;

    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.registry().has<StatusEffects>(entity));
    EXPECT_TRUE(world.registry().get<StatusEffects>(entity).burns.empty());
    EXPECT_FALSE(world.registry().get<StatusEffects>(entity).freeze.has_value());
}

TEST(WorldTest, BlessingConfigScalesEffectValuesWithLevel) {
    EXPECT_EQ(life_steal_percent(1), 8);
    EXPECT_EQ(life_steal_percent(5), 16);
    EXPECT_EQ(critical_strike_percent(1), 15);
    EXPECT_EQ(critical_strike_percent(5), 31);
    EXPECT_EQ(critical_strike_damage_percent(1), 175);
    EXPECT_EQ(critical_strike_damage_percent(5), 235);
    EXPECT_EQ(burn_damage_per_tick(1), 6);
    EXPECT_EQ(burn_damage_per_tick(5), 14);
    EXPECT_FLOAT_EQ(burn_duration_seconds(5).count(), 4.5f);
    EXPECT_EQ(freeze_percent(1), 15);
    EXPECT_EQ(freeze_percent(5), 31);
    EXPECT_FLOAT_EQ(freeze_duration_seconds(5).count(), 1.6f);
    EXPECT_EQ(chain_lightning_damage_percent(1), 50);
    EXPECT_EQ(chain_lightning_damage_percent(5), 90);
    EXPECT_EQ(chain_lightning_target_count(1), 1);
    EXPECT_EQ(chain_lightning_target_count(5), 5);
    EXPECT_FLOAT_EQ(gameplay_config::blessing::chain_lightning::JumpRadius, 9.0f);
}

TEST(WorldTest, CreatePlayerInitializesAttackComponentsFromConfig) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 30,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.75f},
            .windup_seconds = DeltaTime{0.15f},
            .active_seconds = DeltaTime{0.05f},
            .recovery_seconds = DeltaTime{0.20f},
            .movement_multiplier = 0.25f,
        },
    });

    ASSERT_TRUE(world.registry().has<AttackRequest>(entity));
    ASSERT_TRUE(world.registry().has<AttackDefinition>(entity));
    ASSERT_TRUE(world.registry().has<AttackCooldown>(entity));
    ASSERT_TRUE(world.registry().has<AttackState>(entity));

    const auto& attack = world.registry().get<AttackDefinition>(entity);
    const auto& cooldown = world.registry().get<AttackCooldown>(entity);
    const auto& attack_state = world.registry().get<AttackState>(entity);
    EXPECT_FALSE(world.registry().get<AttackRequest>(entity).requested);
    EXPECT_EQ(attack.kind, AttackKind::Melee);
    EXPECT_EQ(attack.damage, 30);
    EXPECT_FLOAT_EQ(attack.range, 2.0f);
    EXPECT_FLOAT_EQ(attack.cooldown_seconds.count(), 0.75f);
    EXPECT_FLOAT_EQ(attack.windup_seconds.count(), 0.15f);
    EXPECT_FLOAT_EQ(attack.active_seconds.count(), 0.05f);
    EXPECT_FLOAT_EQ(attack.recovery_seconds.count(), 0.20f);
    EXPECT_FLOAT_EQ(attack.movement_multiplier, 0.25f);
    EXPECT_FLOAT_EQ(cooldown.remaining_seconds.count(), 0.0f);
    EXPECT_EQ(attack_state.phase, AttackPhase::Idle);
    EXPECT_FLOAT_EQ(attack_state.phase_remaining.count(), 0.0f);
    EXPECT_TRUE(attack_state.hit_targets.empty());
    EXPECT_FALSE(attack_state.projectile_spawned);
}

TEST(WorldTest, SetPlayerCommandReturnsFalseForUnknownEntity) {
    World world;

    EXPECT_FALSE(world.set_player_command(Entity{}, PlayerCommand{
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

    EXPECT_FLOAT_EQ(world.registry().get<MoveRequest>(entity).x, 1.0f);
    EXPECT_FLOAT_EQ(world.registry().get<MoveRequest>(entity).y, 0.0f);
    EXPECT_TRUE(world.registry().get<AttackRequest>(entity).requested);
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

TEST(WorldTest, TickStartsAttackTimelineFromAttackRequest) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
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

    const auto& state = world.registry().get<AttackState>(entity);
    EXPECT_EQ(state.phase, AttackPhase::Active);
    EXPECT_EQ(state.context.owner, entity);
    EXPECT_FALSE(world.registry().get<AttackRequest>(entity).requested);
    EXPECT_FLOAT_EQ(world.registry().get<AttackCooldown>(entity).remaining_seconds.count(), 0.75f);
    ASSERT_EQ(attack_event_count_from(world, entity), 1);
    const auto& event = world.attack_events()[0];
    EXPECT_EQ(event.attacker, entity);
    EXPECT_EQ(event.kind, AttackKind::Melee);
    EXPECT_FLOAT_EQ(event.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(event.direction.y, 1.0f);
    EXPECT_NE(event.action_id, InvalidActionID);
}

TEST(WorldTest, TickAttackEventCapturesConfiguredTimeline) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .attack = AttackDefinition{
            .damage = 30,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.75f},
            .windup_seconds = DeltaTime{0.17f},
            .active_seconds = DeltaTime{0.06f},
            .recovery_seconds = DeltaTime{0.23f},
        },
    });
    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));

    world.tick(DeltaTime{0.0f});

    ASSERT_EQ(world.attack_events().size(), 1);
    const auto& event = world.attack_events().front();
    EXPECT_FLOAT_EQ(event.windup_seconds.count(), 0.17f);
    EXPECT_FLOAT_EQ(event.active_seconds.count(), 0.06f);
    EXPECT_FLOAT_EQ(event.recovery_seconds.count(), 0.23f);
}

TEST(WorldTest, TickDoesNotResolveAttackRequestDuringCooldown) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
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
    ASSERT_EQ(world.registry().get<AttackState>(entity).phase, AttackPhase::Active);

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.1f});

    EXPECT_EQ(world.registry().get<AttackState>(entity).phase, AttackPhase::Idle);
    EXPECT_FALSE(world.registry().get<AttackRequest>(entity).requested);
    EXPECT_FLOAT_EQ(world.registry().get<AttackCooldown>(entity).remaining_seconds.count(), 0.65f);
    EXPECT_EQ(attack_event_count_from(world, entity), 1);
}

TEST(WorldTest, TickAllowsAttackAfterCooldownExpires) {
    World world;
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
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

    EXPECT_EQ(world.registry().get<AttackState>(entity).phase, AttackPhase::Active);
    EXPECT_FALSE(world.registry().get<AttackRequest>(entity).requested);
    EXPECT_FLOAT_EQ(world.registry().get<AttackCooldown>(entity).remaining_seconds.count(), 0.75f);
}

TEST(WorldTest, TickDoesNotDamageDuringAttackWindup) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.4f},
            .windup_seconds = DeltaTime{0.2f},
            .active_seconds = DeltaTime{0.1f},
            .recovery_seconds = DeltaTime{0.1f},
        },
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 1.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.registry().get<AttackState>(player).phase, AttackPhase::Windup);
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 50);
    ASSERT_EQ(attack_event_count_from(world, player), 1);

    world.tick(DeltaTime{0.1f});

    EXPECT_EQ(world.registry().get<AttackState>(player).phase, AttackPhase::Windup);
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 50);
}

TEST(WorldTest, TickDamagesWhenAttackWindupEnds) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.4f},
            .windup_seconds = DeltaTime{0.1f},
            .active_seconds = DeltaTime{0.1f},
            .recovery_seconds = DeltaTime{0.2f},
        },
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 1.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});
    world.tick(DeltaTime{0.1f});

    EXPECT_EQ(world.registry().get<AttackState>(player).phase, AttackPhase::Active);
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 30);
    EXPECT_EQ(attack_event_count_from(world, player), 1);
}

TEST(WorldTest, TickDiscardsAttackRequestDuringRecovery) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.4f},
            .windup_seconds = DeltaTime{0.1f},
            .active_seconds = DeltaTime{0.1f},
            .recovery_seconds = DeltaTime{0.2f},
        },
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 1.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});
    world.tick(DeltaTime{0.1f});

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.1f});

    EXPECT_EQ(world.registry().get<AttackState>(player).phase, AttackPhase::Recovery);
    EXPECT_FALSE(world.registry().get<AttackRequest>(player).requested);
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 30);
    EXPECT_EQ(attack_event_count_from(world, player), 1);
}

TEST(WorldTest, TickActiveWindowDamagesEachTargetOnlyOnce) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 0.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
            .active_seconds = DeltaTime{0.3f},
            .recovery_seconds = DeltaTime{0.2f},
        },
    });
    const auto first_monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 1.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });
    const auto second_monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 4.0f},
        .max_health = 50,
        .move_speed = 0.0f,
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});
    EXPECT_EQ(world.registry().get<Health>(first_monster).current_health, 30);
    EXPECT_EQ(world.registry().get<Health>(second_monster).current_health, 50);

    world.registry().get<Transform>(second_monster).position.y = 1.0f;
    world.tick(DeltaTime{0.1f});
    EXPECT_EQ(world.registry().get<Health>(first_monster).current_health, 30);
    EXPECT_EQ(world.registry().get<Health>(second_monster).current_health, 30);

    world.tick(DeltaTime{0.1f});
    EXPECT_EQ(world.registry().get<Health>(first_monster).current_health, 30);
    EXPECT_EQ(world.registry().get<Health>(second_monster).current_health, 30);
    EXPECT_EQ(world.registry().get<AttackState>(player).hit_targets.size(), 2);
}

TEST(WorldTest, TickLongProjectileActiveWindowSpawnsOneProjectile) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 0.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Projectile,
            .damage = 20,
            .range = 10.0f,
            .cooldown_seconds = DeltaTime{0.5f},
            .active_seconds = DeltaTime{0.3f},
            .recovery_seconds = DeltaTime{0.2f},
            .projectile_speed = 5.0f,
        },
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});
    ASSERT_EQ(world.registry().pool<Projectile>().entities().size(), 1);

    world.tick(DeltaTime{0.1f});
    world.tick(DeltaTime{0.1f});
    EXPECT_EQ(world.registry().pool<Projectile>().entities().size(), 1);
}

TEST(WorldTest, TickDamagesMonsterOnMeleeHalfCircleBoundary) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
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
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 30);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, TickDamagesMonsterInFrontOfMeleeAttack) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 1.0f},
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

    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 30);
}

TEST(WorldTest, TickDoesNotDamageMonsterBehindMeleeAttack) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = -1.0f},
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

    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 50);
}

TEST(WorldTest, TickDoesNotDamageMonsterOutsideAttackRange) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 3.0f, .y = 0.0f},
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

    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 50);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, TickDoesNotDamageOtherPlayers) {
    World world;
    auto attacker = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto teammate = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
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

    EXPECT_EQ(world.registry().get<Health>(teammate).current_health, 100);
}

TEST(WorldTest, TickDestroysMonsterWhenHealthReachesZero) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 50,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.5f},
        },
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 1.0f, .y = 0.0f},
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
    EXPECT_FALSE(world.registry().has<Health>(monster));
    EXPECT_FALSE(world.registry().has<Transform>(monster));
    EXPECT_FALSE(world.registry().has<MonsterController>(monster));
}

TEST(WorldTest, DamageEventReducesHealthAndIsCleared) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 15,
        .modified_damage = 15,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 35);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, DamageEventUsesModifiedDamage) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 15,
        .modified_damage = 30,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 20);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, DamageSystemAddsDamageAppliedEventWithActualAmount) {
    World world({damage_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 30.0f, .y = 40.0f},
        .max_health = 10,
        .move_speed = 3.0f,
    });

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 50,
        .modified_damage = 50,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_EQ(world.damage_applied_events().size(), 1);
    const auto& event = world.damage_applied_events()[0];
    EXPECT_EQ(event.source, player);
    EXPECT_EQ(event.target, monster);
    EXPECT_EQ(event.amount, 10);
    EXPECT_EQ(event.source_kind, DamageSourceKind::Attack);
}

TEST(WorldTest, LifeStealHealsSourceFromAppliedAttackDamage) {
    World world({blessing_trigger_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<Health>(player).current_health = 40;
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::LifeSteal,
        .level = 1,
    });

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 20,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health, 41);
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, LifeStealDoesNotTriggerForNonAttackDamage) {
    World world({blessing_trigger_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<Health>(player).current_health = 40;
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::LifeSteal,
        .level = 1,
    });

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 20,
        .source_kind = DamageSourceKind::Burn,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health, 40);
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, LifeStealClampsHealingToMaxHealth) {
    World world({blessing_trigger_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<Health>(player).current_health = gameplay_config::player::MaxHealth - 5;
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::LifeSteal,
        .level = 1,
    });

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 100,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health, gameplay_config::player::MaxHealth);
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, BurnOnHitAddsBurnStatusToTarget) {
    World world({blessing_trigger_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::BurnOnHit,
        .level = 1,
    });

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 20,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_TRUE(world.registry().get<StatusEffects>(player).burns.empty());
    const auto& burns = world.registry().get<StatusEffects>(monster).burns;
    ASSERT_EQ(burns.size(), 1);
    EXPECT_EQ(burns[0].source, player);
    EXPECT_FLOAT_EQ(burns[0].remaining_seconds.count(), burn_duration_seconds(1).count());
    EXPECT_FLOAT_EQ(burns[0].tick_interval_seconds.count(),
                    gameplay_config::blessing::burn_on_hit::TickInterval.count());
    EXPECT_FLOAT_EQ(burns[0].tick_timer_seconds.count(), 0.0f);
    EXPECT_EQ(burns[0].damage_per_tick, burn_damage_per_tick(1));
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, FreezeOnHitAddsFreezeStatusToTarget) {
    World world({blessing_trigger_system}, DefaultWorldBounds, 5);
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::FreezeOnHit,
        .level = 23,
    });

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 20,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_FALSE(world.registry().get<StatusEffects>(player).freeze.has_value());
    const auto& freeze = world.registry().get<StatusEffects>(monster).freeze;
    ASSERT_TRUE(freeze.has_value());
    EXPECT_FLOAT_EQ(freeze->remaining_seconds.count(), freeze_duration_seconds(23).count());
    EXPECT_TRUE(world.registry().get<StatusEffects>(monster).burns.empty());
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, BlessingTriggerDoesNotAddStatusWithoutMatchingBlessing) {
    World world({blessing_trigger_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 20,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_TRUE(world.registry().get<StatusEffects>(monster).burns.empty());
    EXPECT_FALSE(world.registry().get<StatusEffects>(monster).freeze.has_value());
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, FreezeOnHitKeepsLongerExistingDuration) {
    World world({blessing_trigger_system}, DefaultWorldBounds, 5);
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::FreezeOnHit,
        .level = 1,
    });
    world.registry().get<StatusEffects>(monster).freeze = FreezeStatus{
        .remaining_seconds = DeltaTime{5.0f},
    };

    world.add_damage_applied_event(DamageAppliedEvent{
        .source = player,
        .target = monster,
        .amount = 20,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.registry().get<StatusEffects>(monster).freeze.has_value());
    EXPECT_FLOAT_EQ(world.registry().get<StatusEffects>(monster).freeze->remaining_seconds.count(), 5.0f);
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, BurnStatusAddsDamageEventAfterTickInterval) {
    World world({status_effect_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<StatusEffects>(monster).burns.emplace_back(BurnStatus{
        .source = player,
        .remaining_seconds = DeltaTime{3.0f},
        .tick_interval_seconds = DeltaTime{1.0f},
        .tick_timer_seconds = DeltaTime{0.0f},
        .damage_per_tick = 5,
    });

    world.tick(DeltaTime{0.5f});

    EXPECT_TRUE(world.damage_events().empty());
    ASSERT_EQ(world.registry().get<StatusEffects>(monster).burns.size(), 1);
    EXPECT_FLOAT_EQ(world.registry().get<StatusEffects>(monster).burns[0].remaining_seconds.count(), 2.5f);
    EXPECT_FLOAT_EQ(world.registry().get<StatusEffects>(monster).burns[0].tick_timer_seconds.count(), 0.5f);

    world.tick(DeltaTime{0.5f});

    ASSERT_EQ(world.damage_events().size(), 1);
    const auto& event = world.damage_events()[0];
    EXPECT_EQ(event.source, player);
    EXPECT_EQ(event.target, monster);
    EXPECT_EQ(event.base_damage, 5);
    EXPECT_EQ(event.modified_damage, 5);
    EXPECT_EQ(event.source_kind, DamageSourceKind::Burn);
    ASSERT_EQ(world.registry().get<StatusEffects>(monster).burns.size(), 1);
    EXPECT_FLOAT_EQ(world.registry().get<StatusEffects>(monster).burns[0].tick_timer_seconds.count(), 0.0f);
}

TEST(WorldTest, BurnStatusExpiresAndIsRemoved) {
    World world({status_effect_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<StatusEffects>(monster).burns.emplace_back(BurnStatus{
        .source = player,
        .remaining_seconds = DeltaTime{0.25f},
        .tick_interval_seconds = DeltaTime{1.0f},
        .tick_timer_seconds = DeltaTime{0.0f},
        .damage_per_tick = 5,
    });

    world.tick(DeltaTime{0.25f});

    EXPECT_TRUE(world.registry().get<StatusEffects>(monster).burns.empty());
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, FreezeStatusExpiresAndIsRemoved) {
    World world({status_effect_system});
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<StatusEffects>(monster).freeze = FreezeStatus{
        .remaining_seconds = DeltaTime{0.25f},
    };

    world.tick(DeltaTime{0.25f});

    EXPECT_FALSE(world.registry().get<StatusEffects>(monster).freeze.has_value());
}

TEST(WorldTest, CriticalStrikeUsesConfiguredDamagePercent) {
    World world({damage_modify_system, damage_system}, DefaultWorldBounds, 5);
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());
    world.registry().get<BlessingInventory>(player).blessings.emplace_back(BlessingStack{
        .blessing_id = BlessingID::CriticalStrike,
        .level = 23,
    });

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 5,
        .modified_damage = 5,
        .source_kind = DamageSourceKind::Attack,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 25);
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
        .modified_damage = -10,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(monster));
    EXPECT_EQ(world.registry().get<Health>(monster).current_health, 50);
    EXPECT_TRUE(world.damage_events().empty());
    EXPECT_TRUE(world.damage_applied_events().empty());
}

TEST(WorldTest, DamageEventDestroysTargetWhenDamageExceedsHealth) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 999,
        .modified_damage = 999,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_FALSE(world.has_entity(monster));
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, DamageSystemAddsKillEventWhenPlayerKillsMonster) {
    World world({damage_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 50,
        .modified_damage = 50,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_EQ(world.kill_events().size(), 1);
    const auto& event = world.kill_events()[0];
    EXPECT_EQ(event.killer, player);
    EXPECT_EQ(event.victim, monster);
    EXPECT_EQ(event.monster_kind, battle::MonsterKind::Melee);
    ASSERT_EQ(world.death_events().size(), 1);
    const auto& death_event = world.death_events()[0];
    EXPECT_EQ(death_event.victim, monster);
    EXPECT_EQ(death_event.killer, player);
    EXPECT_EQ(death_event.kind, DeathEntityKind::Monster);
    EXPECT_FLOAT_EQ(death_event.position.x, 30.0f);
    EXPECT_FLOAT_EQ(death_event.position.y, 40.0f);
    ASSERT_TRUE(death_event.monster_kind.has_value());
    EXPECT_EQ(death_event.monster_kind.value(), MonsterKind::Melee);
}

TEST(WorldTest, DamageSystemAddsDeathEventWhenMonsterKillsPlayer) {
    World world({damage_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = monster,
        .target = player,
        .base_damage = gameplay_config::player::MaxHealth,
        .modified_damage = gameplay_config::player::MaxHealth,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_EQ(world.death_events().size(), 1);
    const auto& death_event = world.death_events()[0];
    EXPECT_EQ(death_event.victim, player);
    EXPECT_EQ(death_event.killer, monster);
    EXPECT_EQ(death_event.kind, DeathEntityKind::Player);
    EXPECT_FLOAT_EQ(death_event.position.x, 10.0f);
    EXPECT_FLOAT_EQ(death_event.position.y, 20.0f);
    EXPECT_FALSE(death_event.monster_kind.has_value());
    EXPECT_TRUE(world.kill_events().empty());
}

TEST(WorldTest, DamageSystemDoesNotAddKillEventForNonLethalDamage) {
    World world({damage_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = 10,
        .modified_damage = 10,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_TRUE(world.kill_events().empty());
}

TEST(WorldTest, DamageSystemDoesNotAddKillEventForNegativeDamage) {
    World world({damage_system});
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    world.add_damage_event(DamageEvent{
        .source = player,
        .target = monster,
        .base_damage = -10,
        .modified_damage = -10,
    });
    world.tick(DeltaTime{0.0f});

    EXPECT_TRUE(world.kill_events().empty());
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f + gameplay_config::player::MoveSpeed);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickSlowsAndLocksAttackingPlayerWithoutBlockingOtherMovement) {
    World world;
    const auto attacker = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.4f},
            .windup_seconds = DeltaTime{0.2f},
            .active_seconds = DeltaTime{0.1f},
            .recovery_seconds = DeltaTime{0.1f},
            .movement_multiplier = 0.5f,
        },
    });
    const auto other_player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });

    ASSERT_TRUE(world.set_player_command(attacker, PlayerCommand{
                                                       .move_x = 0.0f,
                                                       .move_y = 0.0f,
                                                       .attack_requested = true,
                                                       .dash_requested = false,
                                                   }));
    ASSERT_TRUE(world.set_player_command(other_player, PlayerCommand{
                                                           .move_x = 0.0f,
                                                           .move_y = 1.0f,
                                                           .attack_requested = false,
                                                           .dash_requested = false,
                                                       }));
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.set_player_command(attacker, PlayerCommand{
                                                       .move_x = 1.0f,
                                                       .move_y = 0.0f,
                                                       .attack_requested = false,
                                                       .dash_requested = false,
                                                   }));
    world.tick(DeltaTime{0.1f});

    const auto& attacker_transform = world.registry().get<Transform>(attacker);
    EXPECT_FLOAT_EQ(attacker_transform.position.x, 0.25f);
    EXPECT_FLOAT_EQ(attacker_transform.position.y, 0.0f);
    EXPECT_FLOAT_EQ(attacker_transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(attacker_transform.direction.y, 1.0f);

    const auto& other_transform = world.registry().get<Transform>(other_player);
    EXPECT_FLOAT_EQ(other_transform.position.x, 10.0f);
    EXPECT_FLOAT_EQ(other_transform.position.y, 0.5f);
    EXPECT_FLOAT_EQ(other_transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(other_transform.direction.y, 1.0f);
}

TEST(WorldTest, TickRestoresFullMovementAfterAttackEnds) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .attack = AttackDefinition{
            .damage = 20,
            .range = 2.0f,
            .cooldown_seconds = DeltaTime{0.3f},
            .windup_seconds = DeltaTime{0.1f},
            .active_seconds = DeltaTime{0.1f},
            .recovery_seconds = DeltaTime{0.1f},
            .movement_multiplier = 0.5f,
        },
    });

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = true,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.1f});
    world.tick(DeltaTime{0.1f});
    world.tick(DeltaTime{0.1f});

    ASSERT_EQ(world.registry().get<AttackState>(player).phase, AttackPhase::Idle);
    world.tick(DeltaTime{0.1f});

    const auto& transform = world.registry().get<Transform>(player);
    EXPECT_FLOAT_EQ(transform.position.x, 1.25f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickClampsPlayerPositionToWorldBounds) {
    World world(WorldBounds{
        .min_x = -1.0f,
        .max_x = 1.0f,
        .min_y = -1.0f,
        .max_y = 1.0f,
    });
    auto entity = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x,
                    1.0f - gameplay_config::combat::DefaultCharacterCollisionRadius);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x,
                    10.0f + gameplay_config::player::MoveSpeed * gameplay_config::player::DashSpeedMultiplier);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
    EXPECT_FALSE(world.registry().get<DashRequest>(entity).requested);
    EXPECT_FLOAT_EQ(world.registry().get<DashCooldown>(entity).remaining_seconds.count(), 1.0f);
}

TEST(WorldTest, TickDashUsesFacingDirectionWithoutMoveInput) {
    World world;
    auto entity = world.create_player(default_player_config());

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = -1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.set_player_command(entity, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = true,
                                                 }));
    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x,
                    10.0f - gameplay_config::player::MoveSpeed * gameplay_config::player::DashSpeedMultiplier);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, -1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
    EXPECT_FALSE(world.registry().get<DashRequest>(entity).requested);
    EXPECT_FLOAT_EQ(world.registry().get<DashCooldown>(entity).remaining_seconds.count(), 1.0f);
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x,
                    10.0f + gameplay_config::player::MoveSpeed * gameplay_config::player::DashSpeedMultiplier +
                        gameplay_config::player::MoveSpeed * 0.5f);
    EXPECT_FLOAT_EQ(transform.position.y, 20.0f);
    EXPECT_FALSE(world.registry().get<DashRequest>(entity).requested);
    EXPECT_FLOAT_EQ(world.registry().get<DashCooldown>(entity).remaining_seconds.count(), 0.5f);
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f + gameplay_config::player::MoveSpeed * 0.5f);
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f - gameplay_config::player::MoveSpeed);
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

    const auto& transform = world.registry().get<Transform>(entity);
    const float expected_delta = gameplay_config::player::MoveSpeed / std::sqrt(2.0f);
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 10.0f + gameplay_config::player::MoveSpeed);
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
    EXPECT_FALSE(world.registry().has<Transform>(entity));
    EXPECT_FALSE(world.registry().has<PlayerController>(entity));
    EXPECT_FALSE(world.registry().has<AttackRequest>(entity));
    EXPECT_FALSE(world.registry().has<AttackDefinition>(entity));
    EXPECT_FALSE(world.registry().has<AttackCooldown>(entity));
    EXPECT_FALSE(world.registry().has<AttackState>(entity));
    EXPECT_FALSE(world.registry().has<MoveRequest>(entity));
    EXPECT_FALSE(world.registry().has<MoveIntent>(entity));
    EXPECT_FALSE(world.registry().has<DashRequest>(entity));
    EXPECT_FALSE(world.registry().has<DashIntent>(entity));
    EXPECT_FALSE(world.registry().has<Dash>(entity));
    EXPECT_FALSE(world.registry().has<DashCooldown>(entity));
    EXPECT_FALSE(world.registry().has<PlayerProgress>(entity));
    EXPECT_FALSE(world.registry().has<BlessingInventory>(entity));
    EXPECT_FALSE(world.registry().has<StatusEffects>(entity));
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

    EXPECT_FALSE(world.destroy_entity(Entity{}));
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

    EXPECT_NE(entity, Entity{});
    EXPECT_TRUE(world.has_entity(entity));

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 30.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 40.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
    EXPECT_FALSE(world.registry().has<PlayerController>(entity));
    EXPECT_TRUE(world.registry().has<MonsterController>(entity));
    ASSERT_TRUE(world.registry().has<MonsterIdentity>(entity));
    EXPECT_EQ(world.registry().get<MonsterIdentity>(entity).kind, battle::MonsterKind::Melee);
}

TEST(WorldTest, CreateMonsterInitializesStatusEffects) {
    World world;

    auto entity = world.create_monster(default_monster_config());

    ASSERT_TRUE(world.registry().has<StatusEffects>(entity));
    EXPECT_TRUE(world.registry().get<StatusEffects>(entity).burns.empty());
    EXPECT_FALSE(world.registry().get<StatusEffects>(entity).freeze.has_value());
}

TEST(WorldTest, CreateMonsterInitializesAttackComponentsFromConfig) {
    World world;

    auto entity = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 30.0f, .y = 40.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 12,
            .range = 1.25f,
            .cooldown_seconds = DeltaTime{1.5f},
            .windup_seconds = DeltaTime{0.35f},
            .active_seconds = DeltaTime{0.10f},
            .recovery_seconds = DeltaTime{0.55f},
            .movement_multiplier = 0.0f,
            .projectile_speed = 0.0f,
        },
    });

    ASSERT_TRUE(world.registry().has<AttackRequest>(entity));
    ASSERT_TRUE(world.registry().has<AttackDefinition>(entity));
    ASSERT_TRUE(world.registry().has<AttackCooldown>(entity));
    ASSERT_TRUE(world.registry().has<AttackState>(entity));

    const auto& attack = world.registry().get<AttackDefinition>(entity);
    const auto& attack_state = world.registry().get<AttackState>(entity);
    EXPECT_FALSE(world.registry().get<AttackRequest>(entity).requested);
    EXPECT_EQ(attack.kind, AttackKind::Melee);
    EXPECT_EQ(attack.damage, 12);
    EXPECT_FLOAT_EQ(attack.range, 1.25f);
    EXPECT_FLOAT_EQ(attack.cooldown_seconds.count(), 1.5f);
    EXPECT_FLOAT_EQ(attack.windup_seconds.count(), 0.35f);
    EXPECT_FLOAT_EQ(attack.active_seconds.count(), 0.10f);
    EXPECT_FLOAT_EQ(attack.recovery_seconds.count(), 0.55f);
    EXPECT_FLOAT_EQ(attack.movement_multiplier, 0.0f);
    EXPECT_FLOAT_EQ(world.registry().get<AttackCooldown>(entity).remaining_seconds.count(), 0.0f);
    EXPECT_EQ(attack_state.phase, AttackPhase::Idle);
    EXPECT_FLOAT_EQ(attack_state.phase_remaining.count(), 0.0f);
    EXPECT_TRUE(attack_state.hit_targets.empty());
    EXPECT_FALSE(attack_state.projectile_spawned);
}

TEST(WorldTest, CreateMonsterAttachesConfiguredKitingAIOnly) {
    World world;
    const auto ranged_monster = world.create_monster(CreateMonsterConfig{
        .kind = MonsterKind::Ranged,
        .max_health = 50,
        .move_speed = 3.0f,
        .kiting_ai = KitingAI{
            .retreat_distance = 5.0f,
        },
    });
    const auto melee_monster = world.create_monster(CreateMonsterConfig{
        .kind = MonsterKind::Melee,
        .max_health = 50,
        .move_speed = 3.0f,
    });

    ASSERT_TRUE(world.registry().has<KitingAI>(ranged_monster));
    EXPECT_FLOAT_EQ(world.registry().get<KitingAI>(ranged_monster).retreat_distance, 5.0f);
    EXPECT_FALSE(world.registry().has<KitingAI>(melee_monster));
}

TEST(WorldTest, TickMonsterDoesNotMoveDuringAttackTimeline) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    world.registry().remove<Collider>(player);
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 10,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{2.0f},
            .windup_seconds = DeltaTime{0.5f},
            .active_seconds = DeltaTime{0.2f},
            .recovery_seconds = DeltaTime{0.5f},
            .movement_multiplier = 0.0f,
            .projectile_speed = 0.0f,
        },
    });

    for (const auto phase : {AttackPhase::Windup, AttackPhase::Active, AttackPhase::Recovery}) {
        auto& transform = world.registry().get<Transform>(monster);
        auto& velocity = world.registry().get<Velocity>(monster);
        auto& attack_state = world.registry().get<AttackState>(monster);
        transform.position = Position{.x = 0.0f, .y = 0.0f};
        velocity = Velocity{.x = 3.0f, .y = 0.0f};
        attack_state.phase = phase;
        attack_state.phase_remaining = DeltaTime{1.0f};

        world.tick(DeltaTime{0.1f});

        EXPECT_FLOAT_EQ(transform.position.x, 0.0f);
        EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
        EXPECT_FLOAT_EQ(velocity.x, 0.0f);
        EXPECT_FLOAT_EQ(velocity.y, 0.0f);
    }
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

    const auto& transform = world.registry().get<Transform>(entity);
    EXPECT_FLOAT_EQ(transform.position.x, 30.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 40.0f);
}

TEST(WorldTest, TickMonsterDamagesPlayerInsideAttackRange) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.5f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .collision_radius = 0.01f,
    });
    world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 15,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        },
        .collision_radius = 0.01f,
    });

    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.has_entity(player));
    EXPECT_EQ(world.registry().get<Health>(player).current_health, 85);
    EXPECT_TRUE(world.damage_events().empty());
}

TEST(WorldTest, TickMonsterDoesNotDamagePlayerOutsideAttackRange) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 5.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 1.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 15,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        },
    });

    world.tick(DeltaTime{1.0f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health, 100);
    EXPECT_FLOAT_EQ(world.registry().get<Transform>(monster).position.x, 1.0f);
    EXPECT_FALSE(world.registry().get<AttackRequest>(monster).requested);
}

TEST(WorldTest, TickMonsterAttackUsesCooldown) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.5f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .collision_radius = 0.01f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 15,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        },
        .collision_radius = 0.01f,
    });

    world.tick(DeltaTime{0.0f});
    ASSERT_EQ(world.registry().get<Health>(player).current_health, 85);
    EXPECT_FLOAT_EQ(world.registry().get<AttackCooldown>(monster).remaining_seconds.count(), 1.0f);

    world.tick(DeltaTime{0.5f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health, 85);
    EXPECT_FLOAT_EQ(world.registry().get<AttackCooldown>(monster).remaining_seconds.count(), 0.5f);
}

TEST(WorldTest, TickMonsterDoesNotDamageOtherMonsters) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.9f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
        .collision_radius = 0.1f,
    });
    auto attacker = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 15,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        },
    });
    auto other_monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.5f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 15,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        },
    });
    world.registry().get<AttackState>(attacker) = AttackState{
        .phase = AttackPhase::Active,
        .locked_direction = Direction{.x = 0.0f, .y = 1.0f},
    };

    hit_resolve_system(world, DeltaTime{0.0f});
    damage_system(world, DeltaTime{0.0f});

    EXPECT_EQ(world.registry().get<Health>(other_monster).current_health, 50);
}

TEST(WorldTest, TickMovesMonsterTowardPlayerUsingMonsterMoveSpeed) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 3.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 0.0f);
}

TEST(WorldTest, TickRangedMonsterRetreatsWhenPlayerIsTooClose) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .kind = MonsterKind::Ranged,
        .position = Position{.x = 4.0f, .y = 0.0f},
        .max_health = 35,
        .move_speed = 3.5f,
        .attack = AttackDefinition{
            .kind = AttackKind::Projectile,
            .damage = 12,
            .range = 8.0f,
            .cooldown_seconds = DeltaTime{1.4f},
            .projectile_speed = 18.0f,
        },
        .kiting_ai = KitingAI{
            .retreat_distance = 5.0f,
        },
    });

    world.tick(DeltaTime{1.0f});

    EXPECT_FLOAT_EQ(world.registry().get<Transform>(monster).position.x, 7.5f);
    EXPECT_TRUE(world.registry().pool<Projectile>().empty());
}

TEST(WorldTest, TickRangedMonsterFacesPlayerAndFiresWithinSafeRange) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .kind = MonsterKind::Ranged,
        .position = Position{.x = 6.0f, .y = 0.0f},
        .max_health = 35,
        .move_speed = 3.5f,
        .attack = AttackDefinition{
            .kind = AttackKind::Projectile,
            .damage = 12,
            .range = 8.0f,
            .cooldown_seconds = DeltaTime{1.4f},
            .projectile_speed = 18.0f,
        },
        .kiting_ai = KitingAI{
            .retreat_distance = 5.0f,
        },
    });

    world.tick(DeltaTime{0.0f});

    const auto& monster_transform = world.registry().get<Transform>(monster);
    EXPECT_FLOAT_EQ(monster_transform.direction.x, -1.0f);
    EXPECT_FLOAT_EQ(monster_transform.direction.y, 0.0f);
    ASSERT_EQ(world.registry().pool<Projectile>().entities().size(), 1);
    const auto projectile = world.registry().pool<Projectile>().entities().front();
    EXPECT_FLOAT_EQ(world.registry().get<Velocity>(projectile).x, -18.0f);
    EXPECT_FLOAT_EQ(world.registry().get<Velocity>(projectile).y, 0.0f);
}

TEST(WorldTest, TickMeleeMonsterDoesNotRetreatWithoutKitingAI) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .kind = MonsterKind::Melee,
        .position = Position{.x = 4.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    });

    world.tick(DeltaTime{1.0f});

    EXPECT_FLOAT_EQ(world.registry().get<Transform>(monster).position.x, 1.0f);
    EXPECT_FALSE(world.registry().has<KitingAI>(monster));
}

TEST(WorldTest, TickFrozenMonsterDoesNotMoveTowardPlayer) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    });
    world.registry().get<StatusEffects>(monster).freeze = FreezeStatus{
        .remaining_seconds = DeltaTime{2.0f},
    };

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
    EXPECT_FLOAT_EQ(world.registry().get<Velocity>(monster).x, 0.0f);
    EXPECT_FLOAT_EQ(world.registry().get<Velocity>(monster).y, 0.0f);
    ASSERT_TRUE(world.registry().get<StatusEffects>(monster).freeze.has_value());
    EXPECT_FLOAT_EQ(world.registry().get<StatusEffects>(monster).freeze->remaining_seconds.count(), 1.0f);
}

TEST(WorldTest, TickFrozenMonsterDoesNotDamagePlayerInsideAttackRange) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.5f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
        .attack = AttackDefinition{
            .kind = AttackKind::Melee,
            .damage = 15,
            .range = 1.0f,
            .cooldown_seconds = DeltaTime{1.0f},
            .projectile_speed = 0.0f,
        },
    });
    world.registry().get<StatusEffects>(monster).freeze = FreezeStatus{
        .remaining_seconds = DeltaTime{2.0f},
    };

    world.tick(DeltaTime{0.0f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health, 100);
    EXPECT_FALSE(world.registry().get<AttackRequest>(monster).requested);
}

TEST(WorldTest, TickMonsterMovesAfterFreezeExpires) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    });
    world.registry().get<StatusEffects>(monster).freeze = FreezeStatus{
        .remaining_seconds = DeltaTime{0.5f},
    };

    world.tick(DeltaTime{1.0f});

    EXPECT_FALSE(world.registry().get<StatusEffects>(monster).freeze.has_value());
    EXPECT_FLOAT_EQ(world.registry().get<Transform>(monster).position.x, 3.0f);
    EXPECT_FLOAT_EQ(world.registry().get<Transform>(monster).position.y, 0.0f);
}

TEST(WorldTest, TickClampsMonsterPositionToWorldBounds) {
    World world(WorldBounds{
        .min_x = -1.0f,
        .max_x = 1.0f,
        .min_y = -1.0f,
        .max_y = 1.0f,
    });
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 5.0f,
        .collision_radius = 0.1f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
    EXPECT_GE(transform.position.x, -0.9f);
    EXPECT_LE(transform.position.x, 0.9f);
    EXPECT_GE(transform.position.y, -0.9f);
    EXPECT_LE(transform.position.y, 0.9f);
    EXPECT_TRUE(std::isfinite(transform.position.x));
    EXPECT_TRUE(std::isfinite(transform.position.y));
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
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 2.0f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 2.0f);
    EXPECT_FLOAT_EQ(transform.direction.x, 0.0f);
    EXPECT_FLOAT_EQ(transform.direction.y, 1.0f);
}

TEST(WorldTest, TickDoesNotMoveMonsterWithoutPlayerTarget) {
    World world;
    auto monster = world.create_monster(default_monster_config());

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
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
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    });
    world.tick(DeltaTime{1.0f});
    ASSERT_TRUE(world.destroy_entity(player));

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
    EXPECT_FLOAT_EQ(transform.position.x, 3.0f);
    EXPECT_FLOAT_EQ(transform.position.y, 0.0f);
}

TEST(WorldTest, TickResolvesOverlappingMonsterSpawnWithoutNan) {
    World world;
    auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 100,
        .move_speed = 5.0f,
    });
    auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 10.0f, .y = 20.0f},
        .max_health = 50,
        .move_speed = 3.0f,
    });

    world.tick(DeltaTime{1.0f});

    const auto& transform = world.registry().get<Transform>(monster);
    const auto& player_transform = world.registry().get<Transform>(player);
    EXPECT_TRUE(std::isfinite(transform.position.x));
    EXPECT_TRUE(std::isfinite(transform.position.y));
    EXPECT_TRUE(transform.position.x != player_transform.position.x ||
                transform.position.y != player_transform.position.y);
}

TEST(WorldTest, DestroyMonsterRemovesEntityComponents) {
    World world;
    auto entity = world.create_monster(default_monster_config());

    EXPECT_TRUE(world.destroy_entity(entity));

    EXPECT_FALSE(world.has_entity(entity));
    EXPECT_FALSE(world.registry().has<Transform>(entity));
    EXPECT_FALSE(world.registry().has<MonsterController>(entity));
    EXPECT_FALSE(world.registry().has<MonsterIdentity>(entity));
    EXPECT_FALSE(world.registry().has<StatusEffects>(entity));
}

TEST(WorldTest, HasLivingMonstersReflectsMonsterControllerEntities) {
    World world;

    EXPECT_FALSE(world.has_living_monsters());
    EXPECT_EQ(world.living_monster_count(), 0);

    auto first_monster = world.create_monster(default_monster_config());
    EXPECT_TRUE(world.has_living_monsters());
    EXPECT_EQ(world.living_monster_count(), 1);

    auto second_monster = world.create_monster(default_monster_config());
    EXPECT_TRUE(world.has_living_monsters());
    EXPECT_EQ(world.living_monster_count(), 2);

    ASSERT_TRUE(world.destroy_entity(first_monster));
    EXPECT_TRUE(world.has_living_monsters());
    EXPECT_EQ(world.living_monster_count(), 1);

    ASSERT_TRUE(world.destroy_entity(second_monster));
    EXPECT_FALSE(world.has_living_monsters());
    EXPECT_EQ(world.living_monster_count(), 0);
}

TEST(WorldTest, CreateObstacleAndTrapInitializeCollisionComponents) {
    World world;

    const auto obstacle = world.create_obstacle(CreateObstacleConfig{
        .position = Position{.x = 2.0f, .y = 3.0f},
        .radius = 1.5f,
    });
    const auto trap = world.create_trap(CreateTrapConfig{
        .position = Position{.x = -2.0f, .y = -3.0f},
        .radius = 2.5f,
        .kind = TrapKind::PoisonPool,
    });

    const auto& obstacle_collider = world.registry().get<Collider>(obstacle);
    EXPECT_EQ(obstacle_collider.category, CollisionCategory::Obstacle);
    EXPECT_EQ(obstacle_collider.collision_mask, ObstacleCollisionMask);
    EXPECT_FLOAT_EQ(obstacle_collider.radius, 1.5f);

    const auto& trap_collider = world.registry().get<Collider>(trap);
    EXPECT_EQ(trap_collider.category, CollisionCategory::Trap);
    EXPECT_EQ(trap_collider.collision_mask, TrapCollisionMask);
    EXPECT_FLOAT_EQ(trap_collider.radius, 2.5f);
    EXPECT_EQ(world.registry().get<Trap>(trap).kind, TrapKind::PoisonPool);
}

TEST(WorldTest, SpikesDamagePlayersAndMonstersOnlyWhenEntering) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 10.0f,
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 3.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 0.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 4.0f,
        .kind = TrapKind::Spikes,
    });

    world.tick(DeltaTime{0.0f});
    EXPECT_EQ(world.registry().get<Health>(player).current_health,
              100 - gameplay_config::trap::spikes::Damage);
    EXPECT_EQ(world.registry().get<Health>(monster).current_health,
              100 - gameplay_config::trap::spikes::Damage);

    world.tick(DeltaTime{0.0f});
    EXPECT_EQ(world.registry().get<Health>(player).current_health,
              100 - gameplay_config::trap::spikes::Damage);
    EXPECT_EQ(world.registry().get<Health>(monster).current_health,
              100 - gameplay_config::trap::spikes::Damage);
}

TEST(WorldTest, SpikesDamagePlayerAgainAfterLeavingAndReentering) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 10.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 0.5f,
        .kind = TrapKind::Spikes,
    });

    world.tick(DeltaTime{0.0f});
    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.2f});
    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = -1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.2f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health,
              100 - 2 * gameplay_config::trap::spikes::Damage);
}

TEST(WorldTest, PoisonPoolAppliesOneRefreshableStatusToPlayersAndMonsters) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 0.0f,
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 3.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 0.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 4.0f,
        .kind = TrapKind::PoisonPool,
    });

    world.tick(DeltaTime{0.0f});
    ASSERT_TRUE(world.registry().get<StatusEffects>(player).poison.has_value());
    ASSERT_TRUE(world.registry().get<StatusEffects>(monster).poison.has_value());

    world.tick(DeltaTime{0.2f});
    const auto& poison = world.registry().get<StatusEffects>(player).poison;
    ASSERT_TRUE(poison.has_value());
    EXPECT_FLOAT_EQ(poison->remaining_seconds.count(), gameplay_config::trap::poison_pool::Duration.count());
    EXPECT_FLOAT_EQ(poison->tick_timer_seconds.count(), 0.2f);
    EXPECT_EQ(poison->damage_per_tick, gameplay_config::trap::poison_pool::DamagePerTick);
}

TEST(WorldTest, PoisonPoolContinuesDamagingAfterExitAndExpires) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 10.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 0.1f,
        .kind = TrapKind::PoisonPool,
    });

    world.tick(DeltaTime{0.0f});
    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.1f});
    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 0.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.4f});

    EXPECT_EQ(world.registry().get<Health>(player).current_health,
              100 - gameplay_config::trap::poison_pool::DamagePerTick);
    ASSERT_TRUE(world.registry().get<StatusEffects>(player).poison.has_value());

    world.tick(DeltaTime{2.5f});
    EXPECT_EQ(world.registry().get<Health>(player).current_health,
              100 - 6 * gameplay_config::trap::poison_pool::DamagePerTick);
    EXPECT_FALSE(world.registry().get<StatusEffects>(player).poison.has_value());
}

TEST(WorldTest, SwampSlowsPlayerAndExpiresAfterExit) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 10.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 0.5f,
        .kind = TrapKind::Swamp,
    });

    world.tick(DeltaTime{0.0f});
    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = false,
                                                 }));
    world.tick(DeltaTime{0.2f});
    EXPECT_FLOAT_EQ(world.registry().get<Transform>(player).position.x,
                    2.0f * gameplay_config::trap::swamp::MovementMultiplier);

    world.tick(DeltaTime{gameplay_config::trap::swamp::Duration.count()});
    EXPECT_FALSE(world.registry().get<StatusEffects>(player).swamp.has_value());
}

TEST(WorldTest, SwampSlowsChasingMonster) {
    World world;
    world.create_player(CreatePlayerConfig{
        .position = Position{.x = 10.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 0.0f,
    });
    const auto monster = world.create_monster(CreateMonsterConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 3.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 2.0f,
        .kind = TrapKind::Swamp,
    });

    world.tick(DeltaTime{0.0f});
    world.tick(DeltaTime{0.1f});

    EXPECT_FLOAT_EQ(world.registry().get<Transform>(monster).position.x,
                    0.3f * gameplay_config::trap::swamp::MovementMultiplier);
}

TEST(WorldTest, DashUsesFullSpeedInsideSwamp) {
    World world;
    const auto player = world.create_player(CreatePlayerConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .max_health = 100,
        .move_speed = 10.0f,
    });
    world.create_trap(CreateTrapConfig{
        .position = Position{.x = 0.0f, .y = 0.0f},
        .radius = 2.0f,
        .kind = TrapKind::Swamp,
    });
    world.tick(DeltaTime{0.0f});

    ASSERT_TRUE(world.set_player_command(player, PlayerCommand{
                                                     .move_x = 1.0f,
                                                     .move_y = 0.0f,
                                                     .attack_requested = false,
                                                     .dash_requested = true,
                                                 }));
    world.tick(DeltaTime{0.1f});

    EXPECT_FLOAT_EQ(world.registry().get<Transform>(player).position.x,
                    10.0f * gameplay_config::player::DashSpeedMultiplier * 0.1f);
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
    EXPECT_EQ(entity_snapshot.kind, EntityKind::Player);
    EXPECT_FLOAT_EQ(entity_snapshot.position.x, 10.0f + gameplay_config::player::MoveSpeed);
    EXPECT_FLOAT_EQ(entity_snapshot.position.y, 20.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.direction.x, 1.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.direction.y, 0.0f);
    EXPECT_EQ(entity_snapshot.current_health, gameplay_config::player::MaxHealth);
    EXPECT_EQ(entity_snapshot.max_health, gameplay_config::player::MaxHealth);
}

TEST(WorldTest, SnapshotIncludesPlayersAndMonsters) {
    World world;
    auto player = world.create_player(default_player_config());
    auto monster = world.create_monster(default_monster_config());

    auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 2);
    EXPECT_TRUE((snapshot.entities[0].entity == player && snapshot.entities[1].entity == monster) ||
                (snapshot.entities[0].entity == monster && snapshot.entities[1].entity == player));
    for (const auto& entity_snapshot : snapshot.entities) {
        if (entity_snapshot.entity == player) {
            EXPECT_EQ(entity_snapshot.kind, EntityKind::Player);
        }
        if (entity_snapshot.entity == monster) {
            EXPECT_EQ(entity_snapshot.kind, EntityKind::Monster);
        }
    }
}

TEST(WorldTest, SnapshotIncludesProjectileWithoutHealth) {
    World world;
    auto projectile = world.create_projectile(CreateProjectileConfig{
        .position = Position{.x = 3.0f, .y = 4.0f},
        .direction = Direction{.x = 0.6f, .y = 0.8f},
        .speed = 10.0f,
        .damage = 30,
        .max_distance = 20.0f,
    });

    const auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 1);
    const auto& entity_snapshot = snapshot.entities.front();
    EXPECT_EQ(entity_snapshot.entity, projectile);
    EXPECT_EQ(entity_snapshot.kind, EntityKind::Projectile);
    EXPECT_FLOAT_EQ(entity_snapshot.position.x, 3.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.position.y, 4.0f);
    EXPECT_FLOAT_EQ(entity_snapshot.direction.x, 0.6f);
    EXPECT_FLOAT_EQ(entity_snapshot.direction.y, 0.8f);
    EXPECT_EQ(entity_snapshot.current_health, 0);
    EXPECT_EQ(entity_snapshot.max_health, 0);
}

TEST(WorldTest, SnapshotIncludesObstacleAndTrapPresentationData) {
    World world;
    const auto obstacle = world.create_obstacle(CreateObstacleConfig{
        .position = Position{.x = 2.0f, .y = 3.0f},
        .radius = 1.5f,
    });
    const auto trap = world.create_trap(CreateTrapConfig{
        .position = Position{.x = -2.0f, .y = -3.0f},
        .radius = 2.5f,
        .kind = TrapKind::Swamp,
    });

    const auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 2);
    for (const auto& entity_snapshot : snapshot.entities) {
        if (entity_snapshot.entity == obstacle) {
            EXPECT_EQ(entity_snapshot.kind, EntityKind::Obstacle);
            EXPECT_EQ(entity_snapshot.scene_object_kind, "obstacle");
            EXPECT_FLOAT_EQ(entity_snapshot.collision_radius, 1.5f);
        } else if (entity_snapshot.entity == trap) {
            EXPECT_EQ(entity_snapshot.kind, EntityKind::Trap);
            EXPECT_EQ(entity_snapshot.scene_object_kind, "swamp");
            EXPECT_FLOAT_EQ(entity_snapshot.collision_radius, 2.5f);
        } else {
            FAIL() << "unexpected entity in snapshot";
        }
    }
}

TEST(WorldTest, SnapshotIncludesRangedMonsterKind) {
    World world;
    const auto monster = world.create_monster(CreateMonsterConfig{
        .kind = MonsterKind::Ranged,
        .position = Position{.x = 3.0f, .y = 4.0f},
        .max_health = 35,
        .move_speed = 3.5f,
    });

    const auto snapshot = world.snapshot();

    ASSERT_EQ(snapshot.entities.size(), 1);
    const auto& entity_snapshot = snapshot.entities.front();
    EXPECT_EQ(entity_snapshot.entity, monster);
    EXPECT_EQ(entity_snapshot.kind, EntityKind::Monster);
    ASSERT_TRUE(entity_snapshot.monster_kind.has_value());
    EXPECT_EQ(entity_snapshot.monster_kind.value(), MonsterKind::Ranged);
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
