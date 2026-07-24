#pragma once

#include <cstdint>

namespace battle::ecs {
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

    struct MoveInput {
        float x;
        float y;
    };

    struct PlayerController {};

    struct CharacterStats {
        float move_speed;
    };

    struct Health {
        int current_health;
        int max_health;
    };
}
