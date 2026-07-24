#include "input_system.hpp"
#include "ecs/world.hpp"
#include <cmath>

void battle::ecs::input_system(World& world, DeltaTime) {
    for (auto entity : world.move_inputs().entities()) {
        auto* velocity = world.velocities().try_get(entity);
        auto* stats = world.character_stats().try_get(entity);
        const auto* input = world.move_inputs().try_get(entity);
        if (!velocity || !stats || !input) {
            continue;
        }
        if (input->x == 0 && input->y == 0) {
            velocity->x = 0;
            velocity->y = 0;
            continue;
        }
        float len = std::sqrt(input->x * input->x + input->y * input->y);
        float dir_x = input->x / len;
        float dir_y = input->y / len;
        velocity->x = dir_x * stats->move_speed;
        velocity->y = dir_y * stats->move_speed;
    }
}
