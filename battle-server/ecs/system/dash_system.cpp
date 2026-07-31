#include "dash_system.hpp"

#include "ecs/world.hpp"

void battle::ecs::dash_system(World& world, DeltaTime deltaTime) {
    for (auto entity : world.registry().pool<DashIntent>().entities()) {
        auto intent = world.registry().try_get<DashIntent>(entity);
        auto velocity = world.registry().try_get<Velocity>(entity);
        if (!intent || !intent->active || !velocity) {
            continue;
        }
        velocity->x *= intent->dash_speed_multiplier;
        velocity->y *= intent->dash_speed_multiplier;
    }
}
