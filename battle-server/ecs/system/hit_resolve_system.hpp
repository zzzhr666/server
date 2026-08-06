#pragma once

#include <ecs/time.hpp>

namespace battle::ecs {
    class World;
    /// @brief 将近战攻击意图解析为目标命中和伤害事件。
    void hit_resolve_system(World& world, DeltaTime delta_time);
}
