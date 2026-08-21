#include "move_system.hpp"

#include <algorithm>
#include <cmath>
#include <vector>

#include "ecs/world.hpp"

namespace {
    bool is_character(const battle::ecs::Collider& collider) {
        return collider.category == battle::ecs::CollisionCategory::Monster ||
            collider.category == battle::ecs::CollisionCategory::Player;
    }

    bool is_interactable(const battle::ecs::Collider& lhs, const battle::ecs::Collider& rhs) {
        return (lhs.category & rhs.collision_mask) != 0 &&
            (lhs.collision_mask & rhs.category) != 0;
    }

    bool is_overlap(const battle::ecs::Position& center_a, const battle::ecs::Position& center_b, float radius_a,
                    float radius_b) {
        const float dx = center_b.x - center_a.x;
        const float dy = center_b.y - center_a.y;
        const float radius_sum = radius_a + radius_b;
        return dx * dx + dy * dy < radius_sum * radius_sum;
    }

    float clamp_character_axis(float value, float min, float max, float radius) {
        const float lower = min + radius;
        const float upper = max - radius;
        if (lower > upper) {
            return (min + max) * 0.5f;
        }
        return std::clamp(value, lower, upper);
    }
}

void battle::ecs::move_system(World& world, DeltaTime delta_time) {
    const auto& bounds = world.world_bounds();
    const float delta_seconds = delta_time.count();
    std::vector<Entity> entities{world.registry().pool<Velocity>().entities()};
    std::ranges::sort(entities);
    for (auto entity : entities) {
        auto velocity = world.registry().try_get<Velocity>(entity);
        auto transform = world.registry().try_get<Transform>(entity);
        if (!velocity || !transform) {
            continue;
        }
        const auto* collider = world.registry().try_get<Collider>(entity);
        auto new_position = Position{
            .x = transform->position.x + velocity->x * delta_seconds,
            .y = transform->position.y + velocity->y * delta_seconds,
        };
        if (collider && is_character(*collider)) {
            new_position.x = clamp_character_axis(new_position.x, bounds.min_x, bounds.max_x, collider->radius);
            new_position.y = clamp_character_axis(new_position.y, bounds.min_y, bounds.max_y, collider->radius);
        } else {
            new_position.x = std::clamp(new_position.x, bounds.min_x, bounds.max_x);
            new_position.y = std::clamp(new_position.y, bounds.min_y, bounds.max_y);
        }

        bool can_move = true;
        if (collider && is_character(*collider)) {
            const auto candidates = world.spatial_index().query_circle(new_position, collider->radius);
            for (const auto candidate_entity : candidates) {
                if (candidate_entity == entity || !world.registry().valid(candidate_entity)) {
                    continue;
                }
                const auto* candidate_transform = world.registry().try_get<Transform>(candidate_entity);
                const auto* candidate_collider = world.registry().try_get<Collider>(candidate_entity);
                if (!candidate_collider || !candidate_transform) {
                    continue;
                }
                if (!is_interactable(*collider, *candidate_collider)) {
                    continue;
                }
                if (is_overlap(new_position, candidate_transform->position, collider->radius,
                               candidate_collider->radius)) {
                    can_move = false;
                    break;
                }
            }
        }

        if (can_move) {
            transform->position = new_position;
        }
        if (collider) {
            world.spatial_index().update(entity, transform->position, collider->radius);
        }
        float len = std::sqrt(velocity->x * velocity->x + velocity->y * velocity->y);
        if (len < 0.0001f) {
            continue;
        }
        float dir_x = velocity->x / len;
        float dir_y = velocity->y / len;
        if (auto state = world.registry().try_get<AttackState>(entity)) {
            if (state->phase != AttackPhase::Idle) {
                transform->direction = state->locked_direction;
                continue;
            }
        }
        transform->direction.x = dir_x;
        transform->direction.y = dir_y;
    }
}
