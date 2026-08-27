#pragma once

#include <optional>
#include <vector>

#include "room_layout.hpp"

namespace battle {
    enum class RoomLayoutIssueKind : std::uint8_t {
        EmptyLayoutID,
        InvalidBounds,
        DuplicateDoorID,
        PlayerSpawnOutsideBounds,
        MonsterSpawnOutsideBounds,
        DoorOutsideBounds,
        DuplicateObstacleID,
        InvalidObstacleRadius,
        ObstacleOutsideBounds,
        DuplicateTrapID,
        InvalidTrapRadius,
        TrapOutsideBounds,
        TrapOverlapsObstacle,
    };

    struct RoomLayoutIssue {
        RoomLayoutIssueKind kind{};
        std::optional<RoomDoorID> door_id{};
        std::optional<std::size_t> point_index{};
    };


    /// @brief 校验房间布局的出生点、障碍物、陷阱与边界约束。
    [[nodiscard]] std::vector<RoomLayoutIssue> validate_room_layout(const RoomLayout& layout);
}
