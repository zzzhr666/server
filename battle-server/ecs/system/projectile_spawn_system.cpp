#include "projectile_spawn_system.hpp"

#include "ecs/world.hpp"
#include "ecs/entity/entity.hpp"

void battle::ecs::projectile_spawn_system(World& world, DeltaTime delta_seconds) {
    for (Entity entity : world.registry().pool<AttackIntent>().entities()) {
        auto intent = world.registry().try_get<AttackIntent>(entity);
        auto transform = world.registry().try_get<Transform>(entity);
        if (!transform || !intent) {
            continue;
        }
        if (!intent->active || intent->kind != AttackKind::Projectile) {
            continue;
        }

        world.create_projectile(CreateProjectileConfig{
            .position = transform->position,
            .direction = transform->direction,
            .speed = intent->projectile_speed,
            .damage = intent->damage,
            .max_distance = intent->range,
            .hit_radius = intent->projectile_hit_radius,
            .context = intent->context
        });
    }
}
