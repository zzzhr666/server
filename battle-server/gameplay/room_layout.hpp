#pragma once

#include <cstdint>
#include <string>
#include <vector>

#include "ecs/world.hpp"

namespace battle {
    /// @brief 房间出口在布局中的稳定标识。
    using RoomDoorID = std::uint32_t;

    /// @brief 描述一个出口在房间空间中的位置。
    struct RoomDoor {
        RoomDoorID door_id{};
        ecs::Position position{};
    };

    /// @brief 描述房间的静态边界、出生点和出口位置。
    struct RoomLayout {
        std::string layout_id;
        ecs::WorldBounds bounds{};
        std::vector<ecs::Position> player_spawn_points;
        std::vector<ecs::Position> monster_spawn_points;
        std::vector<RoomDoor> doors;
    };
}
