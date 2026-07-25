#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void damage_system(World& world, DeltaTime delta_time);
}
