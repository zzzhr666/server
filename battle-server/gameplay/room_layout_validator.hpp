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
    };

    struct RoomLayoutIssue {
        RoomLayoutIssueKind kind{};
        std::optional<RoomDoorID> door_id{};
        std::optional<std::size_t> point_index{};
    };


    [[nodiscard]] std::vector<RoomLayoutIssue> validate_room_layout(const RoomLayout& layout);
}
