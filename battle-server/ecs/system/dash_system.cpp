#include "dash_system.hpp"

#include "ecs/world.hpp"

void battle::ecs::dash_system(World& world, DeltaTime deltaTime) {
    for (auto entity : world.registry().pool<DashIntent>().entities()) {
        const auto* intent = world.registry().try_get<DashIntent>(entity);
        auto* velocity = world.registry().try_get<Velocity>(entity);
        if (!intent || !intent->active || !velocity) {
            continue;
        }

        if (velocity->x == 0.0f && velocity->y == 0.0f) {
            const auto* transform = world.registry().try_get<Transform>(entity);
            const auto* stats = world.registry().try_get<CharacterStats>(entity);
            if (!transform || !stats) {
                continue;
            }
            velocity->x = transform->direction.x * stats->move_speed;
            velocity->y = transform->direction.y * stats->move_speed;
        }

        velocity->x *= intent->dash_speed_multiplier;
        velocity->y *= intent->dash_speed_multiplier;
    }
}
