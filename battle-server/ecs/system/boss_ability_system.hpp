#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void boss_ability_system(World& world, DeltaTime delta_time);
}
