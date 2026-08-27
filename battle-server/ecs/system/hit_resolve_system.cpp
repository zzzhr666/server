#include "hit_resolve_system.hpp"

#include <algorithm>

#include "blessing_config.hpp"
#include "blessing_helpers.hpp"
#include "ecs/world.hpp"


namespace {
    void handle_armor_break_status(battle::ecs::World& world, battle::ecs::Entity attacker_entity,
                                   battle::ecs::Entity target_entity) {
        const auto* blessing = battle::ecs::find_blessing(world, attacker_entity, battle::BlessingID::ArmorBreak);
        auto* effect = world.registry().try_get<battle::ecs::StatusEffects>(target_entity);
        if (blessing && effect) {
            const int armor_break_num = battle::ecs::armor_break_armors(blessing->level);
            if (effect->armor_break.has_value()) {
                auto& status = effect->armor_break.value();
                if (status.armor_decreased) {
                    if (auto* stats = world.registry().try_get<battle::ecs::CharacterStats>(target_entity)) {
                        stats->armor += status.armor_break_number - armor_break_num;
                    }
                }
                status.armor_break_number = armor_break_num;
                status.remaining_seconds = battle::gameplay_config::blessing::armor_break::Duration;
            } else {
                effect->armor_break = std::make_optional(battle::ecs::ArmorBreakStatus{
                    .remaining_seconds = battle::gameplay_config::blessing::armor_break::Duration,
                    .armor_break_number = armor_break_num,
                    .armor_decreased = false,
                });
            }
        }
    }
}

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
        const float attack_query_radius = attacker_collider->radius + attack->range;
        for (auto target_entity : world.spatial_index().query_circle(transform->position, attack_query_radius)) {
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
            const float distance_squared = battle::ecs::distance_squared(
                transform->position, target_transform->position);
            const float hit_distance = attacker_collider->radius + attack->range + target_collider->radius;
            if (distance_squared > hit_distance * hit_distance) {
                continue;
            }
            const float to_target_x = target_transform->position.x - transform->position.x;
            const float to_target_y = target_transform->position.y - transform->position.y;
            const float facing_dot_target = state->locked_direction.x * to_target_x + state->locked_direction.y *
                to_target_y;
            if (facing_dot_target < 0.0f) {
                continue;
            }
            handle_armor_break_status(world, attacker_entity, target_entity);
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
