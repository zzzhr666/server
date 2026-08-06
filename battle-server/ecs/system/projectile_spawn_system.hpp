#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 根据投射物攻击意图创建投射物实体。
    void projectile_spawn_system(World& world, DeltaTime delta_seconds);
}
