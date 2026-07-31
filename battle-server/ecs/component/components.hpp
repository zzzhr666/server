#pragma once

#include <cstdint>
#include <optional>
#include <vector>

#include "ecs/entity/entity.hpp"
#include "ecs/time.hpp"
#include "ecs/combat/combat.hpp"
#include "gameplay/blessing.hpp"
#include "gameplay/monster_kind.hpp"

namespace battle::ecs {
    struct PlayerCommand {
        float move_x;
        float move_y;
        bool attack_requested;
        bool dash_requested;
    };

    struct Position {
        float x;
        float y;
    };

    struct Direction {
        float x;
        float y;
    };

    struct Transform {
        Position position;
        Direction direction;
    };

    struct Velocity {
        float x;
        float y;
    };

    struct MoveRequest {
        float x;
        float y;
    };

    struct AttackRequest {
        bool requested;
    };

    struct DashRequest {
        bool requested;
    };

    struct MoveIntent {
        float x;
        float y;
    };

    struct PlayerController {};

    struct MonsterController {};

    struct CharacterStats {
        float move_speed;
    };

    struct Health {
        int current_health;
        int max_health;
    };

    enum class AttackKind {
        Melee,
        Projectile,
    };

    struct AttackIntent {
        bool active{};
        AttackKind kind{};
        int damage{};
        float range{};
        float projectile_speed{};
        CombatContext context;
    };

    struct DashIntent {
        bool active;
        float dash_speed_multiplier;
    };

    struct AttackDefinition {
        AttackKind kind;
        int damage;
        float range;
        DeltaTime cooldown_seconds;
        float projectile_speed;
    };

    struct Dash {
        DeltaTime cooldown_seconds;
        float dash_speed_multiplier;
    };

    struct AttackCooldown {
        DeltaTime remaining_seconds;
    };

    struct DashCooldown {
        DeltaTime remaining_seconds;
    };

    enum class DamageSourceKind {
        Attack,
        Burn,
        ChainLightning,
    };

    struct DamageEvent {
        Entity source{};
        Entity target{};
        int base_damage{};
        int modified_damage{};
        DamageSourceKind source_kind{DamageSourceKind::Attack};
        CombatContext context;
    };

    struct MonsterIdentity {
        MonsterKind kind;
    };

    struct KillEvent {
        Entity killer;
        Entity victim;
        MonsterKind monster_kind{};
    };

    struct PlayerProgress {
        int level;
        int experience;
        int experience_to_next_level;
        int pending_upgrade_choices;
    };

    struct BlessingStack {
        BlessingID blessing_id = BlessingID::BurnOnHit;
        int level = 1;
    };

    struct BlessingInventory {
        std::vector<BlessingStack> blessings;
    };

    struct DamageAppliedEvent {
        Entity source{};
        Entity target{};
        int amount{};
        DamageSourceKind source_kind{DamageSourceKind::Attack};
        CombatContext context;
    };

    struct BurnStatus {
        Entity source{};
        DeltaTime remaining_seconds{0.0f};
        DeltaTime tick_interval_seconds{1.0f};
        DeltaTime tick_timer_seconds{0.0f};
        int damage_per_tick{};
    };

    struct FreezeStatus {
        DeltaTime remaining_seconds{0.0f};
    };

    struct StatusEffects {
        std::vector<BurnStatus> burns;
        std::optional<FreezeStatus> freeze;
    };
}
