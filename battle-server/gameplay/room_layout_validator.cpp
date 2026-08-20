#include "room_layout_validator.hpp"

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
}


std::vector<battle::RoomLayoutIssue> battle::validate_room_layout(const RoomLayout& layout) {
    std::vector<RoomLayoutIssue> issues;
    validate_layout_id(layout.layout_id, issues);
    validate_world_bounds(layout.bounds, issues);
    validate_doors(layout.doors, layout.bounds, issues);
    validate_spawn_points(layout.player_spawn_points, layout.bounds,
                          true, issues);
    validate_spawn_points(layout.monster_spawn_points, layout.bounds,
                          false, issues);
    return issues;
}
