#include "gameplay/spawn_planner.hpp"

#include <cmath>
#include <numbers>

battle::ecs::CreatePlayerConfig battle::SpawnPlanner::player_spawn(std::size_t index) const {
    index %= 4;
    ecs::CreatePlayerConfig config{
        .max_health = 100,
        .move_speed = 5.0f,
    };
    switch (index) {
    case 0: {
        config.x_position = -2.0f;
        break;
    }
    case 1: {
        config.x_position = 2.0f;
        break;
    }
    case 2: {
        config.y_position = -2.0f;
        break;
    }
    case 3: {
        config.y_position = 2.0f;
        break;
    }
    default:
        return config;
    }
    return config;
}


battle::ecs::CreateMonsterConfig battle::SpawnPlanner::monster_spawn(std::size_t index, std::size_t count) const {
    if (count == 0) {
        count = 1;
    }
    double radius = 8.0f;
    double angle = 2 * std::numbers::pi / static_cast<double>(count) * static_cast<double>(index);
    auto x = static_cast<float>(std::cos(angle) * radius);
    auto y = static_cast<float>(std::sin(angle) * radius);
    return {
        .x_position = x,
        .y_position = y,
        .max_health = 50,
        .move_speed = 3.0f,
    };
}
