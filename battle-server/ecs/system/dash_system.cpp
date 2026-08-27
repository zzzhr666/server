#include "dash_system.hpp"

#include <cmath>

#include "ecs/world.hpp"

void battle::ecs::dash_system(World& world, DeltaTime deltaTime) {
    for (auto entity : world.registry().pool<DashIntent>().entities()) {
        const auto* intent = world.registry().try_get<DashIntent>(entity);
        const auto* stats = world.registry().try_get<CharacterStats>(entity);
        auto* velocity = world.registry().try_get<Velocity>(entity);
        if (!intent || !intent->active || !velocity || !stats) {
            continue;
        }

        float direction_x = velocity->x;
        float direction_y = velocity->y;
        if (direction_x == 0.0f && direction_y == 0.0f) {
            const auto* transform = world.registry().try_get<Transform>(entity);
            if (!transform) {
                continue;
            }
            direction_x = transform->direction.x;
            direction_y = transform->direction.y;
        }

        const float length = std::sqrt(direction_x * direction_x + direction_y * direction_y);
        if (length == 0.0f) {
            continue;
        }
        const float dir_x = direction_x / length;
        const float dir_y = direction_y / length;

        velocity->x = dir_x * stats->move_speed * intent->dash_speed_multiplier;
        velocity->y = dir_y * stats->move_speed * intent->dash_speed_multiplier;
    }
}
