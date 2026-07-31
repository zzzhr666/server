#pragma once


#include "ecs/time.hpp"
#include "ecs/entity/entity.hpp"

namespace battle::ecs {
    class World;
    void pre_damage_blessing_system(World& world,DeltaTime delta_seconds);
}
