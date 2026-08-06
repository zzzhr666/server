#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 销毁生命值耗尽的实体，并生成死亡和击杀事件。
    void death_system(World& world, DeltaTime delta_time);
}
