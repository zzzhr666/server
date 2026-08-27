#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    /// @brief 推进 Boss 技能冷却并生成对应攻击或区域效果。
    void boss_ability_system(World& world, DeltaTime delta_time);
}
