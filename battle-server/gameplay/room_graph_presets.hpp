#pragma once

#include "room_graph.hpp"

namespace battle {
    /// @brief 返回用于测试和 Demo 的固定线性地下城房间图。
    [[nodiscard]] DungeonRoomGraph default_dungeon_room_graph();
}
