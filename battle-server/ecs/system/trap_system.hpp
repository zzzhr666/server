#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {

    class World;

    void trap_system(World& world, DeltaTime delta_time);
}
