#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void projectile_spawn_system(World& world, DeltaTime delta_seconds);
}
