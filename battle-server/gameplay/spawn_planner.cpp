#include "gameplay/spawn_planner.hpp"

#include <cmath>
#include <numbers>

#include "weapon.hpp"

battle::ecs::CreatePlayerConfig battle::SpawnPlanner::player_spawn(std::size_t index) const { //NOLINT
    index %= 4;
    auto weapon = weapon_definition(WeaponKind::Sword);
    ecs::CreatePlayerConfig config{
        .max_health = ecs::DefaultPlayerMaxHealth,
        .move_speed = ecs::DefaultPlayerMoveSpeed,
        .attack = weapon.attack,
    };

    switch (index) {
    case 0: {
        config.position.x = -2.0f;
        break;
    }
    case 1: {
        config.position.x = 2.0f;
        break;
    }
    case 2: {
        config.position.y = -2.0f;
        break;
    }
    case 3: {
        config.position.y = 2.0f;
        break;
    }
    default:
        return config;
    }
    return config;
}


battle::ecs::CreateMonsterConfig battle::SpawnPlanner::monster_spawn(std::size_t index, std::size_t count) const { //NOLINT
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
    };
}
