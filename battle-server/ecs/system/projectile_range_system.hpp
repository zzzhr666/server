#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;

    void projectile_range_system(World& world,DeltaTime delta_seconds);
}
