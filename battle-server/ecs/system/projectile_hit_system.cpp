#include "projectile_hit_system.hpp"

#include <vector>

#include "blessing_config.hpp"
#include "blessing_helpers.hpp"
#include "ecs/world.hpp"
#include "ecs/entity/entity.hpp"

namespace {
    void apply_armor_break(battle::ecs::World& world, battle::ecs::Entity source, battle::ecs::Entity target) {
        const auto* blessing = battle::ecs::find_blessing(world, source, battle::BlessingID::ArmorBreak);
        auto* effects = world.registry().try_get<battle::ecs::StatusEffects>(target);
        if (!blessing || !effects) {
            return;
        }
        const int armor_break_num = battle::ecs::armor_break_armors(blessing->level);
        if (effects->armor_break.has_value()) {
            auto& status = effects->armor_break.value();
            if (status.armor_decreased) {
                if (auto* stats = world.registry().try_get<battle::ecs::CharacterStats>(target)) {
                    stats->armor += status.armor_break_number - armor_break_num;
                }
            }
            status.armor_break_number = armor_break_num;
            status.remaining_seconds = battle::gameplay_config::blessing::armor_break::Duration;
            return;
        }
        effects->armor_break = battle::ecs::ArmorBreakStatus{
            .remaining_seconds = battle::gameplay_config::blessing::armor_break::Duration,
            .armor_break_number = armor_break_num,
            .armor_decreased = false,
        };
    }
}

void battle::ecs::projectile_hit_system(World& world, DeltaTime) {
    std::vector<Entity> entities_to_erase;
    for (Entity entity : world.registry().pool<Projectile>().entities()) {
        auto transform = world.registry().try_get<Transform>(entity);
        auto projectile = world.registry().try_get<Projectile>(entity);
        auto collider = world.registry().try_get<Collider>(entity);
        if (!transform || !projectile || !collider) {
            continue;
        }
        for (Entity target : world.spatial_index().query_circle(transform->position, collider->radius)) {
            auto target_transform = world.registry().try_get<Transform>(target);
            auto target_collider = world.registry().try_get<Collider>(target);
            if (!target_transform || !target_collider) {
                continue;
            }
            if ((collider->collision_mask & target_collider->category) == 0 ||
                (target_collider->collision_mask & collider->category) == 0) {
                continue;
            }
            const float distance_squared = battle::ecs::distance_squared(
                transform->position, target_transform->position);
            const float radius_sum = collider->radius + target_collider->radius;
            if (distance_squared <= radius_sum * radius_sum) {
                if (target_collider->category != CollisionCategory::Obstacle) {
                    apply_armor_break(world, projectile->context.owner, target);
                    world.add_damage_event(DamageEvent{
                        .source = projectile->context.owner,
                        .target = target,
                        .base_damage = projectile->damage,
                        .modified_damage = projectile->damage,
                        .source_kind = DamageSourceKind::Attack,
                        .context = projectile->context
                    });
                }
                entities_to_erase.emplace_back(entity);
                break;
            }
        }
    }
    for (const Entity& entity : entities_to_erase) {
        world.destroy_entity(entity);
    }
}
