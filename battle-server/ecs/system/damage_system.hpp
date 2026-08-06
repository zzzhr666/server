#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 应用伤害事件、生成伤害已应用事件，并为死亡系统准备致死状态。
    void damage_system(World& world, DeltaTime delta_time);
}
