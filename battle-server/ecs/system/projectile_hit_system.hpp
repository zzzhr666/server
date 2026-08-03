#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;

    void projectile_hit_system(World& world, DeltaTime);
}
