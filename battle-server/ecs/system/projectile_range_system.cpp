#include "projectile_range_system.hpp"

#include <cmath>
#include <vector>

#include "ecs/world.hpp"
#include "ecs/entity/entity.hpp"

void battle::ecs::projectile_range_system(World& world, DeltaTime delta_seconds) {
    std::vector<Entity> entities_to_erase;
    for (Entity entity : world.registry().pool<Projectile>().entities()) {
        auto velocity = world.registry().try_get<Velocity>(entity);
        auto projectile = world.registry().try_get<Projectile>(entity);
        if (!projectile || !velocity) {
            continue;
        }
        float delta_distance = std::sqrt(velocity->x * velocity->x + velocity->y * velocity->y) * delta_seconds.count();
        projectile->current_distance += delta_distance;
        if (projectile->current_distance >= projectile->max_distance) {
            entities_to_erase.emplace_back(entity);
        }
    }
    for (Entity entity : entities_to_erase) {
        world.destroy_entity(entity);
    }
}
