#include "gameplay/spawn_planner.hpp"

#include <cmath>
#include <numbers>

#include "hero.hpp"

battle::ecs::CreatePlayerConfig battle::SpawnPlanner::player_spawn(std::size_t index) const { //NOLINT
    index %= gameplay_config::spawn::PlayerSlotCount;
    auto hero = hero_definition(HeroKind::Fire);
    ecs::CreatePlayerConfig config{
        .max_health = gameplay_config::player::MaxHealth,
        .move_speed = gameplay_config::player::MoveSpeed,
        .attack = hero.attack,
    };

    switch (index) {
    case 0: {
        config.position.x = -gameplay_config::spawn::PlayerOffset;
        break;
    }
    case 1: {
        config.position.x = gameplay_config::spawn::PlayerOffset;
        break;
    }
    case 2: {
        config.position.y = -gameplay_config::spawn::PlayerOffset;
        break;
    }
    case 3: {
        config.position.y = gameplay_config::spawn::PlayerOffset;
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
    double radius = gameplay_config::spawn::MonsterRadius;
    double angle = 2 * std::numbers::pi / static_cast<double>(count) * static_cast<double>(index);
    auto x = static_cast<float>(std::cos(angle) * radius);
    auto y = static_cast<float>(std::sin(angle) * radius);
    return {
        .position = {.x = x, .y = y},
    };
}
