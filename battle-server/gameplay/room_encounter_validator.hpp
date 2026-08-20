#pragma once

#include <cstddef>
#include <cstdint>
#include <optional>
#include <vector>

#include "room_graph.hpp"

namespace battle {
    /// @brief 标识房间遭遇配置违反的规则。
    enum class RoomEncounterIssueKind : std::uint8_t {
        EncounterMissingForCombatRoom,
        EncounterMissingForBossRoom,
        EncounterUnexpectedForStartRoom,
        EncounterUnexpectedForRewardRoom,
        EmptyMonsterGroup,
        DuplicateMonsterKindGroup,
    };

    /// @brief 描述一个房间遭遇配置问题。
    struct RoomEncounterIssue {
        RoomEncounterIssueKind kind{};
        std::optional<DungeonRoomID> room_id{};
        std::optional<std::size_t> group_index{};
    };

    /// @brief 返回房间图中所有遭遇配置问题。
    [[nodiscard]] std::vector<RoomEncounterIssue> validate_room_encounters(
        const DungeonRoomGraph& graph);
}
