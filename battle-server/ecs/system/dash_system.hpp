#pragma once
#include "ecs/time.hpp"

namespace battle::ecs {
    class World;
    void dash_system(World& world,DeltaTime deltaTime);
}
