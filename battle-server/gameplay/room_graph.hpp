#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace battle {
    /// @brief 局内房间图中稳定的房间节点标识。
    using DungeonRoomID = std::uint32_t;

    /// @brief 区分房间在一局地下城流程中的玩法职责。
    enum class DungeonRoomKind : std::uint8_t {
        Start,
        Combat,
        Elite,
        Reward,
        Boss,
    };

    /// @brief 描述一个玩法房间及其允许前往的后继房间。
    struct DungeonRoomNode {
        DungeonRoomID room_id{};
        DungeonRoomKind kind{DungeonRoomKind::Start};
        std::string layout_id;
        std::vector<DungeonRoomID> next_room_ids;
    };

    /// @brief 保存一局地下城的起始节点和全部有向房间节点。
    struct DungeonRoomGraph {
        DungeonRoomID start_room_id{};
        std::vector<DungeonRoomNode> rooms;

        /// @brief 按稳定 ID 返回图中的只读房间节点，未找到时返回 nullptr。
        [[nodiscard]] const DungeonRoomNode* find_room(DungeonRoomID room_id) const;
    };
}
