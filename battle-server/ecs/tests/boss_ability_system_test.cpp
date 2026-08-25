#include "ecs/system/boss_ability_system.hpp"
#include "ecs/world.hpp"

#include <gtest/gtest.h>

namespace battle::ecs {
namespace {

MapConfig test_map_config() {
    return MapConfig{
        .bounds = WorldBounds{
            .min_x = -50.0f,
            .max_x = 50.0f,
            .min_y = -50.0f,
            .max_y = 50.0f,
        },
        .cell_size = 5.0f,
    };
}

CreateMonsterConfig boss_config(Position position) {
    return CreateMonsterConfig{
        .kind = MonsterKind::Boss,
        .position = position,
        .max_health = gameplay_config::monster::boss::Health,
        .move_speed = gameplay_config::monster::boss::MoveSpeed,
        .collision_radius = gameplay_config::monster::boss::CollisionRadius,
    };
}

CreatePlayerConfig player_config(Position position) {
    return CreatePlayerConfig{
        .position = position,
        .max_health = gameplay_config::player::MaxHealth,
        .move_speed = 0.0f,
    };
}

std::size_t projectile_count_for_owner(const World& world, Entity owner) {
    std::size_t count = 0;
    for (const auto projectile_entity : world.registry().pool<Projectile>().entities()) {
        const auto& projectile = world.registry().get<Projectile>(projectile_entity);
        if (projectile.context.owner == owner) {
            ++count;
        }
    }
    return count;
}

TEST(BossAbilitySystemTest, StartsTripleDashAndQueuesNextSkill) {
    World world({boss_ability_system}, test_map_config(), 7);
    const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
    const auto player = world.create_player(player_config(Position{.x = 3.0f, .y = 0.0f}));

    ASSERT_NE(boss, NullEntity);
    ASSERT_NE(player, NullEntity);

    world.tick(DeltaTime{0.0f});

    const auto& ability = world.registry().get<BossAbilityState>(boss);
    EXPECT_EQ(ability.phase, BossPhase::One);
    EXPECT_EQ(ability.kind, BossAbilityKind::TripleDash);
    EXPECT_EQ(ability.action_phase, AttackPhase::Windup);
    EXPECT_EQ(ability.next_kind, BossAbilityKind::RadialProjectile);
    EXPECT_EQ(ability.target, player);
    EXPECT_EQ(ability.remaining_seconds, gameplay_config::monster::boss::triple_dash::phase_one::Windup);
    EXPECT_EQ(ability.cooldown_remaining_seconds, DeltaTime{0.0f});
    EXPECT_NE(ability.ability_id, InvalidEffectID);
}

TEST(BossAbilitySystemTest, SwitchesToPhaseTwoAtHalfHealth) {
    World world({boss_ability_system}, test_map_config(), 11);
    const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
    const auto player = world.create_player(player_config(Position{.x = 3.0f, .y = 0.0f}));

    ASSERT_NE(boss, NullEntity);
    ASSERT_NE(player, NullEntity);

    auto& health = world.registry().get<Health>(boss);
    health.current_health = health.max_health / 2;
    world.registry().get<BossAbilityState>(boss).cooldown_remaining_seconds = DeltaTime{1.0f};

    world.tick(DeltaTime{0.0f});

    const auto& ability = world.registry().get<BossAbilityState>(boss);
    EXPECT_EQ(ability.phase, BossPhase::Two);
    EXPECT_EQ(ability.kind, BossAbilityKind::None);
}

TEST(BossAbilitySystemTest, TripleDashHitsPlayerAndUsesPhaseSpecificDashCount) {
    {
        World world({boss_ability_system}, test_map_config(), 13);
        const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
        const auto player = world.create_player(player_config(Position{.x = 1.0f, .y = 0.0f}));

        ASSERT_NE(boss, NullEntity);
        ASSERT_NE(player, NullEntity);

        auto& ability = world.registry().get<BossAbilityState>(boss);
        ability.kind = BossAbilityKind::TripleDash;
        ability.phase = BossPhase::One;
        ability.action_phase = AttackPhase::Active;
        ability.remaining_seconds = DeltaTime{0.0f};
        ability.locked_target_position = Position{.x = 3.0f, .y = 0.0f};
        ability.ability_id = world.create_combat_effect();

        world.tick(DeltaTime{0.1f});

        EXPECT_EQ(world.damage_events().size(), 1U);
        EXPECT_EQ(world.damage_events()[0].target, player);
        EXPECT_EQ(world.damage_events()[0].base_damage,
                  gameplay_config::monster::boss::triple_dash::phase_one::Damage);
        EXPECT_EQ(world.registry().get<BossAbilityState>(boss).hit_targets.size(), 1U);
    }

    {
        World world({boss_ability_system}, test_map_config(), 17);
        const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
        const auto player = world.create_player(player_config(Position{.x = 3.0f, .y = 0.0f}));

        ASSERT_NE(boss, NullEntity);
        ASSERT_NE(player, NullEntity);

        auto& ability = world.registry().get<BossAbilityState>(boss);
        ability.kind = BossAbilityKind::TripleDash;
        ability.phase = BossPhase::Two;
        ability.action_phase = AttackPhase::Recovery;
        ability.remaining_seconds = DeltaTime{0.0f};
        ability.sequence_index = 2;
        ability.target = player;
        ability.ability_id = world.create_combat_effect();

        world.tick(DeltaTime{0.1f});

        const auto& state = world.registry().get<BossAbilityState>(boss);
        EXPECT_EQ(state.kind, BossAbilityKind::TripleDash);
        EXPECT_EQ(state.action_phase, AttackPhase::Windup);
        EXPECT_EQ(state.sequence_index, 3U);
        EXPECT_EQ(state.remaining_seconds,
                  gameplay_config::monster::boss::triple_dash::phase_two::Windup);
    }
}

TEST(BossAbilitySystemTest, RadialProjectileSpawnsEightProjectilesAndUsesPhaseTwoVolleyLimit) {
    {
        World world({boss_ability_system}, test_map_config(), 19);
        const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
        const auto player = world.create_player(player_config(Position{.x = 6.0f, .y = 0.0f}));

        ASSERT_NE(boss, NullEntity);
        ASSERT_NE(player, NullEntity);

        auto& ability = world.registry().get<BossAbilityState>(boss);
        ability.kind = BossAbilityKind::RadialProjectile;
        ability.phase = BossPhase::One;
        ability.action_phase = AttackPhase::Active;
        ability.remaining_seconds = DeltaTime{0.0f};
        ability.sequence_index = 0;
        ability.ability_id = world.create_combat_effect();

        world.tick(DeltaTime{0.0f});

        EXPECT_EQ(projectile_count_for_owner(world, boss), 8U);
        const auto& state = world.registry().get<BossAbilityState>(boss);
        EXPECT_EQ(state.kind, BossAbilityKind::RadialProjectile);
        EXPECT_EQ(state.action_phase, AttackPhase::Active);
        EXPECT_EQ(state.sequence_index, 1U);
        EXPECT_EQ(state.remaining_seconds, gameplay_config::monster::boss::radial_projectile::phase_one::Interval);
    }

    {
        World world({boss_ability_system}, test_map_config(), 23);
        const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
        const auto player = world.create_player(player_config(Position{.x = 6.0f, .y = 0.0f}));

        ASSERT_NE(boss, NullEntity);
        ASSERT_NE(player, NullEntity);

        auto& ability = world.registry().get<BossAbilityState>(boss);
        ability.kind = BossAbilityKind::RadialProjectile;
        ability.phase = BossPhase::Two;
        ability.action_phase = AttackPhase::Active;
        ability.remaining_seconds = DeltaTime{0.0f};
        ability.sequence_index = 4;
        ability.ability_id = world.create_combat_effect();

        world.tick(DeltaTime{0.0f});

        EXPECT_EQ(projectile_count_for_owner(world, boss), 8U);
        const auto& state = world.registry().get<BossAbilityState>(boss);
        EXPECT_EQ(state.action_phase, AttackPhase::Recovery);
        EXPECT_EQ(state.sequence_index, 5U);
        EXPECT_EQ(state.remaining_seconds, gameplay_config::monster::boss::radial_projectile::phase_two::Recovery);
    }
}

TEST(BossAbilitySystemTest, TornadoUsesPhaseSpecificRadius) {
    {
        World world({boss_ability_system}, test_map_config(), 29);
        const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
        const auto player = world.create_player(player_config(Position{.x = 11.0f, .y = 0.0f}));

        ASSERT_NE(boss, NullEntity);
        ASSERT_NE(player, NullEntity);

        auto& ability = world.registry().get<BossAbilityState>(boss);
        ability.kind = BossAbilityKind::Tornado;
        ability.phase = BossPhase::One;
        ability.action_phase = AttackPhase::Active;
        ability.remaining_seconds = gameplay_config::monster::boss::tornado::phase_one::Active;
        ability.ability_id = world.create_combat_effect();

        world.tick(DeltaTime{0.0f});

        EXPECT_TRUE(world.damage_events().empty());
    }

    {
        World world({boss_ability_system}, test_map_config(), 31);
        const auto boss = world.create_monster(boss_config(Position{.x = 0.0f, .y = 0.0f}));
        const auto player = world.create_player(player_config(Position{.x = 11.0f, .y = 0.0f}));

        ASSERT_NE(boss, NullEntity);
        ASSERT_NE(player, NullEntity);

        auto& ability = world.registry().get<BossAbilityState>(boss);
        ability.kind = BossAbilityKind::Tornado;
        ability.phase = BossPhase::Two;
        ability.action_phase = AttackPhase::Active;
        ability.remaining_seconds = gameplay_config::monster::boss::tornado::phase_two::Active;
        ability.ability_id = world.create_combat_effect();

        world.tick(DeltaTime{0.0f});

        ASSERT_EQ(world.damage_events().size(), 1U);
        EXPECT_EQ(world.damage_events()[0].target, player);
        EXPECT_EQ(world.damage_events()[0].base_damage, gameplay_config::monster::boss::tornado::phase_two::Damage);
    }
}

} // namespace
} // namespace battle::ecs
