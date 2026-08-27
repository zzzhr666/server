#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {

    class World;

    /// @brief 检测陷阱触发范围并生成伤害或控制事件。
    void trap_system(World& world, DeltaTime delta_time);
}
