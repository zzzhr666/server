#include "move_resolve_system.hpp"

#include <cmath>

#include "ecs/world.hpp"

void battle::ecs::move_resolve_system(World& world, DeltaTime) {
    for (auto entity : world.move_requests().entities()) {
        auto* velocity = world.velocities().try_get(entity);
        auto* stats = world.character_stats().try_get(entity);
        auto* intent = world.move_intents().try_get(entity);
        const auto* request = world.move_requests().try_get(entity);
        if (!velocity || !stats || !intent || !request) {
            continue;
        }
        if (request->x == 0.0f && request->y == 0.0f) {
            intent->x = 0.0f;
            intent->y = 0.0f;
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            continue;
        }
        const float length = std::sqrt(request->x * request->x + request->y * request->y);
        const float direction_x = request->x / length;
        const float direction_y = request->y / length;
        intent->x = direction_x;
        intent->y = direction_y;
        velocity->x = direction_x * stats->move_speed;
        velocity->y = direction_y * stats->move_speed;
    }
}
