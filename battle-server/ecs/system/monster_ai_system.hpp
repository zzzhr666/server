#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void monster_ai_system(World& world, DeltaTime);
}
