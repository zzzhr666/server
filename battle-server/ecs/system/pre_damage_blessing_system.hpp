#pragma once


#include "ecs/time.hpp"
#include "ecs/entity/entity.hpp"

namespace battle::ecs {
    class World;
    /// @brief 在原始伤害应用前生成闪电链等派生伤害事件。
    void pre_damage_blessing_system(World& world,DeltaTime delta_seconds);
}
