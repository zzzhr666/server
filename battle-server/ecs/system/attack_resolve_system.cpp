#include "attack_resolve_system.hpp"

#include <algorithm>
#include "ecs/world.hpp"

void battle::ecs::attack_resolve_system(World& world, DeltaTime delta_time) {
    for (auto entity : world.attack_requests().entities()) {
        auto request = world.attack_requests().try_get(entity);
        const auto attack = world.attack_definitions().try_get(entity);
        auto cooldown = world.attack_cooldowns().try_get(entity);
        auto intent = world.attack_intents().try_get(entity);

        if (!request || !attack || !cooldown || !intent) {
            continue;
        }

        intent->active = false;
        intent->kind = attack->kind;
        intent->damage = 0;
        intent->range = 0.0f;
        intent->projectile_speed = 0.0f;

        cooldown->remaining_seconds -= delta_time;
        if (cooldown->remaining_seconds < DeltaTime{0}) {
            cooldown->remaining_seconds = DeltaTime{0};
        }
        if (!request->requested) {
            continue;
        }
        request->requested = false;
        if (cooldown->remaining_seconds > DeltaTime{0}) {
            continue;
        }
        intent->active = true;
        intent->damage = attack->damage;
        intent->range = attack->range;
        intent->projectile_speed = attack->projectile_speed;
        cooldown->remaining_seconds = attack->cooldown_seconds;
    }
}
