#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;

    constexpr float ProjectileHitRadius = 0.5f;

    void projectile_hit_system(World& world, DeltaTime);
}
