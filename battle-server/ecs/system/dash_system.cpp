#include "dash_system.hpp"

#include "ecs/world.hpp"

void battle::ecs::dash_system(World& world, DeltaTime deltaTime) {
    for (auto entity : world.dash_intents().entities()) {
        auto intent = world.dash_intents().try_get(entity);
        auto velocity = world.velocities().try_get(entity);
        if (!intent || !intent->active || !velocity) {
            continue;
        }
        velocity->x *= intent->dash_speed_multiplier;
        velocity->y *= intent->dash_speed_multiplier;
    }
}
