#include "room_layout_validator.hpp"

#include <cmath>
#include <unordered_set>


namespace {
    bool point_in_bounds(const battle::ecs::Position& position, const battle::ecs::WorldBounds& world_bounds) {
        return position.x >= world_bounds.min_x && position.x <= world_bounds.max_x &&
            position.y >= world_bounds.min_y && position.y <= world_bounds.max_y;
    }

    void validate_layout_id(const std::string& layout_id, std::vector<battle::RoomLayoutIssue>& issues) {
        if (layout_id.empty()) {
            issues.emplace_back(battle::RoomLayoutIssueKind::EmptyLayoutID);
        }
    }

    void validate_world_bounds(const battle::ecs::WorldBounds& world_bounds,
                               std::vector<battle::RoomLayoutIssue>& issues) {
        if (world_bounds.min_x > world_bounds.max_x || world_bounds.min_y > world_bounds.max_y) {
            issues.emplace_back(battle::RoomLayoutIssueKind::InvalidBounds);
        }
    }

    void validate_doors(const std::vector<battle::RoomDoor>& doors, const battle::ecs::WorldBounds& world_bounds,
                        std::vector<battle::RoomLayoutIssue>& issues) {
        std::unordered_set<battle::RoomDoorID> seen_doors;
        std::unordered_set<battle::RoomDoorID> reported_duplicate_doors;
        for (const auto [door_id, position] : doors) {
            if (!seen_doors.insert(door_id).second && reported_duplicate_doors.insert(door_id).second) {
                issues.emplace_back(battle::RoomLayoutIssueKind::DuplicateDoorID,
                                    std::make_optional(door_id));
            }
            if (!point_in_bounds(position, world_bounds)) {
                issues.emplace_back(battle::RoomLayoutIssueKind::DoorOutsideBounds, std::make_optional(door_id));
            }
        }
    }

    void validate_spawn_points(const std::vector<battle::ecs::Position>& spawn_points,
                               const battle::ecs::WorldBounds& world_bounds,
                               bool is_player,
                               std::vector<battle::RoomLayoutIssue>& issues) {
        for (std::size_t point_index = 0; point_index < spawn_points.size(); ++point_index) {
            if (const auto& spawn_point = spawn_points[point_index]; !point_in_bounds(spawn_point, world_bounds)) {
                issues.emplace_back(
                    is_player
                        ? battle::RoomLayoutIssueKind::PlayerSpawnOutsideBounds
                        : battle::RoomLayoutIssueKind::MonsterSpawnOutsideBounds,
                    std::nullopt,
                    std::make_optional(point_index));
            }
        }
    }

    bool in_boundary(battle::ecs::Position center, float radius, const battle::ecs::WorldBounds& world_bounds) {
        return center.x >= world_bounds.min_x + radius && center.x <= world_bounds.max_x - radius &&
            center.y >= world_bounds.min_y + radius && center.y <= world_bounds.max_y - radius;
    }

    void validate_obstacles(const std::vector<battle::RoomObstacle>& obstacles,
                            const battle::ecs::WorldBounds& world_bounds,
                            std::vector<battle::RoomLayoutIssue>& issues) {
        std::unordered_set<battle::RoomObstacleID> seen_obstacles;
        std::unordered_set<battle::RoomObstacleID> reported_duplicate_obstacles;
        for (std::size_t index = 0; index < obstacles.size(); ++index) {
            const auto& obstacle = obstacles[index];
            if (!seen_obstacles.insert(obstacle.obstacle_id).second && reported_duplicate_obstacles.insert(
                obstacle.obstacle_id).second) {
                issues.emplace_back(battle::RoomLayoutIssueKind::DuplicateObstacleID, std::nullopt,
                                    std::make_optional(index));
            }
            if (!std::isfinite(obstacle.radius) || obstacle.radius <= 0.0f) {
                issues.emplace_back(battle::RoomLayoutIssueKind::InvalidObstacleRadius,
                                    std::nullopt, std::make_optional(index));
                continue;
            }
            if (!in_boundary(obstacle.center, obstacle.radius, world_bounds)) {
                issues.emplace_back(battle::RoomLayoutIssueKind::ObstacleOutsideBounds,
                                    std::nullopt, std::make_optional(index));
            }
        }
    }

    void validate_traps(const std::vector<battle::RoomTrap>& traps,
                        const std::vector<battle::RoomObstacle>& obstacles,
                        const battle::ecs::WorldBounds& world_bounds,
                        std::vector<battle::RoomLayoutIssue>& issues) {
        std::unordered_set<battle::RoomTrapID> seen_traps;
        std::unordered_set<battle::RoomTrapID> reported_duplicate_traps;
        for (std::size_t index = 0; index < traps.size(); ++index) {
            const auto& trap = traps[index];
            if (!seen_traps.insert(trap.trap_id).second && reported_duplicate_traps.insert(trap.trap_id).second) {
                issues.emplace_back(battle::RoomLayoutIssueKind::DuplicateTrapID, std::nullopt,
                                    std::make_optional(index));
            }
            if (!std::isfinite(trap.radius) || trap.radius <= 0.0f) {
                issues.emplace_back(battle::RoomLayoutIssueKind::InvalidTrapRadius,
                                    std::nullopt, std::make_optional(index));
                continue;
            }
            if (!in_boundary(trap.center, trap.radius, world_bounds)) {
                issues.emplace_back(battle::RoomLayoutIssueKind::TrapOutsideBounds,
                                    std::nullopt, std::make_optional(index));
            }
            for (const auto& obstacle : obstacles) {
                if (!std::isfinite(obstacle.radius) || obstacle.radius <= 0.0f) {
                    continue;
                }
                const float delta_x = trap.center.x - obstacle.center.x;
                const float delta_y = trap.center.y - obstacle.center.y;
                const float radius_sum = trap.radius + obstacle.radius;
                if (delta_x * delta_x + delta_y * delta_y < radius_sum * radius_sum) {
                    issues.emplace_back(battle::RoomLayoutIssueKind::TrapOverlapsObstacle,
                                        std::nullopt, std::make_optional(index));
                    break;
                }
            }
        }
    }
}


std::vector<battle::RoomLayoutIssue> battle::validate_room_layout(const RoomLayout& layout) {
    std::vector<RoomLayoutIssue> issues;
    validate_layout_id(layout.layout_id, issues);
    validate_world_bounds(layout.bounds, issues);
    validate_doors(layout.doors, layout.bounds, issues);
    validate_spawn_points(layout.player_spawn_points, layout.bounds, true, issues);
    validate_spawn_points(layout.monster_spawn_points, layout.bounds, false, issues);
    validate_obstacles(layout.obstacles, layout.bounds, issues);
    validate_traps(layout.traps, layout.obstacles, layout.bounds, issues);
    return issues;
}
