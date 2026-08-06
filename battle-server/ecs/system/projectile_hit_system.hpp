#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;

    /// @brief 检测投射物与敌对实体的碰撞，并生成伤害事件。
    void projectile_hit_system(World& world, DeltaTime);
}
