#include "move_resolve_system.hpp"

#include <cmath>

#include "ecs/world.hpp"

void battle::ecs::move_resolve_system(World& world, DeltaTime) {
    for (auto entity : world.registry().pool<MoveRequest>().entities()) {
        auto* velocity = world.registry().try_get<Velocity>(entity);
        auto* stats = world.registry().try_get<CharacterStats>(entity);
        auto* intent = world.registry().try_get<MoveIntent>(entity);
        const auto* request = world.registry().try_get<MoveRequest>(entity);
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
