#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 将原始移动输入归一化为移动意图。
    void move_resolve_system(World& world, DeltaTime delta_time);
}
