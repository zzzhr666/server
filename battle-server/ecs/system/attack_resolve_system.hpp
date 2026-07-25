#pragma once

#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void attack_resolve_system(World& world ,DeltaTime delta_time);
}
