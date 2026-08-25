#include "monster_ai_system.hpp"

#include <cmath>
#include <limits>
#include <utility>

#include "ecs/world.hpp"

void battle::ecs::monster_ai_system(World& world, DeltaTime) {
    for (auto entity : world.registry().pool<MonsterController>().entities()) {
        auto* transform = world.registry().try_get<Transform>(entity);
        auto* velocity = world.registry().try_get<Velocity>(entity);
        const auto* stats = world.registry().try_get<CharacterStats>(entity);
        const auto* identity = world.registry().try_get<MonsterIdentity>(entity);
        const auto* attack = world.registry().try_get<AttackDefinition>(entity);
        auto* attack_request = world.registry().try_get<AttackRequest>(entity);
        const auto* collider = world.registry().try_get<Collider>(entity);
        auto* path_following = world.registry().try_get<PathFollowing>(entity);
        if (!transform || !velocity || !stats || !identity || !collider || !path_following) {
            continue;
        }
        if (attack_request) {
            attack_request->requested = false;
        }
        if (identity->kind == MonsterKind::Boss) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            path_following->target = NullEntity;
            path_following->waypoints.clear();
            path_following->current_waypoint = 0;
            continue;
        }
        float movement_multiplier = 1.0f;

        if (const auto status_effects = world.registry().try_get<StatusEffects>(entity)) {
            if (status_effects->freeze.has_value()) {
                velocity->x = 0.0f;
                velocity->y = 0.0f;
                continue;
            }
            if (status_effects->swamp.has_value()) {
                movement_multiplier *= status_effects->swamp.value().movement_multiplier;
            }
        }
        const auto* attack_state = world.registry().try_get<AttackState>(entity);
        if (attack_state && attack_state->phase != AttackPhase::Idle) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            continue;
        }

        const auto x = transform->position.x;
        const auto y = transform->position.y;
        auto nearest_distance_squared = std::numeric_limits<float>::max();
        Position target_position{};
        bool has_target = false;
        Entity target_entity{NullEntity};
        for (auto players_entity : world.registry().pool<PlayerController>().entities()) {
            const auto* player_transform = world.registry().try_get<Transform>(players_entity);
            if (!player_transform) {
                continue;
            }
            const float distance_squared = battle::ecs::distance_squared(
                player_transform->position, transform->position);
            if (nearest_distance_squared > distance_squared) {
                nearest_distance_squared = distance_squared;
                target_position = player_transform->position;
                target_entity = players_entity;
                has_target = true;
            }
        }

        if (!has_target) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            path_following->target = NullEntity;
            path_following->waypoints.clear();
            path_following->current_waypoint = 0;
            continue;
        }
        const auto delta_x = target_position.x - x;
        const auto delta_y = target_position.y - y;
        const auto target_distance_squared = distance_squared(target_position, transform->position);
        const auto distance = std::sqrt(target_distance_squared);
        const float attack_range = attack ? attack->range : 0.0f;
        if (distance == 0.0f) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            path_following->target = NullEntity;
            path_following->waypoints.clear();
            path_following->current_waypoint = 0;
            continue;
        }
        const float direction_x = delta_x / distance;
        const float direction_y = delta_y / distance;

        if (const auto* kiting_ai = world.registry().try_get<KitingAI>(entity); kiting_ai && distance < kiting_ai->
            retreat_distance) {
            velocity->x = -direction_x * stats->move_speed * movement_multiplier;
            velocity->y = -direction_y * stats->move_speed * movement_multiplier;
            continue;
        }
        transform->direction = {.x = direction_x, .y = direction_y};
        if (attack_request && distance <= attack_range) {
            velocity->x = 0.0f;
            velocity->y = 0.0f;
            attack_request->requested = true;
            continue;
        }

        constexpr auto refind_distance = gameplay_config::monster::PathFollowingRefindDistance;
        const auto target_moved_squared = distance_squared(target_position, path_following->target_position);
        const bool path_finished = path_following->waypoints.empty() ||
            path_following->current_waypoint >= path_following->waypoints.size();
        if (path_following->target != target_entity || path_finished ||
            target_moved_squared > refind_distance * refind_distance) {
            auto path_info = world.navigation_grid().find_path(transform->position, target_position, collider->radius);
            if (!path_info.has_value()) {
                velocity->x = 0.0f;
                velocity->y = 0.0f;
                path_following->target = NullEntity;
                path_following->waypoints.clear();
                path_following->current_waypoint = 0;
                continue;
            }
            path_following->waypoints = std::move(path_info.value());
            path_following->current_waypoint = path_following->waypoints.size() > 1 ? 1 : 0;
            path_following->target = target_entity;
            path_following->target_position = target_position;
        }

        if (path_following->current_waypoint < path_following->waypoints.size()) {
            constexpr auto reach_distance = gameplay_config::monster::PathFollowingWaypointReachDistance;
            constexpr auto reach_distance_squared = reach_distance * reach_distance;
            while (path_following->current_waypoint < path_following->waypoints.size() &&
                distance_squared(path_following->waypoints[path_following->current_waypoint],
                                 transform->position) < reach_distance_squared) {
                ++path_following->current_waypoint;
            }
            if (path_following->current_waypoint >= path_following->waypoints.size()) {
                velocity->x = 0.0f;
                velocity->y = 0.0f;
                path_following->target = NullEntity;
                path_following->waypoints.clear();
                path_following->current_waypoint = 0;
                continue;
            }
            const auto& current_target = path_following->waypoints[path_following->current_waypoint];
            const auto waypoint_delta_x = current_target.x - transform->position.x;
            const auto waypoint_delta_y = current_target.y - transform->position.y;
            const auto waypoint_distance = std::sqrt(distance_squared(current_target, transform->position));
            velocity->x = waypoint_delta_x / waypoint_distance * stats->move_speed * movement_multiplier;
            velocity->y = waypoint_delta_y / waypoint_distance * stats->move_speed * movement_multiplier;
            continue;
        }

        velocity->x = delta_x / distance * stats->move_speed * movement_multiplier;
        velocity->y = delta_y / distance * stats->move_speed * movement_multiplier;
    }
}
