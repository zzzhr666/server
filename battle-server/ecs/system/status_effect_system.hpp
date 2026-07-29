#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void status_effect_system(World& world,DeltaTime delta_time);
}
