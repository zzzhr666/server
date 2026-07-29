#include "damage_system.hpp"

#include <algorithm>

#include "ecs/world.hpp"

void battle::ecs::damage_system(World& world, DeltaTime) {
    for (const auto& event : world.damage_events()) {
        auto* health = world.health().try_get(event.target);
        if (!health) {
            continue;
        }
        const int final_damage = std::clamp(event.modified_damage, 0, health->current_health);
        const int before_health = health->current_health;
        health->current_health = std::clamp(health->current_health - final_damage, 0, health->max_health);
        if (final_damage > 0) {
            world.add_damage_applied_event({
                .source = event.source,
                .target = event.target,
                .amount = final_damage,
                .source_kind = event.source_kind,
            });
        }
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
    world.clear_damage_events();
}
