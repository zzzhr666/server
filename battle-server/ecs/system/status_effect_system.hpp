#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 推进燃烧、冰冻等持续状态，并生成相应的伤害或控制效果。
    void status_effect_system(World& world,DeltaTime delta_time);
}
