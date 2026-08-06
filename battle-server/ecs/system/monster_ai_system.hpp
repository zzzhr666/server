#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 为怪物生成追击、拉开距离和攻击请求。
    void monster_ai_system(World& world, DeltaTime);
}
