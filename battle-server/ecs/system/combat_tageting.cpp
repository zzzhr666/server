#include "combat_tageting.hpp"

#include "ecs/world.hpp"
#include "ecs/component/components.hpp"

bool battle::ecs::is_monster(const World& world, Entity entity) {
    return world.registry().has<MonsterController>(entity);
}

bool battle::ecs::is_player(const World& world, Entity entity) {
    return world.registry().has<battle::ecs::PlayerController>(entity);
}

bool battle::ecs::is_enemy(const World& world, Entity attacker, Entity target) {
    return (is_player(world, attacker) && is_monster(world, target)) ||
        (is_player(world, target) && is_monster(world, attacker));
}
