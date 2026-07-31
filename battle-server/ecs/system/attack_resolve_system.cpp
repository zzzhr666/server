#include "attack_resolve_system.hpp"

#include <algorithm>
#include "ecs/world.hpp"

void battle::ecs::attack_resolve_system(World& world, DeltaTime delta_time) {
    for (auto entity : world.registry().pool<AttackRequest>().entities()) {
        auto request = world.registry().try_get<AttackRequest>(entity);
        const auto attack = world.registry().try_get<AttackDefinition>(entity);
        auto cooldown = world.registry().try_get<AttackCooldown>(entity);
        auto intent = world.registry().try_get<AttackIntent>(entity);

        if (!request || !attack || !cooldown || !intent) {
            continue;
        }

        intent->active = false;
        intent->kind = attack->kind;
        intent->damage = 0;
        intent->range = 0.0f;
        intent->projectile_speed = 0.0f;
        intent->context = {};

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
        intent->context = CombatContext {
            .owner = entity,
            .emitter = entity,
            .action_state = world.crete_combat_action(),
            .effect_id = world.create_combat_effect(),
        };
        cooldown->remaining_seconds = attack->cooldown_seconds;
    }
}
