#pragma once
#include "ecs/time.hpp"


namespace battle::ecs {
    class World;
    /// @brief 将冲刺请求在冷却允许时转换为冲刺意图。
    void dash_resolve_system(World& world,DeltaTime delta_time);
}
