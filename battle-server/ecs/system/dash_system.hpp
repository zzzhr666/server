#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 把冲刺意图施加到本 tick 的移动速度。
    void dash_system(World& world,DeltaTime deltaTime);
}
