#include "projectile_spawn_system.hpp"

#include "ecs/world.hpp"
#include "ecs/entity/entity.hpp"

void battle::ecs::projectile_spawn_system(World& world, DeltaTime) {
    for (Entity entity : world.registry().pool<AttackState>().entities()) {
        auto transform = world.registry().try_get<Transform>(entity);
        const auto attack = world.registry().try_get<AttackDefinition>(entity);
        const auto state = world.registry().try_get<AttackState>(entity);
        if (!transform || !attack || !state) {
            continue;
        }
        if (state->phase != AttackPhase::Active || attack->kind != AttackKind::Projectile ||
            state->projectile_spawned) {
            continue;
        }

        world.create_projectile(CreateProjectileConfig{
            .position = transform->position,
            .direction = state->locked_direction,
            .speed = attack->projectile_speed,
            .damage = attack->damage,
            .max_distance = attack->range,
            .hit_radius = attack->projectile_hit_radius,
            .context = state->context
        });
        state->projectile_spawned = true;
    }
}
