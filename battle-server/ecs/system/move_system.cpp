#include "move_system.hpp"
#include "ecs/world.hpp"
#include <cmath>

void battle::ecs::move_system(World& world, DeltaTime delta_time) {
    const float delta_seconds = delta_time.count();
    for (auto entity : world.velocities().entities()) {
        auto velocity = world.velocities().try_get(entity);
        auto transform = world.transforms().try_get(entity);
        if (!velocity || !transform) {
            continue;
        }
        transform->position.x += velocity->x * delta_seconds;
        transform->position.y += velocity->y * delta_seconds;
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
