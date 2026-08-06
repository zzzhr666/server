#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 将攻击请求和冷却转换为近战攻击意图或投射物生成事件。
    void attack_resolve_system(World& world ,DeltaTime delta_time);
}
