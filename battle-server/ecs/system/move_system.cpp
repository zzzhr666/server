#include "move_system.hpp"

#include <algorithm>
#include <array>
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

    float distance_squared(const battle::ecs::Position& lhs, const battle::ecs::Position& rhs) {
        const float dx = lhs.x - rhs.x;
        const float dy = lhs.y - rhs.y;
        return dx * dx + dy * dy;
    }

    bool is_boss_dash(const battle::ecs::World& world, battle::ecs::Entity entity) {
        const auto* identity = world.registry().try_get<battle::ecs::MonsterIdentity>(entity);
        const auto* ability = world.registry().try_get<battle::ecs::BossAbilityState>(entity);
        return identity && identity->kind == battle::MonsterKind::Boss && ability &&
            ability->kind == battle::ecs::BossAbilityKind::TripleDash &&
            ability->action_phase == battle::ecs::AttackPhase::Active;
    }

    float clamp_character_axis(float value, float min, float max, float radius) {
        const float lower = min + radius;
        const float upper = max - radius;
        if (lower > upper) {
            return (min + max) * 0.5f;
        }
        return std::clamp(value, lower, upper);
    }

    bool can_place_character(const battle::ecs::World& world, battle::ecs::Entity entity,
                             battle::ecs::Position position, const battle::ecs::Collider& collider,
                             battle::ecs::Entity ignored_entity) {
        const auto& bounds = world.world_bounds();
        if (position.x - collider.radius < bounds.min_x || position.x + collider.radius > bounds.max_x ||
            position.y - collider.radius < bounds.min_y || position.y + collider.radius > bounds.max_y) {
            return false;
        }
        for (const auto candidate : world.spatial_index().query_circle(position, collider.radius)) {
            if (candidate == entity || candidate == ignored_entity || !world.registry().valid(candidate)) {
                continue;
            }
            const auto* candidate_transform = world.registry().try_get<battle::ecs::Transform>(candidate);
            const auto* candidate_collider = world.registry().try_get<battle::ecs::Collider>(candidate);
            if (!candidate_transform || !candidate_collider || !is_interactable(collider, *candidate_collider)) {
                continue;
            }
            if (is_overlap(position, candidate_transform->position, collider.radius, candidate_collider->radius)) {
                return false;
            }
        }
        return true;
    }

    void separate_players_from_dashing_bosses(battle::ecs::World& world) {
        std::vector<battle::ecs::Entity> bosses{world.registry().pool<battle::ecs::BossAbilityState>().entities()};
        std::ranges::sort(bosses);
        for (const auto boss : bosses) {
            if (!is_boss_dash(world, boss)) {
                continue;
            }
            const auto* boss_transform = world.registry().try_get<battle::ecs::Transform>(boss);
            const auto* boss_collider = world.registry().try_get<battle::ecs::Collider>(boss);
            const auto* boss_velocity = world.registry().try_get<battle::ecs::Velocity>(boss);
            if (!boss_transform || !boss_collider || !boss_velocity) {
                continue;
            }
            float direction_x = boss_velocity->x;
            float direction_y = boss_velocity->y;
            const float direction_length = std::sqrt(direction_x * direction_x + direction_y * direction_y);
            if (direction_length > 0.0001f) {
                direction_x /= direction_length;
                direction_y /= direction_length;
            } else {
                direction_x = 1.0f;
                direction_y = 0.0f;
            }
            constexpr float diagonal = 0.70710678f;
            const std::array<battle::ecs::Position, 8> directions{
                {
                    {direction_x, direction_y},
                    {(direction_x - direction_y) * diagonal, (direction_x + direction_y) * diagonal},
                    {(direction_x + direction_y) * diagonal, (direction_y - direction_x) * diagonal},
                    {-direction_y, direction_x},
                    {direction_y, -direction_x},
                    {(-direction_x - direction_y) * diagonal, (direction_x - direction_y) * diagonal},
                    {(-direction_x + direction_y) * diagonal, (-direction_x - direction_y) * diagonal},
                    {-direction_x, -direction_y},
                }
            };
            const auto candidates = world.spatial_index().query_circle(boss_transform->position, boss_collider->radius);
            for (const auto player : candidates) {
                if (!world.registry().has<battle::ecs::PlayerController>(player)) {
                    continue;
                }
                auto* player_transform = world.registry().try_get<battle::ecs::Transform>(player);
                const auto* player_collider = world.registry().try_get<battle::ecs::Collider>(player);
                if (!player_transform || !player_collider ||
                    !is_overlap(boss_transform->position, player_transform->position, boss_collider->radius,
                                player_collider->radius)) {
                    continue;
                }
                const float separation_distance = boss_collider->radius + player_collider->radius + 0.01f;
                for (const auto direction : directions) {
                    const battle::ecs::Position separated_position{
                        .x = boss_transform->position.x + direction.x * separation_distance,
                        .y = boss_transform->position.y + direction.y * separation_distance,
                    };
                    if (!can_place_character(world, player, separated_position, *player_collider, boss)) {
                        continue;
                    }
                    player_transform->position = separated_position;
                    world.spatial_index().update(player, separated_position, player_collider->radius);
                    break;
                }
            }
        }
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
        if (collider && is_character(*collider) && !is_boss_dash(world, entity)) {
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
                    const bool already_overlapping = is_overlap(transform->position, candidate_transform->position,
                                                                collider->radius, candidate_collider->radius);
                    if (!already_overlapping ||
                        distance_squared(new_position, candidate_transform->position) <=
                        distance_squared(transform->position, candidate_transform->position)) {
                        can_move = false;
                        break;
                    }
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
    separate_players_from_dashing_bosses(world);
}
