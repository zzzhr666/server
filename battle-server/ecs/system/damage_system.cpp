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
        const int before_health = health->current_health;
        health->current_health = std::clamp(health->current_health - final_damage, 0, health->max_health);
        if (before_health > 0 && health->current_health == 0 && world.monster_controllers().has(event.target)) {
            if (auto identity = world.monster_identities().try_get(event.target)) {
                world.add_kill_event({
                    .killer = event.source,
                    .victim = event.target,
                    .monster_kind = identity->kind,
                });
            }
        }
    }
    world.damage_events().clear();
}
