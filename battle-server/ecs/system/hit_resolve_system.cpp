#include "hit_resolve_system.hpp"

#include "ecs/world.hpp"


namespace {
    bool is_monster(const battle::ecs::World& world, battle::ecs::Entity entity) {
        return world.registry().has<battle::ecs::MonsterController>(entity);
    }

    bool is_player(const battle::ecs::World& world, battle::ecs::Entity entity) {
        return world.registry().has<battle::ecs::PlayerController>(entity);
    }

    bool is_enemy(const battle::ecs::World& world, battle::ecs::Entity attacker, battle::ecs::Entity target) {
        return (is_player(world, attacker) && is_monster(world, target)) || (is_player(world, target) && is_monster(
            world, attacker));
    }
}

void battle::ecs::hit_resolve_system(World& world, DeltaTime) {
    for (auto attacker_entity : world.registry().pool<AttackIntent>().entities()) {
        auto intent = world.registry().try_get<AttackIntent>(attacker_entity);
        auto transform = world.registry().try_get<Transform>(attacker_entity);
        if (!intent || !transform) {
            continue;
        }

        if (!intent->active) {
            continue;
        }
        if (intent->kind != battle::ecs::AttackKind::Melee) {
            continue;
        }
        for (auto target_entity : world.registry().pool<Health>().entities()) {
            if (attacker_entity == target_entity) {
                continue;
            }
            if (!is_enemy(world, attacker_entity, target_entity)) {
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
            if (distance > intent->range * intent->range) {
                continue;
            }
            world.add_damage_event(DamageEvent{
                .source = attacker_entity,
                .target = target_entity,
                .base_damage = intent->damage,
                .modified_damage = intent->damage,
                .source_kind = DamageSourceKind::Attack,
                .context = intent->context
            });
        }
    }
}
