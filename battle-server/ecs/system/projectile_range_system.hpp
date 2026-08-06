#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;

    /// @brief 移除已超过最大飞行距离的投射物。
    void projectile_range_system(World& world,DeltaTime delta_seconds);
}
