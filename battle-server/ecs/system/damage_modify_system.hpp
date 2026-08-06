#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 在伤害落地前处理暴击等数值修正。
    void damage_modify_system(World& world, DeltaTime);
}
