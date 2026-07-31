#pragma once
#include "ecs/entity/entity.hpp"

namespace battle::ecs {
    class World;
    bool is_monster(const World& world, Entity entity);

    bool is_player(const World& world, Entity entity);
    bool is_enemy(const World& world, Entity attacker, Entity target);
}
