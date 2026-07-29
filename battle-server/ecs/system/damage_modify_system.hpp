#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void damage_modify_system(World& world, DeltaTime);
}
