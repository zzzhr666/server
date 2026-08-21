#include "projectile_hit_system.hpp"

#include <vector>

#include "ecs/world.hpp"
#include "ecs/entity/entity.hpp"

void battle::ecs::projectile_hit_system(World& world, DeltaTime) {
    std::vector<Entity> entities_to_erase;
    for (Entity entity : world.registry().pool<Projectile>().entities()) {
        auto transform = world.registry().try_get<Transform>(entity);
        auto projectile = world.registry().try_get<Projectile>(entity);
        auto collider = world.registry().try_get<Collider>(entity);
        if (!transform || !projectile || !collider) {
            continue;
        }
        for (Entity target : world.registry().pool<Health>().entities()) {
            auto target_transform = world.registry().try_get<Transform>(target);
            auto target_collider = world.registry().try_get<Collider>(target);
            if (!target_transform || !target_collider) {
                continue;
            }
            if ((collider->collision_mask & target_collider->category) == 0 ||
                (target_collider->collision_mask & collider->category) == 0) {
                continue;
            }
            const float delta_x = transform->position.x - target_transform->position.x;
            const float delta_y = transform->position.y - target_transform->position.y;
            const float distance_squared = delta_x * delta_x + delta_y * delta_y;
            const float radius_sum = collider->radius + target_collider->radius;
            if (distance_squared <= radius_sum * radius_sum) {
                world.add_damage_event(DamageEvent{
                    .source = projectile->context.owner,
                    .target = target,
                    .base_damage = projectile->damage,
                    .modified_damage = projectile->damage,
                    .source_kind = DamageSourceKind::Attack,
                    .context = projectile->context
                });
                entities_to_erase.emplace_back(entity);
                break;
            }
        }
    }
    for (const Entity& entity : entities_to_erase) {
        world.destroy_entity(entity);
    }
}
