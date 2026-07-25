#include "death_system.hpp"

#include <vector>

#include "ecs/world.hpp"

void battle::ecs::death_system(World& world, DeltaTime) {
    std::vector<Entity> dead_entities;
    for (auto entity : world.health().entities()) {
        const auto* health = world.health().try_get(entity);
        if (health && health->current_health <= 0) {
            dead_entities.push_back(entity);
        }
    }

    for (auto entity : dead_entities) {
        world.destroy_entity(entity);
    }
}
