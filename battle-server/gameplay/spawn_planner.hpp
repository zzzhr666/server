#pragma once

#include <cstddef>
#include <cstdint>

#include "ecs/world.hpp"


namespace battle {
    /// @brief SpawnPlanner 根据 roster 或波次序号生成确定性的出生位置。
    class SpawnPlanner {
    public:
        /// @brief 返回指定 roster 位置的玩家出生配置。
        [[nodiscard]] ecs::CreatePlayerConfig player_spawn(std::size_t index) const;
        /// @brief 返回指定波内序号怪物的出生配置。
        [[nodiscard]] ecs::CreateMonsterConfig monster_spawn(std::size_t index, std::size_t count) const;
    };
}
