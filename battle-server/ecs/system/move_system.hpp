#pragma once

#include "ecs/time.hpp"


namespace battle::ecs {
    class World;
    /// @brief 根据移动意图、速度和世界边界更新实体位置。
    void move_system(World& world, DeltaTime delta_time);
}
