#include "hit_resolve_system.hpp"

#include <algorithm>

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
        const auto attacker_collider = world.registry().try_get<Collider>(attacker_entity);
        if (!attacker_collider) {
            continue;
        }
        for (auto target_entity : world.spatial_index().query_circle(transform->position, attack->range)) {
            if (!world.registry().valid(target_entity) || attacker_entity == target_entity) {
                continue;
            }
            if (std::ranges::find(state->hit_targets, target_entity) != state->hit_targets.end()) {
                continue;
            }
            auto target_transform = world.registry().try_get<Transform>(target_entity);
            auto target_health = world.registry().try_get<Health>(target_entity);
            auto target_collider = world.registry().try_get<Collider>(target_entity);
            if (!target_transform || !target_health || !target_collider ||
                !are_opposing_characters(*attacker_collider, *target_collider)) {
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
