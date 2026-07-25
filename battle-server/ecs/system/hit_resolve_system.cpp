#include "hit_resolve_system.hpp"

#include "ecs/world.hpp"


namespace {
    bool is_monster(const battle::ecs::World& world, battle::ecs::Entity entity) {
        return world.monster_controllers().has(entity);
    }

    bool is_player(const battle::ecs::World& world, battle::ecs::Entity entity) {
        return world.player_controllers().has(entity);
    }

    bool is_enemy(const battle::ecs::World& world, battle::ecs::Entity attacker, battle::ecs::Entity target) {
        return (is_player(world, attacker) && is_monster(world, target)) || (is_player(world, target) && is_monster(
            world, attacker));
    }
}

void battle::ecs::hit_resolve_system(World& world, DeltaTime) {
    for (auto attacker_entity : world.attack_intents().entities()) {
        auto intent = world.attack_intents().try_get(attacker_entity);
        auto transform = world.transforms().try_get(attacker_entity);
        if (!intent || !transform) {
            continue;
        }

        if (!intent->active) {
            continue;
        }
        for (auto target_entity : world.health().entities()) {
            if (attacker_entity == target_entity) {
                continue;
            }
            if (!is_enemy(world, attacker_entity, target_entity)) {
                continue;
            }
            auto target_transform = world.transforms().try_get(target_entity);
            auto target_health = world.health().try_get(target_entity);
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
            });
        }
    }
}
