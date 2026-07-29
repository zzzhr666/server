#include "monster_ai_system.hpp"

#include <cmath>
#include <limits>

#include "ecs/world.hpp"

void battle::ecs::monster_ai_system(World& world, DeltaTime) {
    for (auto entity : world.monster_controllers().entities()) {
        const auto* transform = world.transforms().try_get(entity);
        auto* velocity = world.velocities().try_get(entity);
        const auto* stats = world.character_stats().try_get(entity);
        const auto* attack = world.attack_definitions().try_get(entity);
        auto* attack_request = world.attack_requests().try_get(entity);
        if (!transform || !velocity || !stats) {
            continue;
        }
        if (attack_request) {
            attack_request->requested = false;
        }
        const auto status_effects = world.status_effects().try_get(entity);
        if (status_effects && status_effects->freeze.has_value()) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            continue;
        }

        const auto x = transform->position.x;
        const auto y = transform->position.y;
        auto nearest_distance_squared = std::numeric_limits<float>::max();
        Position target_position{};
        bool has_target = false;
        for (auto players_entity : world.player_controllers().entities()) {
            const auto* player_transform = world.transforms().try_get(players_entity);
            if (!player_transform) {
                continue;
            }
            const float delta_x = player_transform->position.x - x;
            const float delta_y = player_transform->position.y - y;
            const float distance_squared = delta_x * delta_x + delta_y * delta_y;
            if (nearest_distance_squared > distance_squared) {
                nearest_distance_squared = distance_squared;
                target_position = player_transform->position;
                has_target = true;
            }
        }

        if (!has_target) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            continue;
        }

        const auto delta_x = target_position.x - x;
        const auto delta_y = target_position.y - y;
        const auto distance = std::sqrt(delta_x * delta_x + delta_y * delta_y);
        const float attack_range = attack ? attack->range : 0.0f;
        if (attack_request && distance <= attack_range) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            attack_request->requested = true;
            continue;
        }
        if (distance == 0.0f) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            continue;
        }

        velocity->x = delta_x / distance * stats->move_speed;
        velocity->y = delta_y / distance * stats->move_speed;
    }
}
