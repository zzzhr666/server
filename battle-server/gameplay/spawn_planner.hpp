#pragma once

#include <cstddef>
#include <cstdint>

#include "ecs/world.hpp"


namespace battle {
    class SpawnPlanner {
    public:
        [[nodiscard]] ecs::CreatePlayerConfig player_spawn(std::size_t index) const;
        [[nodiscard]] ecs::CreateMonsterConfig monster_spawn(std::size_t index, std::size_t count) const;
    };
}
