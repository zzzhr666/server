#pragma once

#include <cstdint>
#include <optional>
#include <vector>

#include "room_graph.hpp"

namespace battle {
    /// @brief 标识房间图配置违反的结构规则。
    enum class DungeonRoomGraphIssueKind : std::uint8_t {
        DuplicateRoomID,
        StartRoomNotFound,
        StartRoomKindMismatch,
        ExitRoomNotFound,
        SelfLoop,
        DuplicateExit,
        CycleDetected,
        BossHasExit,
        BossRoomNotFound,
        RoomUnreachableFromStart,
    };

    /// @brief 描述一个与具体房间节点关联的校验问题。
    struct DungeonRoomGraphIssue {
        DungeonRoomGraphIssueKind kind{};
        std::optional<DungeonRoomID> room_id{};
        /// @brief 边相关问题关联的目标房间；节点自身问题不填写。
        std::optional<DungeonRoomID> target_room_id{};
    };

    /// @brief 按校验规则和房间配置顺序返回房间图中的全部结构问题。
    [[nodiscard]] std::vector<DungeonRoomGraphIssue> validate_room_graph(const DungeonRoomGraph& graph);
}
