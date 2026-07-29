#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    struct StatusEffects;
    struct DamageAppliedEvent;
    class World;

    void blessing_trigger_system(World& world, DeltaTime);

}
