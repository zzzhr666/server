#include "move_system.hpp"

#include <algorithm>
#include <cmath>

#include "ecs/world.hpp"

void battle::ecs::move_system(World& world, DeltaTime delta_time) {
    const auto& bounds = world.world_bounds();
    const float delta_seconds = delta_time.count();
    for (auto entity : world.registry().pool<Velocity>().entities()) {
        auto velocity = world.registry().try_get<Velocity>(entity);
        auto transform = world.registry().try_get<Transform>(entity);
        if (!velocity || !transform) {
            continue;
        }
        transform->position.x += velocity->x * delta_seconds;
        transform->position.y += velocity->y * delta_seconds;
        transform->position.x = std::clamp(transform->position.x, bounds.min_x, bounds.max_x);
        transform->position.y = std::clamp(transform->position.y, bounds.min_y, bounds.max_y);
        float len = std::sqrt(velocity->x * velocity->x + velocity->y * velocity->y);
        if (len < 0.0001f) {
            continue;
        }
        float dir_x = velocity->x / len;
        float dir_y = velocity->y / len;
        transform->direction.x = dir_x;
        transform->direction.y = dir_y;
    }
}
