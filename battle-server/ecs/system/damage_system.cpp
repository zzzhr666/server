#include "damage_system.hpp"

#include <algorithm>

#include "ecs/world.hpp"

void battle::ecs::damage_system(World& world, DeltaTime) {
    for (const auto& event : world.damage_events()) {
        auto* health = world.health().try_get(event.target);
        if (!health) {
            continue;
        }
        const int final_damage = std::max(event.base_damage, 0);
        health->current_health -= std::min(final_damage, health->current_health);
    }
    world.damage_events().clear();
}
