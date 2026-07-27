#pragma once

#include <cstdint>

#include "ecs/entity/entity.hpp"
#include "ecs/time.hpp"

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

    struct AttackIntent {
        bool active;
        int damage;
        float range;
    };

    struct DashIntent {
        bool active;
        float dash_speed_multiplier;
    };

    struct MeleeAttack {
        int damage;
        float range;
        DeltaTime cooldown_seconds;
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

    struct DamageEvent {
        Entity source;
        Entity target;
        int base_damage;
    };

}
