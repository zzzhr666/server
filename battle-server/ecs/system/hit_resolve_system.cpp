#include "hit_resolve_system.hpp"

#include <algorithm>

#include "combat_targeting.hpp"
#include "ecs/world.hpp"


void battle::ecs::hit_resolve_system(World& world, DeltaTime) {
    for (auto attacker_entity : world.registry().pool<AttackState>().entities()) {
        auto transform = world.registry().try_get<Transform>(attacker_entity);
        auto attack = world.registry().try_get<AttackDefinition>(attacker_entity);
        auto state = world.registry().try_get<AttackState>(attacker_entity);
        if (!transform || !state || !attack) {
            continue;
        }

        if (state->phase != AttackPhase::Active || attack->kind != AttackKind::Melee) {
            continue;
        }
        for (auto target_entity : world.registry().pool<Health>().entities()) {
            if (attacker_entity == target_entity) {
                continue;
            }
            if (!is_enemy(world, attacker_entity, target_entity)) {
                continue;
            }
            if (std::find(state->hit_targets.begin(), state->hit_targets.end(), target_entity) !=
                state->hit_targets.end()) {
                continue;
            }
            auto target_transform = world.registry().try_get<Transform>(target_entity);
            auto target_health = world.registry().try_get<Health>(target_entity);
            if (!target_transform || !target_health) {
                continue;
            }
            float delta_x = transform->position.x - target_transform->position.x;
            float delta_y = transform->position.y - target_transform->position.y;
            float distance = delta_x * delta_x + delta_y * delta_y;
            if (distance > attack->range * attack->range) {
                continue;
            }
            const float to_target_x = target_transform->position.x - transform->position.x;
            const float to_target_y = target_transform->position.y - transform->position.y;
            const float facing_dot_target = state->locked_direction.x * to_target_x + state->locked_direction.y *
                to_target_y;
            if (facing_dot_target < 0.0f) {
                continue;
            }
            world.add_damage_event(DamageEvent{
                .source = attacker_entity,
                .target = target_entity,
                .base_damage = attack->damage,
                .modified_damage = attack->damage,
                .source_kind = DamageSourceKind::Attack,
                .context = state->context
            });
            state->hit_targets.emplace_back(target_entity);
        }
    }
}
