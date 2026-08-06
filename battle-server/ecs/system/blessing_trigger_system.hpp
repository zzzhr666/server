#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    struct StatusEffects;
    struct DamageAppliedEvent;
    class World;

    /// @brief 消费已落地伤害，触发吸血、燃烧和冰冻等后置祝福效果。
    void blessing_trigger_system(World& world, DeltaTime);

}
