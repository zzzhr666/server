#pragma once

#include <cstdint>

namespace battle::ecs {
    struct Position {
        float x;
        float y;
        Position& operator += (const Position& position) {
            x += position.x;
            y += position.y;
            return *this;
        }
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

    struct MoveInput {
        float x;
        float y;
    };

    struct PlayerController {
        std::int64_t player_id;
    };
    struct CharacterStats {
        float move_speed;
    };

    struct Health {
        int current_health;
        int max_health;
    };
}
